package service

import (
	"context"
    logger "recharge-go/pkg/log"
	"time"
)

// RechargeWorker 充值工作器
type RechargeWorker struct {
	rechargeService RechargeService
	stopChan        chan struct{}
	interval        time.Duration
	batchSize       int
}

// NewRechargeWorker 创建充值工作器
func NewRechargeWorker(rechargeService RechargeService, interval time.Duration, batchSize int) *RechargeWorker {
	return &RechargeWorker{
		rechargeService: rechargeService,
		stopChan:        make(chan struct{}),
		interval:        interval,
		batchSize:       batchSize,
	}
}

// Start 启动充值工作器
func (w *RechargeWorker) Start(ctx context.Context) {
	logger.WithContextCategory(ctx, "recharge_worker").Info("充值工作器启动")

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.WithContextCategory(ctx, "recharge_worker").Info("充值工作器停止")
			return
		case <-ticker.C:
			// 获取待处理的充值任务
			tasks, err := w.rechargeService.GetPendingTasks(ctx, w.batchSize)
			if err != nil {
				logger.WithContextCategory(ctx, "recharge_worker").Error("获取待处理任务失败", logger.ErrorV2(err))
				continue
			}

			if len(tasks) == 0 {
				logger.WithContextCategory(ctx, "recharge_worker").Info("没有待处理的充值任务")
				continue
			}

			logger.WithContextCategory(ctx, "recharge_worker").Info("开始处理充值任务", logger.IntV2("task_count", len(tasks)))

			// 处理每个任务
			for _, task := range tasks {
				if err := w.rechargeService.ProcessRechargeTask(ctx, task); err != nil {
					logger.WithContextCategory(ctx, "recharge_worker").Error("处理充值任务失败", logger.Int64V2("order_id", task.ID), logger.ErrorV2(err))
					continue
				}
			}

			logger.WithContextCategory(ctx, "recharge_worker").Info("本轮充值任务处理完成", logger.IntV2("processed_count", len(tasks)))
		}
	}
}

// Stop 停止工作器
func (w *RechargeWorker) Stop() {
	close(w.stopChan)
}
