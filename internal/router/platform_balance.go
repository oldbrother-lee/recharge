package router

import (
	"recharge-go/internal/controller"
	"recharge-go/internal/middleware"
	"recharge-go/internal/service"
	"recharge-go/internal/service/recharge"
	"recharge-go/pkg/database"

	"github.com/gin-gonic/gin"
)

// RegisterPlatformBalanceRoutes 注册平台余额相关路由
func RegisterPlatformBalanceRoutes(r *gin.RouterGroup, userService *service.UserService) {
	// 创建平台管理器
	platformManager := recharge.NewManager(database.DB)
	if err := platformManager.LoadPlatforms(); err != nil {
		panic(err)
	}

	// 创建控制器
	platformBalanceController := controller.NewPlatformBalanceController(platformManager)

	// 注册路由
	balance := r.Group("/platform-balance")
	balance.Use(middleware.Auth(), middleware.CheckSuperAdmin(userService))
	{
		balance.GET("/:platform", platformBalanceController.QueryBalance)
	}
}
