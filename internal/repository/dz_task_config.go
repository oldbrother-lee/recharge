package repository

import (
	"recharge-go/internal/model"

	"gorm.io/gorm"
)

type DzTaskConfigRepository struct {
    db *gorm.DB
}

func NewDzTaskConfigRepository(db *gorm.DB) *DzTaskConfigRepository {
	return &DzTaskConfigRepository{
		db: db,
	}
}

// Create 创建得众任务配置
func (r *DzTaskConfigRepository) Create(config *model.DzTaskConfig) error {
	return r.db.Create(config).Error
}

// Update 更新得众任务配置
func (r *DzTaskConfigRepository) Update(config *model.DzTaskConfig) error {
	return r.db.Save(config).Error
}

// Delete 删除得众任务配置
func (r *DzTaskConfigRepository) Delete(id int64) error {
	return r.db.Delete(&model.DzTaskConfig{}, id).Error
}

// GetByID 根据ID获取得众任务配置
func (r *DzTaskConfigRepository) GetByID(id int64) (*model.DzTaskConfig, error) {
    var config model.DzTaskConfig
    err := r.db.
        Select("id, platform_id, platform_name, platform_account_id, platform_account, product_id, product_name, isp, CAST(face_value AS SIGNED) AS face_value, poll_interval_sec, concurrency, enabled, created_at, updated_at").
        First(&config, id).Error
    if err != nil {
        return nil, err
    }
    return &config, nil
}

// List 获取得众任务配置列表
func (r *DzTaskConfigRepository) List(page, pageSize int, platformAccountID *int64) ([]*model.DzTaskConfig, int64, error) {
    var configs []*model.DzTaskConfig
    var total int64

    query := r.db.Model(&model.DzTaskConfig{})
    
    if platformAccountID != nil {
        query = query.Where("platform_account_id = ?", *platformAccountID)
    }

    // 获取总数
    if err := query.Count(&total).Error; err != nil {
        return nil, 0, err
    }

    // 分页查询
    offset := (page - 1) * pageSize
    if err := query.
        Select("id, platform_id, platform_name, platform_account_id, platform_account, product_id, product_name, isp, CAST(face_value AS SIGNED) AS face_value, poll_interval_sec, concurrency, enabled, created_at, updated_at").
        Offset(offset).Limit(pageSize).Find(&configs).Error; err != nil {
        return nil, 0, err
    }

    return configs, total, nil
}

// GetEnabledConfigs 获取所有启用的得众任务配置
func (r *DzTaskConfigRepository) GetEnabledConfigs() ([]model.DzTaskConfig, error) {
    var configs []model.DzTaskConfig
    err := r.db.
        Select("id, platform_id, platform_name, platform_account_id, platform_account, product_id, product_name, isp, CAST(face_value AS SIGNED) AS face_value, poll_interval_sec, concurrency, enabled, created_at, updated_at").
        Where("enabled = ?", 1).Find(&configs).Error
    if err != nil {
        return nil, err
    }
    return configs, nil
}

// BatchCreate 批量创建得众任务配置
func (r *DzTaskConfigRepository) BatchCreate(configs []*model.DzTaskConfig) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&configs).Error; err != nil {
			return err
		}
		return nil
	})
}

// UpdatePartial 部分更新得众任务配置
func (r *DzTaskConfigRepository) UpdatePartial(id int64, updates map[string]interface{}) error {
	return r.db.Model(&model.DzTaskConfig{}).Where("id = ?", id).Updates(updates).Error
}