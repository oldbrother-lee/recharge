package task

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"go.uber.org/zap"

	"recharge-go/internal/model"
	notificationModel "recharge-go/internal/model/notification"
	"recharge-go/pkg/queue"
	notificationService "recharge-go/internal/service/notification"
	"recharge-go/internal/service"
)

// NotificationTask 通知任务处理器
type NotificationTask struct {
	notificationService notificationService.NotificationService
	platformService     *service.PlatformService
	queue               queue.Queue
	queueName           string
	maxRetries          int
	batchSize           int
	workerCount         int                                      // 工作协程数量
	jobChan             chan *notificationModel.NotificationRecord // 任务通道
	logger              *zap.Logger
}

// NewNotificationTask 创建通知任务处理器
func NewNotificationTask(
	notificationService notificationService.NotificationService,
	platformService *service.PlatformService,
	queue queue.Queue,
	maxRetries int,
	logger *zap.Logger,
) *NotificationTask {
	return &NotificationTask{
		notificationService: notificationService,
		platformService:     platformService,
		queue:               queue,
		queueName:           "notification_queue",
		maxRetries:          maxRetries,
		batchSize:           10,                                              // 每次处理的通知数量
		workerCount:         1,                                               // 默认5个工作协程
		jobChan:             make(chan *notificationModel.NotificationRecord, 100), // 任务通道缓冲区大小100
		logger:              logger,
	}
}

// Start 启动通知任务处理器
func (t *NotificationTask) Start(ctx context.Context) error {
	t.logger.Info("starting notification task processor")

	// 启动工作协程池
	for i := 0; i < t.workerCount; i++ {
		go t.worker(ctx, i)
	}

	// 启动重试任务（可选，混合模式下重试直接由worker处理）
	// go t.startRetryTask(ctx)

	for {
		select {
		case <-ctx.Done():
			t.logger.Info("notification task processor stopped")
			return nil
		default:
			// 从队列Pop一个通知
			value, err := t.queue.Pop(ctx, t.queueName)
			if err != nil {
				t.logger.Error("Pop notification from queue failed", zap.Error(err))
				time.Sleep(time.Second)
				continue
			}
			if value == nil {
				time.Sleep(2 * time.Second)
				continue
			}
			// 解析通知记录
			var record notificationModel.NotificationRecord
			switch v := value.(type) {
			case string:
				if err := json.Unmarshal([]byte(v), &record); err != nil {
					t.logger.Error("队列值反序列化失败", zap.Error(err), zap.String("raw", v))
					continue
				}
			case []byte:
				if err := json.Unmarshal(v, &record); err != nil {
					t.logger.Error("队列值反序列化失败", zap.Error(err), zap.String("raw", string(v)))
					continue
				}
			default:
				t.logger.Error("队列值类型错误", zap.Any("type", value))
				continue
			}
			// 分发到worker
			select {
			case t.jobChan <- &record:
				t.logger.Info("通知已分发到工作协程", zap.Int64("notification_id", record.ID), zap.Int64("order_id", record.OrderID), zap.Int("retry_count", record.RetryCount), zap.String("platform_code", record.PlatformCode))
			case <-ctx.Done():
				return nil
			}
		}
	}
}

// worker 工作协程
func (t *NotificationTask) worker(ctx context.Context, id int) {
	t.logger.Info("worker started", zap.Int("worker_id", id))
	for {
		select {
		case <-ctx.Done():
			t.logger.Info("worker stopped", zap.Int("worker_id", id))
			return
		case record := <-t.jobChan:
			if err := t.processSingleNotification(ctx, record, id); err != nil {
				t.logger.Error("process notification failed",
					zap.Error(err),
					zap.Int("worker_id", id),
					zap.Int64("notification_id", record.ID),
					zap.Int64("order_id", record.OrderID),
					zap.Int("retry_count", record.RetryCount),
					zap.String("platform_code", record.PlatformCode),
				)
			}
		}
	}
}

// processSingleNotification 处理单个通知
func (t *NotificationTask) processSingleNotification(ctx context.Context, record *notificationModel.NotificationRecord, workerID int) error {
	// 处理前查数据库最新状态，只有status=1才处理
	dbRecord, err := t.notificationService.GetNotification(ctx, record.ID)
	if err != nil {
		t.logger.Error("获取通知记录失败",
			zap.Error(err),
			zap.Int64("notification_id", record.ID),
			zap.Int64("order_id", record.OrderID),
			zap.Int("retry_count", record.RetryCount),
			zap.String("platform_code", record.PlatformCode),
		)
		return err
	}
	if dbRecord.Status != 1 {
		t.logger.Info("通知状态不是待处理，跳过",
			zap.Int64("notification_id", dbRecord.ID),
			zap.Int64("order_id", dbRecord.OrderID),
			zap.Int("status", dbRecord.Status),
			zap.Int("retry_count", dbRecord.RetryCount),
			zap.String("platform_code", dbRecord.PlatformCode),
		)
		return nil
	}
	// 获取订单信息：优先使用快照数据，如果没有快照则查询数据库
	var order *model.Order
	if dbRecord.OrderSnapshot != "" {
		// 使用快照数据
		order = &model.Order{}
		if err := json.Unmarshal([]byte(dbRecord.OrderSnapshot), order); err != nil {
			t.logger.Error("反序列化订单快照失败",
				zap.Error(err),
				zap.Int64("notification_id", dbRecord.ID),
				zap.Int64("order_id", dbRecord.OrderID),
				zap.Int("retry_count", dbRecord.RetryCount),
				zap.String("platform_code", dbRecord.PlatformCode),
			)
			return err
		}
		// 确保使用记录时的状态
		if dbRecord.TargetStatus > 0 {
			order.Status = model.OrderStatus(dbRecord.TargetStatus)
		}
		t.logger.Info("使用订单快照数据发送通知",
			zap.Int64("notification_id", dbRecord.ID),
			zap.Int64("order_id", dbRecord.OrderID),
			zap.Int("target_status", dbRecord.TargetStatus),
			zap.String("platform_code", dbRecord.PlatformCode),
		)
	} else {
		// 兼容旧数据：查询数据库
		var err error
		order, err = t.platformService.GetOrder(ctx, dbRecord.OrderID)
		if err != nil {
			t.logger.Error("获取订单信息失败",
				zap.Error(err),
				zap.Int64("notification_id", dbRecord.ID),
				zap.Int64("order_id", dbRecord.OrderID),
				zap.Int("retry_count", dbRecord.RetryCount),
				zap.String("platform_code", dbRecord.PlatformCode),
			)
			// 如果是 record not found，可以直接标记为失败，避免无意义重试
			if strings.Contains(err.Error(), "record not found") {
				err2 := t.notificationService.UpdateNotificationStatus(ctx, dbRecord.ID, 3)
				if err2 != nil {
					t.logger.Error("更新通知状态失败", zap.Error(err2), zap.Int64("notification_id", dbRecord.ID), zap.Int64("order_id", dbRecord.OrderID), zap.Int("retry_count", dbRecord.RetryCount), zap.String("platform_code", dbRecord.PlatformCode))
				}
				t.logger.Info("订单不存在，通知已标记为失败", zap.Int64("notification_id", dbRecord.ID), zap.Int64("order_id", dbRecord.OrderID), zap.Int("retry_count", dbRecord.RetryCount), zap.String("platform_code", dbRecord.PlatformCode))
			}
			return err
		}
		t.logger.Warn("使用数据库查询订单数据（兼容旧通知记录）",
			zap.Int64("notification_id", dbRecord.ID),
			zap.Int64("order_id", dbRecord.OrderID),
			zap.String("platform_code", dbRecord.PlatformCode),
		)
	}
	// 发送通知
	if err := t.platformService.SendNotification(ctx, order); err != nil {
		// 记录通知发送失败的详细错误信息
		t.logger.Error("通知发送失败",
			zap.Error(err),
			zap.Int64("notification_id", dbRecord.ID),
			zap.Int64("order_id", dbRecord.OrderID),
			zap.String("order_number", order.OrderNumber),
			zap.String("platform_code", dbRecord.PlatformCode),
			zap.String("notification_type", dbRecord.NotificationType),
			zap.Int("retry_count", dbRecord.RetryCount),
			zap.String("callback_url", order.PlatformCallbackURL),
		)

		// 业务终态错误关键字
		if strings.Contains(err.Error(), "此订单已做单失败") {
			t.logger.Error("遇到终态业务错误，不再重试",
				zap.Error(err),
				zap.Int64("notification_id", dbRecord.ID),
				zap.Int64("order_id", dbRecord.OrderID),
				zap.String("order_number", order.OrderNumber),
				zap.String("platform_code", dbRecord.PlatformCode),
				zap.String("notification_type", dbRecord.NotificationType),
				zap.Int("retry_count", dbRecord.RetryCount),
			)
			// 标记为失败
			if err := t.notificationService.UpdateNotificationStatus(ctx, dbRecord.ID, 3); err != nil {
				t.logger.Error("更新通知状态失败", zap.Error(err), zap.Int64("notification_id", dbRecord.ID), zap.Int64("order_id", dbRecord.OrderID), zap.String("order_number", order.OrderNumber), zap.Int("retry_count", dbRecord.RetryCount), zap.String("platform_code", dbRecord.PlatformCode))
			}
			return nil
		}
		// 如果处理失败且未超过最大重试次数，则重试
		if dbRecord.RetryCount < t.maxRetries {
			// 使用指数退避策略计算重试间隔
			retryInterval := time.Duration(1<<uint(dbRecord.RetryCount)) * time.Minute
			nextRetryTime := time.Now().Add(retryInterval)
			t.logger.Info("准备重试通知",
				zap.Int64("notification_id", dbRecord.ID),
				zap.Int64("order_id", dbRecord.OrderID),
				zap.String("order_number", order.OrderNumber),
				zap.Int("retry_count", dbRecord.RetryCount),
				zap.Time("next_retry_time", nextRetryTime),
				zap.Duration("retry_interval", retryInterval),
				zap.String("platform_code", dbRecord.PlatformCode),
			)
			// 更新通知记录状态和重试时间
			if err := t.notificationService.RetryFailedNotification(ctx, dbRecord.ID); err != nil {
				t.logger.Error("重试通知失败",
					zap.Error(err),
					zap.Int64("notification_id", dbRecord.ID),
					zap.Int64("order_id", dbRecord.OrderID),
					zap.String("order_number", order.OrderNumber),
					zap.Int("retry_count", dbRecord.RetryCount),
					zap.String("platform_code", dbRecord.PlatformCode),
				)
				return err
			}
			// 重新入队
			if err := t.queue.Push(ctx, t.queueName, dbRecord); err != nil {
				t.logger.Error("重新入队失败",
					zap.Error(err),
					zap.Int64("notification_id", dbRecord.ID),
					zap.Int64("order_id", dbRecord.OrderID),
					zap.String("order_number", order.OrderNumber),
					zap.Int("retry_count", dbRecord.RetryCount),
					zap.String("platform_code", dbRecord.PlatformCode),
					zap.String("queue_name", t.queueName),
				)
				return err
			}
			t.logger.Info("通知已重新入队",
				zap.Int64("notification_id", dbRecord.ID),
				zap.Int64("order_id", dbRecord.OrderID),
				zap.String("order_number", order.OrderNumber),
				zap.Int("retry_count", dbRecord.RetryCount),
				zap.String("queue_name", t.queueName),
				zap.Time("next_retry_time", nextRetryTime),
				zap.String("platform_code", dbRecord.PlatformCode),
			)
		} else {
			t.logger.Info("通知已达到最大重试次数，不再重试",
				zap.Int64("notification_id", dbRecord.ID),
				zap.Int64("order_id", dbRecord.OrderID),
				zap.String("order_number", order.OrderNumber),
				zap.Int("retry_count", dbRecord.RetryCount),
				zap.Int("max_retries", t.maxRetries),
				zap.String("platform_code", dbRecord.PlatformCode),
			)
			// 更新通知状态为失败
			if err := t.notificationService.UpdateNotificationStatus(ctx, dbRecord.ID, 3); err != nil {
				t.logger.Error("更新通知状态失败",
					zap.Error(err),
					zap.Int64("notification_id", dbRecord.ID),
					zap.Int64("order_id", dbRecord.OrderID),
					zap.String("order_number", order.OrderNumber),
					zap.Int("retry_count", dbRecord.RetryCount),
					zap.String("platform_code", dbRecord.PlatformCode),
				)
				return err
			}
			t.logger.Info("通知已从队列中移除",
				zap.Int64("notification_id", dbRecord.ID),
				zap.Int64("order_id", dbRecord.OrderID),
				zap.String("order_number", order.OrderNumber),
				zap.Int("retry_count", dbRecord.RetryCount),
				zap.String("queue_name", t.queueName),
				zap.String("platform_code", dbRecord.PlatformCode),
			)
		}
	} else {
		t.logger.Info("发送通知成功",
			zap.Int64("notification_id", dbRecord.ID),
			zap.Int64("order_id", dbRecord.OrderID),
			zap.String("order_number", order.OrderNumber),
			zap.String("platform_code", dbRecord.PlatformCode),
			zap.String("notification_type", dbRecord.NotificationType),
			zap.Int("retry_count", dbRecord.RetryCount),
		)
		// 更新通知状态为成功
		if err := t.notificationService.UpdateNotificationStatus(ctx, dbRecord.ID, 3); err != nil {
			t.logger.Error("更新通知状态失败",
				zap.Error(err),
				zap.Int64("notification_id", dbRecord.ID),
				zap.Int64("order_id", dbRecord.OrderID),
				zap.String("order_number", order.OrderNumber),
				zap.Int("retry_count", dbRecord.RetryCount),
				zap.String("platform_code", dbRecord.PlatformCode),
			)
			return err
		}
		t.logger.Info("通知处理成功",
			zap.Int64("notification_id", dbRecord.ID),
			zap.Int64("order_id", dbRecord.OrderID),
			zap.String("order_number", order.OrderNumber),
			zap.String("platform_code", dbRecord.PlatformCode),
			zap.String("notification_type", dbRecord.NotificationType),
			zap.Int("retry_count", dbRecord.RetryCount),
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
		t.logger.Error("获取待处理通知失败", zap.Error(err))
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
			t.logger.Info("通知已分发到工作协程",
				zap.Int64("notification_id", record.ID),
				zap.Int64("order_id", record.OrderID),
				zap.Int("retry_count", record.RetryCount),
				zap.String("platform_code", record.PlatformCode),
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
				t.logger.Error("get failed notifications failed", zap.Error(err))
				continue
			}

			// 分发重试任务到工作协程
			for _, record := range records {
				if record.RetryCount < t.maxRetries {
					select {
					case t.jobChan <- record:
						t.logger.Info("重试通知已分发到工作协程",
							zap.Int64("notification_id", record.ID),
							zap.Int64("order_id", record.OrderID),
							zap.Int("retry_count", record.RetryCount),
							zap.String("platform_code", record.PlatformCode),
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
	t.logger.Info("notification task processor stopped")
}
