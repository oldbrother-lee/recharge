package router

import (
	"recharge-go/internal/controller"

	"github.com/gin-gonic/gin"
)

// RegisterXianyinkeOrderRoutes 注册闲赢客订单相关路由
func RegisterXianyinkeOrderRoutes(r *gin.RouterGroup, ctrl *controller.XianyinkeOrderController) {
	xyk := r.Group("/xianyinke/order/:userid")
	{
		xyk.POST("", ctrl.CreateOrder)
	}
}
