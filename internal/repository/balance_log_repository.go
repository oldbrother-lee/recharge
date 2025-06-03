package repository

import (
	"context"
	"recharge-go/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// BalanceLogRepository 余额流水仓储
// 负责 balance_logs 表的增查和用户余额原子操作

type BalanceLogRepository struct {
	db *gorm.DB
}

func NewBalanceLogRepository(db *gorm.DB) *BalanceLogRepository {
	return &BalanceLogRepository{db: db}
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

// AddBalance 用户余额增加（带事务）
func (r *BalanceLogRepository) AddBalance(ctx context.Context, userID int64, amount float64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return tx.Model(&model.User{}).Where("id = ?", userID).UpdateColumn("balance", gorm.Expr("balance + ?", amount)).Error
	})
}

// SubBalance 用户余额扣减（带事务，校验余额充足）
func (r *BalanceLogRepository) SubBalance(ctx context.Context, userID int64, amount float64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var user model.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", userID).First(&user).Error; err != nil {
			return err
		}
		if user.Balance < amount {
			return gorm.ErrInvalidTransaction // 余额不足
		}
		return tx.Model(&model.User{}).Where("id = ?", userID).UpdateColumn("balance", gorm.Expr("balance - ?", amount)).Error
	})
}

// DeleteByOrderIDs 批量删除余额日志
func (r *BalanceLogRepository) DeleteByOrderIDs(ctx context.Context, orderIDs []int64) error {
	return r.db.Where("order_id IN ?", orderIDs).Delete(&model.BalanceLog{}).Error
}
