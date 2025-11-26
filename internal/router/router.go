package router

import (
	"recharge-go/internal/controller"
	"recharge-go/internal/handler"
	"recharge-go/internal/middleware"
	"recharge-go/internal/repository"
	"recharge-go/internal/service"
	"recharge-go/internal/service/platform"
	"recharge-go/pkg/database"
	"recharge-go/pkg/log"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetupRouter(
	userController *controller.UserController,
	permissionController *controller.PermissionController,
	roleController *controller.RoleController,
	productController *controller.ProductController,
	userService *service.UserService,
	phoneLocationController *controller.PhoneLocationController,
	productTypeController *controller.ProductTypeController,
	platformController *controller.PlatformController,
	platformAPIController *controller.PlatformAPIController,
	platformAPIParamController *controller.PlatformAPIParamController,
	productAPIRelationController *controller.ProductAPIRelationController,
	userLogController *controller.UserLogController,
	userGradeController *controller.UserGradeController,
	rechargeHandler *handler.RechargeHandler,
	retryService *service.RetryService,
	userRepo *repository.UserRepository,
	statisticsController *controller.StatisticsController,
	callbackController *controller.CallbackController,
	mf178OrderController *controller.MF178OrderController,
	orderController *controller.OrderController,
	notificationHandler *handler.NotificationHandler,
	systemConfigController *controller.SystemConfigController,
	db *gorm.DB,
) *gin.Engine {
	r := gin.New()

	// Global middleware
	r.Use(middleware.CORS())
	r.Use(log.GinLogger())
	r.Use(log.GinRecovery())

	// API routes
	api := r.Group("/api/v1")
	{
		// Public user routes
		RegisterUserRoutes(api, userController, userService, userLogController)

		// 通知路由
		notificationHandler.RegisterRoutes(api)

		// MF178订单接口
		RegisterMF178OrderRoutes(api, mf178OrderController)

		// 可客帮订单接口 - 不需要认证
		RegisterKekebangOrderRoutes(api)

		// 闲赢客订单接口 - 不需要认证
		RegisterXianyinkeOrderRoutes(api)

		// 代充订单接口 - 不需要认证
		RegisterDaichongOrderRoutes(api)

		// 外部订单接口 - 不需要认证
		RegisterExternalOrderRoutes(api, db)

		// 回调接口 - 不需要认证
		callback := api.Group("/callback")
		{
			callback.POST("/kekebang/:userid", callbackController.HandleKekebangCallback)
			callback.POST("/mishi/:userid", callbackController.HandleMishiCallback)
			callback.POST("/dayuanren/:userid", callbackController.HandleDayuanrenCallback)
			callback.POST("/chongzhi/:userid", callbackController.HandleChongzhiCallback)
			callback.POST("/payc2/:userid", callbackController.HandlePayc2Callback)
			callback.POST("/lingshi/:userid", callbackController.HandleLingshiCallback)
		}

		// Protected routes
		auth := api.Group("")
		auth.Use(middleware.Auth())
		{
			// Protected user routes
			RegisterProtectedUserRoutes(auth, userController, userService, userLogController)

			// Permission routes
			RegisterPermissionRoutes(auth, permissionController)

			// Role routes
			RegisterRoleRoutes(auth, roleController)

			// Product routes
			RegisterProductRoutes(auth, productController, userService)

			// Phone location routes
			RegisterPhoneLocationRoutes(auth, phoneLocationController, userService)

			// Product type routes
			RegisterProductTypeRoutes(auth, productTypeController, userService)

			// Platform routes
			RegisterPlatformRoutes(auth, platformController, userService)

			// Platform API routes
			RegisterPlatformAPIRoutes(auth, platformAPIController, userService)

			// Platform API param routes
			RegisterPlatformAPIParamRoutes(auth, platformAPIParamController, userService)

			// Product API relation routes
			RegisterProductAPIRelationRoutes(auth, productAPIRelationController)

			// User grade routes
			RegisterUserGradeRoutes(auth, userGradeController)

			// Order routes
			RegisterOrderRoutes(auth, userService)

			// Recharge routes
			recharge := auth.Group("/recharge")
			{
				recharge.POST("/callback/:platform", rechargeHandler.HandleCallback)
			}

			// 余额相关接口（仅管理员可访问）
			RegisterBalanceRoutes(auth, database.DB, userRepo, userService)

			// 平台余额查询接口（仅管理员可访问）
			RegisterPlatformBalanceRoutes(auth, userService)

			// 授信相关接口（仅管理员可访问）
			creditLogRepo := repository.NewCreditLogRepository(database.DB)
			creditService := service.NewCreditService(userRepo, creditLogRepo)
			creditController := controller.NewCreditController(creditService)
			RegisterCreditRoutes(auth, creditController)

			// 统计相关路由
			RegisterStatisticsRoutes(auth, statisticsController)

			// Task config routes
			RegisterTaskConfigRoutes(auth)

			// 只允许管理员访问
			RegisterDaichongOrderRoutes(auth)

			// 平台账号相关接口（包含拉单功能）
			RegisterPlatformAccountRoutes(api, userService)
			//通知路由

			// 推单状态相关接口
			platformAccountRepo := repository.NewPlatformAccountRepository(database.DB)
			pushStatusService := platform.NewPushStatusService(platformAccountRepo)
			pushStatusController := controller.NewPlatformPushStatusController(pushStatusService)

			// 注册路由
			pushStatus := auth.Group("/platform/push-status")
			{
				pushStatus.GET("/:account_id", pushStatusController.GetPushStatus)
				pushStatus.PUT("/:account_id", pushStatusController.UpdatePushStatus)
			}

			// 订单相关路由
			orderGroup := auth.Group("/orders")
			{
				orderGroup.GET("/statistics", orderController.GetOrderStatistics)
			}

			// API密钥管理路由
			apiKeyRepo := repository.NewExternalAPIKeyRepository(database.DB)
			apiKeyController := controller.NewExternalAPIKeyController(apiKeyRepo, userRepo)
			RegisterExternalAPIKeyRoutes(auth, apiKeyController, userService)
		}

		// 注册系统配置路由
		RegisterSystemConfigRoutes(api, systemConfigController)
	}

	return r
}

// 注册平台账号相关接口
func RegisterPlatformAccountRoutes(r *gin.RouterGroup, userService *service.UserService) {
	// 这里直接初始化 repository/service/controller，实际项目可根据依赖注入优化
	platformAccountRepo := repository.NewPlatformAccountRepository(database.DB)
	platformAccountSvc := service.NewPlatformAccountService(platformAccountRepo)
	platformAccountCtrl := controller.NewPlatformAccountController(platformAccountSvc)

	// 平台账号变体相关
	variantRepo := repository.NewPlatformAccountVariantRepository(database.DB)
	variantSvc := service.NewPlatformAccountVariantService(variantRepo, platformAccountRepo)
	variantCtrl := controller.NewPlatformAccountVariantController(variantSvc)

	// 基础平台账号接口
	r.POST("/platform/account/bind_user", platformAccountCtrl.BindUser)
	r.GET("/platform/account/list", platformAccountCtrl.List)

	// 拉单相关接口
	pullOrder := r.Group("/platform/account/pull-order")
	{
		pullOrder.GET("/accounts", platformAccountCtrl.GetPullOrderAccounts)
		pullOrder.GET("/accounts/:id", platformAccountCtrl.GetPullOrderAccount)
		pullOrder.PUT("/accounts/:id/config", platformAccountCtrl.UpdatePullOrderConfig)
	}

	// 平台账号变体接口
	variants := r.Group("/platform/account/variants")
	{
		variants.POST("", variantCtrl.Create)
		variants.PUT("/:id", variantCtrl.Update)
		variants.DELETE("/:id", variantCtrl.Delete)
		variants.GET("/:id", variantCtrl.GetByID)
		variants.GET("", variantCtrl.List)
		variants.GET("/platform-account/:platform_account_id", variantCtrl.GetByPlatformAccount)
	}
}

// 注册推单状态相关接口
// func RegisterPushStatusRoutes(r *gin.RouterGroup) {
// 	// 初始化依赖
// 	platformAccountRepo := repository.NewPlatformAccountRepository(database.DB)
// 	pushStatusService := platform.NewPushStatusService(platformAccountRepo)
// 	pushStatusController := controller.NewPlatformPushStatusController(pushStatusService)

// 	// 注册路由
// 	pushStatus := r.Group("/platform/push-status")
// 	{
// 		pushStatus.GET("/:account_id", pushStatusController.GetPushStatus)
// 		pushStatus.PUT("/:account_id", pushStatusController.UpdatePushStatus)
// 	}
// }
