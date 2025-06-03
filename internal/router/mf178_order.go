package router

import (
	"recharge-go/internal/controller"
	"recharge-go/internal/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterMF178OrderRoutes 注册MF178订单相关路由
func RegisterMF178OrderRoutes(r *gin.RouterGroup, mf178OrderController *controller.MF178OrderController) {
	mf178Order := r.Group("/mf178/order/:userid", middleware.MF178Auth())
	{
		mf178Order.POST("", mf178OrderController.CreateOrder)
		mf178Order.POST("/query", mf178OrderController.QueryOrder)
	}
}
