package router

import (
	"recharge-go/internal/controller"
	"recharge-go/internal/service"

	"github.com/gin-gonic/gin"
)

// RegisterPlatformAPIParamRoutes 注册平台接口参数路由
func RegisterPlatformAPIParamRoutes(r *gin.RouterGroup, controller *controller.PlatformAPIParamController, userService *service.UserService) {
	params := r.Group("/platform/api/params")
	{
		params.POST("", controller.CreateParam)
		params.PUT("/:id", controller.UpdateParam)
		params.DELETE("/:id", controller.DeleteParam)
		params.GET("/:id", controller.GetParam)
		params.GET("", controller.ListParams)
	}
}
