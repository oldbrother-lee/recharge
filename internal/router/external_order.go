package router

import (
	"recharge-go/configs"
	"recharge-go/internal/controller"
	"recharge-go/internal/middleware"
	"recharge-go/internal/repository"
	notificationRepo "recharge-go/internal/repository/notification"
	"recharge-go/internal/service"
	"recharge-go/internal/signature"
	"recharge-go/pkg/lock"
	"recharge-go/pkg/logger"
	"recharge-go/pkg/queue"
	"recharge-go/pkg/redis"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterExternalOrderRoutes 注册外部订单相关路由
func RegisterExternalOrderRoutes(r *gin.RouterGroup, db *gorm.DB) {
	// 创建仓库
	orderRepo := repository.NewOrderRepository(db)
	platformRepo := repository.NewPlatformRepository(db)
	callbackLogRepo := repository.NewCallbackLogRepository(db)
	notificationRepo := notificationRepo.NewRepository(db)
	queueInstance := queue.NewRedisQueue()
	balanceLogRepo := repository.NewBalanceLogRepository(db)
	platformAccountRepo := repository.NewPlatformAccountRepository(db)
	userRepo := repository.NewUserRepository(db)
	platformAPIRepo := repository.NewPlatformAPIRepository(db)
	productAPIRelationRepo := repository.NewProductAPIRelationRepository(db)
	platformAPIParamRepo := repository.NewPlatformAPIParamRepository(db)
	retryRepo := repository.NewRetryRepository(db)
	productRepo := repository.NewProductRepository(db)
	// 创建外部API密钥仓库
	apiKeyRepo := repository.NewExternalAPIKeyRepository(db)

	// 创建余额服务
	platformAccountBalanceService := service.NewPlatformAccountBalanceService(
		db,
		platformAccountRepo,
		userRepo,
		balanceLogRepo,
	)

	balanceService := service.NewBalanceService(balanceLogRepo, userRepo)

	// 创建手机查询服务
	phoneQueryService := service.NewPhoneQueryService(configs.GetConfig())
	
	// 创建余额查询记录仓库
	balanceQueryRecordRepo := repository.NewBalanceQueryRecordRepository(db)
	
	// 创建系统配置服务
	systemConfigRepo := repository.NewSystemConfigRepository(db)
	systemConfigService := service.NewSystemConfigService(systemConfigRepo)

	// 创建订单异常服务
	orderExceptionRepo := repository.NewOrderExceptionRepository(db)
	orderExceptionService := service.NewOrderExceptionService(orderExceptionRepo, orderRepo, logger.Log)

	// 创建统一订单服务（暂时不传入retryService）
	unifiedOrderService := service.NewUnifiedOrderService(
		orderRepo,
		balanceQueryRecordRepo,
		phoneQueryService,
		balanceService, // 使用 balanceService 替代 userBalanceService
		notificationRepo,
		queueInstance,
		db,
		logger.Log,
		systemConfigService,
		productRepo,
		nil, // retryService 稍后设置
		orderExceptionService,
	)
	
	// 先创建充值服务（因为订单服务需要它）
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
		platformAccountBalanceService,
		balanceService,
		phoneQueryService, // 添加手机查询服务
		balanceQueryRecordRepo, // 添加余额查询记录仓库
		unifiedOrderService, // 添加统一订单服务
		systemConfigService, // 添加系统配置服务
		notificationRepo,
		queueInstance,
	)

	// 创建分布式锁管理器
	distributedLock := lock.NewRedisDistributedLock(redis.GetClient())
	refundLockManager := lock.NewRefundLockManager(distributedLock)

	// 创建统一退款服务
	unifiedRefundService := service.NewUnifiedRefundService(
		db,
		userRepo,
		orderRepo,
		balanceLogRepo,
		refundLockManager,
		balanceService,
		platformAccountBalanceService,
	)

	// 创建授信服务
	creditLogRepo := repository.NewCreditLogRepository(db)
	creditService := service.NewCreditService(userRepo, creditLogRepo)

	// 创建订单服务
	orderService := service.NewOrderService(
		orderRepo,
		balanceLogRepo,
		userRepo,
		rechargeService,
		unifiedRefundService,
		refundLockManager,
		notificationRepo,
		queueInstance,
		db,
		productRepo,
		creditService,
	)

	// 设置相互依赖
	rechargeService.SetOrderService(orderService)

	// 创建重试服务
	retryService := service.NewRetryService(
		retryRepo,
		orderRepo,
		platformRepo,
		productRepo,
		productAPIRelationRepo,
		rechargeService,
		orderService,
	)

	// 创建认证中间件
	authMiddleware := middleware.NewExternalAuthMiddleware(apiKeyRepo)

	// 创建商品服务
	productService := service.NewProductService(productRepo)

	// 创建外部订单日志repository
	externalOrderLogRepo := repository.NewExternalOrderLogRepository(db)

	// 创建控制器
	externalOrderController := controller.NewExternalOrderController(orderService, productService, externalOrderLogRepo)
	// 创建统一订单处理服务
	// 统一订单服务已在上面创建
	
	// 创建签名验证器（使用基础处理器）
	signValidator := signature.NewBaseSignatureHandler(&signature.Config{})
	
	// 重用之前创建的队列实例
	
	externalCallbackController := controller.NewExternalCallbackController(orderService, unifiedOrderService, apiKeyRepo, externalOrderLogRepo, signValidator, retryService, productRepo, queueInstance)
	externalRefundController := controller.NewExternalRefundController(orderService)

	// 注册外部订单API路由（需要认证）
	externalOrder := r.Group("/external/order")
	externalOrder.Use(authMiddleware.ExternalAuth())
	{
		externalOrder.POST("", externalOrderController.CreateOrder)
		externalOrder.GET("/query", externalOrderController.GetOrder)
		externalOrder.POST("/refund", externalRefundController.ProcessRefund)
	}

	// 注册回调路由（不需要认证中间件，但需要签名验证）
	externalCallback := r.Group("/external/callback")
	{
		externalCallback.POST("/order", externalCallbackController.HandleCallback)
	}
}
