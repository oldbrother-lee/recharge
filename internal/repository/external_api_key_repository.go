package repository

import (
	"recharge-go/internal/model"

	"gorm.io/gorm"
)

// ExternalAPIKeyRepository 外部API密钥仓库接口
type ExternalAPIKeyRepository interface {
	GetByAppID(appID string) (*model.ExternalAPIKey, error)
	GetByAppKey(appKey string) (*model.ExternalAPIKey, error)
	GetByUserID(userID int64, offset, limit int) ([]*model.ExternalAPIKey, int64, error)
	Create(apiKey *model.ExternalAPIKey) error
	Update(apiKey *model.ExternalAPIKey) error
	Delete(id int64) error
	List(offset, limit int) ([]*model.ExternalAPIKey, int64, error)
}

// externalAPIKeyRepository 外部API密钥仓库实现
type externalAPIKeyRepository struct {
	db *gorm.DB
}

// NewExternalAPIKeyRepository 创建外部API密钥仓库
func NewExternalAPIKeyRepository(db *gorm.DB) ExternalAPIKeyRepository {
	return &externalAPIKeyRepository{db: db}
}

// GetByAppID 根据应用ID获取API密钥
func (r *externalAPIKeyRepository) GetByAppID(appID string) (*model.ExternalAPIKey, error) {
	var apiKey model.ExternalAPIKey
	err := r.db.Where("app_id = ?", appID).First(&apiKey).Error
	if err != nil {
		return nil, err
	}
	return &apiKey, nil
}

// GetByAppKey 根据应用密钥获取API密钥
func (r *externalAPIKeyRepository) GetByAppKey(appKey string) (*model.ExternalAPIKey, error) {
	var apiKey model.ExternalAPIKey
	err := r.db.Where("app_key = ?", appKey).First(&apiKey).Error
	if err != nil {
		return nil, err
	}
	return &apiKey, nil
}

// GetByUserID 根据用户ID获取API密钥列表
func (r *externalAPIKeyRepository) GetByUserID(userID int64, offset, limit int) ([]*model.ExternalAPIKey, int64, error) {
	var apiKeys []*model.ExternalAPIKey
	var total int64

	// 获取总数
	err := r.db.Model(&model.ExternalAPIKey{}).Where("user_id = ?", userID).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	// 获取列表
	err = r.db.Where("user_id = ?", userID).Offset(offset).Limit(limit).Find(&apiKeys).Error
	if err != nil {
		return nil, 0, err
	}

	return apiKeys, total, nil
}

// Create 创建API密钥
func (r *externalAPIKeyRepository) Create(apiKey *model.ExternalAPIKey) error {
	return r.db.Create(apiKey).Error
}

// Update 更新API密钥
func (r *externalAPIKeyRepository) Update(apiKey *model.ExternalAPIKey) error {
	return r.db.Save(apiKey).Error
}

// Delete 删除API密钥
func (r *externalAPIKeyRepository) Delete(id int64) error {
	return r.db.Delete(&model.ExternalAPIKey{}, id).Error
}

// List 获取API密钥列表
func (r *externalAPIKeyRepository) List(offset, limit int) ([]*model.ExternalAPIKey, int64, error) {
	var apiKeys []*model.ExternalAPIKey
	var total int64

	// 获取总数
	err := r.db.Model(&model.ExternalAPIKey{}).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	// 获取列表
	err = r.db.Offset(offset).Limit(limit).Find(&apiKeys).Error
	if err != nil {
		return nil, 0, err
	}

	return apiKeys, total, nil
}
