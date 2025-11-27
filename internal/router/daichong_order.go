package router

import (
	"recharge-go/internal/controller"
	"recharge-go/internal/repository"
	"recharge-go/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterDaichongOrderRoutes 注册代充订单相关路由
func RegisterDaichongOrderRoutes(r *gin.RouterGroup, db *gorm.DB) {
	// 初始化依赖
	daichongOrderRepo := repository.NewDaichongOrderRepository(db)
	daichongOrderSvc := service.NewDaichongOrderService(daichongOrderRepo)
	daichongOrderCtrl := controller.NewDaichongOrderController(daichongOrderSvc)

	orderGroup := r.Group("/daichong-order")
	{
		orderGroup.POST("", daichongOrderCtrl.Create)
		orderGroup.GET("/:id", daichongOrderCtrl.GetByID)
		orderGroup.PUT("", daichongOrderCtrl.Update)
		orderGroup.DELETE("/:id", daichongOrderCtrl.Delete)
		orderGroup.GET("", daichongOrderCtrl.List)
	}
}
