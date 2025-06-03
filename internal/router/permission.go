package router

import (
	"recharge-go/internal/controller"
	"recharge-go/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterPermissionRoutes(r *gin.RouterGroup, permissionController *controller.PermissionController) {
	// Protected routes
	auth := r.Group("/permissions")
	auth.Use(middleware.Auth())
	{
		auth.POST("", permissionController.Create)
		auth.PUT("/:id", permissionController.Update)
		auth.DELETE("/:id", permissionController.Delete)
		auth.GET("", permissionController.GetAllPermissions)
		auth.GET("/tree", permissionController.GetTree)
		auth.GET("/menus", permissionController.GetMenuPermissions)
		auth.GET("/buttons", permissionController.GetButtonPermissions)
	}
}
