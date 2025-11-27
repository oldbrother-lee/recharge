package router

import (
	"recharge-go/internal/controller"
	"recharge-go/internal/middleware"
	"recharge-go/internal/service"

	"github.com/gin-gonic/gin"
)

// RegisterOrderExceptionRoutes 注册订单异常路由
func RegisterOrderExceptionRoutes(r *gin.RouterGroup, orderExceptionController *controller.OrderExceptionController, userService *service.UserService) {

	// 订单异常管理路由组 - 需要管理员权限
	exceptions := r.Group("/order-exceptions")
	exceptions.Use(middleware.Auth())
	exceptions.Use(middleware.CheckSuperAdmin(userService))
	{
		// 获取异常列表
		exceptions.GET("", orderExceptionController.List)

		// 获取异常详情
		// exceptions.GET("/:id", orderExceptionController.GetByID)

		// 更新异常状态
		exceptions.PUT("/:id/status", orderExceptionController.UpdateStatus)

		// 获取待处理异常数量
		exceptions.GET("/pending-count", orderExceptionController.GetPendingCount)

		// 获取异常统计信息
		exceptions.GET("/statistics", orderExceptionController.GetStatistics)
	}

	// 代理/认证用户可访问的异常统计路由（不强制管理员权限）
	userExceptions := r.Group("/order-exceptions")
	userExceptions.Use(middleware.Auth())
	{
		// 获取当前用户或指定用户（仅管理员可指定）的异常统计信息
		userExceptions.GET("/user-statistics", orderExceptionController.GetStatisticsByUser)
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
