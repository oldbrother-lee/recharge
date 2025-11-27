package router

import (
	"reflect"
	"time"

	"recharge-go/internal/controller"
	"recharge-go/internal/repository"
	"recharge-go/internal/service"
	"recharge-go/internal/service/platform"
	"recharge-go/pkg/database"
	"recharge-go/pkg/log"
	logger "recharge-go/pkg/log"
	"recharge-go/pkg/metrics"
	pkgMiddleware "recharge-go/pkg/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SetupRouterV2 使用优化后的依赖注入设置路由
func SetupRouterV2(
	securityMiddleware *pkgMiddleware.SecurityMiddleware,
	metricsManager *metrics.MetricsManager,
	controllersInterface interface{}, // 控制器接口，避免循环导入
	userService interface{}, // 用户服务
	userLogController interface{}, // 用户日志控制器
	db *gorm.DB,
) *gin.Engine {
	r := gin.New()

	// 使用反射获取控制器字段
	controllersValue := reflect.ValueOf(controllersInterface)
	if controllersValue.Kind() == reflect.Ptr {
		controllersValue = controllersValue.Elem()
	}

	// 获取所有控制器
	userController := getControllerByName(controllersValue, "User")
	permissionController := getControllerByName(controllersValue, "Permission")
	roleController := getControllerByName(controllersValue, "Role")
	productController := getControllerByName(controllersValue, "Product")
	phoneLocationController := getControllerByName(controllersValue, "PhoneLocation")
	productTypeController := getControllerByName(controllersValue, "ProductType")
	platformController := getControllerByName(controllersValue, "Platform")
	platformAPIController := getControllerByName(controllersValue, "PlatformAPI")
	platformAPIParamController := getControllerByName(controllersValue, "PlatformAPIParam")
	platformPushStatusController := getControllerByName(controllersValue, "PlatformPushStatus")
	productAPIRelationController := getControllerByName(controllersValue, "ProductAPIRelation")
	userGradeController := getControllerByName(controllersValue, "UserGrade")
	mf178OrderController := getControllerByName(controllersValue, "MF178Order")
	callbackController := getControllerByName(controllersValue, "Callback")
	creditController := getControllerByName(controllersValue, "Credit")
	statisticsController := getControllerByName(controllersValue, "Statistics")
	systemConfigController := getControllerByName(controllersValue, "SystemConfig")
	externalAPIKeyController := getControllerByName(controllersValue, "ExternalAPIKey")
	phoneQueryController := getControllerByName(controllersValue, "PhoneQuery")
	// userLogController := getControllerByName(controllersValue, "UserLog") // 从参数获取

	// 类型断言
	userSvc, ok := userService.(*service.UserService)
	if !ok {
		logger.GetCategoryLogger("router").Error("SetupRouterV2.type_assert_failed",
			logger.StringV2("target", "UserService"),
			logger.StringV2("got_type", reflect.TypeOf(userService).String()),
		)
		return nil
	}

	userLogCtrl, ok := userLogController.(*controller.UserLogController)
	if !ok {
		logger.GetCategoryLogger("router").Error("SetupRouterV2.type_assert_failed",
			logger.StringV2("target", "UserLogController"),
			logger.StringV2("got_type", reflect.TypeOf(userLogController).String()),
		)
		return nil
	}

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
		if uc := assertUserController(userController); uc != nil {
			RegisterUserRoutes(v1, uc, userSvc, userLogCtrl)
		}

		// MF178订单接口
		if moc := assertMF178OrderController(mf178OrderController); moc != nil {
			RegisterMF178OrderRoutes(v1, moc)
		}

		// 可客帮订单接口 - 不需要认证
		RegisterKekebangOrderRoutes(v1)

		// 闲赢客订单接口 - 不需要认证
		RegisterXianyinkeOrderRoutes(v1)

		// 代充订单接口 - 不需要认证
		RegisterDaichongOrderRoutes(v1)

		// 外部订单接口 - 不需要认证
		RegisterExternalOrderRoutes(v1, db)

		// 回调接口 - 不需要认证
		if cc := assertCallbackController(callbackController); cc != nil {
			callback := v1.Group("/callback")
			{
				callback.POST("/kekebang/:userid", cc.HandleKekebangCallback)
				callback.POST("/mishi/:userid", cc.HandleMishiCallback)
				callback.POST("/dayuanren/:userid", cc.HandleDayuanrenCallback)
				callback.POST("/chongzhi/:userid", cc.HandleChongzhiCallback)
				callback.POST("/payc2/:userid", cc.HandlePayc2Callback)
				callback.POST("/lingshi/:userid", cc.HandleLingshiCallback)
				callback.POST("/kasushou/:userid", cc.HandleKasushouCallback)
				callback.POST("/shangteng/:userid", cc.HandleShangtengCallback)
			}
		}

		// 平台账号相关接口已在下方的认证路由组中定义

		// 公共系统配置接口 - 不需要认证
		if scc := assertSystemConfigController(systemConfigController); scc != nil {
			public := v1.Group("/public")
			{
				public.GET("/system/name", scc.GetSystemName)
				public.GET("/system/basic-info", scc.GetSystemInfo)
			}
		}

		// Protected routes（在受保护分组挂载 JWTAuth）
		auth := v1.Group("")
		auth.Use(securityMiddleware.JWTAuth())
		{
			// Protected user routes
			if uc := assertUserController(userController); uc != nil {
				RegisterProtectedUserRoutes(auth, uc, userSvc, userLogCtrl)
			}

			// Permission routes
			if pc := assertPermissionController(permissionController); pc != nil {
				RegisterPermissionRoutes(auth, pc)
			}

			// Role routes
			if rc := assertRoleController(roleController); rc != nil {
				RegisterRoleRoutes(auth, rc)
			}

			// Product routes
			if pc := assertProductController(productController); pc != nil {
				RegisterProductRoutes(auth, pc, userSvc)
			}

			// Phone location routes
			if plc := assertPhoneLocationController(phoneLocationController); plc != nil {
				RegisterPhoneLocationRoutes(auth, plc, userSvc)
			}

			// Product type routes
			if ptc := assertProductTypeController(productTypeController); ptc != nil {
				RegisterProductTypeRoutes(auth, ptc, userSvc)
			}

			// Platform routes
			if pc := assertPlatformController(platformController); pc != nil {
				RegisterPlatformRoutes(auth, pc, userSvc)
			}

			// Platform API routes
			if pac := assertPlatformAPIController(platformAPIController); pac != nil {
				RegisterPlatformAPIRoutes(auth, pac, userSvc)
			}

			// Platform API param routes
			if papc := assertPlatformAPIParamController(platformAPIParamController); papc != nil {
				RegisterPlatformAPIParamRoutes(auth, papc, userSvc)
			}

			// Platform push status routes
			if ppsc := assertPlatformPushStatusController(platformPushStatusController); ppsc != nil {
				RegisterPlatformPushStatusRoutes(auth, ppsc)
			}

			// Product API relation routes
			if parc := assertProductAPIRelationController(productAPIRelationController); parc != nil {
				RegisterProductAPIRelationRoutes(auth, parc)
			}

			// User grade routes
			if ugc := assertUserGradeController(userGradeController); ugc != nil {
				RegisterUserGradeRoutes(auth, ugc)
			}

			// Order routes
			RegisterOrderRoutes(auth, userSvc)

			// Recharge routes
			recharge := auth.Group("/recharge")
			{
				// TODO: 添加充值回调处理
				recharge.POST("/callback/:platform", func(c *gin.Context) {
					c.JSON(200, gin.H{"message": "Recharge callback placeholder"})
				})
			}

			// 余额相关接口（仅管理员可访问）
			userRepo := repository.NewUserRepository(database.DB)
			if us, ok := userService.(*service.UserService); ok {
				RegisterBalanceRoutes(auth, database.DB, userRepo, us)
			}

			// 统一拉单配置路由（仅管理员）
			RegisterPullTaskConfigRoutes(auth)

			// 平台余额查询接口（仅管理员可访问）
			RegisterPlatformBalanceRoutes(auth, userSvc)

			// 授信相关接口（仅管理员可访问）
			if cc := assertCreditController(creditController); cc != nil {
				RegisterCreditRoutes(auth, cc)
			}

			// 统计相关路由
			if sc := assertStatisticsController(statisticsController); sc != nil {
				RegisterStatisticsRoutes(auth, sc)
			}

			// Task config routes
			RegisterTaskConfigRoutes(auth)

			// Task routes (包含task-config路由)
			// 需要创建平台服务实例
			platformTokenRepo := repository.NewPlatformTokenRepository(database.DB)
			platformRepo := repository.NewPlatformRepository(database.DB)
			platformSvc := platform.NewService(platformTokenRepo, platformRepo)
			RegisterTaskRoutes(auth, platformSvc)

			// 注册得众平台任务配置路由
			RegisterDzTaskRoutes(auth)

			// System config routes
			if scc := assertSystemConfigController(systemConfigController); scc != nil {
				RegisterSystemConfigRoutes(auth, scc)
			}

			// External API Key routes
			if eakc := assertExternalAPIKeyController(externalAPIKeyController); eakc != nil {
				RegisterExternalAPIKeyRoutes(auth, eakc, userSvc)
			}

			// Phone Query routes
			if pqc := assertPhoneQueryController(phoneQueryController); pqc != nil {
				RegisterPhoneQueryRoutes(auth, pqc, userSvc)
			}

			// Order Exception routes
			RegisterOrderExceptionRoutes(auth, userSvc)

			// 平台账号拉单功能路由
			platformAccountRepo := repository.NewPlatformAccountRepository(db)
			platformAccountSvc := service.NewPlatformAccountService(platformAccountRepo)
			platformAccountCtrl := controller.NewPlatformAccountController(platformAccountSvc)

			// 平台账号变体相关
			variantRepo := repository.NewPlatformAccountVariantRepository(db)
			variantSvc := service.NewPlatformAccountVariantService(variantRepo, platformAccountRepo)
			variantCtrl := controller.NewPlatformAccountVariantController(variantSvc)

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

// getControllerByName 通过反射安全地获取控制器
func getControllerByName(controllersValue reflect.Value, name string) interface{} {
	field := controllersValue.FieldByName(name)
	if !field.IsValid() || field.IsNil() {
		return nil
	}
	return field.Interface()
}

// 类型断言辅助函数
func assertUserController(ctrl interface{}) *controller.UserController {
	if ctrl == nil {
		return nil
	}
	if uc, ok := ctrl.(*controller.UserController); ok {
		return uc
	}
	return nil
}

func assertPermissionController(ctrl interface{}) *controller.PermissionController {
	if ctrl == nil {
		return nil
	}
	if pc, ok := ctrl.(*controller.PermissionController); ok {
		return pc
	}
	return nil
}

func assertRoleController(ctrl interface{}) *controller.RoleController {
	if ctrl == nil {
		return nil
	}
	if rc, ok := ctrl.(*controller.RoleController); ok {
		return rc
	}
	return nil
}

func assertProductController(ctrl interface{}) *controller.ProductController {
	if ctrl == nil {
		return nil
	}
	if pc, ok := ctrl.(*controller.ProductController); ok {
		return pc
	}
	return nil
}

func assertPhoneLocationController(ctrl interface{}) *controller.PhoneLocationController {
	if ctrl == nil {
		return nil
	}
	if plc, ok := ctrl.(*controller.PhoneLocationController); ok {
		return plc
	}
	return nil
}

func assertProductTypeController(ctrl interface{}) *controller.ProductTypeController {
	if ctrl == nil {
		return nil
	}
	if ptc, ok := ctrl.(*controller.ProductTypeController); ok {
		return ptc
	}
	return nil
}

func assertPlatformController(ctrl interface{}) *controller.PlatformController {
	if ctrl == nil {
		return nil
	}
	if pc, ok := ctrl.(*controller.PlatformController); ok {
		return pc
	}
	return nil
}

func assertPlatformAPIController(ctrl interface{}) *controller.PlatformAPIController {
	if ctrl == nil {
		return nil
	}
	if pac, ok := ctrl.(*controller.PlatformAPIController); ok {
		return pac
	}
	return nil
}

func assertPlatformAPIParamController(ctrl interface{}) *controller.PlatformAPIParamController {
	if ctrl == nil {
		return nil
	}
	if papc, ok := ctrl.(*controller.PlatformAPIParamController); ok {
		return papc
	}
	return nil
}

func assertPlatformPushStatusController(ctrl interface{}) *controller.PlatformPushStatusController {
	if ctrl == nil {
		return nil
	}
	if ppsc, ok := ctrl.(*controller.PlatformPushStatusController); ok {
		return ppsc
	}
	return nil
}

func assertProductAPIRelationController(ctrl interface{}) *controller.ProductAPIRelationController {
	if ctrl == nil {
		return nil
	}
	if parc, ok := ctrl.(*controller.ProductAPIRelationController); ok {
		return parc
	}
	return nil
}

func assertUserGradeController(ctrl interface{}) *controller.UserGradeController {
	if ctrl == nil {
		return nil
	}
	if ugc, ok := ctrl.(*controller.UserGradeController); ok {
		return ugc
	}
	return nil
}

func assertMF178OrderController(ctrl interface{}) *controller.MF178OrderController {
	if ctrl == nil {
		return nil
	}
	if moc, ok := ctrl.(*controller.MF178OrderController); ok {
		return moc
	}
	return nil
}

func assertCallbackController(ctrl interface{}) *controller.CallbackController {
	if ctrl == nil {
		return nil
	}
	if cc, ok := ctrl.(*controller.CallbackController); ok {
		return cc
	}
	return nil
}

func assertCreditController(ctrl interface{}) *controller.CreditController {
	if ctrl == nil {
		return nil
	}
	if cc, ok := ctrl.(*controller.CreditController); ok {
		return cc
	}
	return nil
}

func assertStatisticsController(ctrl interface{}) *controller.StatisticsController {
	if ctrl == nil {
		return nil
	}
	if sc, ok := ctrl.(*controller.StatisticsController); ok {
		return sc
	}
	return nil
}

func assertUserLogController(ctrl interface{}) *controller.UserLogController {
	if ctrl == nil {
		return nil
	}
	if ulc, ok := ctrl.(*controller.UserLogController); ok {
		return ulc
	}
	return nil
}

func assertSystemConfigController(ctrl interface{}) *controller.SystemConfigController {
	if ctrl == nil {
		return nil
	}
	if scc, ok := ctrl.(*controller.SystemConfigController); ok {
		return scc
	}
	return nil
}

func assertExternalAPIKeyController(ctrl interface{}) *controller.ExternalAPIKeyController {
	if ctrl == nil {
		return nil
	}
	if eakc, ok := ctrl.(*controller.ExternalAPIKeyController); ok {
		return eakc
	}
	return nil
}

func assertPhoneQueryController(ctrl interface{}) *controller.PhoneQueryController {
	if ctrl == nil {
		return nil
	}
	if c, ok := ctrl.(*controller.PhoneQueryController); ok {
		return c
	}
	return nil
}

func assertUserService(svc interface{}) *service.UserService {
	if svc == nil {
		return nil
	}
	if s, ok := svc.(*service.UserService); ok {
		return s
	}
	return nil
}
