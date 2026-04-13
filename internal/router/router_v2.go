package router

import (
	"time"

	"recharge-go/internal/controller"
	"recharge-go/internal/service"
	"recharge-go/internal/service/platform"
	"recharge-go/pkg/log"
	"recharge-go/pkg/metrics"
	pkgMiddleware "recharge-go/pkg/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ControllerSet struct {
	User                   *controller.UserController
	Permission             *controller.PermissionController
	Role                   *controller.RoleController
	Product                *controller.ProductController
	PhoneLocation          *controller.PhoneLocationController
	ProductType            *controller.ProductTypeController
	Platform               *controller.PlatformController
	PlatformAPI            *controller.PlatformAPIController
	PlatformAPIParam       *controller.PlatformAPIParamController
	PlatformPushStatus     *controller.PlatformPushStatusController
	ProductAPIRelation     *controller.ProductAPIRelationController
	UserGrade              *controller.UserGradeController
	MF178Order             *controller.MF178OrderController
	Callback               *controller.CallbackController
	Credit                 *controller.CreditController
	Statistics             *controller.StatisticsController
	SystemConfig           *controller.SystemConfigController
	ExternalAPIKey         *controller.ExternalAPIKeyController
	PhoneQuery             *controller.PhoneQueryController
	KekebangOrder          *controller.KekebangOrderController
	XianyinkeOrder         *controller.XianyinkeOrderController
	OrderException         *controller.OrderExceptionController
	PlatformAccount        *controller.PlatformAccountController
	PlatformAccountVariant *controller.PlatformAccountVariantController
	Balance                *controller.BalanceController
}

// SetupRouterV2 使用优化后的依赖注入设置路由
func SetupRouterV2(
	securityMiddleware *pkgMiddleware.SecurityMiddleware,
	metricsManager *metrics.MetricsManager,
	controllers ControllerSet,
	userService *service.UserService,
	userLogController *controller.UserLogController,
	platformSvc *platform.Service,
	db *gorm.DB,
) *gin.Engine {
	r := gin.New()

	userLogCtrl := userLogController
	userSvc := userService

	// Global middleware
	r.Use(securityMiddleware.RequestID())
	r.Use(securityMiddleware.CORS())
	r.Use(securityMiddleware.Security())
	r.Use(metricsManager.HTTPMetricsMiddleware())
	r.Use(securityMiddleware.RateLimit())
	r.Use(log.GinLogger())
	r.Use(log.GinRecovery())

	// 404 handler
	r.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{"code": 1004, "message": "Not Found", "data": nil})
	})

	// API routes
	v1 := r.Group("/api/v1")
	{
		// Public user routes
		if controllers.User != nil {
			RegisterUserRoutes(v1, controllers.User, userSvc, userLogCtrl)
		}

		// MF178订单接口
		if controllers.MF178Order != nil {
			RegisterMF178OrderRoutes(v1, controllers.MF178Order)
		}

		// 可客帮订单接口 - 不需要认证
		if controllers.KekebangOrder != nil {
			RegisterKekebangOrderRoutes(v1, controllers.KekebangOrder)
		}

		// 闲赢客订单接口 - 不需要认证
		if controllers.XianyinkeOrder != nil {
			RegisterXianyinkeOrderRoutes(v1, controllers.XianyinkeOrder)
		}

		// 代充订单接口 - 不需要认证
		RegisterDaichongOrderRoutes(v1, db)

		// 外部订单接口 - 不需要认证
		RegisterExternalOrderRoutes(v1, db)

		// 回调接口 - 不需要认证
		if controllers.Callback != nil {
			callback := v1.Group("/callback")
			{
				callback.POST("/kekebang/:userid", controllers.Callback.HandleKekebangCallback)
				callback.POST("/mishi/:userid", controllers.Callback.HandleMishiCallback)
				callback.POST("/dayuanren/:userid", controllers.Callback.HandleDayuanrenCallback)
				callback.POST("/chongzhi/:userid", controllers.Callback.HandleChongzhiCallback)
				callback.POST("/payc2/:userid", controllers.Callback.HandlePayc2Callback)
				callback.POST("/lingshi/:userid", controllers.Callback.HandleLingshiCallback)
				callback.POST("/kasushou/:userid", controllers.Callback.HandleKasushouCallback)
				callback.POST("/shangteng/:userid", controllers.Callback.HandleShangtengCallback)
				callback.POST("/turbo/:userid", controllers.Callback.HandleTurboCallback)
			}
		}

		// 平台账号相关接口已在下方的认证路由组中定义

		// 公共系统配置接口 - 不需要认证
		if controllers.SystemConfig != nil {
			public := v1.Group("/public")
			{
				public.GET("/system/name", controllers.SystemConfig.GetSystemName)
				public.GET("/system/basic-info", controllers.SystemConfig.GetSystemInfo)
			}
		}

		// Protected routes（在受保护分组挂载 JWTAuth）
		auth := v1.Group("")
		auth.Use(securityMiddleware.JWTAuth())
		{
			// Protected user routes
			if controllers.User != nil {
				RegisterProtectedUserRoutes(auth, controllers.User, userSvc, userLogCtrl)
			}

			// Permission routes
			if controllers.Permission != nil {
				RegisterPermissionRoutes(auth, controllers.Permission)
			}

			// Role routes
			if controllers.Role != nil {
				RegisterRoleRoutes(auth, controllers.Role)
			}

			// Product routes
			if controllers.Product != nil {
				RegisterProductRoutes(auth, controllers.Product, userSvc)
			}

			// Phone location routes
			if controllers.PhoneLocation != nil {
				RegisterPhoneLocationRoutes(auth, controllers.PhoneLocation, userSvc)
			}

			// Product type routes
			if controllers.ProductType != nil {
				RegisterProductTypeRoutes(auth, controllers.ProductType, userSvc)
			}

			// Platform routes
			if controllers.Platform != nil {
				RegisterPlatformRoutes(auth, controllers.Platform, userSvc)
			}

			// Platform API routes
			if controllers.PlatformAPI != nil {
				RegisterPlatformAPIRoutes(auth, controllers.PlatformAPI, userSvc)
			}

			// Platform API param routes
			if controllers.PlatformAPIParam != nil {
				RegisterPlatformAPIParamRoutes(auth, controllers.PlatformAPIParam, userSvc)
			}

			// Platform push status routes
			if controllers.PlatformPushStatus != nil {
				RegisterPlatformPushStatusRoutes(auth, controllers.PlatformPushStatus)
			}

			// Product API relation routes
			if controllers.ProductAPIRelation != nil {
				RegisterProductAPIRelationRoutes(auth, controllers.ProductAPIRelation)
			}

			// User grade routes
			if controllers.UserGrade != nil {
				RegisterUserGradeRoutes(auth, controllers.UserGrade)
			}

			// Order routes
			RegisterOrderRoutes(auth, db, userSvc)

			// Recharge routes
			recharge := auth.Group("/recharge")
			{
				// TODO: 添加充值回调处理
				recharge.POST("/callback/:platform", func(c *gin.Context) {
					c.JSON(200, gin.H{"message": "Recharge callback placeholder"})
				})
			}

			// 余额相关接口（仅管理员可访问）
			RegisterBalanceRoutes(auth, controllers.Balance, userSvc)

			// 统一拉单配置路由（仅管理员）
			RegisterPullTaskConfigRoutes(auth, db)

			// 平台余额查询接口（仅管理员可访问）
			RegisterPlatformBalanceRoutes(auth, db, userSvc)

			// 授信相关接口（仅管理员可访问）
			if controllers.Credit != nil {
				RegisterCreditRoutes(auth, controllers.Credit)
			}

			// 统计相关路由
			if controllers.Statistics != nil {
				RegisterStatisticsRoutes(auth, controllers.Statistics)
			}

			// Task config routes
			RegisterTaskConfigRoutes(auth)

			// Task routes (包含task-config路由)
			RegisterTaskRoutes(auth, db, platformSvc)

			// 注册得众平台任务配置路由
			RegisterDzTaskRoutes(auth, db)

			// System config routes
			if controllers.SystemConfig != nil {
				RegisterSystemConfigRoutes(auth, controllers.SystemConfig)
			}

			// External API Key routes
			if controllers.ExternalAPIKey != nil {
				RegisterExternalAPIKeyRoutes(auth, controllers.ExternalAPIKey, userSvc)
			}

			// Phone Query routes
			if controllers.PhoneQuery != nil {
				RegisterPhoneQueryRoutes(auth, controllers.PhoneQuery, userSvc)
			}

			if controllers.OrderException != nil {
				RegisterOrderExceptionRoutes(auth, controllers.OrderException, userSvc)
			}

			// 平台账号功能路由（注入控制器）
			platformAccountCtrl := controllers.PlatformAccount
			variantCtrl := controllers.PlatformAccountVariant

			// 拉单相关接口
			pullOrder := auth.Group("/platform/account/pull-order")
			{
				pullOrder.GET("/accounts", platformAccountCtrl.GetPullOrderAccounts)
				pullOrder.GET("/accounts/:id", platformAccountCtrl.GetPullOrderAccount)
				pullOrder.PUT("/accounts/:id/config", platformAccountCtrl.UpdatePullOrderConfig)
			}

			// 平台账号基本接口
			auth.POST("/platform/account/bind_user", platformAccountCtrl.BindUser)
			auth.GET("/platform/account/list", platformAccountCtrl.List)

			// 平台账号变体接口
			variants := auth.Group("/platform/account/variants")
			{
				variants.POST("", variantCtrl.Create)
				variants.PUT("/:id", variantCtrl.Update)
				variants.DELETE("/:id", variantCtrl.Delete)
				variants.GET("/:id", variantCtrl.GetByID)
				variants.GET("", variantCtrl.List)
				variants.GET("/platform-account/:platform_account_id", variantCtrl.GetByPlatformAccount)
			}

			// TODO: 以下路由对应的控制器暂未初始化，需要对应的服务支持
			// 只允许管理员访问
			// RegisterDaichongOrderRoutes(auth)
		}
	}

	// 指标监控端点
	r.GET("/metrics", gin.WrapH(metricsManager.GetHandler()))

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":    "healthy",
			"message":   "Service is running",
			"timestamp": time.Now().Unix(),
		})
	})

	return r
}
