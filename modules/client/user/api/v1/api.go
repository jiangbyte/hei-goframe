package v1

import (
	middleware "hei-gin/core/auth/middleware"
	"hei-gin/core/log"
	"hei-gin/core/result"
	clientuser "hei-gin/modules/client/user"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers all client user routes on the given gin engine.
func RegisterRoutes(r *gin.Engine) {
	// GET /api/v1/client-user/page
	r.GET("/api/v1/client-user/page",
		middleware.HeiCheckPermission([]string{"client:user:page"}),
		pageHandler,
	)

	// POST /api/v1/client-user/create
	r.POST("/api/v1/client-user/create",
		middleware.HeiCheckPermission([]string{"client:user:create"}),
		createHandler,
	)

	// POST /api/v1/client-user/modify
	r.POST("/api/v1/client-user/modify",
		middleware.HeiCheckPermission([]string{"client:user:modify"}),
		modifyHandler,
	)

	// POST /api/v1/client-user/remove
	r.POST("/api/v1/client-user/remove",
		middleware.HeiCheckPermission([]string{"client:user:remove"}),
		removeHandler,
	)

	// GET /api/v1/client-user/detail
	r.GET("/api/v1/client-user/detail",
		middleware.HeiCheckPermission([]string{"client:user:detail"}),
		detailHandler,
	)

	// GET /api/v1/client-user/current
	r.GET("/api/v1/client-user/current",
		middleware.HeiCheckLogin("CONSUMER"),
		currentHandler,
	)

	// POST /api/v1/client-user/update-profile
	r.POST("/api/v1/client-user/update-profile",
		middleware.HeiCheckLogin("CONSUMER"),
		log.SysLog("C端用户更新个人信息"),
		middleware.NoRepeat(3000),
		updateProfileHandler,
	)

	// POST /api/v1/client-user/update-avatar
	r.POST("/api/v1/client-user/update-avatar",
		middleware.HeiCheckLogin("CONSUMER"),
		log.SysLog("C端用户更新头像"),
		updateAvatarHandler,
	)

	// POST /api/v1/client-user/update-password
	r.POST("/api/v1/client-user/update-password",
		middleware.HeiCheckLogin("CONSUMER"),
		log.SysLog("C端用户修改密码"),
		middleware.NoRepeat(3000),
		updatePasswordHandler,
	)
}

// pageHandler handles GET /api/v1/client-user/page
func pageHandler(c *gin.Context) {
	var param clientuser.ClientUserPageParam
	if err := c.ShouldBindQuery(&param); err != nil {
		c.JSON(200, result.Failure(c, "参数错误: "+err.Error(), 400, nil))
		return
	}

	data := clientuser.Page(c, &param)
	c.JSON(200, data)
}

// createHandler handles POST /api/v1/client-user/create
func createHandler(c *gin.Context) {
	var param clientuser.ClientUserCreateParam
	if err := c.ShouldBindJSON(&param); err != nil {
		c.JSON(200, result.Failure(c, "参数错误: "+err.Error(), 400, nil))
		return
	}

	clientuser.Create(c, &param)
	c.JSON(200, result.Success(c, nil))
}

// modifyHandler handles POST /api/v1/client-user/modify
func modifyHandler(c *gin.Context) {
	var param clientuser.ClientUserModifyParam
	if err := c.ShouldBindJSON(&param); err != nil {
		c.JSON(200, result.Failure(c, "参数错误: "+err.Error(), 400, nil))
		return
	}

	clientuser.Modify(c, &param)
	c.JSON(200, result.Success(c, nil))
}

// removeHandler handles POST /api/v1/client-user/remove
func removeHandler(c *gin.Context) {
	var param struct {
		IDs []string `json:"ids"`
	}
	if err := c.ShouldBindJSON(&param); err != nil {
		c.JSON(200, result.Failure(c, "参数错误: "+err.Error(), 400, nil))
		return
	}

	clientuser.Remove(c, param.IDs)
	c.JSON(200, result.Success(c, nil))
}

// detailHandler handles GET /api/v1/client-user/detail
func detailHandler(c *gin.Context) {
	id := c.Query("id")
	vo := clientuser.Detail(c, id)
	c.JSON(200, result.Success(c, vo))
}

// currentHandler handles GET /api/v1/client-user/current
func currentHandler(c *gin.Context) {
	vo := clientuser.Current(c)
	c.JSON(200, result.Success(c, vo))
}

// updateProfileHandler handles POST /api/v1/client-user/update-profile
func updateProfileHandler(c *gin.Context) {
	var param clientuser.UpdateProfileParam
	if err := c.ShouldBindJSON(&param); err != nil {
		c.JSON(200, result.Failure(c, "参数错误: "+err.Error(), 400, nil))
		return
	}

	clientuser.UpdateProfile(c, &param)
	c.JSON(200, result.Success(c, nil))
}

// updateAvatarHandler handles POST /api/v1/client-user/update-avatar
func updateAvatarHandler(c *gin.Context) {
	var param clientuser.UpdateAvatarParam
	if err := c.ShouldBindJSON(&param); err != nil {
		c.JSON(200, result.Failure(c, "参数错误: "+err.Error(), 400, nil))
		return
	}

	clientuser.UpdateAvatar(c, &param)
	c.JSON(200, result.Success(c, nil))
}

// updatePasswordHandler handles POST /api/v1/client-user/update-password
func updatePasswordHandler(c *gin.Context) {
	var param clientuser.UpdatePasswordParam
	if err := c.ShouldBindJSON(&param); err != nil {
		c.JSON(200, result.Failure(c, "参数错误: "+err.Error(), 400, nil))
		return
	}

	clientuser.UpdatePassword(c, &param)
	c.JSON(200, result.Success(c, nil))
}
