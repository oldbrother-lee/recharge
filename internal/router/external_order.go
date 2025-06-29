package router

import (
	"recharge-go/internal/controller"
	"recharge-go/internal/middleware"
	"recharge-go/internal/repository"
	notificationRepo "recharge-go/internal/repository/notification"
	"recharge-go/internal/service"
	"recharge-go/pkg/queue"

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
	balanceService := service.NewPlatformAccountBalanceService(
		db,
		platformAccountRepo,
		userRepo,
		balanceLogRepo,
	)

	userBalanceService := service.NewBalanceService(balanceLogRepo, userRepo)

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
		balanceService,
		userBalanceService,
		notificationRepo,
		queueInstance,
	)

	// 创建订单服务
	orderService := service.NewOrderService(
		orderRepo,
		balanceLogRepo,
		userRepo,
		rechargeService,
		db,
	)

	// 创建认证中间件
	authMiddleware := middleware.NewExternalAuthMiddleware(apiKeyRepo)

	// 创建商品服务
	productService := service.NewProductService(productRepo)

	// 创建外部订单日志repository
	externalOrderLogRepo := repository.NewExternalOrderLogRepository(db)

	// 创建控制器
	externalOrderController := controller.NewExternalOrderController(orderService, productService, externalOrderLogRepo)
	externalCallbackController := controller.NewExternalCallbackController(orderService, apiKeyRepo, externalOrderLogRepo)
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
