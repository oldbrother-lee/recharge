package service

import (
	"context"
	"errors"
	"recharge-go/internal/model"
	"recharge-go/internal/repository"
	"time"
)

// BalanceService 余额相关业务逻辑

type BalanceService struct {
	repo     *repository.BalanceLogRepository
	userRepo *repository.UserRepository
}

func NewBalanceService(repo *repository.BalanceLogRepository, userRepo *repository.UserRepository) *BalanceService {
	return &BalanceService{repo: repo, userRepo: userRepo}
}

// Recharge 余额充值
func (s *BalanceService) Recharge(ctx context.Context, userID int64, amount float64, remark, operator string) error {
	if amount <= 0 {
		return errors.New("充值金额必须大于0")
	}
	// 获取充值前余额
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	before := user.Balance
	// 增加余额
	if err := s.repo.AddBalance(ctx, userID, amount); err != nil {
		return err
	}
	// 写入流水
	log := &model.BalanceLog{
		UserID:        userID,
		Amount:        amount,
		Type:          1, // 收入
		Style:         4, // 充值
		Balance:       before + amount,
		BalanceBefore: before,
		Remark:        remark,
		Operator:      operator,
		CreatedAt:     time.Now(),
	}
	return s.repo.CreateLog(ctx, log)
}

// Deduct 余额扣款
func (s *BalanceService) Deduct(ctx context.Context, userID int64, amount float64, style int, remark, operator string) error {
	if amount <= 0 {
		return errors.New("扣款金额必须大于0")
	}
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	before := user.Balance
	if before < amount {
		return errors.New("余额不足")
	}
	if err := s.repo.SubBalance(ctx, userID, amount); err != nil {
		return err
	}
	log := &model.BalanceLog{
		UserID:        userID,
		Amount:        -amount,
		Type:          2,     // 支出
		Style:         style, // 业务类型
		Balance:       before - amount,
		BalanceBefore: before,
		Remark:        remark,
		Operator:      operator,
		CreatedAt:     time.Now(),
	}
	return s.repo.CreateLog(ctx, log)
}

// ListLogs 查询余额流水
func (s *BalanceService) ListLogs(ctx context.Context, userID int64, offset, limit int) ([]model.BalanceLog, int64, error) {
	return s.repo.ListLogs(ctx, userID, offset, limit)
}
