package service

import (
	"context"
	"errors"
	"recharge-go/internal/model"
	"recharge-go/internal/repository"
	"time"
	
	"gorm.io/gorm"
)

// BalanceService 余额相关业务逻辑

type BalanceService struct {
	repo     *repository.BalanceLogRepository
	userRepo *repository.UserRepository
	db       *gorm.DB
}

func NewBalanceService(repo *repository.BalanceLogRepository, userRepo *repository.UserRepository) *BalanceService {
	return &BalanceService{
		repo:     repo,
		userRepo: userRepo,
		db:       repo.GetDB(), // 需要添加GetDB方法
	}
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

// Refund 余额退款
func (s *BalanceService) Refund(ctx context.Context, userID int64, amount float64, orderID int64, remark, operator string) error {
	if amount <= 0 {
		return errors.New("退款金额必须大于0")
	}
	
	// 使用事务确保余额更新和日志记录的原子性
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return s.RefundWithTx(ctx, tx, userID, amount, orderID, remark, operator)
	})
}

// RefundWithTx 在指定事务中进行余额退款（使用原子性更新避免竞态条件）
func (s *BalanceService) RefundWithTx(ctx context.Context, tx *gorm.DB, userID int64, amount float64, orderID int64, remark, operator string) error {
	if amount <= 0 {
		return errors.New("退款金额必须大于0")
	}
	
	// 幂等性校验：检查是否已存在该订单的退款记录
	var existCount int64
	if err := tx.Model(&model.BalanceLog{}).Where("order_id = ? AND user_id = ? AND style = ?", orderID, userID, 2).Count(&existCount).Error; err != nil {
		return err
	}
	if existCount > 0 {
		// 已存在退款记录，跳过重复退款
		return nil
	}
	
	// 关键改进：使用原子性更新避免读取-计算-写入的竞态条件
	// 先原子性更新余额
	result := tx.Model(&model.User{}).
		Where("id = ?", userID).
		Update("balance", gorm.Expr("balance + ?", amount))
	
	if result.Error != nil {
		return result.Error
	}
	
	if result.RowsAffected == 0 {
		return errors.New("用户不存在")
	}
	
	// 获取更新后的余额（在同一事务中确保数据一致性）
	var user model.User
	if err := tx.Where("id = ?", userID).First(&user).Error; err != nil {
		return err
	}
	
	afterBalance := user.Balance
	beforeBalance := afterBalance - amount
	
	// 写入流水
	log := &model.BalanceLog{
		UserID:        userID,
		Amount:        amount,
		Type:          1, // 收入
		Style:         2, // 退款
		Balance:       afterBalance,
		BalanceBefore: beforeBalance,
		Remark:        remark,
		Operator:      operator,
		OrderID:       orderID,
		CreatedAt:     time.Now(),
	}
	return tx.Create(log).Error
}

// ListLogs 查询余额流水
func (s *BalanceService) ListLogs(ctx context.Context, userID int64, offset, limit int) ([]model.BalanceLog, int64, error) {
	return s.repo.ListLogs(ctx, userID, offset, limit)
}
