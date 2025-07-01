package service

import (
	"context"
	"errors"
	"fmt"
	"recharge-go/internal/model"
	"recharge-go/internal/repository"
	"recharge-go/pkg/logger"
	"time"

	"gorm.io/gorm"
)

// PlatformAccountBalanceService 平台账号余额服务
type PlatformAccountBalanceService struct {
	db                  *gorm.DB
	platformAccountRepo *repository.PlatformAccountRepository
	userRepo            *repository.UserRepository
	balanceLogRepo      *repository.BalanceLogRepository
}

// NewPlatformAccountBalanceService 创建平台账号余额服务实例
func NewPlatformAccountBalanceService(
	db *gorm.DB,
	platformAccountRepo *repository.PlatformAccountRepository,
	userRepo *repository.UserRepository,
	balanceLogRepo *repository.BalanceLogRepository,
) *PlatformAccountBalanceService {
	return &PlatformAccountBalanceService{
		db:                  db,
		platformAccountRepo: platformAccountRepo,
		userRepo:            userRepo,
		balanceLogRepo:      balanceLogRepo,
	}
}

// DeductBalance 扣除余额，支持授信额度
func (s *PlatformAccountBalanceService) DeductBalance(ctx context.Context, accountID int64, amount float64, orderID int64, remark string) error {
	logger.Info("开始扣除本地账号余额",
		"platform_account_id", accountID,
		"amount", amount,
		"order_id", orderID,
		"remark", remark)

	// 开启事务确保操作原子性
	tx := s.db.Begin()
	if tx.Error != nil {
		logger.Error("开启事务失败", "error", tx.Error, "order_id", orderID)
		return tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			logger.Error("panic, 事务回滚", "panic", r)
		}
	}()

	// 1. 在事务中进行幂等性校验：同一用户、同一订单、同一扣款操作只允许扣款一次
	var existCount int64
	err := tx.Model(&model.BalanceLog{}).
		Where("order_id = ? AND platform_account_id = ? AND style = ?", orderID, accountID, model.BalanceStyleOrderDeduct).
		Count(&existCount).Error
	if err != nil {
		tx.Rollback()
		logger.Error("幂等性校验失败", "error", err, "order_id", orderID)
		return err
	}
	if existCount > 0 {
		tx.Rollback()
		logger.Info("已存在扣款日志，跳过重复扣款", "order_id", orderID, "account_id", accountID)
		return nil
	}

	// 2. 获取平台账号信息 - 在事务中直接查询，避免连接不一致问题
	var account model.PlatformAccount
	err = tx.Preload("Platform").Where("id = ?", accountID).First(&account).Error
	if err != nil {
		tx.Rollback()
		logger.Error("获取平台账号信息失败", "error", err, "account_id", accountID)
		return err
	}

	// 3. 获取本地用户账号（通过 bind_user_id 字段）
	if account.BindUserID == nil {
		tx.Rollback()
		logger.Error("平台账号未绑定本地用户", "account_id", accountID)
		return errors.New("平台账号未绑定本地用户")
	}
	userID := *account.BindUserID

	// 4. 使用行锁获取用户信息，防止并发修改
	var user model.User
	if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("id = ?", userID).First(&user).Error; err != nil {
		tx.Rollback()
		logger.Error("获取本地用户账号失败", "error", err, "user_id", userID)
		return err
	}

	// 5. 检查余额+授信额度
	available := user.Balance + user.Credit
	if available < amount {
		tx.Rollback()
		logger.Error("余额和授信额度均不足", "user_id", userID, "current_balance", user.Balance, "credit", user.Credit, "required_amount", amount)
		return errors.New("余额和授信额度均不足")
	}

	// 6. 扣减余额（余额优先，余额不足自动用授信额度）
	before := user.Balance
	user.Balance -= amount
	if err := tx.Model(&model.User{}).Where("id = ?", userID).Update("balance", user.Balance).Error; err != nil {
		tx.Rollback()
		logger.Error("更新本地用户余额失败", "error", err, "user_id", userID)
		return err
	}

	// 7. 计算本次用掉的授信额度
	creditUsed := 0.0
	if user.Balance < 0 {
		creditUsed = -user.Balance
	}

	// 8. 写用户余额变动日志
	userLog := &model.BalanceLog{
		UserID:            userID,
		OrderID:           orderID,
		PlatformAccountID: accountID,
		PlatformID:        account.PlatformID,
		PlatformCode:      account.Platform.Code,
		PlatformName:      account.Platform.Name,
		Amount:            -amount,
		Type:              model.BalanceTypeExpense,
		Style:             model.BalanceStyleOrderDeduct,
		Balance:           user.Balance,
		BalanceBefore:     before,
		Remark:            fmt.Sprintf("%s（本次用掉授信额度：%.2f）", remark, creditUsed),
		Operator:          "system",
		CreatedAt:         time.Now(),
	}
	if err := tx.Create(userLog).Error; err != nil {
		tx.Rollback()
		logger.Error("创建用户余额变动日志失败", "error", err, "user_id", userID)
		return err
	}

	// 9. 提交事务
	if err := tx.Commit().Error; err != nil {
		logger.Error("提交事务失败", "error", err, "order_id", orderID)
		return err
	}

	logger.Info("扣除本地账号余额成功", "user_id", userID, "amount", amount, "balance_before", before, "balance_after", user.Balance, "credit_used", creditUsed)
	return nil
}

// RefundBalance 退款到用户余额（使用原子性更新避免竞态条件）
func (s *PlatformAccountBalanceService) RefundBalance(ctx context.Context, userID int64, amount float64, orderID int64, remark string) error {
	if amount <= 0 {
		return errors.New("退款金额必须大于0")
	}
	
	// 基于用户ID和订单ID的幂等性校验，防止重复退款
	var existCount int64
	if err := s.db.Model(&model.BalanceLog{}).Where("user_id = ? AND order_id = ? AND style = ?", userID, orderID, 2).Count(&existCount).Error; err != nil {
		return err
	}
	if existCount > 0 {
		// 已存在退款记录，跳过重复退款
		return nil
	}
	
	return s.db.Transaction(func(tx *gorm.DB) error {
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
		
		// 记录用户余额变动日志
		log := &model.BalanceLog{
			UserID:        userID,
			Amount:        amount,
			Type:          1, // 收入
			Style:         2, // 退款
			Balance:       afterBalance,
			BalanceBefore: beforeBalance,
			Remark:        remark,
			Operator:      "system",
			OrderID:       orderID,
			CreatedAt:     time.Now(),
		}
		return tx.Create(log).Error
	})
}

// GetBalanceLogs 获取余额变动记录
func (s *PlatformAccountBalanceService) GetBalanceLogs(ctx context.Context, accountID int64, offset, limit int) ([]*model.BalanceLog, int64, error) {
	// 1. 获取平台账号信息
	account, err := s.platformAccountRepo.GetByID(accountID)
	if err != nil {
		return nil, 0, err
	}

	// 2. 获取本地用户账号（通过 bind_user_id 字段）
	if account.BindUserID == nil {
		return nil, 0, errors.New("平台账号未绑定本地用户")
	}
	userID := *account.BindUserID

	// 3. 查询用户余额变动记录
	var logs []*model.BalanceLog
	var total int64

	if err := s.db.Model(&model.BalanceLog{}).
		Where("user_id = ?", userID).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := s.db.Where("user_id = ?", userID).
		Offset(offset).Limit(limit).
		Order("create_time DESC").
		Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// AdjustBalance 手动调整余额
func (s *PlatformAccountBalanceService) AdjustBalance(ctx context.Context, accountID int64, amount float64, style int, remark string, operator string) error {
	logger.Info("开始手动调整余额",
		"account_id", accountID,
		"amount", amount,
		"style", style,
		"remark", remark,
		"operator", operator)

	// 开启事务
	tx := s.db.Begin()
	if tx.Error != nil {
		logger.Error("开启事务失败",
			"error", tx.Error,
			"account_id", accountID)
		return tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 1. 获取平台账号信息
	account, err := s.platformAccountRepo.GetByID(accountID)
	if err != nil {
		tx.Rollback()
		logger.Error("获取平台账号信息失败",
			"error", err,
			"account_id", accountID)
		return err
	}

	// 2. 获取本地用户账号（通过 bind_user_id 字段）
	if account.BindUserID == nil {
		tx.Rollback()
		logger.Error("平台账号未绑定本地用户", "account_id", accountID)
		return errors.New("平台账号未绑定本地用户")
	}
	userID := *account.BindUserID

	// 3. 调整余额（使用原子性更新避免竞态条件）
	result := tx.Model(&model.User{}).
		Where("id = ?", userID).
		Update("balance", gorm.Expr("balance + ?", amount))
	
	if result.Error != nil {
		tx.Rollback()
		logger.Error("更新本地用户余额失败",
			"error", result.Error,
			"user_id", userID)
		return result.Error
	}
	
	if result.RowsAffected == 0 {
		tx.Rollback()
		logger.Error("用户不存在", "user_id", userID)
		return errors.New("用户不存在")
	}
	
	// 获取更新后的余额（在同一事务中确保数据一致性）
	var updatedUser model.User
	if err := tx.Where("id = ?", userID).First(&updatedUser).Error; err != nil {
		tx.Rollback()
		logger.Error("获取更新后用户信息失败", "error", err, "user_id", userID)
		return err
	}
	
	afterBalance := updatedUser.Balance
	beforeBalance := afterBalance - amount

	// 4. 写用户余额变动日志
	userLog := &model.BalanceLog{
		UserID:            userID,
		PlatformAccountID: accountID,
		PlatformID:        account.PlatformID,
		PlatformCode:      account.Platform.Code,
		PlatformName:      account.Platform.Name,
		Amount:            amount,
		Type:              model.BalanceTypeIncome,
		Style:             style,
		Balance:           afterBalance,
		BalanceBefore:     beforeBalance,
		Remark:            remark,
		Operator:          operator,
		CreatedAt:         time.Now(),
	}
	if err := tx.Create(userLog).Error; err != nil {
		tx.Rollback()
		logger.Error("创建用户余额变动日志失败",
			"error", err,
			"user_id", userID)
		return err
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		logger.Error("提交事务失败",
			"error", err,
			"account_id", accountID)
		return err
	}

	logger.Info("手动调整余额成功",
		"user_id", userID,
		"amount", amount,
		"balance_before", beforeBalance,
		"balance_after", afterBalance)
	return nil
}

// DeleteByOrderIDs 批量删除指定订单ID的余额日志
func (s *PlatformAccountBalanceService) DeleteByOrderIDs(ctx context.Context, orderIDs []int64) error {
	return s.balanceLogRepo.DeleteByOrderIDs(ctx, orderIDs)
}
