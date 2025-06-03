package repository

import (
	"context"
	"errors"
	"recharge-go/internal/model"

	"gorm.io/gorm"
)

type ExternalOrderLogRepository interface {
	Create(ctx context.Context, log *model.ExternalOrderLog) error
	UpdateStatus(ctx context.Context, platform, orderID string, status int, errorMsg string) error
	GetLogs(ctx context.Context, req struct {
		Platform  string
		OrderID   string
		Mobile    string
		Status    int
		StartTime string
		EndTime   string
		Page      int
		PageSize  int
	}) ([]*model.ExternalOrderLog, int64, error)
	GetByPlatformAndOrderID(ctx context.Context, platform, orderID string) (*model.ExternalOrderLog, error)
}

type externalOrderLogRepository struct {
	db *gorm.DB
}

func NewExternalOrderLogRepository(db *gorm.DB) ExternalOrderLogRepository {
	return &externalOrderLogRepository{db: db}
}

func (r *externalOrderLogRepository) Create(ctx context.Context, log *model.ExternalOrderLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *externalOrderLogRepository) UpdateStatus(ctx context.Context, platform, orderID string, status int, errorMsg string) error {
	return r.db.WithContext(ctx).
		Model(&model.ExternalOrderLog{}).
		Where("platform = ? AND order_id = ?", platform, orderID).
		Updates(map[string]interface{}{
			"status":    status,
			"error_msg": errorMsg,
		}).Error
}

func (r *externalOrderLogRepository) GetLogs(ctx context.Context, req struct {
	Platform  string
	OrderID   string
	Mobile    string
	Status    int
	StartTime string
	EndTime   string
	Page      int
	PageSize  int
}) ([]*model.ExternalOrderLog, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.ExternalOrderLog{})

	if req.Platform != "" {
		query = query.Where("platform = ?", req.Platform)
	}
	if req.OrderID != "" {
		query = query.Where("order_id = ?", req.OrderID)
	}
	if req.Mobile != "" {
		query = query.Where("mobile = ?", req.Mobile)
	}
	if req.Status != 0 {
		query = query.Where("status = ?", req.Status)
	}
	if req.StartTime != "" {
		query = query.Where("create_time >= ?", req.StartTime)
	}
	if req.EndTime != "" {
		query = query.Where("create_time <= ?", req.EndTime)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var logs []*model.ExternalOrderLog
	if err := query.Offset((req.Page - 1) * req.PageSize).
		Limit(req.PageSize).
		Order("create_time DESC").
		Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

func (r *externalOrderLogRepository) GetByPlatformAndOrderID(ctx context.Context, platform, orderID string) (*model.ExternalOrderLog, error) {
	var log model.ExternalOrderLog
	err := r.db.WithContext(ctx).
		Where("platform = ? AND order_id = ?", platform, orderID).
		First(&log).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &log, nil
}
