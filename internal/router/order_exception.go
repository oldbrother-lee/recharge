package router

import (
	"recharge-go/internal/controller"
	"recharge-go/internal/middleware"
	"recharge-go/internal/repository"
	"recharge-go/internal/service"
	"recharge-go/pkg/database"
	"recharge-go/pkg/logger"

	"github.com/gin-gonic/gin"
)

// RegisterOrderExceptionRoutes 注册订单异常路由
func RegisterOrderExceptionRoutes(r *gin.RouterGroup, userService *service.UserService) {
	// 初始化依赖
	orderExceptionRepo := repository.NewOrderExceptionRepository(database.DB)
	orderRepo := repository.NewOrderRepository(database.DB)
	orderExceptionService := service.NewOrderExceptionService(orderExceptionRepo, orderRepo, logger.Log)
	orderExceptionController := controller.NewOrderExceptionController(orderExceptionService, logger.Log)

	// 订单异常管理路由组 - 需要管理员权限
	exceptions := r.Group("/order-exceptions")
	exceptions.Use(middleware.Auth())
	exceptions.Use(middleware.CheckSuperAdmin(userService))
	{
		// 获取异常列表
		exceptions.GET("", orderExceptionController.List)

		// 获取异常详情
		exceptions.GET("/:id", orderExceptionController.GetByID)

		// 更新异常状态
		exceptions.PUT("/:id/status", orderExceptionController.UpdateStatus)

		// 获取待处理异常数量
		exceptions.GET("/pending-count", orderExceptionController.GetPendingCount)

		// 获取异常统计信息
		exceptions.GET("/statistics", orderExceptionController.GetStatistics)
	}

	// 订单相关的异常查询路由
	orders := r.Group("/orders")
	orders.Use(middleware.Auth())
	orders.Use(middleware.CheckSuperAdmin(userService))
	{
		// 根据订单ID获取异常记录
		orders.GET("/:order_id/exceptions", orderExceptionController.GetByOrderID)
	}
}