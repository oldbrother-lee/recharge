package router

import (
	"recharge-go/internal/controller"
	"recharge-go/internal/handler"
	"recharge-go/internal/repository"
	notificationRepo "recharge-go/internal/repository/notification"
	"recharge-go/internal/service"
	"recharge-go/internal/service/platform"
	"recharge-go/pkg/database"
	"recharge-go/pkg/queue"
	"time"

	"github.com/gin-gonic/gin"
)

// RegisterTaskRoutes 依赖注入 platformSvc
func RegisterTaskRoutes(r *gin.RouterGroup, platformSvc *platform.Service) {
	db := database.DB
	taskConfigRepo := repository.NewTaskConfigRepository(db)
	taskOrderRepo := repository.NewTaskOrderRepository(db)
	daichongOrderRepo := repository.NewDaichongOrderRepository(db)
	orderRepo := repository.NewOrderRepository(db)
	notificationRepo := notificationRepo.NewRepository(db)
	var queueInstance queue.Queue = queue.NewRedisQueue()

	// 创建充值服务
	platformRepo := repository.NewPlatformRepository(db)
	platformAPIRepo := repository.NewPlatformAPIRepository(db)
	productAPIRelationRepo := repository.NewProductAPIRelationRepository(db)
	platformAPIParamRepo := repository.NewPlatformAPIParamRepository(db)
	retryRepo := repository.NewRetryRepository(db)
	callbackLogRepo := repository.NewCallbackLogRepository(db)
	platformAccountRepo := repository.NewPlatformAccountRepository(db)
	userRepo := repository.NewUserRepository(db)
	balanceLogRepo := repository.NewBalanceLogRepository(db)
	balanceService := service.NewPlatformAccountBalanceService(
		db,
		platformAccountRepo,
		userRepo,
		balanceLogRepo,
	)

	userBalanceService := service.NewBalanceService(balanceLogRepo, userRepo)

	productRepo := repository.NewProductRepository(db)
	rechargeService := service.NewRechargeService(
		db,
		orderRepo,
		platformRepo,
		platformAPIRepo,
		retryRepo,
		callbackLogRepo,
		productAPIRelationRepo,
		productRepo,
		platformAPIParamRepo,
		balanceService,
		userBalanceService,
		notificationRepo,
		queueInstance,
	)

	orderService := service.NewOrderService(
		orderRepo,
		rechargeService,
		notificationRepo,
		queueInstance,
		balanceLogRepo,
		userRepo,
		productRepo,
	)

	taskConfig := &service.TaskConfig{
		Interval:      5 * time.Minute, // 每5分钟执行一次
		MaxRetries:    3,               // 最大重试3次
		RetryDelay:    1 * time.Minute, // 重试间隔1分钟
		MaxConcurrent: 5,               // 最大并发5个任务
		APIKey:        "",              // API密钥
		UserID:        "",              // 用户ID
		BaseURL:       "",              // API基础URL
	}

	taskOrderHandler := handler.NewTaskOrderHandler(taskOrderRepo)
	taskConfigService := service.NewTaskConfigService(taskConfigRepo)
	taskConfigController := controller.NewTaskConfigController(taskConfigService)
	taskSvc := service.NewTaskService(
		taskConfigRepo,
		taskOrderRepo,
		orderRepo,
		daichongOrderRepo,
		platformSvc,
		orderService,
		taskConfig,
		platformAccountRepo,
	)

	// 启动自动取单任务
	taskSvc.StartTask()

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
