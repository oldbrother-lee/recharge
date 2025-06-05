package router

import (
	"reflect"
	"time"

	"recharge-go/internal/controller"
	"recharge-go/internal/repository"
	"recharge-go/internal/service"
	"recharge-go/pkg/database"
	"recharge-go/pkg/logger"
	"recharge-go/pkg/metrics"
	"recharge-go/pkg/middleware"

	"github.com/gin-gonic/gin"
)

// SetupRouterV2 使用优化后的依赖注入设置路由
func SetupRouterV2(
	securityMiddleware *middleware.SecurityMiddleware,
	metricsManager *metrics.MetricsManager,
	controllersInterface interface{}, // 控制器接口，避免循环导入
	userService interface{}, // 用户服务
	userLogController interface{}, // 用户日志控制器
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
	// userLogController := getControllerByName(controllersValue, "UserLog") // 从参数获取

	// 类型断言
	userSvc, ok := userService.(*service.UserService)
	if !ok {
		logger.Error("Failed to assert userService type")
		return nil
	}

	userLogCtrl, ok := userLogController.(*controller.UserLogController)
	if !ok {
		logger.Error("Failed to assert userLogController type")
		return nil
	}

	// Global middleware
	r.Use(securityMiddleware.RequestID())
	r.Use(securityMiddleware.CORS())
	r.Use(securityMiddleware.Security())
	r.Use(metricsManager.HTTPMetricsMiddleware())
	r.Use(securityMiddleware.RateLimit())
	r.Use(logger.GinLogger())
	r.Use(logger.GinRecovery())

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

		// 代充订单接口 - 不需要认证
		RegisterDaichongOrderRoutes(v1)

		// 外部订单接口 - 不需要认证
		RegisterExternalOrderRoutes(v1)

		// 回调接口 - 不需要认证
		if cc := assertCallbackController(callbackController); cc != nil {
			callback := v1.Group("/callback")
			{
				callback.POST("/kekebang/:userid", cc.HandleKekebangCallback)
				callback.POST("/mishi/:userid", cc.HandleMishiCallback)
				callback.POST("/dayuanren/:userid", cc.HandleDayuanrenCallback)
			}
		}

		// 平台账号相关接口
		RegisterPlatformAccountRoutes(v1)

		// 公共系统配置接口 - 不需要认证
		if scc := assertSystemConfigController(systemConfigController); scc != nil {
			public := v1.Group("/public")
			{
				public.GET("/system/name", scc.GetSystemName)
				public.GET("/system/basic-info", scc.GetSystemInfo)
			}
		}

		// Protected routes
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

			// 平台余额查询接口（仅管理员可访问）
			RegisterPlatformBalanceRoutes(auth, nil)

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

			// System config routes
			if scc := assertSystemConfigController(systemConfigController); scc != nil {
				RegisterSystemConfigRoutes(auth, scc)
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
