package repository

import (
	"context"
	"recharge-go/internal/model"

	"gorm.io/gorm"
)

// UserLogRepository 用户日志仓库
type UserLogRepository struct {
	db *gorm.DB
}

// NewUserLogRepository 创建用户日志仓库
func NewUserLogRepository(db *gorm.DB) *UserLogRepository {
	return &UserLogRepository{
		db: db,
	}
}

// Create 创建用户日志
func (r *UserLogRepository) Create(ctx context.Context, log *model.UserLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

// GetByID 根据ID获取用户日志
func (r *UserLogRepository) GetByID(ctx context.Context, id int64) (*model.UserLog, error) {
	var log model.UserLog
	err := r.db.WithContext(ctx).First(&log, id).Error
	if err != nil {
		return nil, err
	}
	return &log, nil
}

// List 获取用户日志列表
func (r *UserLogRepository) List(ctx context.Context, userID, targetID int64, action string, page, size int) ([]model.UserLog, int64, error) {
	var logs []model.UserLog
	var total int64

	query := r.db.WithContext(ctx).Model(&model.UserLog{})

	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}
	if targetID > 0 {
		query = query.Where("target_id = ?", targetID)
	}
	if action != "" {
		query = query.Where("action = ?", action)
	}

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Offset((page - 1) * size).Limit(size).Order("created_at DESC").Find(&logs).Error
	if err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}
