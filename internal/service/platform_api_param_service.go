package service

import (
	"context"
	"errors"
	"recharge-go/internal/model"
	"recharge-go/internal/repository"
)

// PlatformAPIParamService 平台接口参数服务接口
type PlatformAPIParamService interface {
	// CreateParam 创建平台接口参数
	CreateParam(ctx context.Context, param *model.PlatformAPIParam) error
	// UpdateParam 更新平台接口参数
	UpdateParam(ctx context.Context, param *model.PlatformAPIParam) error
	// DeleteParam 删除平台接口参数
	DeleteParam(ctx context.Context, id int64) error
	// GetParam 获取平台接口参数详情
	GetParam(ctx context.Context, id int64) (*model.PlatformAPIParam, error)
	// ListParams 获取平台接口参数列表
	ListParams(ctx context.Context, apiID int64, page, pageSize int) ([]*model.PlatformAPIParam, int64, error)
}

// platformAPIParamService 平台接口参数服务实现
type platformAPIParamService struct {
	repo repository.PlatformAPIParamRepository
}

// NewPlatformAPIParamService 创建平台接口参数服务实例
func NewPlatformAPIParamService(repo repository.PlatformAPIParamRepository) PlatformAPIParamService {
	return &platformAPIParamService{repo: repo}
}

func (s *platformAPIParamService) CreateParam(ctx context.Context, param *model.PlatformAPIParam) error {
	return s.repo.Create(ctx, param)
}

func (s *platformAPIParamService) UpdateParam(ctx context.Context, param *model.PlatformAPIParam) error {
	// 检查参数是否存在
	existing, err := s.repo.GetByID(ctx, param.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return errors.New("参数不存在")
	}
	return s.repo.Update(ctx, param)
}

func (s *platformAPIParamService) DeleteParam(ctx context.Context, id int64) error {
	// 检查参数是否存在
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return errors.New("参数不存在")
	}

	return s.repo.Delete(ctx, id)
}

func (s *platformAPIParamService) GetParam(ctx context.Context, id int64) (*model.PlatformAPIParam, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *platformAPIParamService) ListParams(ctx context.Context, apiID int64, page, pageSize int) ([]*model.PlatformAPIParam, int64, error) {
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

	return s.repo.List(ctx, apiID, page, pageSize)
}
