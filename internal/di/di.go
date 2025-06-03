package di

import (
	"recharge-go/internal/repository"
	notificationRepo "recharge-go/internal/repository/notification"
	"recharge-go/internal/service"
	notificationService "recharge-go/internal/service/notification"
	"recharge-go/internal/service/platform"
	"recharge-go/pkg/database"
	"recharge-go/pkg/logger"
	"recharge-go/pkg/queue"
	"time"
)

type App struct {
	PermissionService         *service.PermissionService
	DaichongOrderService      *service.DaichongOrderService
	RebateService             service.RebateService
	OrderUpgradeService       service.OrderUpgradeService
	PlatformAPIService        service.PlatformAPIService
	UserService               *service.UserService
	TaskService               *service.TaskService
	RechargeService           service.RechargeService
	BalanceService            *service.BalanceService
	NotificationService       notificationService.NotificationService
	DistributionService       *service.DistributionService
	RoleService               *service.RoleService
	PlatformService           *service.PlatformService
	PushStatusService         *platform.PushStatusService
	UserTagService            *service.UserTagService
	TaskConfigService         *service.TaskConfigService
	UserGradeService          *service.UserGradeService
	UserLogService            *service.UserLogService
	PlatformAccountService    *service.PlatformAccountService
	CreditService             *service.CreditService
	PlatformAPIParamService   service.PlatformAPIParamService
	RewardService             service.RewardService
	ProductTypeService        *service.ProductTypeService
	StatisticsService         service.StatisticsService
	PhoneLocationService      *service.PhoneLocationService
	ProductAPIRelationService service.ProductAPIRelationService
	ProductService            *service.ProductService
	StatisticsTask            *service.StatisticsTask
	OrderService              service.OrderService
	RetryService              *service.RetryService
}

func NewApp() *App {
	db := database.DB
	log := logger.Log
	q := queue.NewRedisQueue()

	// 仓库初始化
	permissionRepo := repository.NewPermissionRepository(db)
	daichongOrderRepo := repository.NewDaichongOrderRepository(db)
	rebateRepo := repository.NewRebateRepository(db)
	platformTokenRepo := repository.NewPlatformTokenRepository()
	orderUpgradeRepo := repository.NewOrderUpgradeRepository(db)
	platformAccountRepo := repository.NewPlatformAccountRepository(db)
	userRepo := repository.NewUserRepository(db)
	userGradeRepo := repository.NewUserGradeRepository(db)
	userTagRepo := repository.NewUserTagRepository(db)
	userTagRelationRepo := repository.NewUserTagRelationRepository(db)
	userGradeRelationRepo := repository.NewUserGradeRelationRepository(db)
	platformAPIParamRepo := repository.NewPlatformAPIParamRepository(db)
	platformAPIRepo := repository.NewPlatformAPIRepository(db)
	taskOrderRepo := repository.NewTaskOrderRepository()
	notificationRepo := notificationRepo.NewRepository(db)
	distributionRepo := repository.NewDistributionRepository(db)
	roleRepo := repository.NewRoleRepository(db)
	orderRepo := repository.NewOrderRepository(db)
	taskConfigRepo := repository.NewTaskConfigRepository()
	callbackLogRepo := repository.NewCallbackLogRepository(db)
	productRepo := repository.NewProductRepository(db)
	retryRepo := repository.NewRetryRepository(db)
	userLogRepo := repository.NewUserLogRepository(db)
	creditLogRepo := repository.NewCreditRepository(db)
	platformRepo := repository.NewPlatformRepository(db)
	balanceLogRepo := repository.NewBalanceLogRepository(db)
	rewardRepo := repository.NewRewardRepository(db)
	productTypeRepo := repository.NewProductTypeRepository(db)
	productTypeCategoryRepo := repository.NewProductTypeCategoryRepository(db)
	phoneLocationRepo := repository.NewPhoneLocationRepository(db)
	orderStatisticsRepo := repository.NewOrderStatisticsRepository(db)
	productAPIRelationRepo := repository.NewProductAPIRelationRepository(db)

	// 服务初始化
	permissionService := service.NewPermissionService(permissionRepo)
	daichongOrderService := service.NewDaichongOrderService(daichongOrderRepo)
	rebateService := service.NewRebateService(rebateRepo)
	orderUpgradeService := service.NewOrderUpgradeService(orderUpgradeRepo, platformRepo, platformAPIRepo, productRepo, productAPIRelationRepo, platformAPIParamRepo)
	platformAPIService := service.NewPlatformAPIService(platformAPIRepo)
	userService := service.NewUserService(userRepo, userGradeRepo, userTagRepo, userTagRelationRepo, userGradeRelationRepo, userLogRepo)
	balanceService := service.NewBalanceService(balanceLogRepo, userRepo)
	notificationService := notificationService.NewNotificationService(notificationRepo)
	distributionService := service.NewDistributionService(distributionRepo)
	roleService := service.NewRoleService(roleRepo)
	platformService := service.NewPlatformService(platformTokenRepo, platformRepo)
	pushStatusService := platform.NewPushStatusService(platformAccountRepo)
	userTagService := service.NewUserTagService(userTagRepo, userTagRelationRepo)
	taskConfigService := service.NewTaskConfigService(taskConfigRepo)
	userGradeService := service.NewUserGradeService(userGradeRepo, userGradeRelationRepo)
	userLogService := service.NewUserLogService(userLogRepo)
	platformAccountService := service.NewPlatformAccountService(platformAccountRepo)
	platformAccountBalanceService := service.NewPlatformAccountBalanceService(db, platformAccountRepo, userRepo, balanceLogRepo)
	creditService := service.NewCreditService(userRepo, creditLogRepo)
	platformAPIParamService := service.NewPlatformAPIParamService(platformAPIParamRepo)
	rewardService := service.NewRewardService(rewardRepo)
	productTypeService := service.NewProductTypeService(productTypeRepo, productTypeCategoryRepo)
	statisticsService := service.NewStatisticsService(orderStatisticsRepo, orderRepo)
	phoneLocationService := service.NewPhoneLocationService(phoneLocationRepo)
	productAPIRelationService := service.NewProductAPIRelationService(productAPIRelationRepo)
	productService := service.NewProductService(productRepo)
	statisticsTask := service.NewStatisticsTask(statisticsService, log)

	// 先用 nil 占位，后补全循环依赖
	orderService := service.NewOrderService(orderRepo, nil, notificationRepo, q)
	rechargeService := service.NewRechargeService(db, orderRepo, platformRepo, platformAPIRepo, retryRepo, callbackLogRepo, productAPIRelationRepo, productRepo, platformAPIParamRepo, platformAccountBalanceService, notificationRepo, q)
	retryService := service.NewRetryService(retryRepo, orderRepo, platformRepo, productRepo, productAPIRelationRepo, rechargeService, orderService)
	orderService.SetRechargeService(rechargeService)

	// 初始化 platformSvc
	platformSvc := platform.NewService(platformTokenRepo, platformRepo)
	// 初始化 taskConfig
	taskConfig := &service.TaskConfig{
		Interval:      5 * time.Minute,
		MaxRetries:    3,
		RetryDelay:    1 * time.Minute,
		MaxConcurrent: 5,
		APIKey:        "",
		UserID:        "",
		BaseURL:       "",
	}
	// 初始化 TaskService
	taskService := service.NewTaskService(taskConfigRepo, taskOrderRepo, orderRepo, daichongOrderRepo, platformSvc, orderService, taskConfig)

	return &App{
		PermissionService:         permissionService,
		DaichongOrderService:      daichongOrderService,
		RebateService:             rebateService,
		OrderUpgradeService:       orderUpgradeService,
		PlatformAPIService:        platformAPIService,
		UserService:               userService,
		TaskService:               taskService,
		RechargeService:           rechargeService,
		BalanceService:            balanceService,
		NotificationService:       notificationService,
		DistributionService:       distributionService,
		RoleService:               roleService,
		PlatformService:           platformService,
		PushStatusService:         pushStatusService,
		UserTagService:            userTagService,
		TaskConfigService:         taskConfigService,
		UserGradeService:          userGradeService,
		UserLogService:            userLogService,
		PlatformAccountService:    platformAccountService,
		CreditService:             creditService,
		PlatformAPIParamService:   platformAPIParamService,
		RewardService:             rewardService,
		ProductTypeService:        productTypeService,
		StatisticsService:         statisticsService,
		PhoneLocationService:      phoneLocationService,
		ProductAPIRelationService: productAPIRelationService,
		ProductService:            productService,
		StatisticsTask:            statisticsTask,
		OrderService:              orderService,
		RetryService:              retryService,
	}
}
