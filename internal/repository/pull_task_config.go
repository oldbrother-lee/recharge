package repository

import (
	"context"
	"recharge-go/internal/model"

	"gorm.io/gorm"
)

type PullTaskConfigRepository interface {
	Create(ctx context.Context, cfg *model.PullTaskConfig) error
	Update(ctx context.Context, cfg *model.PullTaskConfig) error
	Delete(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (*model.PullTaskConfig, error)
	List(ctx context.Context, platformAccountID int64, platformCode string, page, pageSize int) ([]model.PullTaskConfig, int64, error)
}

type pullTaskConfigRepository struct {
	db *gorm.DB
}

func NewPullTaskConfigRepository(db *gorm.DB) PullTaskConfigRepository {
	return &pullTaskConfigRepository{db: db}
}

func (r *pullTaskConfigRepository) Create(ctx context.Context, cfg *model.PullTaskConfig) error {
	return r.db.WithContext(ctx).Create(cfg).Error
}

func (r *pullTaskConfigRepository) Update(ctx context.Context, cfg *model.PullTaskConfig) error {
	return r.db.WithContext(ctx).Save(cfg).Error
}

func (r *pullTaskConfigRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.PullTaskConfig{}, id).Error
}

func (r *pullTaskConfigRepository) GetByID(ctx context.Context, id int64) (*model.PullTaskConfig, error) {
	var cfg model.PullTaskConfig
	if err := r.db.WithContext(ctx).First(&cfg, id).Error; err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (r *pullTaskConfigRepository) List(ctx context.Context, platformAccountID int64, platformCode string, page, pageSize int) ([]model.PullTaskConfig, int64, error) {
	var list []model.PullTaskConfig
	var total int64
	q := r.db.WithContext(ctx).Model(&model.PullTaskConfig{})
	if platformAccountID > 0 {
		q = q.Where("platform_account_id = ?", platformAccountID)
	}
	if platformCode != "" {
		q = q.Where("platform_code = ?", platformCode)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}