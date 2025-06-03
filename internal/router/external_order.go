package router

import (
	"recharge-go/internal/controller"
	"recharge-go/internal/repository"
	notificationRepo "recharge-go/internal/repository/notification"
	"recharge-go/internal/service"
	"recharge-go/pkg/database"
	"recharge-go/pkg/queue"

	"github.com/gin-gonic/gin"
)

// RegisterExternalOrderRoutes 注册外部订单相关路由
func RegisterExternalOrderRoutes(r *gin.RouterGroup) {
	// 创建服务实例
	orderRepo := repository.NewOrderRepository(database.DB)
	platformRepo := repository.NewPlatformRepository(database.DB)
	callbackLogRepo := repository.NewCallbackLogRepository(database.DB)
	// manager := recharge.NewManager(database.DB)

	// 创建通知仓库
	notificationRepo := notificationRepo.NewRepository(database.DB)

	// 创建队列实例
	queueInstance := queue.NewRedisQueue()

	// 创建订单服务
	orderService := service.NewOrderService(
		orderRepo,
		nil, // 先传入 nil，后面再设置
		notificationRepo,
		queueInstance,
	)

	// 初始化余额服务
	platformAccountRepo := repository.NewPlatformAccountRepository(database.DB)
	userRepo := repository.NewUserRepository(database.DB)
	balanceLogRepo := repository.NewBalanceLogRepository(database.DB)
	balanceService := service.NewPlatformAccountBalanceService(
		database.DB,
		platformAccountRepo,
		userRepo,
		balanceLogRepo,
	)
	platformAPIRepo := repository.NewPlatformAPIRepository(database.DB)
	productAPIRelationRepo := repository.NewProductAPIRelationRepository(database.DB)
	platformAPIParamRepo := repository.NewPlatformAPIParamRepository(database.DB)
	retryRepo := repository.NewRetryRepository(database.DB)
	productRepo := repository.NewProductRepository(database.DB)

	// 创建充值服务
	rechargeService := service.NewRechargeService(
		database.DB,
		orderRepo,
		platformRepo,
		platformAPIRepo,
		retryRepo,
		callbackLogRepo,
		productAPIRelationRepo,
		productRepo,
		platformAPIParamRepo,
		balanceService,
		notificationRepo,
		queueInstance,
	)

	// 设置 orderService 的 rechargeService
	orderService.SetRechargeService(rechargeService)

	// 创建控制器
	externalOrderController := controller.NewExternalOrderController(orderService)

	// 注册路由
	externalOrder := r.Group("/external/order")
	{
		externalOrder.POST("", externalOrderController.CreateOrder)
		externalOrder.GET("/:id", externalOrderController.GetOrder)
	}
}
