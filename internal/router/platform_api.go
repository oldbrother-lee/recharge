package router

import (
	"recharge-go/internal/controller"
	"recharge-go/internal/service"

	"github.com/gin-gonic/gin"
)

// RegisterPlatformAPIRoutes 注册平台接口路由
func RegisterPlatformAPIRoutes(r *gin.RouterGroup, controller *controller.PlatformAPIController, userService *service.UserService) {
	apis := r.Group("/platform/api")
	{
		apis.POST("", controller.CreateAPI)
		apis.PUT("/:id", controller.UpdateAPI)
		apis.DELETE("/:id", controller.DeleteAPI)
		apis.GET("/:id", controller.GetAPI)
		apis.GET("", controller.ListAPIs)
	}
}
