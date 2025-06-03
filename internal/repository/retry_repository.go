package repository

import (
	"context"
	"recharge-go/internal/model"

	"gorm.io/gorm"
)

type RetryRepository interface {
	Create(ctx context.Context, record *model.OrderRetryRecord) error
	Update(ctx context.Context, record *model.OrderRetryRecord) error
	GetByID(ctx context.Context, id int64) (*model.OrderRetryRecord, error)
	GetPendingRetries(ctx context.Context) ([]*model.OrderRetryRecord, error)
	GetByOrderID(ctx context.Context, orderID int64) ([]*model.OrderRetryRecord, error)
}

type retryRepository struct {
	db *gorm.DB
}

func NewRetryRepository(db *gorm.DB) RetryRepository {
	return &retryRepository{db: db}
}

func (r *retryRepository) Create(ctx context.Context, record *model.OrderRetryRecord) error {
	return r.db.WithContext(ctx).Create(record).Error
}

func (r *retryRepository) Update(ctx context.Context, record *model.OrderRetryRecord) error {
	return r.db.WithContext(ctx).Save(record).Error
}

func (r *retryRepository) GetByID(ctx context.Context, id int64) (*model.OrderRetryRecord, error) {
	var record model.OrderRetryRecord
	err := r.db.WithContext(ctx).First(&record, id).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *retryRepository) GetPendingRetries(ctx context.Context) ([]*model.OrderRetryRecord, error) {
	var records []*model.OrderRetryRecord
	err := r.db.WithContext(ctx).
		Where("status IN (0, 1) AND next_retry_time <= NOW()").
		Order("next_retry_time ASC").
		Find(&records).Error
	return records, err
}

func (r *retryRepository) GetByOrderID(ctx context.Context, orderID int64) ([]*model.OrderRetryRecord, error) {
	var records []*model.OrderRetryRecord
	err := r.db.WithContext(ctx).
		Where("order_id = ?", orderID).
		Find(&records).Error
	return records, err
}
