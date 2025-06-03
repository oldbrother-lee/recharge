package main

import (
	"recharge-go/internal/controller"
	"recharge-go/internal/repository"
	"recharge-go/internal/router"
	"recharge-go/internal/service"
	"recharge-go/internal/service/platform"
	"recharge-go/pkg/database"
	"recharge-go/pkg/logger"
)

func main() {
	// 初始化日志
	if err := logger.InitLogger("server"); err != nil {
		panic(err)
	}
	defer logger.Close()

	// 初始化数据库
	if err := database.InitDB(); err != nil {
		logger.Error("Failed to initialize database", "error", err)
		panic(err)
	}

	// 初始化服务
	userRepo := repository.NewUserRepository(database.GetDB())
	userGradeRepo := repository.NewUserGradeRepository(database.GetDB())
	userTagRepo := repository.NewUserTagRepository(database.GetDB())
	userTagRelationRepo := repository.NewUserTagRelationRepository(database.GetDB())
	userGradeRelationRepo := repository.NewUserGradeRelationRepository(database.GetDB())
	userLogRepo := repository.NewUserLogRepository(database.GetDB())
	orderRepo := repository.NewOrderRepository(database.GetDB())
	platformRepo := repository.NewPlatformRepository(database.GetDB())

	userService := service.NewUserService(userRepo, userGradeRepo, userTagRepo, userTagRelationRepo, userGradeRelationRepo, userLogRepo)
	platformService := service.NewPlatformService(platformRepo, orderRepo)
	platformSvc := platform.NewService()

	// 初始化控制器
	platformController := controller.NewPlatformController(platformService, platformSvc)

	// 初始化统计相关依赖
	orderStatsRepo := repository.NewOrderStatisticsRepository(database.GetDB())
	statisticsService := service.NewStatisticsService(orderStatsRepo, orderRepo)
	statisticsController := controller.NewStatisticsController(statisticsService)

	// 初始化并启动统计任务
	statisticsTask := service.NewStatisticsTask(statisticsService, logger.Log)
	statisticsTask.Start()

	// 初始化路由
	r := router.SetupRouter(
		nil, // userController
		nil, // permissionController
		nil, // roleController
		nil, // productController
		userService,
		nil, // phoneLocationController
		nil, // productTypeController
		platformController,
		nil, // platformAPIController
		nil, // platformAPIParamController
		nil, // productAPIRelationController
		nil, // userLogController
		nil, // userGradeController
		nil, // rechargeHandler
		nil, // retryService
		userRepo,
		statisticsController,
	)

	// 启动服务器
	if err := r.Run(":8080"); err != nil {
		logger.Error("Failed to start server", "error", err)
		panic(err)
	}
}
