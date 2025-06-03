package router

import (
	"recharge-go/internal/controller"
	"recharge-go/internal/middleware"
	"recharge-go/internal/repository"
	"recharge-go/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterBalanceRoutes 注册余额相关接口
func RegisterBalanceRoutes(r *gin.RouterGroup, db *gorm.DB, userRepo *repository.UserRepository, userService *service.UserService) {
	balanceLogRepo := repository.NewBalanceLogRepository(db)
	balanceService := service.NewBalanceService(balanceLogRepo, userRepo)
	balanceController := controller.NewBalanceController(balanceService)

	api := r.Group("/balance", middleware.CheckSuperAdmin(userService))
	{
		api.POST("/recharge", balanceController.Recharge)
		api.POST("/deduct", balanceController.Deduct)
		api.GET("/logs", balanceController.ListLogs)
	}
}
