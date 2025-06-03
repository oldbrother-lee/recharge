package router

import (
	"recharge-go/internal/controller"
	"recharge-go/internal/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterCreditRoutes 注册授信相关路由
func RegisterCreditRoutes(r *gin.RouterGroup, creditController *controller.CreditController) {
	credit := r.Group("/credit")
	credit.Use(middleware.Auth()) // 所有授信相关接口都需要登录
	{
		// 设置授信额度
		credit.POST("/set", creditController.SetCredit)
		// 获取授信日志列表
		credit.GET("/logs", creditController.GetCreditLogs)
		// 获取用户授信统计
		credit.GET("/stats", creditController.GetUserCreditStats)
	}
}
