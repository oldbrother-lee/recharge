package repository

import (
	"context"
	"recharge-go/internal/model"

	"gorm.io/gorm"
)

type OrderUpgradeRepository interface {
	Create(ctx context.Context, order *model.OrderUpgrade) error
	GetByID(ctx context.Context, id int64) (*model.OrderUpgrade, error)
	GetByUserID(ctx context.Context, userID int64, page, pageSize int) ([]*model.OrderUpgrade, int64, error)
	UpdateStatus(ctx context.Context, id int64, status int) error
}

type orderUpgradeRepository struct {
	db *gorm.DB
}

func NewOrderUpgradeRepository(db *gorm.DB) OrderUpgradeRepository {
	return &orderUpgradeRepository{
		db: db,
	}
}

func (r *orderUpgradeRepository) Create(ctx context.Context, order *model.OrderUpgrade) error {
	return r.db.WithContext(ctx).Create(order).Error
}

func (r *orderUpgradeRepository) GetByID(ctx context.Context, id int64) (*model.OrderUpgrade, error) {
	var order model.OrderUpgrade
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&order).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

func (r *orderUpgradeRepository) GetByUserID(ctx context.Context, userID int64, page, pageSize int) ([]*model.OrderUpgrade, int64, error) {
	var orders []*model.OrderUpgrade
	var total int64

	offset := (page - 1) * pageSize
	err := r.db.WithContext(ctx).Model(&model.OrderUpgrade{}).
		Where("user_id = ?", userID).
		Count(&total).
		Offset(offset).
		Limit(pageSize).
		Order("created_at DESC").
		Find(&orders).Error

	if err != nil {
		return nil, 0, err
	}
	return orders, total, nil
}

func (r *orderUpgradeRepository) UpdateStatus(ctx context.Context, id int64, status int) error {
	return r.db.WithContext(ctx).Model(&model.OrderUpgrade{}).
		Where("id = ?", id).
		Update("status", status).Error
}
