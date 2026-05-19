package username_api

import (
	"github.com/gin-gonic/gin"

	"hei-gin/core/auth/middleware"
	"hei-gin/core/log"

	"hei-gin/modules/client/auth/username"
)

// RegisterRoutes registers consumer username-based auth routes (login/register/logout).
func RegisterRoutes(r *gin.Engine) {
	r.POST("/api/v1/public/c/login", username.DoLogin)
	r.POST("/api/v1/public/c/register",
		log.SysLog("注册"),
		middleware.NoRepeat(5000),
		username.DoRegister,
	)
	r.POST("/api/v1/c/logout",
		middleware.HeiCheckLogin("CONSUMER"),
		username.DoLogout,
	)
}
