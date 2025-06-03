package repository

import (
	"context"
	"recharge-go/internal/model"

	"gorm.io/gorm"
)

// PlatformAPIRepository 平台接口仓库接口
type PlatformAPIRepository interface {
	// Create 创建平台接口
	Create(ctx context.Context, api *model.PlatformAPI) error
	// Update 更新平台接口
	Update(ctx context.Context, api *model.PlatformAPI) error
	// Delete 删除平台接口
	Delete(ctx context.Context, id int64) error
	// GetByID 根据ID获取平台接口
	GetByID(ctx context.Context, id int64) (*model.PlatformAPI, error)
	// GetByCode 根据代码获取平台接口
	GetByCode(ctx context.Context, code string) (*model.PlatformAPI, error)
	// List 获取平台接口列表
	List(ctx context.Context, page, pageSize int) ([]*model.PlatformAPI, int64, error)
}

// platformAPIRepository 平台接口仓库实现
type platformAPIRepository struct {
	db *gorm.DB
}

// NewPlatformAPIRepository 创建平台接口仓库实例
func NewPlatformAPIRepository(db *gorm.DB) PlatformAPIRepository {
	return &platformAPIRepository{db: db}
}

func (r *platformAPIRepository) Create(ctx context.Context, api *model.PlatformAPI) error {
	return r.db.WithContext(ctx).Create(api).Error
}

func (r *platformAPIRepository) Update(ctx context.Context, api *model.PlatformAPI) error {
	return r.db.WithContext(ctx).Save(api).Error
}

func (r *platformAPIRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.PlatformAPI{}, id).Error
}

func (r *platformAPIRepository) GetByID(ctx context.Context, id int64) (*model.PlatformAPI, error) {
	var api model.PlatformAPI
	err := r.db.WithContext(ctx).First(&api, id).Error
	if err != nil {
		return nil, err
	}
	return &api, nil
}

func (r *platformAPIRepository) GetByCode(ctx context.Context, code string) (*model.PlatformAPI, error) {
	var api model.PlatformAPI
	err := r.db.WithContext(ctx).Where("code = ?", code).First(&api).Error
	if err != nil {
		return nil, err
	}
	return &api, nil
}

func (r *platformAPIRepository) List(ctx context.Context, page, pageSize int) ([]*model.PlatformAPI, int64, error) {
	var apis []*model.PlatformAPI
	var total int64

	// 计算偏移量
	offset := (page - 1) * pageSize

	// 查询总数
	if err := r.db.WithContext(ctx).Model(&model.PlatformAPI{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 查询列表
	if err := r.db.WithContext(ctx).
		Offset(offset).
		Limit(pageSize).
		Order("id DESC").
		Find(&apis).Error; err != nil {
		return nil, 0, err
	}

	return apis, total, nil
}
