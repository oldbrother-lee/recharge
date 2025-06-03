package repository

import (
	"context"
	"recharge-go/internal/model"

	"gorm.io/gorm"
)

// CallbackLogRepository 回调日志仓库接口
type CallbackLogRepository interface {
	// Create 创建回调日志
	Create(ctx context.Context, log *model.CallbackLog) error
	// GetByOrderID 根据订单号获取回调日志
	GetByOrderID(ctx context.Context, orderID string) ([]*model.CallbackLog, error)
	// GetByOrderIDAndType 根据订单号和回调类型获取回调日志
	GetByOrderIDAndType(ctx context.Context, orderID, callbackType string) (*model.CallbackLog, error)
}

// CallbackLogRepositoryImpl 回调日志仓库实现
type CallbackLogRepositoryImpl struct {
	db *gorm.DB
}

// NewCallbackLogRepository 创建回调日志仓库
func NewCallbackLogRepository(db *gorm.DB) CallbackLogRepository {
	return &CallbackLogRepositoryImpl{db: db}
}

// Create 创建回调日志
func (r *CallbackLogRepositoryImpl) Create(ctx context.Context, log *model.CallbackLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

// GetByOrderID 根据订单号获取回调日志
func (r *CallbackLogRepositoryImpl) GetByOrderID(ctx context.Context, orderID string) ([]*model.CallbackLog, error) {
	var logs []*model.CallbackLog
	err := r.db.WithContext(ctx).Where("order_id = ?", orderID).Find(&logs).Error
	return logs, err
}

// GetByOrderIDAndType 根据订单号和回调类型获取回调日志
func (r *CallbackLogRepositoryImpl) GetByOrderIDAndType(ctx context.Context, orderID, callbackType string) (*model.CallbackLog, error) {
	var log model.CallbackLog
	err := r.db.WithContext(ctx).Where("order_id = ? AND callback_type = ?", orderID, callbackType).First(&log).Error
	if err != nil {
		return nil, err
	}
	return &log, nil
}
