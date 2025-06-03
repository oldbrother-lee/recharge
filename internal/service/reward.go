package service

import (
	"context"
	"recharge-go/internal/model"
	"recharge-go/internal/repository"
)

type RewardService interface {
	CreateReward(ctx context.Context, reward *model.Reward) error
	GetRewardByID(ctx context.Context, id int64) (*model.Reward, error)
	GetRewardsByUserID(ctx context.Context, userID int64, page, pageSize int) ([]*model.Reward, int64, error)
	UpdateRewardStatus(ctx context.Context, id int64, status int) error
}

type rewardService struct {
	rewardRepo repository.RewardRepository
}

func NewRewardService(rewardRepo repository.RewardRepository) RewardService {
	return &rewardService{
		rewardRepo: rewardRepo,
	}
}

func (s *rewardService) CreateReward(ctx context.Context, reward *model.Reward) error {
	return s.rewardRepo.Create(ctx, reward)
}

func (s *rewardService) GetRewardByID(ctx context.Context, id int64) (*model.Reward, error) {
	return s.rewardRepo.GetByID(ctx, id)
}

func (s *rewardService) GetRewardsByUserID(ctx context.Context, userID int64, page, pageSize int) ([]*model.Reward, int64, error) {
	return s.rewardRepo.GetByUserID(ctx, userID, page, pageSize)
}

func (s *rewardService) UpdateRewardStatus(ctx context.Context, id int64, status int) error {
	return s.rewardRepo.UpdateStatus(ctx, id, status)
}
