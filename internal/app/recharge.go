package app

import (
	"context"
	"log"
	"time"

	"recharge-go/internal/handler"
	"recharge-go/internal/service"
)

// RechargeApp 充值应用
type RechargeApp struct {
	container       *Container
	retryHandler    *handler.RetryHandler
	rechargeHandler *handler.RechargeHandler
	rechargeWorker  *service.RechargeWorker
	ctx             context.Context
	cancel          context.CancelFunc
}

// NewRechargeApp 创建新的充值应用
func NewRechargeApp(container *Container) *RechargeApp {
	return &RechargeApp{
		container: container,
	}
}

// Start 启动充值工作器
func (r *RechargeApp) Initialize() error {
	// 创建重试处理器
	r.retryHandler = handler.NewRetryHandler(
		r.container.GetServices().Retry,
	)

	// 创建充值处理器
	r.rechargeHandler = handler.NewRechargeHandler(
		r.container.GetServices().Recharge,
	)

	return nil
}

// Start 启动充值应用
func (r *RechargeApp) Start(ctx context.Context) error {
	if err := r.Initialize(); err != nil {
		return err
	}

	// 获取配置中的批量处理数量，如果未配置或为0则使用默认值10
	batchSize := r.container.GetConfig().Task.BatchSize
	if batchSize <= 0 {
		batchSize = 10
	}

	// 创建充值工作器
	r.rechargeWorker = service.NewRechargeWorker(
		r.container.GetServices().Recharge,
		time.Second*5, // 每5秒检查一次
		batchSize,     // 从配置读取批量处理数量
	)

	// 创建上下文
	r.ctx, r.cancel = context.WithCancel(ctx)

	// 启动充值工作器
	go r.rechargeWorker.Start(r.ctx)

	log.Println("充值应用启动成功")
	return nil
}

// Stop 停止充值应用
func (r *RechargeApp) Stop(ctx context.Context) error {
	log.Println("正在停止充值工作器...")

	// 停止充值工作器
	if r.cancel != nil {
		r.cancel()
	}

	// 关闭容器资源
	return r.container.Close()
}
