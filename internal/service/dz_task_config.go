package service

import (
	"context"
	"fmt"
	"recharge-go/internal/model"
	"recharge-go/internal/repository"
)

type DzTaskConfigService struct {
	dzTaskConfigRepo *repository.DzTaskConfigRepository
}

func NewDzTaskConfigService(dzTaskConfigRepo *repository.DzTaskConfigRepository) *DzTaskConfigService {
	return &DzTaskConfigService{
		dzTaskConfigRepo: dzTaskConfigRepo,
	}
}

// Create 创建得众任务配置
func (s *DzTaskConfigService) Create(ctx context.Context, config *model.DzTaskConfig) error {
	return s.dzTaskConfigRepo.Create(config)
}

// Update 更新得众任务配置
func (s *DzTaskConfigService) Update(ctx context.Context, config *model.DzTaskConfig) error {
	return s.dzTaskConfigRepo.Update(config)
}

// UpdatePartial 部分更新得众任务配置
func (s *DzTaskConfigService) UpdatePartial(ctx context.Context, req *model.UpdateDzTaskConfigRequest) error {
	if req.ID == nil {
		return fmt.Errorf("id is required")
	}

	// 先获取现有配置
	config, err := s.dzTaskConfigRepo.GetByID(*req.ID)
	if err != nil {
		return err
	}

	// 只更新非 nil 的字段
	if req.ProductID != nil {
		config.ProductID = *req.ProductID
	}
	if req.PlatformID != nil {
		config.PlatformID = *req.PlatformID
	}
	if req.PlatformAccountID != nil {
		config.PlatformAccountID = *req.PlatformAccountID
	}
	if req.ISP != nil {
		config.ISP = *req.ISP
	}
	if req.FaceValue != nil {
		config.FaceValue = *req.FaceValue
	}
	if req.PollIntervalSec != nil {
		config.PollIntervalSec = *req.PollIntervalSec
	}
	if req.Concurrency != nil {
		config.Concurrency = *req.Concurrency
	}
	if req.Enabled != nil {
		config.Enabled = *req.Enabled
	}

	return s.dzTaskConfigRepo.Update(config)
}

// Delete 删除得众任务配置
func (s *DzTaskConfigService) Delete(ctx context.Context, id int64) error {
	return s.dzTaskConfigRepo.Delete(id)
}

// GetByID 根据ID获取得众任务配置
func (s *DzTaskConfigService) GetByID(ctx context.Context, id int64) (*model.DzTaskConfig, error) {
	return s.dzTaskConfigRepo.GetByID(id)
}

// List 获取得众任务配置列表
func (s *DzTaskConfigService) List(ctx context.Context, page, pageSize int, platformAccountID *int64) ([]*model.DzTaskConfig, int64, error) {
	return s.dzTaskConfigRepo.List(page, pageSize, platformAccountID)
}

// BatchCreate 批量创建得众任务配置
func (s *DzTaskConfigService) BatchCreate(ctx context.Context, configs []*model.DzTaskConfig) error {
	return s.dzTaskConfigRepo.BatchCreate(configs)
}

// GetEnabledConfigs 获取所有启用的得众任务配置
func (s *DzTaskConfigService) GetEnabledConfigs(ctx context.Context) ([]model.DzTaskConfig, error) {
	return s.dzTaskConfigRepo.GetEnabledConfigs()
}