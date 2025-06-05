package repository

import (
	"recharge-go/internal/model"

	"gorm.io/gorm"
)

type SystemConfigRepository struct {
	db *gorm.DB
}

func NewSystemConfigRepository(db *gorm.DB) *SystemConfigRepository {
	return &SystemConfigRepository{
		db: db,
	}
}

// Create 创建系统配置
func (r *SystemConfigRepository) Create(config *model.SystemConfig) error {
	return r.db.Create(config).Error
}

// Update 更新系统配置
func (r *SystemConfigRepository) Update(config *model.SystemConfig) error {
	return r.db.Save(config).Error
}

// Delete 删除系统配置
func (r *SystemConfigRepository) Delete(id int64) error {
	return r.db.Delete(&model.SystemConfig{}, id).Error
}

// GetByID 根据ID获取系统配置
func (r *SystemConfigRepository) GetByID(id int64) (*model.SystemConfig, error) {
	var config model.SystemConfig
	err := r.db.First(&config, id).Error
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// GetByKey 根据配置键获取系统配置
func (r *SystemConfigRepository) GetByKey(key string) (*model.SystemConfig, error) {
	var config model.SystemConfig
	err := r.db.Where("config_key = ? AND status = ?", key, 1).First(&config).Error
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// GetList 获取系统配置列表
func (r *SystemConfigRepository) GetList(page, pageSize int, configKey string) ([]model.SystemConfig, int64, error) {
	var configs []model.SystemConfig
	var total int64

	query := r.db.Model(&model.SystemConfig{})

	// 如果有配置键搜索条件
	if configKey != "" {
		query = query.Where("config_key LIKE ?", "%"+configKey+"%")
	}

	// 获取总数
	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err = query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&configs).Error
	if err != nil {
		return nil, 0, err
	}

	return configs, total, nil
}

// GetAllEnabled 获取所有启用的系统配置
func (r *SystemConfigRepository) GetAllEnabled() ([]model.SystemConfig, error) {
	var configs []model.SystemConfig
	err := r.db.Where("status = ?", 1).Find(&configs).Error
	if err != nil {
		return nil, err
	}
	return configs, nil
}

// UpdateByKey 根据配置键更新配置值
func (r *SystemConfigRepository) UpdateByKey(key, value string) error {
	return r.db.Model(&model.SystemConfig{}).Where("config_key = ?", key).Update("config_value", value).Error
}

// BatchUpdateByKeys 批量更新配置
func (r *SystemConfigRepository) BatchUpdateByKeys(configs map[string]string) error {
	tx := r.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	for key, value := range configs {
		if err := tx.Model(&model.SystemConfig{}).Where("config_key = ?", key).Update("config_value", value).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}
