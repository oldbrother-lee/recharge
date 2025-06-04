package repository

import (
	"recharge-go/internal/model"

	"gorm.io/gorm"
)

type TaskConfigRepository struct {
	db *gorm.DB
}

func NewTaskConfigRepository(db *gorm.DB) *TaskConfigRepository {
	return &TaskConfigRepository{
		db: db,
	}
}

// Create 创建任务配置
func (r *TaskConfigRepository) Create(config *model.TaskConfig) error {
	return r.db.Create(config).Error
}

// Update 更新任务配置
func (r *TaskConfigRepository) Update(config *model.TaskConfig) error {
	return r.db.Save(config).Error
}

// Delete 删除任务配置
func (r *TaskConfigRepository) Delete(id int64) error {
	return r.db.Delete(&model.TaskConfig{}, id).Error
}

// GetByID 根据ID获取任务配置
func (r *TaskConfigRepository) GetByID(id int64) (*model.TaskConfig, error) {
	var config model.TaskConfig
	err := r.db.First(&config, id).Error
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// List 获取任务配置列表
func (r *TaskConfigRepository) List(page, pageSize int, platformAccountID *int64) ([]*model.TaskConfig, int64, error) {
	var configs []*model.TaskConfig
	var total int64

	offset := (page - 1) * pageSize

	db := r.db.Model(&model.TaskConfig{})
	if platformAccountID != nil {
		db = db.Where("platform_account_id = ?", *platformAccountID)
	}

	// 获取总数
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取分页数据
	if err := db.Offset(offset).Limit(pageSize).Find(&configs).Error; err != nil {
		return nil, 0, err
	}

	return configs, total, nil
}

// GetEnabledConfigs 获取所有启用的取单任务配置
func (r *TaskConfigRepository) GetEnabledConfigs() ([]model.TaskConfig, error) {
	var configs []model.TaskConfig
	err := r.db.Where("status = ?", 1).Find(&configs).Error
	if err != nil {
		return nil, err
	}
	return configs, nil
}

// Upsert: 如果 ChannelID 存在则更新，否则插入
func (r *TaskConfigRepository) Upsert(config *model.TaskConfig) error {
	var existing model.TaskConfig
	err := r.db.Where("channel_id = ?", config.ChannelID).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		return r.db.Create(config).Error
	} else if err != nil {
		return err
	}
	// 存在则更新
	return r.db.Model(&existing).Updates(config).Error
}

// BatchCreate 批量创建任务配置
func (r *TaskConfigRepository) BatchCreate(configs []*model.TaskConfig) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&configs).Error; err != nil {
			return err
		}
		return nil
	})
}
