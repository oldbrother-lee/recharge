package router

import (
	"recharge-go/internal/controller"

	"github.com/gin-gonic/gin"
)

// RegisterKekebangOrderRoutes 注册可客帮订单相关路由
func RegisterKekebangOrderRoutes(r *gin.RouterGroup, ctrl *controller.KekebangOrderController) {
	kekebangOrder := r.Group("/kekebang/order/:userid")
	{
		kekebangOrder.POST("", ctrl.CreateOrder)
		kekebangOrder.POST("/query", ctrl.QueryOrder)
	}
}
