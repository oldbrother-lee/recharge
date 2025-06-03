package task

import (
	"context"
	"recharge-go/internal/service"
	"recharge-go/pkg/logger"
	"time"
)

// RechargeTask 充值任务处理器
type RechargeTask struct {
	rechargeService service.RechargeService
}

// NewRechargeTask 创建充值任务处理器
func NewRechargeTask(rechargeService service.RechargeService) *RechargeTask {
	return &RechargeTask{
		rechargeService: rechargeService,
	}
}

// Start 启动充值任务处理器
func (t *RechargeTask) Start(ctx context.Context) error {
	logger.Info("【充值任务处理器启动】")

	// 启动主处理循环
	for {
		select {
		case <-ctx.Done():
			logger.Info("【充值任务处理器停止】")
			return nil
		default:
			// 从充值队列获取订单
			orderID, err := t.rechargeService.PopFromRechargeQueue(ctx)
			if err != nil {
				logger.Error("【从充值队列获取订单失败】",
					"error", err)
				time.Sleep(time.Second) // 发生错误时暂停一秒
				continue
			}

			if orderID == 0 {
				// 如果队列为空，休眠 5 秒
				logger.Debug("【充值队列为空，等待5秒】")
				time.Sleep(5 * time.Second)
				continue
			}

			logger.Info("【从充值队列获取到订单】",
				"order_id", orderID)

			// 获取订单信息
			order, err := t.rechargeService.GetOrderByID(ctx, orderID)
			if err != nil {
				logger.Error("【获取订单信息失败】",
					"error", err,
					"order_id", orderID)
				// 从处理中队列移除
				if err := t.rechargeService.RemoveFromProcessingQueue(ctx, orderID); err != nil {
					logger.Error("【从处理队列移除失败】",
						"error", err,
						"order_id", orderID)
				}
				continue
			}

			logger.Info("【获取订单信息成功】",
				"order_id", orderID,
				"order_number", order.OrderNumber,
				"status", order.Status)

			// 处理充值任务
			if err := t.rechargeService.ProcessRechargeTask(ctx, order); err != nil {
				logger.Error("【处理充值任务失败】",
					"error", err,
					"order_id", orderID,
					"order_number", order.OrderNumber)
				// 从处理中队列移除
				if err := t.rechargeService.RemoveFromProcessingQueue(ctx, orderID); err != nil {
					logger.Error("【从处理队列移除失败】",
						"error", err,
						"order_id", orderID)
				}
				continue
			}

			logger.Info("【充值任务处理成功】",
				"order_id", orderID,
				"order_number", order.OrderNumber)
		}
	}
}

// Stop 停止充值任务处理器
func (t *RechargeTask) Stop() {
	// 清理资源
	logger.Info("【充值任务处理器停止】")
}
