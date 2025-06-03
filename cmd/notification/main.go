package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"recharge-go/internal/config"
	"recharge-go/internal/repository"
	notificationRepo "recharge-go/internal/repository/notification"
	"recharge-go/internal/service"
	notificationService "recharge-go/internal/service/notification"
	"recharge-go/internal/task"
	"recharge-go/pkg/database"
	"recharge-go/pkg/logger"
	"recharge-go/pkg/queue"
	"recharge-go/pkg/redis"
	"syscall"

	"go.uber.org/zap"
)

func main() {
	// 初始化日志
	if err := logger.InitLogger("notification"); err != nil {
		panic(fmt.Sprintf("初始化日志失败: %v", err))
	}
	defer logger.Close()

	// 加载配置
	cfg, err := config.LoadConfig("configs/config.yaml")
	if err != nil {
		logger.Log.Fatal("加载配置失败", zap.Error(err))
	}

	// 初始化数据库连接
	if err := database.InitDB(); err != nil {
		logger.Log.Fatal("初始化数据库失败", zap.Error(err))
	}

	// 初始化Redis连接
	if err := redis.InitRedis(
		cfg.Redis.Host,
		cfg.Redis.Port,
		cfg.Redis.Password,
		cfg.Redis.DB,
	); err != nil {
		logger.Log.Fatal("初始化Redis失败", zap.Error(err))
	}

	// 创建仓储实例
	recordRepo := notificationRepo.NewRepository(database.DB)
	platformRepo := repository.NewPlatformRepository(database.DB)
	orderRepo := repository.NewOrderRepository(database.DB)

	// 创建队列实例
	queueInstance := queue.NewRedisQueue()

	// 创建服务实例
	notificationService := notificationService.NewNotificationService(recordRepo, queueInstance)
	platformService := service.NewPlatformService(platformRepo, orderRepo)

	// 创建通知任务处理器
	notificationTask := task.NewNotificationTask(
		notificationService,
		platformService,
		queueInstance,
		cfg.Notification.MaxRetries,
	)

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动通知任务处理器
	go func() {
		if err := notificationTask.Start(ctx); err != nil {
			logger.Log.Error("通知任务处理器启动失败", zap.Error(err))
			cancel()
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// 停止通知任务处理器
	notificationTask.Stop()

	// 关闭Redis连接
	if err := redis.Close(); err != nil {
		logger.Log.Error("关闭Redis连接失败", zap.Error(err))
	}

	logger.Log.Info("通知服务已关闭")
}
