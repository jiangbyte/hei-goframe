package middleware

import (
	"hei-gin/core/auth"

	"github.com/gin-gonic/gin"
)

// HeiCheckLogin returns a middleware that checks if the user is logged in.
// loginType defaults to "BUSINESS". Pass "CONSUMER" for client-side users.
func HeiCheckLogin(loginType ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		lt := "BUSINESS"
		if len(loginType) > 0 {
			lt = loginType[0]
		}
		var isLogin bool
		if lt == "CONSUMER" {
			tool := &auth.HeiClientAuthTool{}
			isLogin = tool.IsLogin(c)
		} else {
			isLogin = auth.IsLogin(c)
		}
		if !isLogin {
			c.Abort()
			c.JSON(200, gin.H{"code": 401, "message": "未授权/未登录", "success": false})
			return
		}
		c.Next()
	}
}

// HeiClientCheckLogin returns a middleware that checks if the CONSUMER user is logged in.
func HeiClientCheckLogin() gin.HandlerFunc {
	return HeiCheckLogin("CONSUMER")
}
