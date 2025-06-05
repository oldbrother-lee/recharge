package app

import (
	"recharge-go/internal/controller"
	"recharge-go/internal/handler"
)

// Controllers 控制器集合
type Controllers struct {
	User               *controller.UserController
	Permission         *controller.PermissionController
	Role               *controller.RoleController
	Product            *controller.ProductController
	PhoneLocation      *controller.PhoneLocationController
	ProductType        *controller.ProductTypeController
	Platform           *controller.PlatformController
	PlatformAPI        *controller.PlatformAPIController
	PlatformAPIParam   *controller.PlatformAPIParamController
	PlatformPushStatus *controller.PlatformPushStatusController
	ProductAPIRelation *controller.ProductAPIRelationController
	UserLog            *controller.UserLogController
	UserGrade          *controller.UserGradeController
	Statistics         *controller.StatisticsController
	Callback           *controller.CallbackController
	MF178Order         *controller.MF178OrderController
	Order              *controller.OrderController
	Credit             *controller.CreditController
	SystemConfig       *controller.SystemConfigController

	// Handlers
	Recharge     *handler.RechargeHandler
	Notification *handler.NotificationHandler
}

// initControllers 初始化所有控制器
func (c *Container) initControllers() {
	c.controllers = &Controllers{
		User:          controller.NewUserController(c.services.User, c.services.UserGrade, c.services.UserTag),
		PhoneLocation: controller.NewPhoneLocationController(c.services.PhoneLocation),
		Statistics:    controller.NewStatisticsController(c.services.Statistics),
		Callback:      controller.NewCallbackController(c.services.Recharge, c.repositories.Platform, c.repositories.Order),
		MF178Order:    controller.NewMF178OrderController(c.services.Order, c.services.Recharge),
		Order:         controller.NewOrderController(c.services.Order),
		Platform:      controller.NewPlatformController(c.services.Platform, c.services.PlatformSvc),
		UserGrade:     controller.NewUserGradeController(c.services.UserGrade),

		// 恢复以下控制器
		Permission:  controller.NewPermissionController(c.services.Permission),
		Role:        controller.NewRoleController(c.services.Role),
		Product:     controller.NewProductController(c.services.Product),
		ProductType: controller.NewProductTypeController(c.services.ProductType),
		PlatformAPI: controller.NewPlatformAPIController(c.services.PlatformAPI, c.services.Platform),
		// 恢复以下控制器
		PlatformAPIParam:   controller.NewPlatformAPIParamController(c.services.PlatformAPIParam),
		PlatformPushStatus: controller.NewPlatformPushStatusController(c.services.PlatformPushStatus),
		ProductAPIRelation: controller.NewProductAPIRelationController(c.services.ProductAPIRelation),
		UserLog:            controller.NewUserLogController(c.services.UserLog),
		Credit:             controller.NewCreditController(c.services.Credit),
		SystemConfig:       controller.NewSystemConfigController(c.services.SystemConfig),

		// Handlers
		Recharge:     handler.NewRechargeHandler(c.services.Recharge),
		Notification: handler.NewNotificationHandler(c.services.Notification),
	}
}

// GetControllers 获取控制器集合
func (c *Container) GetControllers() *Controllers {
	return c.controllers
}
