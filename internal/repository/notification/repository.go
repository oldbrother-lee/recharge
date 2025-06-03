package notification

import (
	"context"
	"recharge-go/internal/model/notification"
	"time"

	"gorm.io/gorm"
)

// Repository 通知记录仓库接口
type Repository interface {
	Create(ctx context.Context, record *notification.NotificationRecord) error
	UpdateStatus(ctx context.Context, id int64, status int) error
	GetByID(ctx context.Context, id int64) (*notification.NotificationRecord, error)
	GetPendingRecords(ctx context.Context, limit int) ([]*notification.NotificationRecord, error)
	List(ctx context.Context, params map[string]interface{}, page, pageSize int) ([]*notification.NotificationRecord, int64, error)
	Update(ctx context.Context, record *notification.NotificationRecord) error
}

// RepositoryImpl 通知记录仓库实现
type RepositoryImpl struct {
	db *gorm.DB
}

// NewRepository 创建通知记录仓库实例
func NewRepository(db *gorm.DB) Repository {
	return &RepositoryImpl{db: db}
}

// Create 创建通知记录
func (r *RepositoryImpl) Create(ctx context.Context, record *notification.NotificationRecord) error {
	return r.db.WithContext(ctx).Create(record).Error
}

// UpdateStatus 更新通知状态
func (r *RepositoryImpl) UpdateStatus(ctx context.Context, id int64, status int) error {
	if status == 3 {
		return r.db.WithContext(ctx).Model(&notification.NotificationRecord{}).
			Where("id = ?", id).
			Updates(map[string]interface{}{
				"status":     status,
				"success_at": time.Now(),
			}).Error
	}
	return r.db.WithContext(ctx).Model(&notification.NotificationRecord{}).
		Where("id = ?", id).
		Update("status", status).Error
}

// GetByID 根据ID获取通知记录
func (r *RepositoryImpl) GetByID(ctx context.Context, id int64) (*notification.NotificationRecord, error) {
	var record notification.NotificationRecord
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// GetPendingRecords 获取待处理的通知记录
func (r *RepositoryImpl) GetPendingRecords(ctx context.Context, limit int) ([]*notification.NotificationRecord, error) {
	var records []*notification.NotificationRecord
	err := r.db.WithContext(ctx).
		Where("status = ?", 1). // 待处理状态
		Limit(limit).
		Find(&records).Error
	if err != nil {
		return nil, err
	}
	return records, nil
}

// List 获取通知记录列表
func (r *RepositoryImpl) List(ctx context.Context, params map[string]interface{}, page, pageSize int) ([]*notification.NotificationRecord, int64, error) {
	var records []*notification.NotificationRecord
	var total int64

	query := r.db.WithContext(ctx).Model(&notification.NotificationRecord{})

	// 添加查询条件
	for key, value := range params {
		query = query.Where(key+" = ?", value)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	err := query.Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&records).Error
	if err != nil {
		return nil, 0, err
	}

	return records, total, nil
}

// Update 更新通知记录
func (r *RepositoryImpl) Update(ctx context.Context, record *notification.NotificationRecord) error {
	return r.db.WithContext(ctx).Save(record).Error
}
