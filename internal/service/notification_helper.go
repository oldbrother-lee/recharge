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
	// 幂等校验：按 (order_id, notification_type, target_status) 查找最近一条通知
	var existing notificationModel.NotificationRecord
	err := h.db.WithContext(ctx).
		Where("order_id = ? AND notification_type = ? AND target_status = ?", order.ID, "order_status_changed", int(newStatus)).
		Order("id DESC").
		First(&existing).Error
	if err == nil {
		switch existing.Status {
		case 3: // 已成功
			logger.Info("已存在成功的状态变更通知，跳过重复创建",
				"order_id", order.ID,
				"target_status", int(newStatus),
				"notification_id", existing.ID)
			return nil
		case 1, 2: // 待处理/处理中，复用并重推
			logger.Info("已存在待处理/处理中通知，复用并重推到队列",
				"order_id", order.ID,
				"target_status", int(newStatus),
				"notification_id", existing.ID)
			if pushErr := h.queue.Push(ctx, "notification_queue", &existing); pushErr != nil {
				logger.Error("重推通知到队列失败",
					"order_id", order.ID,
					"notification_id", existing.ID,
					"error", pushErr)
				return pushErr
			}
			logger.Info("重推通知到队列成功",
				"order_id", order.ID,
				"notification_id", existing.ID)
			return nil
		case 4: // 失败，重置为待处理并重推
			logger.Info("存在失败通知，重置为待处理并重推",
				"order_id", order.ID,
				"target_status", int(newStatus),
				"notification_id", existing.ID)
			if updErr := h.notificationRepo.UpdateStatus(ctx, existing.ID, 1); updErr != nil {
				logger.Error("重置失败通知状态失败",
					"order_id", order.ID,
					"notification_id", existing.ID,
					"error", updErr)
				// 继续尝试推送
			}
			if pushErr := h.queue.Push(ctx, "notification_queue", &existing); pushErr != nil {
				logger.Error("重推失败通知到队列失败",
					"order_id", order.ID,
					"notification_id", existing.ID,
					"error", pushErr)
				return pushErr
			}
			logger.Info("失败通知重推成功",
				"order_id", order.ID,
				"notification_id", existing.ID)
			return nil
		}
	} else if err != gorm.ErrRecordNotFound {
		// 查询错误，记录日志但不中断创建流程
		logger.Error("查询现有通知记录失败，继续走创建流程",
			"order_id", order.ID,
			"target_status", int(newStatus),
			"error", err)
	}

	// 序列化订单快照
	orderData, mErr := json.Marshal(order)
	if mErr != nil {
		logger.Error("序列化订单快照失败", "order_id", order.ID, "error", mErr)
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
				logger.Warn("检测到通知记录唯一键冲突，复用已有记录",
					"order_id", order.ID,
					"target_status", int(newStatus),
					"error", err)
				var exist2 notificationModel.NotificationRecord
				if qErr := tx.
					Where("order_id = ? AND notification_type = ? AND target_status = ?", order.ID, "order_status_changed", int(newStatus)).
					Order("id DESC").
					First(&exist2).Error; qErr == nil {
					logger.Info("复用已有通知记录并推送到队列",
						"order_id", order.ID,
						"notification_id", exist2.ID)
					if pushErr := h.queue.Push(ctx, "notification_queue", &exist2); pushErr != nil {
						logger.Error("推送已有通知到队列失败",
							"order_id", order.ID,
							"notification_id", exist2.ID,
							"error", pushErr)
						return pushErr
					}
					return nil
				}
				// 查询不到则返回原始错误
				return err
			}
			logger.Error("创建通知记录失败", "order_id", order.ID, "error", err)
			return err
		}

		// 推送通知到队列
		logger.Info("准备推送通知到队列",
			"order_id", order.ID,
			"new_status", int(newStatus),
			"notification_id", notification.ID)

		if err := h.queue.Push(ctx, "notification_queue", notification); err != nil {
			logger.Error("推送通知到队列失败",
				"order_id", order.ID,
				"notification_id", notification.ID,
				"error", err)
			return err
		}

		logger.Info("推送通知到队列成功",
			"order_id", order.ID,
			"notification_id", notification.ID)
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

	logger.Info("订单回调通知已推送到队列", "order_id", orderID, "order_number", order.OrderNumber, "platform_code", platformCode)
	return nil
}