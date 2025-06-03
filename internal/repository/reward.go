package repository

import (
	"context"
	"recharge-go/internal/model"

	"gorm.io/gorm"
)

type RewardRepository interface {
	Create(ctx context.Context, reward *model.Reward) error
	GetByID(ctx context.Context, id int64) (*model.Reward, error)
	GetByUserID(ctx context.Context, userID int64, page, pageSize int) ([]*model.Reward, int64, error)
	UpdateStatus(ctx context.Context, id int64, status int) error
}

type rewardRepository struct {
	db *gorm.DB
}

func NewRewardRepository(db *gorm.DB) RewardRepository {
	return &rewardRepository{
		db: db,
	}
}

func (r *rewardRepository) Create(ctx context.Context, reward *model.Reward) error {
	return r.db.WithContext(ctx).Create(reward).Error
}

func (r *rewardRepository) GetByID(ctx context.Context, id int64) (*model.Reward, error) {
	var reward model.Reward
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&reward).Error
	if err != nil {
		return nil, err
	}
	return &reward, nil
}

func (r *rewardRepository) GetByUserID(ctx context.Context, userID int64, page, pageSize int) ([]*model.Reward, int64, error) {
	var rewards []*model.Reward
	var total int64

	offset := (page - 1) * pageSize
	err := r.db.WithContext(ctx).Model(&model.Reward{}).
		Where("user_id = ?", userID).
		Count(&total).
		Offset(offset).
		Limit(pageSize).
		Order("created_at DESC").
		Find(&rewards).Error

	if err != nil {
		return nil, 0, err
	}
	return rewards, total, nil
}

func (r *rewardRepository) UpdateStatus(ctx context.Context, id int64, status int) error {
	return r.db.WithContext(ctx).Model(&model.Reward{}).
		Where("id = ?", id).
		Update("status", status).Error
}
