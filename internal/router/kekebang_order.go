package router

import (
	"recharge-go/internal/controller"
	"recharge-go/internal/repository"
	notificationRepo "recharge-go/internal/repository/notification"
	"recharge-go/internal/service"
	"recharge-go/pkg/database"
	"recharge-go/pkg/lock"
	"recharge-go/pkg/queue"
	"recharge-go/pkg/redis"

	"github.com/gin-gonic/gin"
)

// RegisterKekebangOrderRoutes 注册可客帮订单相关路由
func RegisterKekebangOrderRoutes(r *gin.RouterGroup) {
	// 获取数据库连接
	db := database.DB
	
	// 初始化仓库
	orderRepo := repository.NewOrderRepository(db)
	platformRepo := repository.NewPlatformRepository(db)
	callbackLogRepo := repository.NewCallbackLogRepository(db)

	// 创建通知仓库
	notificationRepo := notificationRepo.NewRepository(db)

	// 创建队列实例
	queueInstance := queue.NewRedisQueue()

	// 初始化repository
	platformAccountRepo := repository.NewPlatformAccountRepository(db)
	userRepo := repository.NewUserRepository(db)
	balanceLogRepo := repository.NewBalanceLogRepository(db)
	productRepo := repository.NewProductRepository(db)

	// 创建余额服务
	balanceService := service.NewBalanceService(balanceLogRepo, userRepo)

	// 创建平台账户余额服务
	platformAccountBalanceService := service.NewPlatformAccountBalanceService(db, platformAccountRepo, userRepo, balanceLogRepo)

	// 创建分布式锁管理器
	distributedLock := lock.NewRedisDistributedLock(redis.GetClient())
	refundLockManager := lock.NewRefundLockManager(distributedLock)

	// 创建统一退款服务
	unifiedRefundService := service.NewUnifiedRefundService(db, userRepo, orderRepo, balanceLogRepo, refundLockManager, balanceService, platformAccountBalanceService)

	// 创建订单服务
	orderService := service.NewOrderService(
		orderRepo,
		balanceLogRepo,
		userRepo,
		nil, // 先传入 nil，后面再设置
		unifiedRefundService,
		refundLockManager,
		notificationRepo,
		queueInstance,
		database.DB,
	)

	// 初始化充值服务需要的额外仓库

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
		platformAccountBalanceService,
		balanceService,
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
