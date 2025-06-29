// cmd/worker/main.go - 充值工作器启动文件
package main

import (
	"context"
	"os"
	"os/signal"
	"recharge-go/internal/config"
	"recharge-go/internal/repository"
	"recharge-go/internal/repository/notification"
	"recharge-go/internal/service"
	"recharge-go/internal/service/recharge"
	"recharge-go/internal/task"
	"recharge-go/pkg/database"
	"recharge-go/pkg/lock"
	"recharge-go/pkg/logger"
	"recharge-go/pkg/queue"
	"recharge-go/pkg/redis"
	"syscall"
)

func main() {
	// 初始化配置
	cfg := config.GetConfig()

	// 初始化日志
	if err := logger.InitLogger("recharge"); err != nil {
		panic(err)
	}

	// 初始化数据库连接
	if err := database.InitDB(); err != nil {
		panic(err)
	}
	db := database.DB

	// 初始化Redis连接
	if err := redis.InitRedis(cfg.Redis.Host, cfg.Redis.Port, cfg.Redis.Password, cfg.Redis.DB); err != nil {
		panic(err)
	}

	// 初始化仓库
	orderRepo := repository.NewOrderRepository(db)
	platformRepo := repository.NewPlatformRepository(db)
	productAPIRelationRepo := repository.NewProductAPIRelationRepository(db)
	callbackLogRepo := repository.NewCallbackLogRepository(db)
	retryRepo := repository.NewRetryRepository(db)
	platformAPIParamRepo := repository.NewPlatformAPIParamRepository(db)

	platformAccountRepo := repository.NewPlatformAccountRepository(db)
	userRepo := repository.NewUserRepository(db)
	balanceLogRepo := repository.NewBalanceLogRepository(db)
	productRepo := repository.NewProductRepository(db)

	// 初始化平台管理器
	platformManager := recharge.NewManager(db)
	if err := platformManager.LoadPlatforms(); err != nil {
		panic(err)
	}

	// 初始化队列
	queue := queue.NewRedisQueue()

	// 初始化余额服务
	balanceService := service.NewPlatformAccountBalanceService(
		db,
		platformAccountRepo,
		userRepo,
		balanceLogRepo,
	)

	// 初始化用户余额服务
	userBalanceService := service.NewBalanceService(balanceLogRepo, userRepo)

	// 初始化平台API仓库
	platformAPIRepo := repository.NewPlatformAPIRepository(db)

	// 初始化通知仓库
	notificationRepoInstance := notification.NewRepository(db)

	// 初始化分布式锁管理器
	distributedLock := lock.NewRedisDistributedLock(redis.GetClient())
	refundLockManager := lock.NewRefundLockManager(distributedLock)

	// 初始化平台账户余额服务
	platformAccountBalanceService := service.NewPlatformAccountBalanceService(db, platformAccountRepo, userRepo, balanceLogRepo)

	// 初始化统一退款服务
	unifiedRefundService := service.NewUnifiedRefundService(db, userRepo, orderRepo, balanceLogRepo, refundLockManager, userBalanceService, platformAccountBalanceService)

	// 初始化服务
	orderService := service.NewOrderService(orderRepo, balanceLogRepo, userRepo, nil, unifiedRefundService, refundLockManager, notificationRepoInstance, queue, db)
	rechargeService := service.NewRechargeService(
		db,
		orderRepo,
		platformRepo,
		platformAPIRepo,
		retryRepo,
		callbackLogRepo,
		productAPIRelationRepo,
		productRepo,
		platformAPIParamRepo,
		balanceService,
		userBalanceService,
		notificationRepoInstance,
		queue,
	)
	retryService := service.NewRetryService(retryRepo, orderRepo, platformRepo, productRepo, productAPIRelationRepo, rechargeService, orderService)

	// 设置充值服务
	orderService.SetRechargeService(rechargeService)

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 初始化任务
	retryTask := task.NewRetryTask(retryService)
	rechargeTask := task.NewRechargeTask(rechargeService)

	// 启动任务
	go retryTask.Start()
	go rechargeTask.Start(ctx)

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// 停止任务
	retryTask.Stop()
	rechargeTask.Stop()
}
