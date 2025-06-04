package router

import (
	"recharge-go/internal/controller"
	"recharge-go/internal/middleware"
	"recharge-go/internal/service"

	"github.com/gin-gonic/gin"
)

// RegisterUserRoutes 注册用户相关路由
func RegisterUserRoutes(r *gin.RouterGroup, userController *controller.UserController, userService *service.UserService, userLogController *controller.UserLogController) {
	// Public routes - register directly to parent group
	r.POST("/user/login", userController.Login)
	r.POST("/user/register", userController.Register)
	r.POST("/user/refresh-token", userController.RefreshToken)
}

// RegisterProtectedUserRoutes 注册需要认证的用户路由
func RegisterProtectedUserRoutes(r *gin.RouterGroup, userController *controller.UserController, userService *service.UserService, userLogController *controller.UserLogController) {
	// User profile routes
	user := r.Group("/user")
	{
		user.GET("/profile", userController.GetProfile)
		user.PUT("/profile", userController.UpdateProfile)
		user.POST("/change-password", userController.ChangePassword)
		user.GET("/info", userController.GetUserInfo)
		user.GET("/list", userController.GetUserList)
		user.PUT("/status", userController.UpdateUserStatus)
		user.GET("/grade", userController.GetUserGrade)
		user.GET("/tags", userController.GetUserTags)
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
		users.POST("/:id/roles", userController.AssignRoles)
		users.GET("/:id/roles", userController.GetUserRoles)
		users.DELETE("/:id/roles/:role_id", userController.RemoveRole)
	}

	// User log routes
	userLogs := r.Group("/user-logs")
	userLogs.Use(middleware.Auth(), middleware.CheckSuperAdmin(userService))
	{
		userLogs.POST("", userLogController.CreateLog)
		userLogs.GET("/:id", userLogController.GetLogByID)
		userLogs.GET("", userLogController.ListLogs)
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
