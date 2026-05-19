package middleware

import (
	"strings"

	"hei-gin/core/auth"

	"github.com/gin-gonic/gin"
)

// HeiCheckRole returns a middleware that checks the user has the required roles.
// mode defaults to "AND" (all roles required). Pass "OR" for any role.
// This middleware is for BUSINESS login type.
func HeiCheckRole(roles []string, mode ...string) gin.HandlerFunc {
	m := "AND"
	if len(mode) > 0 {
		m = mode[0]
	}
	return heiCheckRoleInner("BUSINESS", roles, m)
}

// HeiClientCheckRole returns a middleware that checks the CONSUMER user has the required roles.
// mode defaults to "AND" (all roles required). Pass "OR" for any role.
func HeiClientCheckRole(roles []string, mode ...string) gin.HandlerFunc {
	m := "AND"
	if len(mode) > 0 {
		m = mode[0]
	}
	return heiCheckRoleInner("CONSUMER", roles, m)
}

// heiCheckRoleInner is a shared implementation for both BUSINESS and CONSUMER role checks.
func heiCheckRoleInner(loginType string, roles []string, mode string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check login first
		var isLogin bool
		if loginType == "CONSUMER" {
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

		// Check role
		if mode == "OR" {
			if !auth.HasRoleOr(c, loginType, roles...) {
				c.Abort()
				c.JSON(200, gin.H{"code": 403, "message": "缺少角色: " + strings.Join(roles, ","), "success": false})
				return
			}
		} else {
			if !auth.HasRoleAnd(c, loginType, roles...) {
				c.Abort()
				c.JSON(200, gin.H{"code": 403, "message": "缺少角色: " + strings.Join(roles, ","), "success": false})
				return
			}
		}
		c.Next()
	}
}
