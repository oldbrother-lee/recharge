package router

import (
	"recharge-go/internal/controller"
	"recharge-go/internal/repository"
	"recharge-go/internal/service"
	"recharge-go/pkg/database"
	"recharge-go/pkg/redis"

	"github.com/gin-gonic/gin"
)

// RegisterDzTaskRoutes 注册得众平台任务配置路由
func RegisterDzTaskRoutes(r *gin.RouterGroup) {
	db := database.DB
	dzTaskConfigRepo := repository.NewDzTaskConfigRepository(db)
	dzTaskConfigService := service.NewDzTaskConfigService(dzTaskConfigRepo)
	
	// 创建TaskConfigNotifier
	redisClient := redis.GetClient()
	taskConfigNotifier := service.NewTaskConfigNotifier(redisClient)
	dzTaskConfigController := controller.NewDzTaskConfigController(dzTaskConfigService, taskConfigNotifier)

	// 得众平台任务配置路由
	dzTaskConfigGroup := r.Group("/dz-task-config")
	{
		dzTaskConfigGroup.POST("", dzTaskConfigController.Create)
		dzTaskConfigGroup.PUT("", dzTaskConfigController.Update)
		dzTaskConfigGroup.DELETE("/:id", dzTaskConfigController.Delete)
		dzTaskConfigGroup.GET("/:id", dzTaskConfigController.Get)
		dzTaskConfigGroup.GET("", dzTaskConfigController.List)
	}
}