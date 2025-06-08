package task

import (
	"context"
	"encoding/json"
	model "recharge-go/internal/model/notification"
	"recharge-go/internal/service"
	svc "recharge-go/internal/service/notification"
	"recharge-go/pkg/logger"
	"recharge-go/pkg/queue"
	"strings"
	"time"
)

// NotificationTask 通知任务处理器
type NotificationTask struct {
	notificationService svc.NotificationService
	platformService     *service.PlatformService
	queue               queue.Queue
	queueName           string
	maxRetries          int
	batchSize           int
	workerCount         int                            // 工作协程数量
	jobChan             chan *model.NotificationRecord // 任务通道
}

// NewNotificationTask 创建通知任务处理器
func NewNotificationTask(
	notificationService svc.NotificationService,
	platformService *service.PlatformService,
	queue queue.Queue,
	maxRetries int,
) *NotificationTask {
	return &NotificationTask{
		notificationService: notificationService,
		platformService:     platformService,
		queue:               queue,
		queueName:           "notification_queue",
		maxRetries:          maxRetries,
		batchSize:           10,                                        // 每次处理的通知数量
		workerCount:         1,                                         // 默认5个工作协程
		jobChan:             make(chan *model.NotificationRecord, 100), // 任务通道缓冲区大小100
	}
}

// Start 启动通知任务处理器
func (t *NotificationTask) Start(ctx context.Context) error {
	logger.Info("starting notification task processor")

	// 启动工作协程池
	for i := 0; i < t.workerCount; i++ {
		go t.worker(ctx, i)
	}

	// 启动重试任务（可选，混合模式下重试直接由worker处理）
	// go t.startRetryTask(ctx)

	for {
		select {
		case <-ctx.Done():
			logger.Info("notification task processor stopped")
			return nil
		default:
			// 从队列Pop一个通知
			value, err := t.queue.Pop(ctx, t.queueName)
			if err != nil {
				logger.Error("Pop notification from queue failed", "error", err)
				time.Sleep(time.Second)
				continue
			}
			if value == nil {
				time.Sleep(2 * time.Second)
				continue
			}
			// 解析通知记录
			var record model.NotificationRecord
			switch v := value.(type) {
			case string:
				if err := json.Unmarshal([]byte(v), &record); err != nil {
					logger.Error("队列值反序列化失败", "error", err, "raw", v)
					continue
				}
			case []byte:
				if err := json.Unmarshal(v, &record); err != nil {
					logger.Error("队列值反序列化失败", "error", err, "raw", string(v))
					continue
				}
			default:
				logger.Error("队列值类型错误", "type", value)
				continue
			}
			// 分发到worker
			select {
			case t.jobChan <- &record:
				logger.Info("通知已分发到工作协程", "notification_id", record.ID, "order_id", record.OrderID, "retry_count", record.RetryCount, "platform_code", record.PlatformCode)
			case <-ctx.Done():
				return nil
			}
		}
	}
}

// worker 工作协程
func (t *NotificationTask) worker(ctx context.Context, id int) {
	logger.Info("worker started", "worker_id", id)
	for {
		select {
		case <-ctx.Done():
			logger.Info("worker stopped", "worker_id", id)
			return
		case record := <-t.jobChan:
			if err := t.processSingleNotification(ctx, record, id); err != nil {
				logger.Error("process notification failed",
					"error", err,
					"worker_id", id,
					"notification_id", record.ID,
					"order_id", record.OrderID,
					"retry_count", record.RetryCount,
					"platform_code", record.PlatformCode,
				)
			}
		}
	}
}

// processSingleNotification 处理单个通知
func (t *NotificationTask) processSingleNotification(ctx context.Context, record *model.NotificationRecord, workerID int) error {
	// 处理前查数据库最新状态，只有status=1才处理
	dbRecord, err := t.notificationService.GetNotification(ctx, record.ID)
	if err != nil {
		logger.Error("获取通知记录失败",
			"error", err,
			"notification_id", record.ID,
			"order_id", record.OrderID,
			"retry_count", record.RetryCount,
			"platform_code", record.PlatformCode,
		)
		return err
	}
	if dbRecord.Status != 1 {
		logger.Info("通知已被处理，跳过", "notification_id", record.ID, "order_id", record.OrderID, "status", dbRecord.Status)
		return nil
	}
	// 获取订单信息
	order, err := t.platformService.GetOrder(ctx, dbRecord.OrderID)
	if err != nil {
		logger.Error("获取订单信息失败",
			"error", err,
			"notification_id", dbRecord.ID,
			"order_id", dbRecord.OrderID,
			"retry_count", dbRecord.RetryCount,
			"platform_code", dbRecord.PlatformCode,
		)
		// 如果是 record not found，可以直接标记为失败，避免无意义重试
		if strings.Contains(err.Error(), "record not found") {
			err2 := t.notificationService.UpdateNotificationStatus(ctx, dbRecord.ID, 3)
			if err2 != nil {
				logger.Error("更新通知状态失败", "error", err2, "notification_id", dbRecord.ID, "order_id", dbRecord.OrderID, "retry_count", dbRecord.RetryCount, "platform_code", dbRecord.PlatformCode)
			}
			logger.Info("订单不存在，通知已标记为失败", "notification_id", dbRecord.ID, "order_id", dbRecord.OrderID, "retry_count", dbRecord.RetryCount, "platform_code", dbRecord.PlatformCode)
		}
		return err
	}
	// 发送通知
	if err := t.platformService.SendNotification(ctx, order); err != nil {
		// 记录通知发送失败的详细错误信息
		logger.Error("通知发送失败",
			"error", err,
			"notification_id", dbRecord.ID,
			"order_id", dbRecord.OrderID,
			"order_number", order.OrderNumber,
			"platform_code", dbRecord.PlatformCode,
			"notification_type", dbRecord.NotificationType,
			"retry_count", dbRecord.RetryCount,
			"callback_url", order.PlatformCallbackURL,
		)

		// 业务终态错误关键字
		if strings.Contains(err.Error(), "此订单已做单失败") {
			logger.Error("遇到终态业务错误，不再重试",
				"error", err,
				"notification_id", dbRecord.ID,
				"order_id", dbRecord.OrderID,
				"order_number", order.OrderNumber,
				"platform_code", dbRecord.PlatformCode,
				"notification_type", dbRecord.NotificationType,
				"retry_count", dbRecord.RetryCount,
			)
			// 标记为失败
			if err := t.notificationService.UpdateNotificationStatus(ctx, dbRecord.ID, 3); err != nil {
				logger.Error("更新通知状态失败", "error", err, "notification_id", dbRecord.ID, "order_id", dbRecord.OrderID, "order_number", order.OrderNumber, "retry_count", dbRecord.RetryCount, "platform_code", dbRecord.PlatformCode)
			}
			return nil
		}
		// 如果处理失败且未超过最大重试次数，则重试
		if dbRecord.RetryCount < t.maxRetries {
			// 使用指数退避策略计算重试间隔
			retryInterval := time.Duration(1<<uint(dbRecord.RetryCount)) * time.Minute
			nextRetryTime := time.Now().Add(retryInterval)
			logger.Info("准备重试通知",
				"notification_id", dbRecord.ID,
				"order_id", dbRecord.OrderID,
				"order_number", order.OrderNumber,
				"retry_count", dbRecord.RetryCount,
				"next_retry_time", nextRetryTime,
				"retry_interval", retryInterval,
				"platform_code", dbRecord.PlatformCode,
			)
			// 更新通知记录状态和重试时间
			if err := t.notificationService.RetryFailedNotification(ctx, dbRecord.ID); err != nil {
				logger.Error("重试通知失败",
					"error", err,
					"notification_id", dbRecord.ID,
					"order_id", dbRecord.OrderID,
					"order_number", order.OrderNumber,
					"retry_count", dbRecord.RetryCount,
					"platform_code", dbRecord.PlatformCode,
				)
				return err
			}
			// 重新入队
			if err := t.queue.Push(ctx, t.queueName, dbRecord); err != nil {
				logger.Error("重新入队失败",
					"error", err,
					"notification_id", dbRecord.ID,
					"order_id", dbRecord.OrderID,
					"order_number", order.OrderNumber,
					"retry_count", dbRecord.RetryCount,
					"platform_code", dbRecord.PlatformCode,
					"queue_name", t.queueName,
				)
				return err
			}
			logger.Info("通知已重新入队",
				"notification_id", dbRecord.ID,
				"order_id", dbRecord.OrderID,
				"order_number", order.OrderNumber,
				"retry_count", dbRecord.RetryCount,
				"queue_name", t.queueName,
				"next_retry_time", nextRetryTime,
				"platform_code", dbRecord.PlatformCode,
			)
		} else {
			logger.Info("通知已达到最大重试次数，不再重试",
				"notification_id", dbRecord.ID,
				"order_id", dbRecord.OrderID,
				"order_number", order.OrderNumber,
				"retry_count", dbRecord.RetryCount,
				"max_retries", t.maxRetries,
				"platform_code", dbRecord.PlatformCode,
			)
			// 更新通知状态为失败
			if err := t.notificationService.UpdateNotificationStatus(ctx, dbRecord.ID, 3); err != nil {
				logger.Error("更新通知状态失败",
					"error", err,
					"notification_id", dbRecord.ID,
					"order_id", dbRecord.OrderID,
					"order_number", order.OrderNumber,
					"retry_count", dbRecord.RetryCount,
					"platform_code", dbRecord.PlatformCode,
				)
				return err
			}
			logger.Info("通知已从队列中移除",
				"notification_id", dbRecord.ID,
				"order_id", dbRecord.OrderID,
				"order_number", order.OrderNumber,
				"retry_count", dbRecord.RetryCount,
				"queue_name", t.queueName,
				"platform_code", dbRecord.PlatformCode,
			)
		}
	} else {
		logger.Info("发送通知成功",
			"notification_id", dbRecord.ID,
			"order_id", dbRecord.OrderID,
			"order_number", order.OrderNumber,
			"platform_code", dbRecord.PlatformCode,
			"notification_type", dbRecord.NotificationType,
			"retry_count", dbRecord.RetryCount,
		)
		// 更新通知状态为成功
		if err := t.notificationService.UpdateNotificationStatus(ctx, dbRecord.ID, 3); err != nil {
			logger.Error("更新通知状态失败",
				"error", err,
				"notification_id", dbRecord.ID,
				"order_id", dbRecord.OrderID,
				"order_number", order.OrderNumber,
				"retry_count", dbRecord.RetryCount,
				"platform_code", dbRecord.PlatformCode,
			)
			return err
		}
		logger.Info("通知处理成功",
			"notification_id", dbRecord.ID,
			"order_id", dbRecord.OrderID,
			"order_number", order.OrderNumber,
			"platform_code", dbRecord.PlatformCode,
			"notification_type", dbRecord.NotificationType,
			"retry_count", dbRecord.RetryCount,
		)
	}
	return nil
}

// processNotifications 处理通知
func (t *NotificationTask) processNotifications(ctx context.Context) (bool, error) {
	// 批量获取待处理的通知
	records, _, err := t.notificationService.ListNotifications(ctx, map[string]interface{}{
		"status": 1, // 待处理状态
	}, 1, t.batchSize)
	if err != nil {
		logger.Error("获取待处理通知失败", "error", err)
		return false, err
	}

	if len(records) == 0 {
		// logger.Info("没有待处理的通知") // 可以注释掉或降低为 debug 级别
		return false, nil
	}

	// 分发任务到工作协程
	for _, record := range records {
		select {
		case t.jobChan <- record:
			logger.Info("通知已分发到工作协程",
				"notification_id", record.ID,
				"order_id", record.OrderID,
				"retry_count", record.RetryCount,
				"platform_code", record.PlatformCode,
			)
		case <-ctx.Done():
			return false, ctx.Err()
		}
	}

	return true, nil
}

// startRetryTask 启动重试任务
func (t *NotificationTask) startRetryTask(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// 获取所有待重试的通知记录
			records, _, err := t.notificationService.ListNotifications(ctx, map[string]interface{}{
				"status": 4, // 失败状态
			}, 1, t.batchSize)
			if err != nil {
				logger.Error("get failed notifications failed", "error", err)
				continue
			}

			// 分发重试任务到工作协程
			for _, record := range records {
				if record.RetryCount < t.maxRetries {
					select {
					case t.jobChan <- record:
						logger.Info("重试通知已分发到工作协程",
							"notification_id", record.ID,
							"order_id", record.OrderID,
							"retry_count", record.RetryCount,
							"platform_code", record.PlatformCode,
						)
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}
}

// Stop 停止通知任务处理器
func (t *NotificationTask) Stop() {
	// 清理资源
	close(t.jobChan)
	logger.Info("notification task processor stopped")
}
