package router

import (
	"recharge-go/internal/controller"
	"recharge-go/internal/middleware"
	"recharge-go/internal/service"

	"github.com/gin-gonic/gin"
)

// RegisterPhoneLocationRoutes registers phone location related routes
func RegisterPhoneLocationRoutes(r *gin.RouterGroup, phoneLocationController *controller.PhoneLocationController, userService *service.UserService) {
	phoneLocations := r.Group("/phone-locations")
	{
		// 需要管理员权限的路由
		admin := phoneLocations.Group("")
		admin.Use(middleware.CheckSuperAdmin(userService))
		{
			admin.POST("", phoneLocationController.Create)
			admin.PUT("/:id", phoneLocationController.Update)
			admin.DELETE("/:id", phoneLocationController.Delete)
		}

		// 公开路由
		phoneLocations.GET("", phoneLocationController.List)
	}
}
