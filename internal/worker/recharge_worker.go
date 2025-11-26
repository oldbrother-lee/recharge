package worker

import (
	"context"
    "recharge-go/internal/service"
    logger "recharge-go/pkg/log"
	"time"
)

// RechargeWorker 充值工作器
type RechargeWorker struct {
	rechargeService service.RechargeService
	stopChan        chan struct{}
}

// NewRechargeWorker 创建充值工作器
func NewRechargeWorker(rechargeService service.RechargeService) *RechargeWorker {
	return &RechargeWorker{
		rechargeService: rechargeService,
		stopChan:        make(chan struct{}),
	}
}

// Start 启动工作器
func (w *RechargeWorker) Start() {
	logger.InfoV2("充值工作器启动")
	go w.processQueue()
	go w.checkRechargingOrders()
}

// Stop 停止工作器
func (w *RechargeWorker) Stop() {
	logger.InfoV2("充值工作器停止")
	close(w.stopChan)
}

// processQueue 处理队列
func (w *RechargeWorker) processQueue() {
	// 使用可取消的上下文，随 stopChan 关闭而取消
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		select {
		case <-w.stopChan:
			cancel()
		case <-ctx.Done():
		}
	}()

	for {
		select {
		case <-w.stopChan:
			return
		default:
			// 从队列中获取订单ID
			orderID, err := w.rechargeService.PopFromRechargeQueue(ctx)
			if err != nil {
				logger.ErrorLogV2("从队列获取订单失败", logger.ErrorV2(err))
				// time.Sleep(time.Second)
				select {
				case <-w.stopChan:
					return
				case <-time.After(time.Second):
				}
				continue
			}

			// 获取订单信息
			order, err := w.rechargeService.GetOrderByID(ctx, orderID)
			if err != nil {
				logger.ErrorLogV2("获取订单信息失败", logger.Int64V2("order_id", orderID), logger.ErrorV2(err))
				continue
			}

			// 处理充值任务
			if err := w.rechargeService.ProcessRechargeTask(ctx, order); err != nil {
				logger.ErrorLogV2("处理充值任务失败", logger.Int64V2("order_id", orderID), logger.ErrorV2(err))
				// 如果处理失败，将订单重新放入队列
				if err := w.rechargeService.PushToRechargeQueue(ctx, orderID); err != nil {
					logger.ErrorLogV2("重新放入队列失败", logger.Int64V2("order_id", orderID), logger.ErrorV2(err))
				}
				// time.Sleep(time.Second)
				select {
				case <-w.stopChan:
					return
				case <-time.After(time.Second):
				}
				continue
			}

			logger.InfoV2("充值任务处理完成", logger.Int64V2("order_id", orderID))
		}
	}
}

// checkRechargingOrders 定期检查充值中订单
func (w *RechargeWorker) checkRechargingOrders() {
	logger.InfoV2("【充值中订单检查器】启动定时检查任务", logger.StringV2("interval", "3分钟"))
	ticker := time.NewTicker(3 * time.Minute)
	defer ticker.Stop()

	// 使用可取消的上下文，随 stopChan 关闭而取消
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		select {
		case <-w.stopChan:
			cancel()
		case <-ctx.Done():
		}
	}()

	for {
		select {
		case <-w.stopChan:
			logger.InfoV2("【充值中订单检查器】收到停止信号，检查任务停止")
			return
		case <-ticker.C:
			logger.InfoV2("【充值中订单检查器】定时器触发，开始新一轮检查")
			if err := w.rechargeService.CheckRechargingOrders(ctx); err != nil {
				logger.ErrorLogV2("【充值中订单检查器】检查失败", logger.ErrorV2(err))
			}
		}
	}
}
