package service

import (
	"context"
	"recharge-go/internal/model"
	"recharge-go/internal/repository"
)

type RebateService interface {
	CreateRebate(ctx context.Context, rebate *model.Rebate) error
	GetRebateByID(ctx context.Context, id int64) (*model.Rebate, error)
	GetRebatesByUserID(ctx context.Context, userID int64, page, pageSize int) ([]*model.Rebate, int64, error)
	UpdateRebateStatus(ctx context.Context, id int64, status int) error
}

type rebateService struct {
	rebateRepo repository.RebateRepository
}

func NewRebateService(rebateRepo repository.RebateRepository) RebateService {
	return &rebateService{
		rebateRepo: rebateRepo,
	}
}

func (s *rebateService) CreateRebate(ctx context.Context, rebate *model.Rebate) error {
	return s.rebateRepo.Create(ctx, rebate)
}

func (s *rebateService) GetRebateByID(ctx context.Context, id int64) (*model.Rebate, error) {
	return s.rebateRepo.GetByID(ctx, id)
}

func (s *rebateService) GetRebatesByUserID(ctx context.Context, userID int64, page, pageSize int) ([]*model.Rebate, int64, error) {
	return s.rebateRepo.GetByUserID(ctx, userID, page, pageSize)
}

func (s *rebateService) UpdateRebateStatus(ctx context.Context, id int64, status int) error {
	return s.rebateRepo.UpdateStatus(ctx, id, status)
}
