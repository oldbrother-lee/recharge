package router

import (
	"recharge-go/internal/controller"

	"github.com/gin-gonic/gin"
)

// RegisterSystemConfigRoutes 注册系统配置路由
func RegisterSystemConfigRoutes(r *gin.RouterGroup, systemConfigController *controller.SystemConfigController) {
	// 系统配置路由组
	systemConfig := r.Group("/system-config")
	{
		// CRUD 操作
		systemConfig.POST("", systemConfigController.Create)
		systemConfig.PUT("/:id", systemConfigController.Update)
		systemConfig.DELETE("/:id", systemConfigController.Delete)
		systemConfig.GET("/:id", systemConfigController.GetByID)
		systemConfig.GET("/key/:key", systemConfigController.GetByKey)
		systemConfig.GET("", systemConfigController.GetList)

		// 批量更新配置
		systemConfig.PUT("/batch", systemConfigController.BatchUpdate)

		// 系统名称相关
		systemConfig.PUT("/system-name", systemConfigController.UpdateSystemName)
		systemConfig.GET("/system-name", systemConfigController.GetSystemName)

		// 系统信息
		systemConfig.GET("/system-info", systemConfigController.GetSystemInfo)
	}

	// 兼容前端现有API的路由组
	systemManage := r.Group("/systemManage")
	{
		// CRUD 操作
		systemManage.POST("", systemConfigController.Create)
		systemManage.PUT("/:id", systemConfigController.Update)
		systemManage.DELETE("/:id", systemConfigController.Delete)
		systemManage.GET("/:id", systemConfigController.GetByID)
		systemManage.GET("/key/:key", systemConfigController.GetByKey)
		systemManage.GET("", systemConfigController.GetList)

		// 系统设置相关
		systemManage.GET("/settings", systemConfigController.GetSystemInfo)
		systemManage.PUT("/settings/system-name", systemConfigController.UpdateSystemName)
		systemManage.PUT("/settings/batch", systemConfigController.BatchUpdate)
	}
}
