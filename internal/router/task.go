package router

import (
	"recharge-go/internal/controller"
	"recharge-go/internal/handler"
	"recharge-go/internal/repository"
	"recharge-go/internal/service"
	"recharge-go/internal/service/platform"
	"recharge-go/pkg/database"
	"recharge-go/pkg/redis"

	"github.com/gin-gonic/gin"
)

// RegisterTaskRoutes 依赖注入 platformSvc
func RegisterTaskRoutes(r *gin.RouterGroup, platformSvc *platform.Service) {
	db := database.DB
	taskConfigRepo := repository.NewTaskConfigRepository(db)
	taskOrderRepo := repository.NewTaskOrderRepository(db)

	taskOrderHandler := handler.NewTaskOrderHandler(taskOrderRepo)
	taskConfigService := service.NewTaskConfigService(taskConfigRepo)
	// 创建TaskConfigNotifier
	redisClient := redis.GetClient()
	taskConfigNotifier := service.NewTaskConfigNotifier(redisClient)
	taskConfigController := controller.NewTaskConfigController(taskConfigService, taskConfigNotifier)

	// 取单任务配置路由
	taskConfigGroup := r.Group("/task-config")
	{
		taskConfigGroup.POST("", taskConfigController.Create)
		taskConfigGroup.PUT("", taskConfigController.Update)
		taskConfigGroup.DELETE("/:id", taskConfigController.Delete)
		taskConfigGroup.GET("/:id", taskConfigController.Get)
		taskConfigGroup.GET("", taskConfigController.List)
	}

	// 取单任务订单路由
	taskOrder := r.Group("/task-order")
	{
		taskOrder.GET("", taskOrderHandler.List)
		taskOrder.GET("/:order_number", taskOrderHandler.GetByOrderNumber)
	}
}

// InitTaskRouter 注册任务相关路由，platformSvc 由 main.go 注入
func InitTaskRouter(r *gin.Engine, platformSvc *platform.Service) {
	// 这里用 platformSvc 注册 controller 或 handler
	// 示例：
	// ctrl := controller.NewTaskController(platformSvc)
	// r.POST("/task/xxx", ctrl.XXX)
}
