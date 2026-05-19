package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"

	"hei-gin/config"
	"hei-gin/core/db"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
)

// baseAuthTool is the shared authentication implementation for both BUSINESS and CONSUMER.
// It is parameterized by loginType and generates Redis key prefixes accordingly.
type baseAuthTool struct {
	expire    int
	tokenName string
	secret    string
	algorithm string
	loginType string
}

func newBaseAuthTool(loginType string) *baseAuthTool {
	t := &baseAuthTool{loginType: loginType}
	t.ensureConfig()
	return t
}

// ensureConfig initializes default values from the global config if not already set.
func (t *baseAuthTool) ensureConfig() {
	if t.secret != "" {
		return
	}
	if config.C == nil {
		return
	}
	t.expire = config.C.JWT.ExpireSeconds
	t.tokenName = config.C.JWT.TokenName
	t.secret = config.C.JWT.SecretKey
	t.algorithm = config.C.JWT.Algorithm
}

func (t *baseAuthTool) tokenURLSafe(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func (t *baseAuthTool) getRedis() *redis.Client {
	return db.Redis
}

func (t *baseAuthTool) getTokenKey(token string) string {
	return "hei:auth:" + t.loginType + ":token:" + token
}

func (t *baseAuthTool) getSessionKey(userID string) string {
	return "hei:auth:" + t.loginType + ":session:" + userID
}

func (t *baseAuthTool) getDisableKey(loginID string) string {
	return "hei:auth:" + t.loginType + ":disable:" + loginID
}

// Init sets custom expire and token name. Falls back to config if values are zero/empty.
func (t *baseAuthTool) Init(expire int, tokenName string) {
	t.ensureConfig()
	if expire > 0 {
		t.expire = expire
	}
	if tokenName != "" {
		t.tokenName = tokenName
	}
}

// GetLoginType returns the login type identifier.
func (t *baseAuthTool) GetLoginType() string {
	t.ensureConfig()
	return t.loginType
}

// GetTokenName returns the HTTP header name used to carry the token.
func (t *baseAuthTool) GetTokenName() string {
	t.ensureConfig()
	return t.tokenName
}

// GetTokenValue extracts the token from the request header.
func (t *baseAuthTool) GetTokenValue(c *gin.Context) string {
	t.ensureConfig()
	if c == nil {
		return ""
	}
	return c.GetHeader(t.tokenName)
}

// Login authenticates a user by user ID, stores token data in Redis, and returns the signed JWT token.
func (t *baseAuthTool) Login(c *gin.Context, id string, extra map[string]any) (string, error) {
	t.ensureConfig()

	now := time.Now()
	jti := t.tokenURLSafe(32)

	claims := jwt.MapClaims{
		"jti": jti,
		"iat": jwt.NewNumericDate(now),
	}
	token := jwt.NewWithClaims(jwt.GetSigningMethod(t.algorithm), claims)
	signedToken, err := token.SignedString([]byte(t.secret))
	if err != nil {
		return "", err
	}

	tokenData := map[string]any{
		"user_id":    id,
		"type":       t.loginType,
		"created_at": now.Format("2006-01-02 15:04:05"),
		"extra":      extra,
	}
	if extra == nil {
		tokenData["extra"] = map[string]any{}
	}

	tokenDataJSON, err := json.Marshal(tokenData)
	if err != nil {
		return "", err
	}

	redisClient := t.getRedis()
	ctx := context.Background()

	err = redisClient.SetEx(ctx, t.getTokenKey(signedToken), tokenDataJSON, time.Duration(t.expire)*time.Second).Err()
	if err != nil {
		return "", err
	}

	sessionKey := t.getSessionKey(id)
	err = redisClient.SAdd(ctx, sessionKey, signedToken).Err()
	if err != nil {
		return "", err
	}
	err = redisClient.Expire(ctx, sessionKey, time.Duration(t.expire)*time.Second).Err()
	if err != nil {
		return "", err
	}

	return signedToken, nil
}

// Logout invalidates the current session. If loginID is provided, it kicks out all sessions for that user.
func (t *baseAuthTool) Logout(c *gin.Context, loginID ...string) {
	t.ensureConfig()

	if len(loginID) > 0 {
		t.Kickout(loginID[0])
		return
	}

	token := t.GetTokenValue(c)
	if token == "" {
		return
	}

	data := t.getTokenData(token)
	if data != nil {
		userID, _ := data["user_id"].(string)
		if userID != "" {
			redisClient := t.getRedis()
			ctx := context.Background()
			sessionKey := t.getSessionKey(userID)
			_ = redisClient.SRem(ctx, sessionKey, token).Err()
		}
	}

	redisClient := t.getRedis()
	ctx := context.Background()
	tokenKey := t.getTokenKey(token)
	_ = redisClient.Del(ctx, tokenKey).Err()
}

// Kickout deletes all tokens and session data for the given login ID.
func (t *baseAuthTool) Kickout(loginID string) {
	t.ensureConfig()

	redisClient := t.getRedis()
	ctx := context.Background()
	sessionKey := t.getSessionKey(loginID)

	tokens, err := redisClient.SMembers(ctx, sessionKey).Result()
	if err != nil {
		return
	}

	for _, token := range tokens {
		tokenKey := t.getTokenKey(token)
		_ = redisClient.Del(ctx, tokenKey).Err()
	}

	_ = redisClient.Del(ctx, sessionKey).Err()
}

// KickoutToken removes a specific token from the user's session set and deletes its data.
func (t *baseAuthTool) KickoutToken(loginID, token string) {
	t.ensureConfig()

	redisClient := t.getRedis()
	ctx := context.Background()
	sessionKey := t.getSessionKey(loginID)
	tokenKey := t.getTokenKey(token)

	_ = redisClient.SRem(ctx, sessionKey, token).Err()
	_ = redisClient.Del(ctx, tokenKey).Err()
}

// IsLogin checks whether the current request carries a valid token.
func (t *baseAuthTool) IsLogin(c *gin.Context) bool {
	loginID := t.GetLoginIDDefaultNull(c)
	return loginID != ""
}

// CheckLogin returns an error if the current request is not authenticated.
func (t *baseAuthTool) CheckLogin(c *gin.Context) error {
	if !t.IsLogin(c) {
		return errors.New("未授权/未登录")
	}
	return nil
}

// GetLoginID extracts and returns the login ID from the current request's token.
func (t *baseAuthTool) GetLoginID(c *gin.Context) string {
	return t.GetLoginIDDefaultNull(c)
}

// GetLoginIDDefaultNull returns the login ID from the token, or empty string if not logged in.
func (t *baseAuthTool) GetLoginIDDefaultNull(c *gin.Context) string {
	token := t.GetTokenValue(c)
	if token == "" {
		return ""
	}
	data := t.decodeToken(token)
	if data == nil {
		return ""
	}
	userID, _ := data["user_id"].(string)
	return userID
}

// GetLoginIDByToken extracts the login ID from the given token value.
func (t *baseAuthTool) GetLoginIDByToken(token string) string {
	if token == "" {
		return ""
	}
	data := t.decodeToken(token)
	if data == nil {
		return ""
	}
	userID, _ := data["user_id"].(string)
	return userID
}

// decodeToken retrieves token data from Redis and verifies the JWT signature.
func (t *baseAuthTool) decodeToken(token string) map[string]any {
	if token == "" {
		return nil
	}

	data := t.getTokenData(token)
	if data == nil {
		return nil
	}

	_, err := jwt.Parse(token, func(tk *jwt.Token) (interface{}, error) {
		if _, ok := tk.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(t.secret), nil
	})
	if err != nil {
		return nil
	}

	return data
}

// getTokenData retrieves the token payload from Redis.
func (t *baseAuthTool) getTokenData(token string) map[string]any {
	if token == "" {
		return nil
	}

	redisClient := t.getRedis()
	ctx := context.Background()
	tokenKey := t.getTokenKey(token)

	data, err := redisClient.Get(ctx, tokenKey).Result()
	if err == redis.Nil {
		return nil
	}
	if err != nil {
		return nil
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		return nil
	}
	return result
}

// GetTokenInfo returns the full token data stored in Redis for the current request.
func (t *baseAuthTool) GetTokenInfo(c *gin.Context) map[string]any {
	token := t.GetTokenValue(c)
	if token == "" {
		return nil
	}
	return t.getTokenData(token)
}

// GetExtra returns a specific extra field from the token data.
func (t *baseAuthTool) GetExtra(c *gin.Context, key string) any {
	data := t.GetTokenInfo(c)
	if data != nil {
		extra, ok := data["extra"].(map[string]any)
		if ok {
			return extra[key]
		}
	}
	return nil
}

// GetSession returns the full token payload for the current request.
func (t *baseAuthTool) GetSession(c *gin.Context) map[string]any {
	token := t.GetTokenValue(c)
	if token == "" {
		return nil
	}
	return t.getTokenData(token)
}

// RenewTimeout extends the token and session timeouts.
func (t *baseAuthTool) RenewTimeout(c *gin.Context, timeout ...int) {
	t.ensureConfig()

	token := t.GetTokenValue(c)
	if token == "" {
		return
	}

	newTimeout := t.expire
	if len(timeout) > 0 && timeout[0] > 0 {
		newTimeout = timeout[0]
	}

	redisClient := t.getRedis()
	ctx := context.Background()
	tokenKey := t.getTokenKey(token)
	_ = redisClient.Expire(ctx, tokenKey, time.Duration(newTimeout)*time.Second).Err()

	loginID := t.GetLoginIDByToken(token)
	if loginID != "" {
		sessionKey := t.getSessionKey(loginID)
		_ = redisClient.Expire(ctx, sessionKey, time.Duration(newTimeout)*time.Second).Err()
	}
}

// GetTokenTimeout returns the remaining TTL (in seconds) of the current token. Returns -1 if not logged in.
func (t *baseAuthTool) GetTokenTimeout(c *gin.Context) int {
	token := t.GetTokenValue(c)
	if token == "" {
		return -1
	}

	redisClient := t.getRedis()
	ctx := context.Background()
	tokenKey := t.getTokenKey(token)

	ttl, err := redisClient.TTL(ctx, tokenKey).Result()
	if err != nil || ttl < 0 {
		return -1
	}
	return int(ttl.Seconds())
}

// GetSessionTimeout returns the remaining TTL (in seconds) of the current session. Returns -1 if not logged in.
func (t *baseAuthTool) GetSessionTimeout(c *gin.Context) int {
	loginID := t.GetLoginIDDefaultNull(c)
	if loginID == "" {
		return -1
	}

	redisClient := t.getRedis()
	ctx := context.Background()
	sessionKey := t.getSessionKey(loginID)

	ttl, err := redisClient.TTL(ctx, sessionKey).Result()
	if err != nil || ttl < 0 {
		return -1
	}
	return int(ttl.Seconds())
}

// GetTokenValueByLoginID returns one token for the given login ID.
func (t *baseAuthTool) GetTokenValueByLoginID(loginID string) string {
	redisClient := t.getRedis()
	ctx := context.Background()
	sessionKey := t.getSessionKey(loginID)

	tokens, err := redisClient.SMembers(ctx, sessionKey).Result()
	if err != nil || len(tokens) == 0 {
		return ""
	}
	return tokens[0]
}

// GetTokenValuesByLoginID returns all tokens for the given login ID.
func (t *baseAuthTool) GetTokenValuesByLoginID(loginID string) []string {
	redisClient := t.getRedis()
	ctx := context.Background()
	sessionKey := t.getSessionKey(loginID)

	tokens, err := redisClient.SMembers(ctx, sessionKey).Result()
	if err != nil {
		return nil
	}
	return tokens
}

// Disable marks a login ID as disabled for the specified duration (in seconds).
func (t *baseAuthTool) Disable(loginID string, timeSeconds int) {
	redisClient := t.getRedis()
	ctx := context.Background()
	disableKey := t.getDisableKey(loginID)
	_ = redisClient.SetEx(ctx, disableKey, "1", time.Duration(timeSeconds)*time.Second).Err()
}

// IsDisable checks whether a login ID is currently disabled.
func (t *baseAuthTool) IsDisable(loginID string) bool {
	redisClient := t.getRedis()
	ctx := context.Background()
	disableKey := t.getDisableKey(loginID)

	exists, err := redisClient.Exists(ctx, disableKey).Result()
	if err != nil {
		return false
	}
	return exists > 0
}

// CheckDisable returns an error if the login ID is currently disabled.
func (t *baseAuthTool) CheckDisable(loginID string) error {
	if t.IsDisable(loginID) {
		return errors.New("账号已被禁用")
	}
	return nil
}

// GetDisableTime returns the remaining disable time (in seconds). Returns -1 if not disabled.
func (t *baseAuthTool) GetDisableTime(loginID string) int {
	redisClient := t.getRedis()
	ctx := context.Background()
	disableKey := t.getDisableKey(loginID)

	ttl, err := redisClient.TTL(ctx, disableKey).Result()
	if err != nil || ttl < 0 {
		return -1
	}
	return int(ttl.Seconds())
}

// UntieDisable removes the disabled status from a login ID.
func (t *baseAuthTool) UntieDisable(loginID string) {
	redisClient := t.getRedis()
	ctx := context.Background()
	disableKey := t.getDisableKey(loginID)
	_ = redisClient.Del(ctx, disableKey).Err()
}
