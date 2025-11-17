package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"recharge-go/internal/model"
	notificationModel "recharge-go/internal/model/notification"
	notificationRepo "recharge-go/internal/repository/notification"
	"recharge-go/pkg/logger"
	"recharge-go/pkg/queue"

	"gorm.io/gorm"
)

// NotificationHelper 统一的通知发送辅助类
type NotificationHelper struct {
	db               *gorm.DB
	notificationRepo notificationRepo.Repository
	queue            queue.Queue
}

// NewNotificationHelper 创建通知辅助实例
func NewNotificationHelper(
	db *gorm.DB,
	notificationRepo notificationRepo.Repository,
	queue queue.Queue,
) *NotificationHelper {
	return &NotificationHelper{
		db:               db,
		notificationRepo: notificationRepo,
		queue:            queue,
	}
}

// SendOrderStatusNotification 发送订单状态变更通知（带幂等性保护）
func (h *NotificationHelper) SendOrderStatusNotification(ctx context.Context, order *model.Order, newStatus model.OrderStatus) error {
	if newStatus != model.OrderStatusSuccess && newStatus != model.OrderStatusFailed {
		logger.WithContext(ctx).Info("跳过非成功/失败状态通知",
			logger.Int64V2("order_id", order.ID),
			logger.IntV2("target_status", int(newStatus)),
		)
		return nil
	}
	// 幂等校验：按 (order_id, notification_type, target_status) 查找最近一条通知
	var existing notificationModel.NotificationRecord
	err := h.db.WithContext(ctx).
		Where("order_id = ? AND notification_type = ? AND target_status = ?", order.ID, "order_status_changed", int(newStatus)).
		Order("id DESC").
		First(&existing).Error
	if err == nil {
		switch existing.Status {
		case 3: // 已成功
			logger.WithContext(ctx).Info("已存在成功的状态变更通知，跳过重复创建",
				logger.Int64V2("order_id", order.ID),
				logger.IntV2("target_status", int(newStatus)),
				logger.Int64V2("notification_id", existing.ID),
			)
			return nil
		case 1, 2: // 待处理/处理中，复用并重推
			logger.WithContext(ctx).Info("已存在待处理/处理中通知，复用并重推到队列",
				logger.Int64V2("order_id", order.ID),
				logger.IntV2("target_status", int(newStatus)),
				logger.Int64V2("notification_id", existing.ID),
			)
			if pushErr := h.queue.Push(ctx, "notification_queue", &existing); pushErr != nil {
				logger.WithContext(ctx).Error("重推通知到队列失败",
					logger.Int64V2("order_id", order.ID),
					logger.Int64V2("notification_id", existing.ID),
					logger.ErrorV2(pushErr),
				)
				return pushErr
			}
			logger.WithContext(ctx).Info("重推通知到队列成功",
				logger.Int64V2("order_id", order.ID),
				logger.Int64V2("notification_id", existing.ID),
			)
			return nil
		case 4: // 失败，重置为待处理并重推
			logger.WithContext(ctx).Info("存在失败通知，重置为待处理并重推",
				logger.Int64V2("order_id", order.ID),
				logger.IntV2("target_status", int(newStatus)),
				logger.Int64V2("notification_id", existing.ID),
			)
			if updErr := h.notificationRepo.UpdateStatus(ctx, existing.ID, 1); updErr != nil {
				logger.WithContext(ctx).Error("重置失败通知状态失败",
					logger.Int64V2("order_id", order.ID),
					logger.Int64V2("notification_id", existing.ID),
					logger.ErrorV2(updErr),
				)
				// 继续尝试推送
			}
			if pushErr := h.queue.Push(ctx, "notification_queue", &existing); pushErr != nil {
				logger.WithContext(ctx).Error("重推失败通知到队列失败",
					logger.Int64V2("order_id", order.ID),
					logger.Int64V2("notification_id", existing.ID),
					logger.ErrorV2(pushErr),
				)
				return pushErr
			}
			logger.WithContext(ctx).Info("失败通知重推成功",
				logger.Int64V2("order_id", order.ID),
				logger.Int64V2("notification_id", existing.ID),
			)
			return nil
		}
	} else if err != gorm.ErrRecordNotFound {
		// 查询错误，记录日志但不中断创建流程
		logger.WithContext(ctx).Error("查询现有通知记录失败，继续走创建流程",
			logger.Int64V2("order_id", order.ID),
			logger.IntV2("target_status", int(newStatus)),
			logger.ErrorV2(err),
		)
	}

	// 序列化订单快照
	orderData, mErr := json.Marshal(order)
	if mErr != nil {
		logger.WithContext(ctx).Error("序列化订单快照失败", logger.Int64V2("order_id", order.ID), logger.ErrorV2(mErr))
		return mErr
	}

	// 创建通知记录（包含订单快照）
	notification := &notificationModel.NotificationRecord{
		OrderID:          order.ID,
		PlatformCode:     order.PlatformCode,
		NotificationType: "order_status_changed",
		Content:          fmt.Sprintf("订单状态已更新为: %d", newStatus),
		OrderSnapshot:    string(orderData), // 保存完整订单快照
		TargetStatus:     int(newStatus),    // 保存目标状态
		Status:           1,                 // 待处理
	}

	// 原子操作：创建通知记录并推送到队列
	return h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(notification).Error; err != nil {
			// 避免唯一约束冲突导致报错，这里做一次容错处理
			if strings.Contains(err.Error(), "Duplicate") || strings.Contains(err.Error(), "UNIQUE") || strings.Contains(err.Error(), "duplicate") {
				logger.WithContext(ctx).Warn("检测到通知记录唯一键冲突，复用已有记录",
					logger.Int64V2("order_id", order.ID),
					logger.IntV2("target_status", int(newStatus)),
					logger.ErrorV2(err),
				)
				var exist2 notificationModel.NotificationRecord
				if qErr := tx.
					Where("order_id = ? AND notification_type = ? AND target_status = ?", order.ID, "order_status_changed", int(newStatus)).
					Order("id DESC").
					First(&exist2).Error; qErr == nil {
					logger.WithContext(ctx).Info("复用已有通知记录并推送到队列",
						logger.Int64V2("order_id", order.ID),
						logger.Int64V2("notification_id", exist2.ID),
					)
					if pushErr := h.queue.Push(ctx, "notification_queue", &exist2); pushErr != nil {
						logger.WithContext(ctx).Error("推送已有通知到队列失败",
							logger.Int64V2("order_id", order.ID),
							logger.Int64V2("notification_id", exist2.ID),
							logger.ErrorV2(pushErr),
						)
						return pushErr
					}
					return nil
				}
				// 查询不到则返回原始错误
				return err
			}
			logger.WithContext(ctx).Error("创建通知记录失败", logger.Int64V2("order_id", order.ID), logger.ErrorV2(err))
			return err
		}

		// 推送通知到队列
		logger.WithContext(ctx).Info("准备推送通知到队列",
			logger.Int64V2("order_id", order.ID),
			logger.IntV2("new_status", int(newStatus)),
			logger.Int64V2("notification_id", notification.ID),
		)

		if err := h.queue.Push(ctx, "notification_queue", notification); err != nil {
			logger.WithContext(ctx).Error("推送通知到队列失败",
				logger.Int64V2("order_id", order.ID),
				logger.Int64V2("notification_id", notification.ID),
				logger.ErrorV2(err),
			)
			return err
		}

		logger.WithContext(ctx).Info("推送通知到队列成功",
			logger.Int64V2("order_id", order.ID),
			logger.Int64V2("notification_id", notification.ID),
		)
		return nil
	})
}

func (h *NotificationHelper) SendOrderPreReport(ctx context.Context, order *model.Order) error {
	var existing notificationModel.NotificationRecord
	err := h.db.WithContext(ctx).
		Where("order_id = ? AND notification_type = ? AND target_status = ?", order.ID, "order_pre_report", int(model.OrderStatusProcessing)).
		Order("id DESC").
		First(&existing).Error
	if err == nil {
		switch existing.Status {
		case 3:
			logger.WithContext(ctx).Info("已存在成功的预上报通知，跳过重复创建",
				logger.Int64V2("order_id", order.ID),
				logger.Int64V2("notification_id", existing.ID),
			)
			return nil
		case 1, 2:
			logger.WithContext(ctx).Info("已存在待处理/处理中预上报通知，复用并重推到队列",
				logger.Int64V2("order_id", order.ID),
				logger.Int64V2("notification_id", existing.ID),
			)
			if pushErr := h.queue.Push(ctx, "notification_queue", &existing); pushErr != nil {
				logger.WithContext(ctx).Error("重推预上报通知到队列失败",
					logger.Int64V2("order_id", order.ID),
					logger.Int64V2("notification_id", existing.ID),
					logger.ErrorV2(pushErr),
				)
				return pushErr
			}
			logger.WithContext(ctx).Info("重推预上报通知到队列成功",
				logger.Int64V2("order_id", order.ID),
				logger.Int64V2("notification_id", existing.ID),
			)
			return nil
		case 4:
			logger.WithContext(ctx).Info("存在失败的预上报通知，重置为待处理并重推",
				logger.Int64V2("order_id", order.ID),
				logger.Int64V2("notification_id", existing.ID),
			)
			if updErr := h.notificationRepo.UpdateStatus(ctx, existing.ID, 1); updErr != nil {
				logger.WithContext(ctx).Error("重置失败的预上报通知状态失败",
					logger.Int64V2("order_id", order.ID),
					logger.Int64V2("notification_id", existing.ID),
					logger.ErrorV2(updErr),
				)
			}
			if pushErr := h.queue.Push(ctx, "notification_queue", &existing); pushErr != nil {
				logger.WithContext(ctx).Error("重推失败预上报通知到队列失败",
					logger.Int64V2("order_id", order.ID),
					logger.Int64V2("notification_id", existing.ID),
					logger.ErrorV2(pushErr),
				)
				return pushErr
			}
			logger.WithContext(ctx).Info("失败预上报通知重推成功",
				logger.Int64V2("order_id", order.ID),
				logger.Int64V2("notification_id", existing.ID),
			)
			return nil
		}
	} else if err != gorm.ErrRecordNotFound {
		logger.WithContext(ctx).Error("查询现有预上报通知记录失败，继续创建",
			logger.Int64V2("order_id", order.ID),
			logger.ErrorV2(err),
		)
	}

	orderData, mErr := json.Marshal(order)
	if mErr != nil {
		logger.WithContext(ctx).Error("序列化订单快照失败", logger.Int64V2("order_id", order.ID), logger.ErrorV2(mErr))
		return mErr
	}

	notification := &notificationModel.NotificationRecord{
		OrderID:          order.ID,
		PlatformCode:     order.PlatformCode,
		NotificationType: "order_pre_report",
		Content:          fmt.Sprintf("订单预上报: %d", model.OrderStatusProcessing),
		OrderSnapshot:    string(orderData),
		TargetStatus:     int(model.OrderStatusProcessing),
		Status:           1,
	}

	return h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(notification).Error; err != nil {
			if strings.Contains(err.Error(), "Duplicate") || strings.Contains(err.Error(), "UNIQUE") || strings.Contains(err.Error(), "duplicate") {
				var exist2 notificationModel.NotificationRecord
				if qErr := tx.
					Where("order_id = ? AND notification_type = ? AND target_status = ?", order.ID, "order_pre_report", int(model.OrderStatusProcessing)).
					Order("id DESC").
					First(&exist2).Error; qErr == nil {
					if pushErr := h.queue.Push(ctx, "notification_queue", &exist2); pushErr != nil {
						return pushErr
					}
					return nil
				}
				return err
			}
			return err
		}
		if err := h.queue.Push(ctx, "notification_queue", notification); err != nil {
			return err
		}
		return nil
	})
}

// SendOrderCallbackNotification 发送订单回调通知
func (h *NotificationHelper) SendOrderCallbackNotification(ctx context.Context, orderID int64, order *model.Order) error {
	// 确定平台代码
	platformCode := order.PlatformCode
	if platformCode == "" {
		platformCode = "system" // 默认值
	}

	// 创建通知任务
	notification := &notificationModel.NotificationRecord{
		OrderID:          orderID,
		PlatformCode:     platformCode,
		NotificationType: "order_callback",
		Content:          fmt.Sprintf("订单 %s 回调通知", order.OrderNumber),
		Status:           1, // 待处理
		RetryCount:       0,
		NextRetryTime:    time.Now().Add(5 * time.Minute),
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	// 保存通知记录
	if err := h.notificationRepo.Create(ctx, notification); err != nil {
		return fmt.Errorf("创建通知记录失败: %w", err)
	}

	// 推送到通知队列
	if err := h.queue.Push(ctx, "notification_queue", notification); err != nil {
		return fmt.Errorf("推送到通知队列失败: %w", err)
	}

	logger.WithContext(ctx).Info("订单回调通知已推送到队列",
		logger.Int64V2("order_id", orderID),
		logger.StringV2("order_number", order.OrderNumber),
		logger.StringV2("platform_code", platformCode),
	)
	return nil
}
