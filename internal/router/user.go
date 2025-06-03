package router

import (
	"recharge-go/internal/controller"
	"recharge-go/internal/middleware"
	"recharge-go/internal/service"

	"github.com/gin-gonic/gin"
)

// RegisterUserRoutes 注册用户相关路由
func RegisterUserRoutes(r *gin.RouterGroup, userController *controller.UserController, userService *service.UserService, userLogController *controller.UserLogController) {
	// Public routes
	public := r.Group("/public")
	{
		public.POST("/login", userController.Login)
		public.POST("/register", userController.Register)
		public.POST("/refresh-token", userController.RefreshToken)
	}

	// User profile routes
	user := r.Group("/user")
	user.Use(middleware.Auth())
	{
		user.GET("/profile", userController.GetProfile)
		user.PUT("/profile", userController.UpdateProfile)
		user.PUT("/password", userController.ChangePassword)
	}

	// User management routes (admin only)
	users := r.Group("/users")
	users.Use(middleware.Auth(), middleware.CheckSuperAdmin(userService))
	{
		users.GET("", userController.ListUsers)
		users.POST("", userController.CreateUser)
		users.GET("/:id", userController.GetUser)
		users.PUT("/:id", userController.UpdateUser)
		users.DELETE("/:id", userController.DeleteUser)
		users.PUT("/:id/status", userController.UpdateUserStatus)
		users.POST("/:id/reset-password", userController.ResetPassword)
		users.POST(":id/roles", userController.AssignRoles)
		users.GET(":id/roles", userController.GetUserRoles)
		users.DELETE(":id/roles/:role_id", userController.RemoveRole)
	}

	// User log routes
	logs := r.Group("/user-logs")
	logs.Use(middleware.Auth(), middleware.CheckSuperAdmin(userService))
	{
		logs.POST("", userLogController.CreateLog)
		logs.GET("/:id", userLogController.GetLogByID)
		logs.GET("", userLogController.ListLogs)
	}

	// User tag routes
	tags := r.Group("/user-tags")
	tags.Use(middleware.Auth(), middleware.CheckSuperAdmin(userService))
	{
		tags.GET("", userController.GetUserTags)
		tags.POST("", userController.AssignUserTag)
		tags.DELETE("/:tag_id", userController.RemoveUserTag)
	}
}
