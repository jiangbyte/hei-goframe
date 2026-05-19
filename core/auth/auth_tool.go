package auth

import "github.com/gin-gonic/gin"

var businessAuth = newBaseAuthTool("BUSINESS")

func Init(expire int, tokenName string)   { businessAuth.Init(expire, tokenName) }
func GetLoginType() string                { return businessAuth.GetLoginType() }
func GetTokenName() string                { return businessAuth.GetTokenName() }
func GetTokenValue(c *gin.Context) string { return businessAuth.GetTokenValue(c) }
func Login(c *gin.Context, id string, extra map[string]any) (string, error) {
	return businessAuth.Login(c, id, extra)
}
func Logout(c *gin.Context, loginID ...string)    { businessAuth.Logout(c, loginID...) }
func Kickout(loginID string)                      { businessAuth.Kickout(loginID) }
func KickoutToken(loginID, token string)          { businessAuth.KickoutToken(loginID, token) }
func IsLogin(c *gin.Context) bool                 { return businessAuth.IsLogin(c) }
func CheckLogin(c *gin.Context) error             { return businessAuth.CheckLogin(c) }
func GetLoginID(c *gin.Context) string            { return businessAuth.GetLoginID(c) }
func GetLoginIDDefaultNull(c *gin.Context) string { return businessAuth.GetLoginIDDefaultNull(c) }
func GetLoginIDByToken(token string) string       { return businessAuth.GetLoginIDByToken(token) }
func GetTokenInfo(c *gin.Context) map[string]any  { return businessAuth.GetTokenInfo(c) }
func GetExtra(c *gin.Context, key string) any     { return businessAuth.GetExtra(c, key) }
func GetSession(c *gin.Context) map[string]any    { return businessAuth.GetSession(c) }
func RenewTimeout(c *gin.Context, timeout ...int) { businessAuth.RenewTimeout(c, timeout...) }
func GetTokenTimeout(c *gin.Context) int          { return businessAuth.GetTokenTimeout(c) }
func GetSessionTimeout(c *gin.Context) int        { return businessAuth.GetSessionTimeout(c) }
func GetTokenValueByLoginID(loginID string) string {
	return businessAuth.GetTokenValueByLoginID(loginID)
}
func GetTokenValuesByLoginID(loginID string) []string {
	return businessAuth.GetTokenValuesByLoginID(loginID)
}
func Disable(loginID string, timeSeconds int) { businessAuth.Disable(loginID, timeSeconds) }
func IsDisable(loginID string) bool           { return businessAuth.IsDisable(loginID) }
func CheckDisable(loginID string) error       { return businessAuth.CheckDisable(loginID) }
func GetDisableTime(loginID string) int       { return businessAuth.GetDisableTime(loginID) }
func UntieDisable(loginID string)             { businessAuth.UntieDisable(loginID) }
