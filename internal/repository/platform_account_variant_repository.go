package repository

import (
	"context"
	"recharge-go/internal/model"

	"gorm.io/gorm"
)

// PlatformAccountVariantRepository 平台账号变体仓库接口
type PlatformAccountVariantRepository interface {
	// 基础CRUD操作
	Create(ctx context.Context, variant *model.PlatformAccountVariant) error
	Update(ctx context.Context, variant *model.PlatformAccountVariant) error
	Delete(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (*model.PlatformAccountVariant, error)
	
	// 查询操作
	GetByPlatformAccountID(ctx context.Context, platformAccountID int64) ([]model.PlatformAccountVariant, error)
	GetEnabledVariants(ctx context.Context, platformAccountID int64) ([]model.PlatformAccountVariant, error)
	GetByISPAndFaceValue(ctx context.Context, platformAccountID int64, isp int, faceValue float64) (*model.PlatformAccountVariant, error)
	List(ctx context.Context, platformAccountID *int64, isp *int, enabled *bool, offset, limit int) ([]model.PlatformAccountVariant, int64, error)
	
	// 拉单相关操作
	UpdateCursor(ctx context.Context, variantID int64, cursor string) error
	UpdateLastPullAt(ctx context.Context, variantID int64) error
	IncrementFailCount(ctx context.Context, variantID int64) error
	ResetFailCount(ctx context.Context, variantID int64) error
}

// platformAccountVariantRepository 平台账号变体仓库实现
type platformAccountVariantRepository struct {
	db *gorm.DB
}

// NewPlatformAccountVariantRepository 创建平台账号变体仓库实例
func NewPlatformAccountVariantRepository(db *gorm.DB) PlatformAccountVariantRepository {
	return &platformAccountVariantRepository{db: db}
}

// Create 创建变体
func (r *platformAccountVariantRepository) Create(ctx context.Context, variant *model.PlatformAccountVariant) error {
	return r.db.WithContext(ctx).Create(variant).Error
}

// Update 更新变体
func (r *platformAccountVariantRepository) Update(ctx context.Context, variant *model.PlatformAccountVariant) error {
	return r.db.WithContext(ctx).Save(variant).Error
}

// Delete 删除变体
func (r *platformAccountVariantRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.PlatformAccountVariant{}, id).Error
}

// GetByID 根据ID获取变体
func (r *platformAccountVariantRepository) GetByID(ctx context.Context, id int64) (*model.PlatformAccountVariant, error) {
	var variant model.PlatformAccountVariant
	if err := r.db.WithContext(ctx).Preload("PlatformAccount").First(&variant, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &variant, nil
}

// GetByPlatformAccountID 根据平台账号ID获取变体列表
func (r *platformAccountVariantRepository) GetByPlatformAccountID(ctx context.Context, platformAccountID int64) ([]model.PlatformAccountVariant, error) {
	var variants []model.PlatformAccountVariant
	if err := r.db.WithContext(ctx).Where("platform_account_id = ?", platformAccountID).Find(&variants).Error; err != nil {
		return nil, err
	}
	return variants, nil
}

// GetEnabledVariants 获取启用的变体列表
func (r *platformAccountVariantRepository) GetEnabledVariants(ctx context.Context, platformAccountID int64) ([]model.PlatformAccountVariant, error) {
	var variants []model.PlatformAccountVariant
	if err := r.db.WithContext(ctx).Where("platform_account_id = ? AND enabled = ?", platformAccountID, true).Find(&variants).Error; err != nil {
		return nil, err
	}
	return variants, nil
}

// GetByISPAndFaceValue 根据运营商和面值获取变体
func (r *platformAccountVariantRepository) GetByISPAndFaceValue(ctx context.Context, platformAccountID int64, isp int, faceValue float64) (*model.PlatformAccountVariant, error) {
	var variant model.PlatformAccountVariant
	if err := r.db.WithContext(ctx).Where("platform_account_id = ? AND isp = ? AND face_value = ?", platformAccountID, isp, faceValue).First(&variant).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &variant, nil
}

// List 获取变体列表
func (r *platformAccountVariantRepository) List(ctx context.Context, platformAccountID *int64, isp *int, enabled *bool, offset, limit int) ([]model.PlatformAccountVariant, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.PlatformAccountVariant{}).Preload("PlatformAccount")
	
	if platformAccountID != nil {
		query = query.Where("platform_account_id = ?", *platformAccountID)
	}
	if isp != nil {
		query = query.Where("isp = ?", *isp)
	}
	if enabled != nil {
		query = query.Where("enabled = ?", *enabled)
	}
	
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	
	var variants []model.PlatformAccountVariant
	if err := query.Offset(offset).Limit(limit).Order("id DESC").Find(&variants).Error; err != nil {
		return nil, 0, err
	}
	
	return variants, total, nil
}

// UpdateCursor 更新拉取游标
func (r *platformAccountVariantRepository) UpdateCursor(ctx context.Context, variantID int64, cursor string) error {
	return r.db.WithContext(ctx).Model(&model.PlatformAccountVariant{}).
		Where("id = ?", variantID).
		Update("cursor_token", cursor).Error
}

// UpdateLastPullAt 更新最后拉取时间
func (r *platformAccountVariantRepository) UpdateLastPullAt(ctx context.Context, variantID int64) error {
	return r.db.WithContext(ctx).Model(&model.PlatformAccountVariant{}).
		Where("id = ?", variantID).
		Update("last_pull_at", gorm.Expr("NOW()")).Error
}

// IncrementFailCount 增加失败计数
func (r *platformAccountVariantRepository) IncrementFailCount(ctx context.Context, variantID int64) error {
	return r.db.WithContext(ctx).Model(&model.PlatformAccountVariant{}).
		Where("id = ?", variantID).
		Update("fail_count", gorm.Expr("fail_count + 1")).Error
}

// ResetFailCount 重置失败计数
func (r *platformAccountVariantRepository) ResetFailCount(ctx context.Context, variantID int64) error {
	return r.db.WithContext(ctx).Model(&model.PlatformAccountVariant{}).
		Where("id = ?", variantID).
		Update("fail_count", 0).Error
}