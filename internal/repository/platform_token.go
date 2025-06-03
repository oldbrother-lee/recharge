package repository

import (
	"recharge-go/internal/model"
	"recharge-go/pkg/database"
	"time"

	"gorm.io/gorm"
)

type PlatformTokenRepository struct {
	db *gorm.DB
}

func NewPlatformTokenRepository() *PlatformTokenRepository {
	return &PlatformTokenRepository{db: database.DB}
}

// Get 获取指定任务配置的token
func (r *PlatformTokenRepository) Get(taskConfigID int64) (*model.PlatformToken, error) {
	var token model.PlatformToken
	err := r.db.Where("task_config_id = ?", taskConfigID).First(&token).Error
	if err != nil {
		return nil, err
	}
	return &token, nil
}

// Save 保存/更新指定任务配置的token
func (r *PlatformTokenRepository) Save(taskConfigID int64, token string) error {
	now := time.Now()
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 先删除旧的token
		if err := tx.Where("task_config_id = ?", taskConfigID).Delete(&model.PlatformToken{}).Error; err != nil {
			return err
		}
		// 创建新的token
		return tx.Create(&model.PlatformToken{
			TaskConfigID: taskConfigID,
			Token:        token,
			LastUsedAt:   now,
		}).Error
	})
}

// UpdateLastUsed 更新token的最后使用时间
func (r *PlatformTokenRepository) UpdateLastUsed(taskConfigID int64) error {
	return r.db.Model(&model.PlatformToken{}).
		Where("task_config_id = ?", taskConfigID).
		Update("last_used_at", time.Now()).
		Error
}

// Delete 删除指定任务配置的token
func (r *PlatformTokenRepository) Delete(taskConfigID int64) error {
	return r.db.Where("task_config_id = ?", taskConfigID).Delete(&model.PlatformToken{}).Error
}
