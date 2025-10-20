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
	"recharge-go/pkg/logger"
)

// NotificationTask 通知任务处理器
type NotificationTask struct {
	notificationService notificationService.NotificationService
	orderService        service.OrderService
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
	orderService service.OrderService,
	platformService *service.PlatformService,
	queue queue.Queue,
	maxRetries int,
	logger *zap.Logger,
) *NotificationTask {
	return &NotificationTask{
		notificationService: notificationService,
		orderService:        orderService,
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
	logger.WithContextCategory(ctx, "notification_task").Info("starting notification task processor")

	// 启动工作协程池
	for i := 0; i < t.workerCount; i++ {
		go t.worker(ctx, i)
	}

	// 启动重试任务（可选，混合模式下重试直接由worker处理）
	// go t.startRetryTask(ctx)

	for {
		select {
		case <-ctx.Done():
			logger.WithContextCategory(ctx, "notification_task").Info("notification task processor stopped")
			return nil
		default:
			// 从队列Pop一个通知
			value, err := t.queue.Pop(ctx, t.queueName)
			if err != nil {
				logger.WithContextCategory(ctx, "notification_task").Error("Pop notification from queue failed", logger.ErrorV2(err))
				select {
				case <-ctx.Done():
					return nil
				case <-time.After(time.Second):
				}
				continue
			}
			if value == nil {
				// time.Sleep(2 * time.Second)
				select {
				case <-ctx.Done():
					return nil
				case <-time.After(2 * time.Second):
				}
				continue
			}
			// 解析通知记录
			var record notificationModel.NotificationRecord
			switch v := value.(type) {
			case string:
				if err := json.Unmarshal([]byte(v), &record); err != nil {
					logger.WithContextCategory(ctx, "notification_task").Error("队列值反序列化失败", logger.ErrorV2(err), logger.StringV2("raw", v))
					continue
				}
			case []byte:
				if err := json.Unmarshal(v, &record); err != nil {
					logger.WithContextCategory(ctx, "notification_task").Error("队列值反序列化失败", logger.ErrorV2(err), logger.StringV2("raw", string(v)))
					continue
				}
			default:
				logger.WithContextCategory(ctx, "notification_task").Error("队列值类型错误", logger.AnyV2("type", value))
				continue
			}
			// 分发到worker
			select {
			case t.jobChan <- &record:
				logger.WithContextCategory(ctx, "notification_task").Info("通知已分发到工作协程", 
				    logger.Int64V2("notification_id", record.ID), 
				    logger.Int64V2("order_id", record.OrderID), 
				    logger.IntV2("retry_count", record.RetryCount), 
				    logger.StringV2("platform_code", record.PlatformCode))
			case <-ctx.Done():
				return nil
			}
		}
	}
}

// worker 工作协程
func (t *NotificationTask) worker(ctx context.Context, id int) {
	logger.WithContextCategory(ctx, "notification_task").Info("worker started", logger.IntV2("worker_id", id))
	for {
		select {
		case <-ctx.Done():
			logger.WithContextCategory(ctx, "notification_task").Info("worker stopped", logger.IntV2("worker_id", id))
			return
		case record := <-t.jobChan:
			if err := t.processSingleNotification(ctx, record, id); err != nil {
				logger.WithContextCategory(ctx, "notification_task").Error("process notification failed",
					logger.ErrorV2(err),
					logger.IntV2("worker_id", id),
					logger.Int64V2("notification_id", record.ID),
					logger.Int64V2("order_id", record.OrderID),
					logger.IntV2("retry_count", record.RetryCount),
					logger.StringV2("platform_code", record.PlatformCode),
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
		// 如果通知记录不存在（可能被清理任务删除），使用队列中的记录信息继续发送通知
		if strings.Contains(err.Error(), "record not found") {
			logger.WithContextCategory(ctx, "notification_task").Warn("通知记录已被删除，使用队列记录继续发送通知",
				logger.Int64V2("notification_id", record.ID),
				logger.Int64V2("order_id", record.OrderID),
				logger.IntV2("retry_count", record.RetryCount),
				logger.StringV2("platform_code", record.PlatformCode),
				logger.StringV2("reason", "可能被数据清理任务删除"),
			)
			// 使用队列中的记录信息发送通知
			return t.sendNotificationWithQueueRecord(ctx, record, workerID)
		}
		// 其他类型的错误才记录ERROR日志
		logger.WithContextCategory(ctx, "notification_task").Error("获取通知记录失败",
			logger.ErrorV2(err),
			logger.Int64V2("notification_id", record.ID),
			logger.Int64V2("order_id", record.OrderID),
			logger.IntV2("retry_count", record.RetryCount),
			logger.StringV2("platform_code", record.PlatformCode),
		)
		return err
	}
	if dbRecord.Status != 1 {
		logger.WithContextCategory(ctx, "notification_task").Info("通知状态不是待处理，跳过",
			logger.Int64V2("notification_id", dbRecord.ID),
			logger.Int64V2("order_id", dbRecord.OrderID),
			logger.IntV2("status", dbRecord.Status),
			logger.IntV2("retry_count", dbRecord.RetryCount),
			logger.StringV2("platform_code", dbRecord.PlatformCode),
		)
		return nil
	}
	// 获取订单信息：优先使用快照数据，如果没有快照则查询数据库
	var order *model.Order
	if dbRecord.OrderSnapshot != "" {
		// 使用快照数据
		order = &model.Order{}
		if err := json.Unmarshal([]byte(dbRecord.OrderSnapshot), order); err != nil {
			logger.WithContextCategory(ctx, "notification_task").Error("反序列化订单快照失败",
				logger.ErrorV2(err),
				logger.Int64V2("notification_id", dbRecord.ID),
				logger.Int64V2("order_id", dbRecord.OrderID),
				logger.IntV2("retry_count", dbRecord.RetryCount),
				logger.StringV2("platform_code", dbRecord.PlatformCode),
			)
			return err
		}
		// 确保使用记录时的状态
		if dbRecord.TargetStatus > 0 {
			order.Status = model.OrderStatus(dbRecord.TargetStatus)
		}
		logger.WithContextCategory(ctx, "notification_task").Info("使用订单快照数据发送通知",
			logger.Int64V2("notification_id", dbRecord.ID),
			logger.Int64V2("order_id", dbRecord.OrderID),
			logger.IntV2("target_status", dbRecord.TargetStatus),
			logger.StringV2("platform_code", dbRecord.PlatformCode),
		)
	} else {
		// 兼容旧数据：查询数据库
		var err error
		order, err = t.platformService.GetOrder(ctx, dbRecord.OrderID)
		if err != nil {
			logger.WithContextCategory(ctx, "notification_task").Error("获取订单信息失败",
				logger.ErrorV2(err),
				logger.Int64V2("notification_id", dbRecord.ID),
				logger.Int64V2("order_id", dbRecord.OrderID),
				logger.IntV2("retry_count", dbRecord.RetryCount),
				logger.StringV2("platform_code", dbRecord.PlatformCode),
			)
			// 如果是 record not found，可以直接标记为失败，避免无意义重试
			if strings.Contains(err.Error(), "record not found") {
				err2 := t.notificationService.UpdateNotificationStatus(ctx, dbRecord.ID, 4)
				if err2 != nil {
					logger.WithContextCategory(ctx, "notification_task").Error("更新通知状态失败", logger.ErrorV2(err2), logger.Int64V2("notification_id", dbRecord.ID), logger.Int64V2("order_id", dbRecord.OrderID), logger.IntV2("retry_count", dbRecord.RetryCount), logger.StringV2("platform_code", dbRecord.PlatformCode))
				}
				logger.WithContextCategory(ctx, "notification_task").Info("订单不存在，通知已标记为失败", logger.Int64V2("notification_id", dbRecord.ID), logger.Int64V2("order_id", dbRecord.OrderID), logger.IntV2("retry_count", dbRecord.RetryCount), logger.StringV2("platform_code", dbRecord.PlatformCode))
			}
			return err
		}
		logger.WithContextCategory(ctx, "notification_task").Warn("使用数据库查询订单数据（兼容旧通知记录）",
			logger.Int64V2("notification_id", dbRecord.ID),
			logger.Int64V2("order_id", dbRecord.OrderID),
			logger.StringV2("platform_code", dbRecord.PlatformCode),
		)
	}
	// 发送通知
	if err := t.platformService.SendNotification(ctx, order); err != nil {
		// 记录通知发送失败的详细错误信息
		logger.WithContextCategory(ctx, "notification_task").Error("通知发送失败",
			logger.ErrorV2(err),
			logger.Int64V2("notification_id", dbRecord.ID),
			logger.Int64V2("order_id", dbRecord.OrderID),
			logger.StringV2("order_number", order.OrderNumber),
			logger.StringV2("platform_code", dbRecord.PlatformCode),
			logger.StringV2("notification_type", dbRecord.NotificationType),
			logger.IntV2("retry_count", dbRecord.RetryCount),
			logger.StringV2("callback_url", order.PlatformCallbackURL),
		)

		// 业务终态错误关键字
		if strings.Contains(err.Error(), "此订单已做单失败") {
			logger.WithContextCategory(ctx, "notification_task").Error("遇到终态业务错误，不再重试",
				logger.ErrorV2(err),
				logger.Int64V2("notification_id", dbRecord.ID),
				logger.Int64V2("order_id", dbRecord.OrderID),
				logger.StringV2("order_number", order.OrderNumber),
				logger.StringV2("platform_code", dbRecord.PlatformCode),
				logger.StringV2("notification_type", dbRecord.NotificationType),
				logger.IntV2("retry_count", dbRecord.RetryCount),
			)
			// 标记为失败
			err2 := t.notificationService.UpdateNotificationStatus(ctx, dbRecord.ID, 4)
			if err2 != nil {
				logger.WithContextCategory(ctx, "notification_task").Error("更新通知状态失败", logger.ErrorV2(err2), logger.Int64V2("notification_id", dbRecord.ID), logger.Int64V2("order_id", dbRecord.OrderID), logger.IntV2("retry_count", dbRecord.RetryCount), logger.StringV2("platform_code", dbRecord.PlatformCode))
			}
			return nil
		}

		// 非终态错误，判断是否需要重试
		if dbRecord.RetryCount < t.maxRetries {
			// 延迟重试（加入到队列）——重试次数在 RetryFailedNotification 中自增
			if err := t.notificationService.RetryFailedNotification(ctx, dbRecord.ID); err != nil {
				logger.WithContextCategory(ctx, "notification_task").Error("加入重试队列失败", logger.ErrorV2(err), logger.Int64V2("notification_id", dbRecord.ID), logger.Int64V2("order_id", dbRecord.OrderID), logger.IntV2("retry_count", dbRecord.RetryCount), logger.StringV2("platform_code", dbRecord.PlatformCode))
			}
			return nil
		}

		// 达到最大重试次数，标记为失败
		if err := t.notificationService.UpdateNotificationStatus(ctx, dbRecord.ID, 4); err != nil {
			logger.WithContextCategory(ctx, "notification_task").Error("更新通知状态失败", logger.ErrorV2(err), logger.Int64V2("notification_id", dbRecord.ID), logger.Int64V2("order_id", dbRecord.OrderID), logger.IntV2("retry_count", dbRecord.RetryCount), logger.StringV2("platform_code", dbRecord.PlatformCode))
		}
		return nil
	}

	// 通知发送成功，标记为已完成
	if err := t.notificationService.UpdateNotificationStatus(ctx, dbRecord.ID, 3); err != nil {
		logger.WithContextCategory(ctx, "notification_task").Error("更新通知状态失败", logger.ErrorV2(err), logger.Int64V2("notification_id", dbRecord.ID), logger.Int64V2("order_id", dbRecord.OrderID), logger.IntV2("retry_count", dbRecord.RetryCount), logger.StringV2("platform_code", dbRecord.PlatformCode))
	}

	return nil
}

// processNotifications 处理待通知的记录（兼容旧逻辑，作为备用）
func (t *NotificationTask) processNotifications(ctx context.Context) (bool, error) {
	// 查询待处理的通知记录
	records, _, err := t.notificationService.ListNotifications(ctx, map[string]interface{}{
		"status": 1, // 待处理
	}, 1, t.batchSize)
	if err != nil {
		return false, err
	}

	if len(records) == 0 {
		return false, nil
	}

	for _, record := range records {
		if err := t.processSingleNotification(ctx, record, 0); err != nil {
			// 记录错误，但继续处理其他记录
			logger.WithContextCategory(ctx, "notification_task").Error("处理通知失败", logger.ErrorV2(err), logger.Int64V2("notification_id", record.ID), logger.Int64V2("order_id", record.OrderID), logger.IntV2("retry_count", record.RetryCount), logger.StringV2("platform_code", record.PlatformCode))
		}
	}
	return true, nil
}

// startRetryTask 定期扫描并加入重试队列（兼容旧逻辑，作为备用）
func (t *NotificationTask) startRetryTask(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(30 * time.Second):
			// 每30秒扫描一次
			records, _, err := t.notificationService.ListNotifications(ctx, map[string]interface{}{
				"status": 4, // 发送失败
			}, 1, 20)
			if err != nil {
				logger.WithContextCategory(ctx, "notification_task").Error("扫描失败的通知记录失败", logger.ErrorV2(err))
				continue
			}
			for _, record := range records {
				// 加入重试队列——重试次数在 RetryFailedNotification 中自增
				if err := t.notificationService.RetryFailedNotification(ctx, record.ID); err != nil {
					logger.WithContextCategory(ctx, "notification_task").Error("加入重试队列失败", logger.ErrorV2(err), logger.Int64V2("notification_id", record.ID), logger.Int64V2("order_id", record.OrderID), logger.IntV2("retry_count", record.RetryCount), logger.StringV2("platform_code", record.PlatformCode))
				}
			}
		}
	}
}

// sendNotificationWithQueueRecord 使用队列中的记录信息发送通知
func (t *NotificationTask) sendNotificationWithQueueRecord(ctx context.Context, record *notificationModel.NotificationRecord, workerID int) error {
	// 获取订单信息（优先使用快照）
	var order *model.Order
	if record.OrderSnapshot != "" {
		order = &model.Order{}
		if err := json.Unmarshal([]byte(record.OrderSnapshot), order); err != nil {
			logger.WithContextCategory(ctx, "notification_task").Error("反序列化订单快照失败", logger.ErrorV2(err), logger.Int64V2("notification_id", record.ID), logger.Int64V2("order_id", record.OrderID), logger.IntV2("retry_count", record.RetryCount), logger.StringV2("platform_code", record.PlatformCode))
			return err
		}
		// 使用记录时的状态
		if record.TargetStatus > 0 {
			order.Status = model.OrderStatus(record.TargetStatus)
		}
	} else {
		// 兼容：从DB查询
		var err error
		order, err = t.orderService.GetOrderByID(ctx, record.OrderID)
		if err != nil {
			logger.WithContextCategory(ctx, "notification_task").Error("获取订单信息失败", logger.ErrorV2(err), logger.Int64V2("notification_id", record.ID), logger.Int64V2("order_id", record.OrderID), logger.IntV2("retry_count", record.RetryCount), logger.StringV2("platform_code", record.PlatformCode))
			return err
		}
	}

	// 发送通知
	if err := t.platformService.SendNotification(ctx, order); err != nil {
		// 记录失败
		logger.WithContextCategory(ctx, "notification_task").Error("通知发送失败", logger.ErrorV2(err), logger.Int64V2("notification_id", record.ID), logger.Int64V2("order_id", record.OrderID), logger.StringV2("order_number", order.OrderNumber), logger.IntV2("retry_count", record.RetryCount), logger.StringV2("platform_code", record.PlatformCode))
		// 重试策略
		if record.RetryCount < t.maxRetries {
			// 加入重试队列——重试次数在 RetryFailedNotification 中自增
			if err := t.notificationService.RetryFailedNotification(ctx, record.ID); err != nil {
				logger.WithContextCategory(ctx, "notification_task").Error("加入重试队列失败", logger.ErrorV2(err), logger.Int64V2("notification_id", record.ID))
			}
			return nil
		}
		// 达到最大重试次数
		if err := t.notificationService.UpdateNotificationStatus(ctx, record.ID, 4); err != nil {
			logger.WithContextCategory(ctx, "notification_task").Error("更新通知状态失败", logger.ErrorV2(err), logger.Int64V2("notification_id", record.ID))
		}
		return nil
	}

	// 成功
	if err := t.notificationService.UpdateNotificationStatus(ctx, record.ID, 3); err != nil {
		logger.WithContextCategory(ctx, "notification_task").Error("更新通知状态失败", logger.ErrorV2(err), logger.Int64V2("notification_id", record.ID))
	}
	return nil
}

// Stop 停止通知任务处理器
func (t *NotificationTask) Stop() {
	// 资源清理
	logger.WithContextCategory(context.Background(), "notification_task").Info("notification task processor stopped")
}
