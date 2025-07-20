package router

import (
	"recharge-go/internal/controller"
	"recharge-go/internal/middleware"
	"recharge-go/internal/service"

	"github.com/gin-gonic/gin"
)

// RegisterPhoneQueryRoutes 注册手机查询相关路由
func RegisterPhoneQueryRoutes(r *gin.RouterGroup, phoneQueryController *controller.PhoneQueryController, userService *service.UserService) {
	phoneQuery := r.Group("/phone")
	{
		// 需要管理员权限的路由
		admin := phoneQuery.Group("")
		admin.Use(middleware.CheckSuperAdmin(userService))
		{
			// 余额查询接口
			admin.POST("/balance", phoneQueryController.QueryBalance)

			// 缴费记录查询接口
			admin.POST("/payment-records", phoneQueryController.QueryPaymentRecords)
		}
	}
}

// RegisterPublicPhoneQueryRoutes 注册公开的手机查询路由（如果需要对外提供API）
func RegisterPublicPhoneQueryRoutes(r *gin.RouterGroup, phoneQueryController *controller.PhoneQueryController) {
	// 公开的手机查询接口（暂时不使用认证中间件）
	public := r.Group("/public/phone")
	// TODO: 如需要外部API认证，需要传入数据库连接并创建 ExternalAuthMiddleware
	// authMiddleware := middleware.NewExternalAuthMiddleware(apiKeyRepo)
	// public.Use(authMiddleware.ExternalAuth())
	{
		// 余额查询接口
		public.POST("/balance", phoneQueryController.QueryBalance)

		// 缴费记录查询接口
		public.POST("/payment-records", phoneQueryController.QueryPaymentRecords)
	}
}