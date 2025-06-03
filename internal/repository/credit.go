package repository

import (
	"recharge-go/internal/model"

	"gorm.io/gorm"
)

// CreditRepository 授信仓库
type CreditRepository struct {
	db *gorm.DB
}

// NewCreditRepository 创建授信仓库
func NewCreditRepository(db *gorm.DB) *CreditRepository {
	return &CreditRepository{db: db}
}

// CreateCreditLog 创建授信日志
func (r *CreditRepository) CreateCreditLog(log *model.CreditLog) error {
	return r.db.Create(log).Error
}

// GetCreditLogs 获取授信日志列表
func (r *CreditRepository) GetCreditLogs(userID int64, changeType string, orderID string, page, size int) ([]*model.CreditLog, int64, error) {
	var logs []*model.CreditLog
	var total int64

	query := r.db.Model(&model.CreditLog{})

	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}
	if changeType != "" {
		query = query.Where("change_type = ?", changeType)
	}
	if orderID != "" {
		query = query.Where("order_id = ?", orderID)
	}

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Offset((page - 1) * size).Limit(size).Order("created_at DESC").Find(&logs).Error
	if err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// GetUserCreditStats 获取用户授信统计
func (r *CreditRepository) GetUserCreditStats(userID int64) (float64, float64, error) {
	var totalUsed, totalRestored float64

	// 统计已使用额度
	err := r.db.Model(&model.CreditLog{}).
		Where("user_id = ? AND type = ?", userID, model.CreditTypeUse).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&totalUsed).Error
	if err != nil {
		return 0, 0, err
	}

	// 统计已恢复额度
	err = r.db.Model(&model.CreditLog{}).
		Where("user_id = ? AND type = ?", userID, model.CreditTypeRestore).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&totalRestored).Error
	if err != nil {
		return 0, 0, err
	}

	return totalUsed, totalRestored, nil
}
