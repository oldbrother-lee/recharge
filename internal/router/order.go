package router

import (
	"recharge-go/internal/controller"
	"recharge-go/internal/middleware"
	"recharge-go/internal/repository"
	notificationRepo "recharge-go/internal/repository/notification"
	"recharge-go/internal/service"
	"recharge-go/pkg/database"
	"recharge-go/pkg/queue"

	"github.com/gin-gonic/gin"
)

// RegisterOrderRoutes 注册订单相关路由
func RegisterOrderRoutes(r *gin.RouterGroup, userService *service.UserService) {
	// 创建服务实例
	orderRepo := repository.NewOrderRepository(database.DB)
	platformRepo := repository.NewPlatformRepository(database.DB)
	callbackLogRepo := repository.NewCallbackLogRepository(database.DB)

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

	// 创建充值服务
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

		// 只允许管理员访问的订单清理接口
		order.DELETE("/cleanup", middleware.CheckSuperAdmin(userService), orderController.CleanupOrders)
	}
}
