package router

import (
    "recharge-go/internal/controller"
    "recharge-go/internal/middleware"
    "recharge-go/internal/service"

    "github.com/gin-gonic/gin"
)

// RegisterBalanceRoutes 注册余额相关接口
func RegisterBalanceRoutes(r *gin.RouterGroup, balanceController *controller.BalanceController, userService *service.UserService) {

    api := r.Group("/balance", middleware.CheckSuperAdmin(userService))
	{
		api.POST("/recharge", balanceController.Recharge)
		api.POST("/deduct", balanceController.Deduct)
		api.GET("/logs", balanceController.ListLogs)
	}
}
