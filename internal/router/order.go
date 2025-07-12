package router

import (
	"recharge-go/internal/controller"
	"recharge-go/internal/middleware"
	"recharge-go/internal/repository"
	notificationRepo "recharge-go/internal/repository/notification"
	"recharge-go/internal/service"
	"recharge-go/pkg/database"
	"recharge-go/pkg/lock"
	"recharge-go/pkg/queue"
	"recharge-go/pkg/redis"

	"github.com/gin-gonic/gin"
)

// RegisterOrderRoutes 注册订单相关路由
func RegisterOrderRoutes(r *gin.RouterGroup, userService *service.UserService) {
	// 获取数据库连接
	db := database.DB
	
	// 创建服务实例
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

	// 创建授信服务
	creditLogRepo := repository.NewCreditLogRepository(db)
	creditService := service.NewCreditService(userRepo, creditLogRepo)

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
		productRepo,
		creditService,
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
		platformAccountBalanceService,
		userBalanceService,
		notificationRepo,
		queueInstance,
	)

	// 设置 orderService 的 rechargeService
	orderService.SetRechargeService(rechargeService)

	// 创建控制器
	orderController := controller.NewOrderController(orderService)

	// 注册路由
	order := r.Group("/order")
	{
		order.GET("/list", orderController.GetOrders)   // 获取订单列表（管理员接口）
		order.GET("/:id", orderController.GetOrderByID) // 获取订单详情
		order.POST("", orderController.CreateOrder)     // 创建订单
		// order.PUT("/:id/status", orderController.UpdateOrderStatus)                // 更新订单状态
		order.GET("/customer/:customer_id", orderController.GetOrdersByCustomerID) // 获取客户订单列表
		// order.POST("/:id/payment", orderController.ProcessOrderPayment)
		// order.POST("/:id/recharge", orderController.ProcessOrderRecharge)
		order.POST("/:id/success", orderController.ProcessOrderSuccess)
		order.POST("/:id/fail", orderController.ProcessOrderFail)
		// order.POST("/:id/refund", orderController.ProcessOrderRefund)
		// order.POST("/:id/cancel", orderController.ProcessOrderCancel)
		// order.POST("/:id/split", orderController.ProcessOrderSplit)
		// order.POST("/:id/partial", orderController.ProcessOrderPartial)
		order.POST("/:id/delete", orderController.DeleteOrder)

		// 批量操作接口
		order.POST("/batch-delete", orderController.BatchDeleteOrders)
		order.POST("/batch-success", orderController.BatchProcessOrderSuccess)
		order.POST("/batch-fail", orderController.BatchProcessOrderFail)
		order.POST("/batch-notification", orderController.BatchSendNotification)

		// 只允许管理员访问的订单清理接口
		order.DELETE("/cleanup", middleware.CheckSuperAdmin(userService), orderController.CleanupOrders)
	}

	// 注册 /orders 路由组（用于统计等批量操作）
	orders := r.Group("/orders")
	{
		orders.GET("/statistics", orderController.GetOrderStatistics) // 获取订单统计
	}
}
