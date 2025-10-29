package repository

import (
	"context"
	"recharge-go/internal/model"

	"gorm.io/gorm"
)

// PullSourceRepository 拉单平台源仓库
 type PullSourceRepository interface {
	GetEnabledSources(ctx context.Context) ([]model.PullSource, error)
	GetVariantsBySource(ctx context.Context, sourceID int64) ([]model.PullVariantConfig, error)
	GetMapByIspDenom(ctx context.Context, sourceID int64, isp int, faceValue float64) (*model.PullProductMap, error)
	GetMapByExternalCode(ctx context.Context, sourceID int64, externalCode string) (*model.PullProductMap, error)
	UpdateVariantCursor(ctx context.Context, variantID int64, cursor string) error
	// 新增：按ID获取变体与源
	GetVariantByID(ctx context.Context, id int64) (*model.PullVariantConfig, error)
	GetSourceByID(ctx context.Context, id int64) (*model.PullSource, error)
}

// PullSourceRepositoryImpl 实现
 type PullSourceRepositoryImpl struct {
	db *gorm.DB
}

func NewPullSourceRepository(db *gorm.DB) *PullSourceRepositoryImpl { return &PullSourceRepositoryImpl{db: db} }

func (r *PullSourceRepositoryImpl) GetEnabledSources(ctx context.Context) ([]model.PullSource, error) {
	var list []model.PullSource
	if err := r.db.WithContext(ctx).Where("enabled = ?", true).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *PullSourceRepositoryImpl) GetVariantsBySource(ctx context.Context, sourceID int64) ([]model.PullVariantConfig, error) {
	var list []model.PullVariantConfig
	if err := r.db.WithContext(ctx).Where("source_id = ? AND enabled = ?", sourceID, true).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *PullSourceRepositoryImpl) GetMapByIspDenom(ctx context.Context, sourceID int64, isp int, faceValue float64) (*model.PullProductMap, error) {
	var m model.PullProductMap
	if err := r.db.WithContext(ctx).Where("source_id = ? AND isp = ? AND face_value = ?", sourceID, isp, faceValue).First(&m).Error; err != nil {
		if err == gorm.ErrRecordNotFound { return nil, nil }
		return nil, err
	}
	return &m, nil
}

func (r *PullSourceRepositoryImpl) GetMapByExternalCode(ctx context.Context, sourceID int64, externalCode string) (*model.PullProductMap, error) {
	var m model.PullProductMap
	if err := r.db.WithContext(ctx).Where("source_id = ? AND external_code = ?", sourceID, externalCode).First(&m).Error; err != nil {
		if err == gorm.ErrRecordNotFound { return nil, nil }
		return nil, err
	}
	return &m, nil
}

func (r *PullSourceRepositoryImpl) UpdateVariantCursor(ctx context.Context, variantID int64, cursor string) error {
	return r.db.WithContext(ctx).Model(&model.PullVariantConfig{}).Where("id = ?", variantID).Update("cursor_token", cursor).Error
}

// 新增：按ID获取变体
func (r *PullSourceRepositoryImpl) GetVariantByID(ctx context.Context, id int64) (*model.PullVariantConfig, error) {
	var v model.PullVariantConfig
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&v).Error; err != nil {
		if err == gorm.ErrRecordNotFound { return nil, nil }
		return nil, err
	}
	return &v, nil
}

// 新增：按ID获取拉单源
func (r *PullSourceRepositoryImpl) GetSourceByID(ctx context.Context, id int64) (*model.PullSource, error) {
	var s model.PullSource
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&s).Error; err != nil {
		if err == gorm.ErrRecordNotFound { return nil, nil }
		return nil, err
	}
	return &s, nil
}