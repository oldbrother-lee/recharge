package repository

import (
	"context"
	"recharge-go/internal/model"

	"gorm.io/gorm"
)

// OrderTraceRepository 订单链路事件仓储
type OrderTraceRepository interface {
	Create(ctx context.Context, e *model.OrderTraceEvent) error
	ListByOrderID(ctx context.Context, orderID int64) ([]model.OrderTraceEvent, error)
}

type orderTraceRepository struct {
	db *gorm.DB
}

// NewOrderTraceRepository 创建订单链路仓储
func NewOrderTraceRepository(db *gorm.DB) OrderTraceRepository {
	return &orderTraceRepository{db: db}
}

func (r *orderTraceRepository) Create(ctx context.Context, e *model.OrderTraceEvent) error {
	return r.db.WithContext(ctx).Create(e).Error
}

func (r *orderTraceRepository) ListByOrderID(ctx context.Context, orderID int64) ([]model.OrderTraceEvent, error) {
	var list []model.OrderTraceEvent
	err := r.db.WithContext(ctx).
		Where("order_id = ?", orderID).
		Order("id ASC").
		Find(&list).Error
	return list, err
}
