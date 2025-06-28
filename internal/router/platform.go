package router

import (
	"recharge-go/internal/controller"
	"recharge-go/internal/middleware"
	"recharge-go/internal/service"

	"github.com/gin-gonic/gin"
)

func RegisterPlatformRoutes(r *gin.RouterGroup, platformController *controller.PlatformController, userService *service.UserService) {
	platforms := r.Group("/platform")
	{
		// 需要管理员权限的路由
		admin := platforms.Group("")
		admin.Use(middleware.CheckSuperAdmin(userService))
		{
			admin.GET("/list", platformController.ListPlatforms)
			admin.POST("", platformController.CreatePlatform)
			admin.PUT("/:id", platformController.UpdatePlatform)
			admin.DELETE("/:id", platformController.DeletePlatform)
			admin.POST("/account", platformController.CreatePlatformAccount)
			admin.PUT("/account/:id", platformController.UpdatePlatformAccount)
			admin.DELETE("/account/:id", platformController.DeletePlatformAccount)
		}

		// 公共路由
		platforms.GET("/:id", platformController.GetPlatform)
		platforms.GET("/accounts/:id", platformController.GetPlatformAccount)
	}

	// 话费帮充接口路由
	platform := r.Group("/platform/xianzhuanxia")
	{
		platform.GET("/channels", platformController.GetChannelList)
	}

	// 蜜蜂平台接口路由
	bee := r.Group("/platform/bee")
	{
		bee.GET("/accounts/:accountId/products", platformController.GetBeeProductList)
		bee.PUT("/accounts/:accountId/products/price", platformController.UpdateBeeProductPrice)
		bee.PUT("/accounts/:accountId/products/province", platformController.UpdateBeeProductProvince)
	}
}
