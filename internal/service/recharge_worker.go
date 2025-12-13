package service

import (
	"context"
	logger "recharge-go/pkg/log"
	"sync"
	"time"
)

// RechargeWorker 充值工作器
type RechargeWorker struct {
	rechargeService RechargeService
	interval        time.Duration // 用于 BRPOP 的超时时间
	concurrency     int           // 并发工作的协程数量
}

// NewRechargeWorker 创建充值工作器
func NewRechargeWorker(rechargeService RechargeService, interval time.Duration, concurrency int) *RechargeWorker {
	return &RechargeWorker{
		rechargeService: rechargeService,
		interval:        interval,
		concurrency:     concurrency,
	}
}

// Start 启动充值工作器
func (w *RechargeWorker) Start(ctx context.Context) {
	logger.WithContextCategory(ctx, "recharge_worker").Info("充值工作器启动",
		logger.IntV2("concurrency", w.concurrency),
		logger.StringV2("mode", "blocking_pop"))

	var wg sync.WaitGroup

	for i := 0; i < w.concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			logger.WithContextCategory(ctx, "recharge_worker").Debug("Worker启动", logger.IntV2("worker_id", workerID))

			for {
				select {
				case <-ctx.Done():
					return
				default:
					// 阻塞获取任务
					orderID, err := w.rechargeService.PopFromRechargeQueueBlocking(ctx, w.interval)
					if err != nil {
						// 错误已在 Service 层记录，这里只需判断是否需要继续
						// 如果是超时（err==nil且orderID==0），继续循环
						// 如果是真正的错误，可能需要简单的退避
						if orderID == 0 {
							continue
						}
						time.Sleep(time.Second) // 发生错误时简单退避
						continue
					}

					if orderID == 0 {
						continue
					}

					// 获取订单详情
					order, err := w.rechargeService.GetOrderByID(ctx, orderID)
					if err != nil {
						logger.WithContextCategory(ctx, "recharge_worker").Error("获取订单详情失败",
							logger.Int64V2("order_id", orderID),
							logger.ErrorV2(err))
						continue
					}

					if order == nil {
						logger.WithContextCategory(ctx, "recharge_worker").Error("订单不存在", logger.Int64V2("order_id", orderID))
						continue
					}

					// 处理任务
					if err := w.rechargeService.ProcessRechargeTask(ctx, order); err != nil {
						logger.WithContextCategory(ctx, "recharge_worker").Error("处理充值任务失败",
							logger.Int64V2("order_id", orderID),
							logger.ErrorV2(err))
					}
				}
			}
		}(i)
	}

	// 等待上下文取消
	<-ctx.Done()
	logger.WithContextCategory(ctx, "recharge_worker").Info("充值工作器正在停止...")
	wg.Wait()
	logger.WithContextCategory(ctx, "recharge_worker").Info("充值工作器已停止")
}

// Stop 停止工作器
func (w *RechargeWorker) Stop() {
	// 空实现，停止通过 context 控制
}
