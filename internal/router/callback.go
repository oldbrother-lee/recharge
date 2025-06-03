package router

import (
	"recharge-go/internal/controller"

	"github.com/gin-gonic/gin"
)

// RegisterCallbackRoutes 注册回调相关路由
func RegisterCallbackRoutes(r *gin.RouterGroup, callbackController *controller.CallbackController) {
	callback := r.Group("/callback")
	{
		callback.POST("/kekebang/:userid", callbackController.HandleKekebangCallback)
		callback.POST("/mishi/:userid", callbackController.HandleMishiCallback)
		callback.POST("/dayuanren/:userid", callbackController.HandleDayuanrenCallback)
	}
}
