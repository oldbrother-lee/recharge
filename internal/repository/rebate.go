package repository

import (
	"context"
	"recharge-go/internal/model"

	"gorm.io/gorm"
)

type RebateRepository interface {
	Create(ctx context.Context, rebate *model.Rebate) error
	GetByID(ctx context.Context, id int64) (*model.Rebate, error)
	GetByUserID(ctx context.Context, userID int64, page, pageSize int) ([]*model.Rebate, int64, error)
	UpdateStatus(ctx context.Context, id int64, status int) error
}

type rebateRepository struct {
	db *gorm.DB
}

func NewRebateRepository(db *gorm.DB) RebateRepository {
	return &rebateRepository{
		db: db,
	}
}

func (r *rebateRepository) Create(ctx context.Context, rebate *model.Rebate) error {
	return r.db.WithContext(ctx).Create(rebate).Error
}

func (r *rebateRepository) GetByID(ctx context.Context, id int64) (*model.Rebate, error) {
	var rebate model.Rebate
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&rebate).Error
	if err != nil {
		return nil, err
	}
	return &rebate, nil
}

func (r *rebateRepository) GetByUserID(ctx context.Context, userID int64, page, pageSize int) ([]*model.Rebate, int64, error) {
	var rebates []*model.Rebate
	var total int64

	offset := (page - 1) * pageSize
	err := r.db.WithContext(ctx).Model(&model.Rebate{}).
		Where("user_id = ?", userID).
		Count(&total).
		Offset(offset).
		Limit(pageSize).
		Order("created_at DESC").
		Find(&rebates).Error

	if err != nil {
		return nil, 0, err
	}
	return rebates, total, nil
}

func (r *rebateRepository) UpdateStatus(ctx context.Context, id int64, status int) error {
	return r.db.WithContext(ctx).Model(&model.Rebate{}).
		Where("id = ?", id).
		Update("status", status).Error
}
