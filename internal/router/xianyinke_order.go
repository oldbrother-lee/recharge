package router

import (
	"recharge-go/configs"
	"recharge-go/internal/controller"
	"recharge-go/internal/repository"
	notificationRepo "recharge-go/internal/repository/notification"
	"recharge-go/internal/service"
	"recharge-go/pkg/database"
    "recharge-go/pkg/lock"
    "recharge-go/pkg/log"
	"recharge-go/pkg/queue"
	"recharge-go/pkg/redis"

	"github.com/gin-gonic/gin"
)

// RegisterXianyinkeOrderRoutes 注册闲赢客订单相关路由
func RegisterXianyinkeOrderRoutes(r *gin.RouterGroup) {
	// 获取数据库连接
	db := database.DB

	// 初始化仓库
	orderRepo := repository.NewOrderRepository(db)
	platformRepo := repository.NewPlatformRepository(db)
	callbackLogRepo := repository.NewCallbackLogRepository(db)

	// 创建通知仓库
	nRepo := notificationRepo.NewRepository(db)

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

	// 创建授信服务
	creditLogRepo := repository.NewCreditLogRepository(db)
	creditService := service.NewCreditService(userRepo, creditLogRepo)

	// 创建余额查询记录仓库
	balanceQueryRecordRepo := repository.NewBalanceQueryRecordRepository(database.DB)

	// 创建订单服务
	orderService := service.NewOrderService(
		orderRepo,
		balanceLogRepo,
		userRepo,
		nil, // 先传入 nil，后面再设置
		unifiedRefundService,
		refundLockManager,
		nRepo,
		queueInstance,
		database.DB,
		productRepo,
		creditService,
		balanceQueryRecordRepo,
	)

	// 初始化充值服务需要的额外仓库
	platformAPIRepo := repository.NewPlatformAPIRepository(database.DB)
	productAPIRelationRepo := repository.NewProductAPIRelationRepository(database.DB)
	platformAPIParamRepo := repository.NewPlatformAPIParamRepository(database.DB)
	retryRepo := repository.NewRetryRepository(database.DB)

	// 创建手机查询服务
	phoneQueryService := service.NewPhoneQueryService(configs.GetConfig())

	// 创建系统配置服务
	systemConfigRepo := repository.NewSystemConfigRepository(database.DB)
	systemConfigService := service.NewSystemConfigService(systemConfigRepo)

	// 创建订单异常服务
	orderExceptionRepo := repository.NewOrderExceptionRepository(database.DB)
    orderExceptionService := service.NewOrderExceptionService(orderExceptionRepo, orderRepo, log.Log)

	// 创建统一订单处理服务（暂时不传入retryService）
	unifiedOrderService := service.NewUnifiedOrderService(
		orderRepo,
		balanceQueryRecordRepo,
		phoneQueryService,
		balanceService,
		nRepo,
		queueInstance,
		database.DB,
        log.Log,
		systemConfigService,
		productRepo,
		nil, // retryService 稍后设置
		orderExceptionService,
	)

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
		phoneQueryService,
		balanceQueryRecordRepo,
		unifiedOrderService,
		systemConfigService,
		nRepo,
		queueInstance,
	)

	// 设置 orderService 的 rechargeService
	orderService.SetRechargeService(rechargeService)

	// 创建控制器
	xianyinkeOrderController := controller.NewXianyinkeOrderController(orderService, rechargeService)

	// 注册路由
	xyk := r.Group("/xianyinke/order/:userid")
	{
		xyk.POST("", xianyinkeOrderController.CreateOrder)
	}
}
