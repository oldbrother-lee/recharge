package repository

import (
	"context"
	"errors"
	"recharge-go/internal/model"

	"gorm.io/gorm"
)

// BalanceLogRepository 余额流水仓储
// 负责 balance_logs 表的增查和用户余额原子操作

type BalanceLogRepository struct {
	db *gorm.DB
}

func NewBalanceLogRepository(db *gorm.DB) *BalanceLogRepository {
	return &BalanceLogRepository{db: db}
}

// GetDB 获取数据库连接
func (r *BalanceLogRepository) GetDB() *gorm.DB {
	return r.db
}

// CreateLog 新增一条余额流水
func (r *BalanceLogRepository) CreateLog(ctx context.Context, log *model.BalanceLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

// ListLogs 查询用户余额流水（分页）
func (r *BalanceLogRepository) ListLogs(ctx context.Context, userID int64, offset, limit int) ([]model.BalanceLog, int64, error) {
	var logs []model.BalanceLog
	var total int64
	db := r.db.WithContext(ctx).Model(&model.BalanceLog{}).Where("user_id = ?", userID)
	db.Count(&total)
	err := db.Order("id desc").Offset(offset).Limit(limit).Find(&logs).Error
	return logs, total, err
}

// ListLogsWithOrder 查询用户余额流水（分页，关联订单号/外部订单号/手机号）
func (r *BalanceLogRepository) ListLogsWithOrder(ctx context.Context, userID int64, offset, limit int) ([]model.BalanceLogWithOrder, int64, error) {
	var total int64
	db := r.db.WithContext(ctx).Model(&model.BalanceLog{}).Where("balance_logs.user_id = ?", userID)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var logs []model.BalanceLogWithOrder
	err := db.Select("balance_logs.*, o.order_number AS order_number, o.out_trade_num AS out_trade_num, o.mobile AS mobile").
		Joins("LEFT JOIN orders o ON balance_logs.order_id = o.id").
		Order("balance_logs.id DESC").
		Offset(offset).Limit(limit).
		Find(&logs).Error
	return logs, total, err
}

// AddBalance 用户余额增加（使用原子性更新避免竞态条件）
func (r *BalanceLogRepository) AddBalance(ctx context.Context, userID int64, amount float64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 使用原子性更新避免读取-计算-写入的竞态条件
		result := tx.Model(&model.User{}).
			Where("id = ?", userID).
			Update("balance", gorm.Expr("balance + ?", amount))
		
		if result.Error != nil {
			return result.Error
		}
		
		if result.RowsAffected == 0 {
			return errors.New("用户不存在")
		}
		
		return nil
	})
}

// SubBalance 用户余额扣减（使用原子性更新和余额校验避免竞态条件）
func (r *BalanceLogRepository) SubBalance(ctx context.Context, userID int64, amount float64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 使用原子性更新，同时在SQL层面校验余额充足
		// 只有当余额充足时才会更新，避免读取-计算-写入的竞态条件
		result := tx.Model(&model.User{}).
			Where("id = ? AND balance >= ?", userID, amount).
			Update("balance", gorm.Expr("balance - ?", amount))
		
		if result.Error != nil {
			return result.Error
		}
		
		if result.RowsAffected == 0 {
			// 检查是用户不存在还是余额不足
			var user model.User
			if err := tx.Where("id = ?", userID).First(&user).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return errors.New("用户不存在")
				}
				return err
			}
			return errors.New("余额不足")
		}
		
		return nil
	})
}

// DeleteByOrderIDs 批量删除余额日志（分批避免占位符过多导致 1390 错误）
func (r *BalanceLogRepository) DeleteByOrderIDs(ctx context.Context, orderIDs []int64) error {
	if len(orderIDs) == 0 {
		return nil
	}
	const batchSize = 1000
	for start := 0; start < len(orderIDs); start += batchSize {
		end := start + batchSize
		if end > len(orderIDs) {
			end = len(orderIDs)
		}
		ids := orderIDs[start:end]
		if err := r.db.WithContext(ctx).Where("order_id IN ?", ids).Delete(&model.BalanceLog{}).Error; err != nil {
			return err
		}
	}
	return nil
}
