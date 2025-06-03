package router

import (
	"recharge-go/internal/controller"
	"recharge-go/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoleRoutes(r *gin.RouterGroup, roleController *controller.RoleController) {
	// Protected routes
	auth := r.Group("/roles")
	auth.Use(middleware.Auth())
	{
		auth.POST("", roleController.Create)
		auth.PUT("/:id", roleController.Update)
		auth.DELETE("/:id", roleController.Delete)
		auth.GET("/:id", roleController.GetByID)
		auth.GET("", roleController.List)
		auth.GET("/all", roleController.GetAll)
		auth.POST("/:id/permissions/:permission_id", roleController.AddPermission)
		auth.DELETE("/:id/permissions/:permission_id", roleController.RemovePermission)
	}
}
