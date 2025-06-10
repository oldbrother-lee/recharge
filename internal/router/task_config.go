package router

import (
	"github.com/gin-gonic/gin"
)

func RegisterTaskConfigRoutes(r *gin.RouterGroup) {
	// 直接调用task.go中的路由注册，避免重复的依赖初始化
	// 因为TaskConfigController需要复杂的依赖注入
	// 实际的task-config路由已在task.go的RegisterTaskRoutes中注册
	// 这里保持空实现以兼容router_v2.go的调用
}
