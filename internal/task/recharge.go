package task

import (
	"context"
	"recharge-go/internal/service"
	logger "recharge-go/pkg/log"
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
	logger.InfoV2("recharge_task_started")

	// 启动主处理循环
	for {
		select {
		case <-ctx.Done():
			logger.InfoV2("recharge_task_stopped")
			return nil
		default:
			// 从充值队列获取订单
			orderID, err := t.rechargeService.PopFromRechargeQueue(ctx)
			if err != nil {
				logger.ErrorLogV2("recharge_queue_pop_failed", logger.ErrorV2(err))
				// time.Sleep(time.Second) // 发生错误时暂停一秒
				select {
				case <-ctx.Done():
					logger.InfoV2("recharge_task_stopped")
					return nil
				case <-time.After(time.Second):
				}
				continue
			}

			if orderID == 0 {
				// 如果队列为空，休眠 5 秒
				logger.DebugV2("recharge_queue_empty_waiting")
				// time.Sleep(5 * time.Second)
				select {
				case <-ctx.Done():
					logger.InfoV2("recharge_task_stopped")
					return nil
				case <-time.After(5 * time.Second):
				}
				continue
			}

			logger.InfoV2("recharge_queue_popped", logger.Int64V2("order_id", orderID))

			// 获取订单信息
			order, err := t.rechargeService.GetOrderByID(ctx, orderID)
			if err != nil {
				logger.ErrorLogV2("get_order_failed", logger.ErrorV2(err), logger.Int64V2("order_id", orderID))
				// 从处理中队列移除
				if err := t.rechargeService.RemoveFromProcessingQueue(ctx, orderID); err != nil {
					logger.ErrorLogV2("processing_queue_remove_failed", logger.ErrorV2(err), logger.Int64V2("order_id", orderID))
				}
				continue
			}

			logger.InfoV2("get_order_success", logger.Int64V2("order_id", orderID), logger.StringV2("order_number", order.OrderNumber), logger.IntV2("status", int(order.Status)))

			// 处理充值任务
			if err := t.rechargeService.ProcessRechargeTask(ctx, order); err != nil {
				logger.ErrorLogV2("process_recharge_task_failed", logger.ErrorV2(err), logger.Int64V2("order_id", orderID), logger.StringV2("order_number", order.OrderNumber))
				// 从处理中队列移除
				if err := t.rechargeService.RemoveFromProcessingQueue(ctx, orderID); err != nil {
					logger.ErrorLogV2("processing_queue_remove_failed", logger.ErrorV2(err), logger.Int64V2("order_id", orderID))
				}
				continue
			}

			logger.InfoV2("process_recharge_task_success", logger.Int64V2("order_id", orderID), logger.StringV2("order_number", order.OrderNumber))
		}
	}
}

// Stop 停止充值任务处理器
func (t *RechargeTask) Stop() {
	// 清理资源
	logger.InfoV2("recharge_task_stopped")
}
