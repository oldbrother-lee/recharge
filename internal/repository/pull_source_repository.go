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
	
	// 拉单源管理接口
	GetSources(ctx context.Context, name, code string, enabled *bool, offset, limit int) ([]model.PullSource, int64, error)
	CreateSource(ctx context.Context, source *model.PullSource) error
	UpdateSource(ctx context.Context, source *model.PullSource) error
	DeleteSource(ctx context.Context, id int64) error
	
	// 变体配置管理接口
	GetVariants(ctx context.Context, sourceID *int64, isp *int, enabled *bool, offset, limit int) ([]model.PullVariantConfig, int64, error)
	CreateVariant(ctx context.Context, variant *model.PullVariantConfig) error
	UpdateVariant(ctx context.Context, variant *model.PullVariantConfig) error
	DeleteVariant(ctx context.Context, id int64) error
	GetAllVariantsBySource(ctx context.Context, sourceID int64) ([]model.PullVariantConfig, error)
	
	// 账号绑定相关接口
	BindUser(ctx context.Context, sourceID int64, userID int64) error
	UnbindUser(ctx context.Context, sourceID int64) error
	GetSourcesWithUserName(ctx context.Context, name, code string, enabled *bool, offset, limit int) ([]model.PullSource, int64, error)
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

// 拉单源管理实现
func (r *PullSourceRepositoryImpl) GetSources(ctx context.Context, name, code string, enabled *bool, offset, limit int) ([]model.PullSource, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.PullSource{})
	
	if name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	if code != "" {
		query = query.Where("code LIKE ?", "%"+code+"%")
	}
	if enabled != nil {
		query = query.Where("enabled = ?", *enabled)
	}
	
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	
	var sources []model.PullSource
	if err := query.Offset(offset).Limit(limit).Order("id DESC").Find(&sources).Error; err != nil {
		return nil, 0, err
	}
	
	return sources, total, nil
}

func (r *PullSourceRepositoryImpl) CreateSource(ctx context.Context, source *model.PullSource) error {
	return r.db.WithContext(ctx).Create(source).Error
}

func (r *PullSourceRepositoryImpl) UpdateSource(ctx context.Context, source *model.PullSource) error {
	return r.db.WithContext(ctx).Save(source).Error
}

func (r *PullSourceRepositoryImpl) DeleteSource(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.PullSource{}, id).Error
}

// 变体配置管理实现
func (r *PullSourceRepositoryImpl) GetVariants(ctx context.Context, sourceID *int64, isp *int, enabled *bool, offset, limit int) ([]model.PullVariantConfig, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.PullVariantConfig{})
	
	if sourceID != nil {
		query = query.Where("source_id = ?", *sourceID)
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
	
	var variants []model.PullVariantConfig
	if err := query.Offset(offset).Limit(limit).Order("id DESC").Find(&variants).Error; err != nil {
		return nil, 0, err
	}
	
	return variants, total, nil
}

func (r *PullSourceRepositoryImpl) CreateVariant(ctx context.Context, variant *model.PullVariantConfig) error {
	return r.db.WithContext(ctx).Create(variant).Error
}

func (r *PullSourceRepositoryImpl) UpdateVariant(ctx context.Context, variant *model.PullVariantConfig) error {
	return r.db.WithContext(ctx).Save(variant).Error
}

func (r *PullSourceRepositoryImpl) DeleteVariant(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.PullVariantConfig{}, id).Error
}

func (r *PullSourceRepositoryImpl) GetAllVariantsBySource(ctx context.Context, sourceID int64) ([]model.PullVariantConfig, error) {
	var variants []model.PullVariantConfig
	if err := r.db.WithContext(ctx).Where("source_id = ?", sourceID).Find(&variants).Error; err != nil {
		return nil, err
	}
	return variants, nil
}

// BindUser 绑定用户到拉单源
func (r *PullSourceRepositoryImpl) BindUser(ctx context.Context, sourceID int64, userID int64) error {
	return r.db.WithContext(ctx).Model(&model.PullSource{}).
		Where("id = ?", sourceID).
		Update("bind_user_id", userID).Error
}

// UnbindUser 解绑拉单源的用户
func (r *PullSourceRepositoryImpl) UnbindUser(ctx context.Context, sourceID int64) error {
	return r.db.WithContext(ctx).Model(&model.PullSource{}).
		Where("id = ?", sourceID).
		Update("bind_user_id", nil).Error
}

// GetSourcesWithUserName 获取拉单源列表（包含绑定用户名）
func (r *PullSourceRepositoryImpl) GetSourcesWithUserName(ctx context.Context, name, code string, enabled *bool, offset, limit int) ([]model.PullSource, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.PullSource{}).
		Select("pull_sources.*, users.username as bind_user_name").
		Joins("LEFT JOIN users ON pull_sources.bind_user_id = users.id")
	
	if name != "" {
		query = query.Where("pull_sources.name LIKE ?", "%"+name+"%")
	}
	if code != "" {
		query = query.Where("pull_sources.code LIKE ?", "%"+code+"%")
	}
	if enabled != nil {
		query = query.Where("pull_sources.enabled = ?", *enabled)
	}
	
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	
	var sources []model.PullSource
	if err := query.Offset(offset).Limit(limit).Order("pull_sources.id DESC").Find(&sources).Error; err != nil {
		return nil, 0, err
	}
	
	return sources, total, nil
}