package service

import (
	"context"
	"errors"
	"recharge-go/internal/model"
	"recharge-go/internal/repository"

	"gorm.io/gorm"
)

// PlatformAPIService 平台接口服务接口
type PlatformAPIService interface {
	// CreateAPI 创建平台接口
	CreateAPI(ctx context.Context, api *model.PlatformAPI) error
	// UpdateAPI 更新平台接口
	UpdateAPI(ctx context.Context, api *model.PlatformAPI) error
	// DeleteAPI 删除平台接口
	DeleteAPI(ctx context.Context, id int64) error
	// GetAPI 获取平台接口详情
	GetAPI(ctx context.Context, id int64) (*model.PlatformAPI, error)
	// GetAPIByCode 根据代码获取平台接口
	GetAPIByCode(ctx context.Context, code string) (*model.PlatformAPI, error)
	// ListAPIs 获取平台接口列表
	ListAPIs(ctx context.Context, page, pageSize int) ([]*model.PlatformAPI, int64, error)
}

// platformAPIService 平台接口服务实现
type platformAPIService struct {
	repo repository.PlatformAPIRepository
}

// NewPlatformAPIService 创建平台接口服务实例
func NewPlatformAPIService(repo repository.PlatformAPIRepository) PlatformAPIService {
	return &platformAPIService{repo: repo}
}

func (s *platformAPIService) CreateAPI(ctx context.Context, api *model.PlatformAPI) error {
	// 检查接口代码是否已存在
	existing, err := s.repo.GetByCode(ctx, api.Code)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if existing != nil {
		return errors.New("接口代码已存在")
	}

	return s.repo.Create(ctx, api)
}

func (s *platformAPIService) UpdateAPI(ctx context.Context, api *model.PlatformAPI) error {
	// 检查接口是否存在
	existing, err := s.repo.GetByID(ctx, api.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return errors.New("接口不存在")
	}

	// 如果修改了接口代码，检查新代码是否已存在
	if api.Code != existing.Code {
		codeExists, err := s.repo.GetByCode(ctx, api.Code)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if codeExists != nil {
			return errors.New("接口代码已存在")
		}
	}

	return s.repo.Update(ctx, api)
}

func (s *platformAPIService) DeleteAPI(ctx context.Context, id int64) error {
	// 检查接口是否存在
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return errors.New("接口不存在")
	}

	return s.repo.Delete(ctx, id)
}

func (s *platformAPIService) GetAPI(ctx context.Context, id int64) (*model.PlatformAPI, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *platformAPIService) GetAPIByCode(ctx context.Context, code string) (*model.PlatformAPI, error) {
	return s.repo.GetByCode(ctx, code)
}

func (s *platformAPIService) ListAPIs(ctx context.Context, page, pageSize int) ([]*model.PlatformAPI, int64, error) {
	// 参数验证
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	return s.repo.List(ctx, page, pageSize)
}
