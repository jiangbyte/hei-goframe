package session

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"hei-gin/core/auth"
	"hei-gin/core/constants"
	"hei-gin/core/db"
	ent "hei-gin/ent/gen"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// scanKeys uses Redis SCAN (cursor loop, 200 per batch) to find all keys matching pattern.
func scanKeys(ctx context.Context, redis *redis.Client, pattern string) ([]string, error) {
	var cursor uint64
	var keys []string
	for {
		batch, nextCursor, err := redis.Scan(ctx, cursor, pattern, 200).Result()
		if err != nil {
			return nil, err
		}
		keys = append(keys, batch...)
		if nextCursor == 0 {
			break
		}
		cursor = nextCursor
	}
	return keys, nil
}

// Analysis returns an overview analysis of all sessions (BUSINESS + CONSUMER).
func Analysis(c *gin.Context) *SessionAnalysisResult {
	ctx := context.Background()
	bKeys, _ := scanKeys(ctx, db.Redis, constants.SESSION_PREFIX_BUSINESS+"*")
	cKeys, _ := scanKeys(ctx, db.Redis, constants.SESSION_PREFIX_CONSUMER+"*")

	bTotal, bNewly, bMax := countTokens(ctx, db.Redis, bKeys, constants.TOKEN_PREFIX_BUSINESS)
	cTotal, cNewly, cMax := countTokens(ctx, db.Redis, cKeys, constants.TOKEN_PREFIX_CONSUMER)

	maxTokenCount := bMax
	if cMax > maxTokenCount {
		maxTokenCount = cMax
	}

	return &SessionAnalysisResult{
		TotalCount:        bTotal + cTotal,
		MaxTokenCount:     maxTokenCount,
		OneHourNewlyAdded: bNewly + cNewly,
		ProportionOfBAndC: fmt.Sprintf("%d/%d", bTotal, cTotal),
	}
}

// countTokens iterates session keys, SMEMBER to get tokens, GET each token's data,
// and returns total token count, count of tokens created within the last hour,
// and the maximum tokens for a single user.
func countTokens(ctx context.Context, redis *redis.Client, sessionKeys []string, tokenPrefix string) (total, oneHourNewlyAdded, maxPerUser int) {
	userTokenCounts := make(map[string]int)
	oneHourAgo := time.Now().Add(-1 * time.Hour)

	for _, sessionKey := range sessionKeys {
		parts := strings.Split(sessionKey, ":")
		userID := parts[len(parts)-1]

		tokens, err := redis.SMembers(ctx, sessionKey).Result()
		if err != nil {
			continue
		}
		userTokenCounts[userID] = len(tokens)

		for _, token := range tokens {
			total++
			tokenKey := tokenPrefix + token
			data, err := redis.Get(ctx, tokenKey).Result()
			if err != nil {
				continue
			}
			var tokenData map[string]any
			if err := json.Unmarshal([]byte(data), &tokenData); err != nil {
				continue
			}
			createdAtStr, _ := tokenData["created_at"].(string)
			if createdAtStr != "" {
				createdAt, err := time.Parse("2006-01-02 15:04:05", createdAtStr)
				if err == nil && createdAt.After(oneHourAgo) {
					oneHourNewlyAdded++
				}
			}
		}
	}

	for _, count := range userTokenCounts {
		if count > maxPerUser {
			maxPerUser = count
		}
	}
	return
}

// countDaily groups tokens by creation day (format "2006-01-02").
func countDaily(ctx context.Context, redis *redis.Client, sessionKeys []string, tokenPrefix string) map[string]int {
	daily := make(map[string]int)
	for _, sessionKey := range sessionKeys {
		tokens, err := redis.SMembers(ctx, sessionKey).Result()
		if err != nil {
			continue
		}
		for _, token := range tokens {
			tokenKey := tokenPrefix + token
			data, err := redis.Get(ctx, tokenKey).Result()
			if err != nil {
				continue
			}
			var tokenData map[string]any
			if err := json.Unmarshal([]byte(data), &tokenData); err != nil {
				continue
			}
			createdAtStr, _ := tokenData["created_at"].(string)
			if createdAtStr != "" {
				createdAt, err := time.Parse("2006-01-02 15:04:05", createdAtStr)
				if err == nil {
					day := createdAt.Format("2006-01-02")
					daily[day]++
				}
			}
		}
	}
	return daily
}

// Page returns a paginated list of BUSINESS sessions as a full gin.H response (manual pagination).
func Page(c *gin.Context, param *SessionPageParam) gin.H {
	ctx := context.Background()
	sessions, err := collectSessions(ctx, db.Redis, constants.SESSION_PREFIX_BUSINESS, constants.TOKEN_PREFIX_BUSINESS, param.Keyword)
	if err != nil {
		sessions = []*SessionPageResult{}
	}

	// Sort by SessionCreateTime DESC (RFC3339 strings are lexicographically sortable)
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].SessionCreateTime > sessions[j].SessionCreateTime
	})

	total := len(sessions)
	current := param.Current
	if current < 1 {
		current = 1
	}
	size := param.Size
	if size < 1 {
		size = 10
	}

	pages := (total + size - 1) / size
	start := (current - 1) * size
	var pageRecords []*SessionPageResult

	if start >= total {
		pageRecords = []*SessionPageResult{}
	} else {
		end := start + size
		if end > total {
			end = total
		}
		pageRecords = sessions[start:end]
	}

	return gin.H{
		"code": 200, "message": "请求成功", "success": true,
		"data": gin.H{
			"records": pageRecords, "total": total,
			"page": current, "size": size, "pages": pages,
		},
	}
}

// collectSessions scans session keys, enriches with token and user data, and optionally filters by keyword.
func collectSessions(ctx context.Context, redis *redis.Client, sessionPrefix, tokenPrefix, keyword string) ([]*SessionPageResult, error) {
	pattern := sessionPrefix + "*"
	keys, err := scanKeys(ctx, redis, pattern)
	if err != nil {
		return nil, err
	}

	var result []*SessionPageResult
	userCache := make(map[string]*ent.SysUser)

	for _, sessionKey := range keys {
		parts := strings.Split(sessionKey, ":")
		userID := parts[len(parts)-1]

		tokens, err := redis.SMembers(ctx, sessionKey).Result()
		if err != nil {
			continue
		}

		tokenCount := len(tokens)

		// Find the first valid token (one whose data still exists in Redis)
		var sessionCreateTime string
		var username string
		found := false

		for _, token := range tokens {
			tokenKey := tokenPrefix + token
			data, err := redis.Get(ctx, tokenKey).Result()
			if err != nil {
				continue
			}
			var tokenData map[string]any
			if err := json.Unmarshal([]byte(data), &tokenData); err != nil {
				continue
			}
			extra, _ := tokenData["extra"].(map[string]any)
			if extra != nil {
				username, _ = extra["username"].(string)
			}
			sessionCreateTime, _ = tokenData["created_at"].(string)
			found = true
			break
		}

		if !found {
			continue
		}

		// Apply keyword filter against the username from token extra
		if keyword != "" && !strings.Contains(username, keyword) {
			continue
		}

		// Get session TTL
		ttl, err := redis.TTL(ctx, sessionKey).Result()
		timeoutSeconds := -1
		if err == nil {
			timeoutSeconds = int(ttl.Seconds())
		}

		// Query SysUser for enrichment (with cache)
		user, ok := userCache[userID]
		if !ok {
			user, err = db.Client.SysUser.Get(ctx, userID)
			if err != nil {
				user = nil
			}
			userCache[userID] = user
		}

		sr := &SessionPageResult{
			UserID:                userID,
			TokenCount:            tokenCount,
			SessionCreateTime:     sessionCreateTime,
			SessionTimeout:        formatTimeout(timeoutSeconds),
			SessionTimeoutSeconds: timeoutSeconds,
		}

		if username != "" {
			sr.Username = &username
		}

		if user != nil {
			sr.Nickname = user.Nickname
			sr.Avatar = user.Avatar
			sr.Status = user.Status
			sr.LastLoginIP = user.LastLoginIP
			if user.LastLoginAt != nil {
				sr.LastLoginTime = user.LastLoginAt.Format("2006-01-02 15:04:05")
			}
		}

		result = append(result, sr)
	}

	return result, nil
}

// Exit force-logouts all sessions for the given user.
func Exit(c *gin.Context, userID string) {
	auth.Kickout(userID)
}

// TokenList returns all active tokens for a given user.
func TokenList(c *gin.Context, userID string) []*SessionTokenResult {
	ctx := context.Background()
	sessionKey := constants.SESSION_PREFIX_BUSINESS + userID
	tokens, err := db.Redis.SMembers(ctx, sessionKey).Result()
	if err != nil || len(tokens) == 0 {
		return []*SessionTokenResult{}
	}

	var results []*SessionTokenResult
	for _, token := range tokens {
		tokenKey := constants.TOKEN_PREFIX_BUSINESS + token
		data, err := db.Redis.Get(ctx, tokenKey).Result()
		if err != nil {
			continue
		}
		var tokenData map[string]any
		if err := json.Unmarshal([]byte(data), &tokenData); err != nil {
			continue
		}

		createdAt, _ := tokenData["created_at"].(string)
		extra, _ := tokenData["extra"].(map[string]any)
		deviceType, _ := extra["device_type"].(string)
		deviceID, _ := extra["device_id"].(string)

		ttl, err := db.Redis.TTL(ctx, tokenKey).Result()
		timeoutSeconds := -1
		if err == nil {
			timeoutSeconds = int(ttl.Seconds())
		}

		results = append(results, &SessionTokenResult{
			Token:          token,
			CreatedAt:      createdAt,
			Timeout:        formatTimeout(timeoutSeconds),
			TimeoutSeconds: timeoutSeconds,
			DeviceType:     deviceType,
			DeviceID:       deviceID,
		})
	}

	return results
}

// ExitToken force-logouts a specific token for a given user.
func ExitToken(c *gin.Context, userID, token string) {
	auth.KickoutToken(userID, token)
}

// ChartData returns bar chart data (last 7 days daily new tokens) and pie chart data (B vs C total).
func ChartData(c *gin.Context) *SessionChartData {
	ctx := context.Background()
	bKeys, _ := scanKeys(ctx, db.Redis, constants.SESSION_PREFIX_BUSINESS+"*")
	cKeys, _ := scanKeys(ctx, db.Redis, constants.SESSION_PREFIX_CONSUMER+"*")

	bTotal, _, _ := countTokens(ctx, db.Redis, bKeys, constants.TOKEN_PREFIX_BUSINESS)
	cTotal, _, _ := countTokens(ctx, db.Redis, cKeys, constants.TOKEN_PREFIX_CONSUMER)

	bDaily := countDaily(ctx, db.Redis, bKeys, constants.TOKEN_PREFIX_BUSINESS)
	cDaily := countDaily(ctx, db.Redis, cKeys, constants.TOKEN_PREFIX_CONSUMER)

	days := lastNDays(7)
	series := make([]int, 7)
	for i, day := range days {
		series[i] = bDaily[day] + cDaily[day]
	}

	return &SessionChartData{
		BarChart: BarChartData{
			Days: days,
			Series: []CategorySeries{
				{Name: "新增在线数", Data: series},
			},
		},
		PieChart: PieChartData{
			Data: []CategoryTotal{
				{Category: "BUSINESS", Total: bTotal},
				{Category: "CONSUMER", Total: cTotal},
			},
		},
	}
}

// formatTimeout converts TTL seconds to a human-readable Chinese string.
func formatTimeout(seconds int) string {
	if seconds < 0 {
		return "已过期"
	}
	if seconds == 0 {
		return "永久"
	}
	if seconds < 60 {
		return fmt.Sprintf("剩余 %d秒", seconds)
	}
	if seconds < 3600 {
		return fmt.Sprintf("剩余 %d分钟", seconds/60)
	}
	if seconds < 86400 {
		return fmt.Sprintf("剩余 %d小时%d分钟", seconds/3600, (seconds%3600)/60)
	}
	return fmt.Sprintf("剩余 %d天%d小时", seconds/86400, (seconds%86400)/3600)
}

// lastNDays returns the last n calendar days in "2006-01-02" format (oldest first).
func lastNDays(n int) []string {
	days := make([]string, n)
	now := time.Now()
	for i := 0; i < n; i++ {
		days[i] = now.AddDate(0, 0, -(n - 1 - i)).Format("2006-01-02")
	}
	return days
}
