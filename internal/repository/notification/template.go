package notification

import (
	"context"
	"recharge-go/internal/model/notification"

	"gorm.io/gorm"
)

// TemplateRepository 通知模板仓库
type TemplateRepository struct {
	db *gorm.DB
}

// NewTemplateRepository 创建通知模板仓库
func NewTemplateRepository(db *gorm.DB) *TemplateRepository {
	return &TemplateRepository{db: db}
}

// Create 创建通知模板
func (r *TemplateRepository) Create(ctx context.Context, template *notification.Template) error {
	return r.db.WithContext(ctx).Create(template).Error
}

// Update 更新通知模板
func (r *TemplateRepository) Update(ctx context.Context, template *notification.Template) error {
	return r.db.WithContext(ctx).Save(template).Error
}

// GetByID 根据ID获取通知模板
func (r *TemplateRepository) GetByID(ctx context.Context, id int64) (*notification.Template, error) {
	var template notification.Template
	err := r.db.WithContext(ctx).First(&template, id).Error
	if err != nil {
		return nil, err
	}
	return &template, nil
}

// GetByPlatformAndType 根据平台和类型获取通知模板
func (r *TemplateRepository) GetByPlatformAndType(ctx context.Context, platformCode string, notificationType int) (*notification.Template, error) {
	var template notification.Template
	err := r.db.WithContext(ctx).
		Where("platform_code = ? AND notification_type = ? AND status = ?", platformCode, notificationType, 1).
		First(&template).Error
	if err != nil {
		return nil, err
	}
	return &template, nil
}

// List 获取通知模板列表
func (r *TemplateRepository) List(ctx context.Context, platformCode string) ([]*notification.Template, error) {
	var templates []*notification.Template
	query := r.db.WithContext(ctx)
	if platformCode != "" {
		query = query.Where("platform_code = ?", platformCode)
	}
	err := query.Find(&templates).Error
	return templates, err
}
