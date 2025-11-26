package notification

import (
	"context"
	"recharge-go/internal/model/notification"
	notificationRepo "recharge-go/internal/repository/notification"
	"recharge-go/pkg/log"
	"recharge-go/pkg/queue"
	"time"
)

// NotificationService 通知服务接口
type NotificationService interface {
	// CreateNotification 创建通知
	CreateNotification(ctx context.Context, record *notification.NotificationRecord) error
	// GetNotificationStatus 获取通知状态
	GetNotificationStatus(ctx context.Context, id int64) (*notification.NotificationRecord, error)
	// ListNotifications 获取通知列表
	ListNotifications(ctx context.Context, params map[string]interface{}, page, pageSize int) ([]*notification.NotificationRecord, int64, error)
	// RetryFailedNotification 重试失败的通知
	RetryFailedNotification(ctx context.Context, id int64) error
	// UpdateNotificationStatus 更新通知状态
	UpdateNotificationStatus(ctx context.Context, id int64, status int) error
	// GetNotification 获取通知记录
	GetNotification(ctx context.Context, id int64) (*notification.NotificationRecord, error)
	// DeleteByOrderIDs 根据订单ID列表删除通知记录
	DeleteByOrderIDs(ctx context.Context, orderIDs []int64) error
}

// notificationService 通知服务实现
type notificationService struct {
	recordRepo notificationRepo.Repository
	queue      queue.Queue
}

// NewNotificationService 创建通知服务实例
func NewNotificationService(recordRepo notificationRepo.Repository, queue queue.Queue) NotificationService {
	return &notificationService{
		recordRepo: recordRepo,
		queue:      queue,
	}
}

// CreateNotification 创建通知
func (s *notificationService) CreateNotification(ctx context.Context, record *notification.NotificationRecord) error {
	return s.recordRepo.Create(ctx, record)
}

// GetNotificationStatus 获取通知状态
func (s *notificationService) GetNotificationStatus(ctx context.Context, id int64) (*notification.NotificationRecord, error) {
	return s.recordRepo.GetByID(ctx, id)
}

// ListNotifications 获取通知列表
func (s *notificationService) ListNotifications(ctx context.Context, params map[string]interface{}, page, pageSize int) ([]*notification.NotificationRecord, int64, error) {
	return s.recordRepo.List(ctx, params, page, pageSize)
}

// RetryFailedNotification 重试失败的通知
func (s *notificationService) RetryFailedNotification(ctx context.Context, id int64) error {
	record, err := s.recordRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// 更新重试次数和下次重试时间（固定1分钟后）
	record.RetryCount++
	record.NextRetryTime = time.Now().Add(1 * time.Minute)
	record.Status = 1 // 重置为待处理状态

	if err := s.recordRepo.Update(ctx, record); err != nil {
		return err
	}

	// 使用 time.AfterFunc 在1分钟后重新入队（保持与现有消费者兼容）
	delay := 1 * time.Minute
	time.AfterFunc(delay, func() {
		// 在回调中使用调用方的ctx可能已经超时/取消，这里创建一个可取消的派生ctx，尊重上层取消但避免使用全局Background
		select {
		case <-ctx.Done():
			// 上下文已取消，不再入队
			log.Info(ctx, "skip_cancelled_notification_retry_enque", log.Int64("notification_id", record.ID))
			return
		default:
		}
		if err := s.queue.Push(ctx, "notification_queue", record); err != nil {
			log.Error(ctx, "notification_retry_enqueue_failed", log.Err(err), log.Int64("notification_id", record.ID))
			return
		}
		log.Info(ctx, "notification_retry_enqueued", log.Int64("notification_id", record.ID), log.Duration("delay", delay))
	})

	return nil
}

// UpdateNotificationStatus 更新通知状态
func (s *notificationService) UpdateNotificationStatus(ctx context.Context, id int64, status int) error {
	return s.recordRepo.UpdateStatus(ctx, id, status)
}

// GetNotification 获取通知记录
func (s *notificationService) GetNotification(ctx context.Context, id int64) (*notification.NotificationRecord, error) {
	return s.recordRepo.GetByID(ctx, id)
}

// DeleteByOrderIDs 根据订单ID列表删除通知记录
func (s *notificationService) DeleteByOrderIDs(ctx context.Context, orderIDs []int64) error {
	return s.recordRepo.DeleteByOrderIDs(ctx, orderIDs)
}
