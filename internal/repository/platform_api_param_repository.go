package repository

import (
	"context"
	"recharge-go/internal/model"

	"gorm.io/gorm"
)

// PlatformAPIParamRepository 平台接口参数仓库接口
type PlatformAPIParamRepository interface {
	// Create 创建平台接口参数
	Create(ctx context.Context, param *model.PlatformAPIParam) error
	// Update 更新平台接口参数
	Update(ctx context.Context, param *model.PlatformAPIParam) error
	// Delete 删除平台接口参数
	Delete(ctx context.Context, id int64) error
	// GetByID 根据ID获取平台接口参数
	GetByID(ctx context.Context, id int64) (*model.PlatformAPIParam, error)
	// List 获取平台接口参数列表
	List(ctx context.Context, apiID int64, page, pageSize int) ([]*model.PlatformAPIParam, int64, error)
}

// platformAPIParamRepository 平台接口参数仓库实现
type platformAPIParamRepository struct {
	db *gorm.DB
}

// NewPlatformAPIParamRepository 创建平台接口参数仓库实例
func NewPlatformAPIParamRepository(db *gorm.DB) PlatformAPIParamRepository {
	return &platformAPIParamRepository{db: db}
}

func (r *platformAPIParamRepository) Create(ctx context.Context, param *model.PlatformAPIParam) error {
	return r.db.WithContext(ctx).Create(param).Error
}

func (r *platformAPIParamRepository) Update(ctx context.Context, param *model.PlatformAPIParam) error {
	return r.db.WithContext(ctx).Save(param).Error
}

func (r *platformAPIParamRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.PlatformAPIParam{}, id).Error
}

func (r *platformAPIParamRepository) GetByID(ctx context.Context, id int64) (*model.PlatformAPIParam, error) {
	var param model.PlatformAPIParam
	err := r.db.WithContext(ctx).First(&param, id).Error
	if err != nil {
		return nil, err
	}
	return &param, nil
}

func (r *platformAPIParamRepository) List(ctx context.Context, apiID int64, page, pageSize int) ([]*model.PlatformAPIParam, int64, error) {
	var params []*model.PlatformAPIParam
	var total int64

	query := r.db.WithContext(ctx).Model(&model.PlatformAPIParam{})
	if apiID > 0 {
		query = query.Where("api_id = ?", apiID)
	}

	// 查询总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 查询列表
	if err := query.
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Order("id DESC").
		Find(&params).Error; err != nil {
		return nil, 0, err
	}

	return params, total, nil
}
