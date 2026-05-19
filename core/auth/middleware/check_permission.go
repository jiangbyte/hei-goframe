package middleware

import (
	"strings"

	"hei-gin/core/auth"

	"github.com/gin-gonic/gin"
)

// HeiCheckPermission returns a middleware that checks the user has the required permissions.
// mode defaults to "AND" (all permissions required). Pass "OR" for any permission.
// This middleware is for BUSINESS login type.
func HeiCheckPermission(permissions []string, mode ...string) gin.HandlerFunc {
	m := "AND"
	if len(mode) > 0 {
		m = mode[0]
	}
	return heiCheckPermissionInner("BUSINESS", permissions, m)
}

// HeiClientCheckPermission returns a middleware that checks the CONSUMER user has the required permissions.
// mode defaults to "AND" (all permissions required). Pass "OR" for any permission.
func HeiClientCheckPermission(permissions []string, mode ...string) gin.HandlerFunc {
	m := "AND"
	if len(mode) > 0 {
		m = mode[0]
	}
	return heiCheckPermissionInner("CONSUMER", permissions, m)
}

// heiCheckPermissionInner is a shared implementation for both BUSINESS and CONSUMER permission checks.
func heiCheckPermissionInner(loginType string, permissions []string, mode string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Register permission for auto-discovery scan
		for _, p := range permissions {
			auth.RegisterPermission(auth.PermissionEntry{
				Code:   p,
				Module: getModuleFromCode(p),
				Name:   "",
			})
		}

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

		// Check permission
		if mode == "OR" {
			if !auth.HasPermissionOr(c, loginType, permissions...) {
				c.Abort()
				c.JSON(200, gin.H{"code": 403, "message": "缺少权限: " + strings.Join(permissions, ","), "success": false})
				return
			}
		} else {
			if !auth.HasPermissionAnd(c, loginType, permissions...) {
				c.Abort()
				c.JSON(200, gin.H{"code": 403, "message": "缺少权限: " + strings.Join(permissions, ","), "success": false})
				return
			}
		}
		c.Next()
	}
}

// getModuleFromCode extracts the module segment from a permission code.
// Example: "user:add" -> "user", "sys:user:view" -> "sys:user"
func getModuleFromCode(code string) string {
	parts := strings.Split(code, ":")
	if len(parts) > 1 {
		return strings.Join(parts[:len(parts)-1], ":")
	}
	return code
}
