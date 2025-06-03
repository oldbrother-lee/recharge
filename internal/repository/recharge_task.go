package repository

import (
	"context"
	"recharge-go/internal/model"

	"gorm.io/gorm"
)

type RechargeTaskRepository interface {
	Create(ctx context.Context, task *model.RechargeTask) error
	UpdateStatus(ctx context.Context, task *model.RechargeTask) error
	GetOrderByTaskID(ctx context.Context, taskID int64) (*model.Order, error)
	UpdateOrderStatus(ctx context.Context, order *model.Order) error
	GetPendingTasks(ctx context.Context, limit int) ([]*model.RechargeTask, error)
}

type rechargeTaskRepository struct {
	db *gorm.DB
}

func NewRechargeTaskRepository(db *gorm.DB) RechargeTaskRepository {
	return &rechargeTaskRepository{db: db}
}

func (r *rechargeTaskRepository) Create(ctx context.Context, task *model.RechargeTask) error {
	return r.db.WithContext(ctx).Create(task).Error
}

func (r *rechargeTaskRepository) UpdateStatus(ctx context.Context, task *model.RechargeTask) error {
	return r.db.WithContext(ctx).Model(task).Updates(map[string]interface{}{
		"status":        task.Status,
		"error_msg":     task.ErrorMsg,
		"retry_times":   task.RetryTimes,
		"next_retry_at": task.NextRetryAt,
		"result":        task.Result,
	}).Error
}

func (r *rechargeTaskRepository) GetOrderByTaskID(ctx context.Context, taskID int64) (*model.Order, error) {
	var task model.RechargeTask
	if err := r.db.WithContext(ctx).First(&task, taskID).Error; err != nil {
		return nil, err
	}

	var order model.Order
	if err := r.db.WithContext(ctx).First(&order, task.OrderID).Error; err != nil {
		return nil, err
	}

	return &order, nil
}

func (r *rechargeTaskRepository) UpdateOrderStatus(ctx context.Context, order *model.Order) error {
	return r.db.WithContext(ctx).Model(order).Update("status", order.Status).Error
}

func (r *rechargeTaskRepository) GetPendingTasks(ctx context.Context, limit int) ([]*model.RechargeTask, error) {
	var tasks []*model.RechargeTask
	err := r.db.WithContext(ctx).
		Where("status = ? AND retry_times < max_retries", model.RechargeTaskStatusPending).
		Limit(limit).
		Find(&tasks).Error
	return tasks, err
}
