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
	repo         *repository.BalanceLogRepository
	userRepo     *repository.UserRepository
	db           *gorm.DB
	creditService *CreditService
}

func NewBalanceService(repo *repository.BalanceLogRepository, userRepo *repository.UserRepository) *BalanceService {
	return &BalanceService{
		repo:     repo,
		userRepo: userRepo,
		db:       repo.GetDB(), // 需要添加GetDB方法
	}
}

// NewBalanceServiceWithCredit 创建带授信功能的余额服务
func NewBalanceServiceWithCredit(repo *repository.BalanceLogRepository, userRepo *repository.UserRepository, creditService *CreditService) *BalanceService {
	return &BalanceService{
		repo:         repo,
		userRepo:     userRepo,
		db:           repo.GetDB(),
		creditService: creditService,
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

// SmartDeduct 智能扣款（优先使用余额，不足时使用授信额度）
func (s *BalanceService) SmartDeduct(ctx context.Context, userID int64, amount float64, style int, remark, operator string) error {
	if amount <= 0 {
		return errors.New("扣款金额必须大于0")
	}

	// 如果没有授信服务，直接使用普通扣款
	if s.creditService == nil {
		return s.Deduct(ctx, userID, amount, style, remark, operator)
	}

	// 获取用户信息
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	// 计算总可用金额（余额 + 授信额度）
	totalAvailable := user.Balance + user.Credit
	if totalAvailable < amount {
		return errors.New("余额和授信额度总和不足")
	}

	// 使用事务确保原子性
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 计算扣款策略
		balanceDeduct := amount
		creditDeduct := 0.0

		if user.Balance < amount {
			// 余额不足，需要使用授信
			balanceDeduct = user.Balance
			creditDeduct = amount - user.Balance
		}

		// 1. 扣除余额（如果有需要扣除的余额）
		if balanceDeduct > 0 {
			// 使用原子性更新扣除余额
			result := tx.Model(&model.User{}).
				Where("id = ? AND balance >= ?", userID, balanceDeduct).
				Update("balance", gorm.Expr("balance - ?", balanceDeduct))

			if result.Error != nil {
				return result.Error
			}

			if result.RowsAffected == 0 {
				return errors.New("余额不足")
			}

			// 创建余额扣款日志
			balanceLog := &model.BalanceLog{
				UserID:        userID,
				Amount:        -balanceDeduct,
				Type:          2,     // 支出
				Style:         style, // 业务类型
				Balance:       user.Balance - balanceDeduct,
				BalanceBefore: user.Balance,
				Remark:        remark + "(余额部分)",
				Operator:      operator,
				CreatedAt:     time.Now(),
			}
			if err := tx.Create(balanceLog).Error; err != nil {
				return err
			}
		}

		// 2. 扣除授信额度（如果需要）
		if creditDeduct > 0 {
			// 使用原子性更新扣除授信额度
			result := tx.Model(&model.User{}).
				Where("id = ? AND credit >= ?", userID, creditDeduct).
				Update("credit", gorm.Expr("credit - ?", creditDeduct))

			if result.Error != nil {
				return result.Error
			}

			if result.RowsAffected == 0 {
				return errors.New("授信额度不足")
			}

			// 创建授信使用日志
			creditLog := &model.CreditLog{
				UserID:       userID,
				Amount:       creditDeduct,
				Type:         model.CreditTypeUse,
				CreditBefore: user.Credit,
				CreditAfter:  user.Credit - creditDeduct,
				Remark:       remark + "(授信部分)",
				Operator:     operator,
				CreatedAt:    time.Now(),
			}
			if err := tx.Create(creditLog).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// ListLogs 查询余额流水
func (s *BalanceService) ListLogs(ctx context.Context, userID int64, offset, limit int) ([]model.BalanceLog, int64, error) {
	return s.repo.ListLogs(ctx, userID, offset, limit)
}
