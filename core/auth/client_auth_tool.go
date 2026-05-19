package auth

import "github.com/gin-gonic/gin"

var consumerAuth = newBaseAuthTool("CONSUMER")

// HeiClientAuthTool provides authentication utilities for consumer (client) users.
// All methods delegate to the underlying consumerAuth singleton.
type HeiClientAuthTool struct{}

// NewHeiClientAuthTool creates a new HeiClientAuthTool.
func NewHeiClientAuthTool() *HeiClientAuthTool { return &HeiClientAuthTool{} }

func (h *HeiClientAuthTool) Init(expire int, tokenName string) { consumerAuth.Init(expire, tokenName) }
func (h *HeiClientAuthTool) GetLoginType() string              { return consumerAuth.GetLoginType() }
func (h *HeiClientAuthTool) GetTokenName() string              { return consumerAuth.GetTokenName() }
func (h *HeiClientAuthTool) GetTokenValue(c *gin.Context) string {
	return consumerAuth.GetTokenValue(c)
}
func (h *HeiClientAuthTool) Login(c *gin.Context, id string, extra map[string]any) (string, error) {
	return consumerAuth.Login(c, id, extra)
}
func (h *HeiClientAuthTool) Logout(c *gin.Context, loginID ...string) {
	consumerAuth.Logout(c, loginID...)
}
func (h *HeiClientAuthTool) Kickout(loginID string) { consumerAuth.Kickout(loginID) }
func (h *HeiClientAuthTool) KickoutToken(loginID, token string) {
	consumerAuth.KickoutToken(loginID, token)
}
func (h *HeiClientAuthTool) IsLogin(c *gin.Context) bool      { return consumerAuth.IsLogin(c) }
func (h *HeiClientAuthTool) CheckLogin(c *gin.Context) error  { return consumerAuth.CheckLogin(c) }
func (h *HeiClientAuthTool) GetLoginID(c *gin.Context) string { return consumerAuth.GetLoginID(c) }
func (h *HeiClientAuthTool) GetLoginIDDefaultNull(c *gin.Context) string {
	return consumerAuth.GetLoginIDDefaultNull(c)
}
func (h *HeiClientAuthTool) GetLoginIDByToken(token string) string {
	return consumerAuth.GetLoginIDByToken(token)
}
func (h *HeiClientAuthTool) GetTokenInfo(c *gin.Context) map[string]any {
	return consumerAuth.GetTokenInfo(c)
}
func (h *HeiClientAuthTool) GetExtra(c *gin.Context, key string) any {
	return consumerAuth.GetExtra(c, key)
}
func (h *HeiClientAuthTool) GetSession(c *gin.Context) map[string]any {
	return consumerAuth.GetSession(c)
}
func (h *HeiClientAuthTool) RenewTimeout(c *gin.Context, timeout ...int) {
	consumerAuth.RenewTimeout(c, timeout...)
}
func (h *HeiClientAuthTool) GetTokenTimeout(c *gin.Context) int {
	return consumerAuth.GetTokenTimeout(c)
}
func (h *HeiClientAuthTool) GetSessionTimeout(c *gin.Context) int {
	return consumerAuth.GetSessionTimeout(c)
}
func (h *HeiClientAuthTool) GetTokenValueByLoginID(loginID string) string {
	return consumerAuth.GetTokenValueByLoginID(loginID)
}
func (h *HeiClientAuthTool) GetTokenValuesByLoginID(loginID string) []string {
	return consumerAuth.GetTokenValuesByLoginID(loginID)
}
func (h *HeiClientAuthTool) Disable(loginID string, timeSeconds int) {
	consumerAuth.Disable(loginID, timeSeconds)
}
func (h *HeiClientAuthTool) IsDisable(loginID string) bool { return consumerAuth.IsDisable(loginID) }
func (h *HeiClientAuthTool) CheckDisable(loginID string) error {
	return consumerAuth.CheckDisable(loginID)
}
func (h *HeiClientAuthTool) GetDisableTime(loginID string) int {
	return consumerAuth.GetDisableTime(loginID)
}
func (h *HeiClientAuthTool) UntieDisable(loginID string) { consumerAuth.UntieDisable(loginID) }
