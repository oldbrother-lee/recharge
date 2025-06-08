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

// RegisterKekebangOrderRoutes 注册可客帮订单相关路由
func RegisterKekebangOrderRoutes(r *gin.RouterGroup) {
	// 初始化仓库
	orderRepo := repository.NewOrderRepository(database.DB)
	platformRepo := repository.NewPlatformRepository(database.DB)
	callbackLogRepo := repository.NewCallbackLogRepository(database.DB)

	// 创建通知仓库
	notificationRepo := notificationRepo.NewRepository(database.DB)

	// 创建队列实例
	queueInstance := queue.NewRedisQueue()

	// 初始化repository
	platformAccountRepo := repository.NewPlatformAccountRepository(database.DB)
	userRepo := repository.NewUserRepository(database.DB)
	balanceLogRepo := repository.NewBalanceLogRepository(database.DB)
	productRepo := repository.NewProductRepository(database.DB)

	// 创建订单服务
	orderService := service.NewOrderService(
		orderRepo,
		nil, // 先传入 nil，后面再设置
		notificationRepo,
		queueInstance,
		balanceLogRepo,
		userRepo,
		productRepo,
	)

	// 初始化充值服务
	balanceService := service.NewPlatformAccountBalanceService(
		database.DB,
		platformAccountRepo,
		userRepo,
		balanceLogRepo,
	)

	userBalanceService := service.NewBalanceService(balanceLogRepo, userRepo)

	platformAPIRepo := repository.NewPlatformAPIRepository(database.DB)
	productAPIRelationRepo := repository.NewProductAPIRelationRepository(database.DB)
	platformAPIParamRepo := repository.NewPlatformAPIParamRepository(database.DB)
	retryRepo := repository.NewRetryRepository(database.DB)

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
		userBalanceService,
		notificationRepo,
		queueInstance,
	)

	// 设置 orderService 的 rechargeService
	orderService.SetRechargeService(rechargeService)

	// 创建控制器
	kekebangOrderController := controller.NewKekebangOrderController(orderService, rechargeService)

	// 注册路由
	kekebangOrder := r.Group("/kekebang/order/:userid")
	{
		kekebangOrder.POST("", kekebangOrderController.CreateOrder)
		kekebangOrder.POST("/query", kekebangOrderController.QueryOrder)
	}
}
