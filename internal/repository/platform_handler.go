package repository

import (
	"context"
	"recharge-go/internal/model"

	"gorm.io/gorm"
)

// PlatformHandlerRepository 平台处理器仓储接口
type PlatformHandlerRepository interface {
	// GetByPlatformName 根据平台名称获取处理器
	GetByPlatformName(ctx context.Context, platformName string) (*model.PlatformHandler, error)
	// Create 创建处理器
	Create(ctx context.Context, handler *model.PlatformHandler) error
	// Update 更新处理器
	Update(ctx context.Context, handler *model.PlatformHandler) error
	// Delete 删除处理器
	Delete(ctx context.Context, id int64) error
	// List 获取处理器列表
	List(ctx context.Context) ([]*model.PlatformHandler, error)
}

// PlatformHandlerRepositoryImpl 平台处理器仓储实现
type PlatformHandlerRepositoryImpl struct {
	db *gorm.DB
}

// NewPlatformHandlerRepository 创建平台处理器仓储
func NewPlatformHandlerRepository(db *gorm.DB) PlatformHandlerRepository {
	return &PlatformHandlerRepositoryImpl{db: db}
}

// GetByPlatformName 根据平台名称获取处理器
func (r *PlatformHandlerRepositoryImpl) GetByPlatformName(ctx context.Context, platformName string) (*model.PlatformHandler, error) {
	var handler model.PlatformHandler
	if err := r.db.Where("platform_name = ? AND status = 1", platformName).First(&handler).Error; err != nil {
		return nil, err
	}
	return &handler, nil
}

// Create 创建处理器
func (r *PlatformHandlerRepositoryImpl) Create(ctx context.Context, handler *model.PlatformHandler) error {
	return r.db.Create(handler).Error
}

// Update 更新处理器
func (r *PlatformHandlerRepositoryImpl) Update(ctx context.Context, handler *model.PlatformHandler) error {
	return r.db.Save(handler).Error
}

// Delete 删除处理器
func (r *PlatformHandlerRepositoryImpl) Delete(ctx context.Context, id int64) error {
	return r.db.Delete(&model.PlatformHandler{}, id).Error
}

// List 获取处理器列表
func (r *PlatformHandlerRepositoryImpl) List(ctx context.Context) ([]*model.PlatformHandler, error) {
	var handlers []*model.PlatformHandler
	if err := r.db.Find(&handlers).Error; err != nil {
		return nil, err
	}
	return handlers, nil
}
