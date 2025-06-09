package service

import (
	"context"
	"fmt"
	"recharge-go/internal/model"
	"recharge-go/internal/repository"
)

type TaskConfigService struct {
	taskConfigRepo *repository.TaskConfigRepository
}

func NewTaskConfigService(taskConfigRepo *repository.TaskConfigRepository) *TaskConfigService {
	return &TaskConfigService{
		taskConfigRepo: taskConfigRepo,
	}
}

type Product struct {
	ProductID string `json:"productId"`
}

type TaskConfigPayload struct {
	ID               int64  `json:"id"`
	ChannelID        int    `json:"channelId"`
	ChannelName      string `json:"channelName"`
	ProductID        string `json:"ProductID"`
	ProductName      string `json:"ProductName"`
	FaceValues       string `json:"FaceValues"`
	MinSettleAmounts string `json:"MinSettleAmounts"`
	Status           int    `json:"status"`
}

// Create 创建任务配置
func (s *TaskConfigService) Create(ctx context.Context, config *model.TaskConfig) error {
	return s.taskConfigRepo.Create(config)
}

// Update 更新任务配置
func (s *TaskConfigService) Update(ctx context.Context, config *model.TaskConfig) error {
	return s.taskConfigRepo.Update(config)
}

// UpdatePartial 部分更新任务配置
func (s *TaskConfigService) UpdatePartial(ctx context.Context, req *model.UpdateTaskConfigRequest) error {
	if req.ID == nil {
		return fmt.Errorf("id is required")
	}

	// 先获取现有配置
	config, err := s.taskConfigRepo.GetByID(*req.ID)
	if err != nil {
		return err
	}

	// 只更新非 nil 的字段
	if req.FaceValues != nil {
		config.FaceValues = *req.FaceValues
	}
	if req.MinSettleAmounts != nil {
		config.MinSettleAmounts = *req.MinSettleAmounts
	}
	if req.Status != nil {
		config.Status = *req.Status
	}
	if req.ChannelID != nil {
		config.ChannelID = *req.ChannelID
	}
	if req.ProductID != nil {
		config.ProductID = *req.ProductID
	}

	return s.taskConfigRepo.Update(config)
}

// Delete 删除任务配置
func (s *TaskConfigService) Delete(ctx context.Context, id int64) error {
	return s.taskConfigRepo.Delete(id)
}

// GetByID 根据ID获取任务配置
func (s *TaskConfigService) GetByID(ctx context.Context, id int64) (*model.TaskConfig, error) {
	return s.taskConfigRepo.GetByID(id)
}

// List 获取任务配置列表
func (s *TaskConfigService) List(ctx context.Context, page, pageSize int, platformAccountID *int64) ([]*model.TaskConfig, int64, error) {
	return s.taskConfigRepo.List(page, pageSize, platformAccountID)
}

// BatchCreate 批量创建任务配置
func (s *TaskConfigService) BatchCreate(ctx context.Context, configs []*model.TaskConfig) error {
	return s.taskConfigRepo.BatchCreate(configs)
}
