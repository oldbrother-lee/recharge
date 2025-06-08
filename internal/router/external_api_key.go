package router

import (
	"recharge-go/internal/controller"
	"recharge-go/internal/service"

	"github.com/gin-gonic/gin"
)

// RegisterExternalAPIKeyRoutes 注册API密钥管理路由
func RegisterExternalAPIKeyRoutes(r *gin.RouterGroup, controller *controller.ExternalAPIKeyController, userService *service.UserService) {
	// API密钥管理路由组 - 需要用户认证
	apiKeys := r.Group("/external-api-keys")
	{
		// 创建API密钥
		apiKeys.POST("", controller.CreateAPIKey)

		// 获取我的API密钥列表
		apiKeys.GET("/my", controller.GetMyAPIKeys)

		// 重新生成API密钥
		apiKeys.POST("/:id/regenerate", controller.RegenerateAPIKey)

		// 更新API密钥状态
		apiKeys.PUT("/:id/status", controller.UpdateAPIKeyStatus)
	}
}
