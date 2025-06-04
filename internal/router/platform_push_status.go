package router

import (
	"recharge-go/internal/controller"

	"github.com/gin-gonic/gin"
)

// RegisterPlatformPushStatusRoutes 注册平台推单状态相关路由
func RegisterPlatformPushStatusRoutes(r *gin.RouterGroup, controller *controller.PlatformPushStatusController) {
	pushStatus := r.Group("/platform/push-status")
	{
		pushStatus.GET("/:account_id", controller.GetPushStatus)
		pushStatus.PUT("/:account_id", controller.UpdatePushStatus)
	}
}
