package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"recharge-go/configs"
	"recharge-go/internal/repository"
	"recharge-go/internal/service"
	"recharge-go/internal/service/platform"
	"recharge-go/pkg/database"
	"recharge-go/pkg/queue"
	"recharge-go/pkg/redis"
	"syscall"
	"time"
)

func main() {
	// 解析命令行参数
	configPath := flag.String("config", "configs/config.yaml", "配置文件路径")
	flag.Parse()

	// 初始化配置
	if err := configs.Init(*configPath); err != nil {
		log.Fatalf("初始化配置失败: %v", err)
	}

	// 获取配置
	cfg := configs.GetConfig()

	// 初始化Redis连接（必须在所有依赖 Redis 的实例化之前）
	if err := redis.InitRedis(
		cfg.Redis.Host,
		cfg.Redis.Port,
		cfg.Redis.Password,
		cfg.Redis.DB,
	); err != nil {
		log.Fatalf("初始化Redis失败: %v", err)
	}

	// 初始化数据库连接
	if err := database.InitDB(); err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}

	// 获取配置
	cfg = configs.GetConfig()

	// 创建任务配置
	taskConfig := &service.TaskConfig{
		Interval:      time.Duration(cfg.Task.Interval) * time.Second,
		MaxRetries:    cfg.Task.MaxRetries,
		RetryDelay:    time.Duration(cfg.Task.RetryDelay) * time.Second,
		MaxConcurrent: cfg.Task.MaxConcurrent,
		APIKey:        cfg.API.Key,
		UserID:        cfg.API.UserID,
		BaseURL:       cfg.API.BaseURL,
	}

	// 初始化依赖
	db := database.DB
	taskConfigRepo := repository.NewTaskConfigRepository()
	taskOrderRepo := repository.NewTaskOrderRepository()
	orderRepo := repository.NewOrderRepository(db)
	platformRepo := repository.NewPlatformRepository(db)
	platformAPIRepo := repository.NewPlatformAPIRepository(db)
	callbackLogRepo := repository.NewCallbackLogRepository(db)
	productAPIRelationRepo := repository.NewProductAPIRelationRepository(db)
	platformAPIParamRepo := repository.NewPlatformAPIParamRepository(db)
	balanceLogRepo := repository.NewBalanceLogRepository(db)
	userRepo := repository.NewUserRepository(db)
	platformAccountRepo := repository.NewPlatformAccountRepository(db)
	balanceService := service.NewPlatformAccountBalanceService(
		db, platformAccountRepo, userRepo, balanceLogRepo,
	)

	notificationRepo := repository.NewNotificationRepository(db)
	queueInstance := queue.NewRedisQueue()
	productRepo := repository.NewProductRepository(db)
	rechargeService := service.NewRechargeService(
		db,
		orderRepo,
		platformRepo,
		platformAPIRepo,
		repository.NewRetryRepository(db),
		callbackLogRepo,
		productAPIRelationRepo,
		productRepo,
		platformAPIParamRepo,
		balanceService,
		notificationRepo,
		queueInstance,
	)
	orderService := service.NewOrderService(
		orderRepo,
		rechargeService,
		notificationRepo,
		queueInstance,
	)
	daichongOrderRepo := repository.NewDaichongOrderRepository(db)
	tokenRepo := repository.NewPlatformTokenRepository()
	platformSvc := platform.NewService(tokenRepo, platformRepo)

	// 创建任务服务
	taskSvc := service.NewTaskService(
		taskConfigRepo,
		taskOrderRepo,
		orderRepo,
		daichongOrderRepo,
		platformSvc,
		orderService,
		taskConfig,
		platformAccountRepo,
	)

	// 启动任务
	taskSvc.StartTask()
	log.Println("任务服务已启动")

	// 等待信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// 优雅关闭
	log.Println("正在关闭任务服务...")
	taskSvc.StopTask()
	log.Println("任务服务已关闭")
}
