package repository

import (
	"context"
	"recharge-go/internal/model"

	"gorm.io/gorm"
)

// CreditLogRepository 授信日志仓储
type CreditLogRepository struct {
	db *gorm.DB
}

// NewCreditLogRepository 创建授信日志仓储
func NewCreditLogRepository(db *gorm.DB) *CreditLogRepository {
	return &CreditLogRepository{
		db: db,
	}
}

// Create 创建授信日志
func (r *CreditLogRepository) Create(ctx context.Context, log *model.CreditLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

// GetByID 根据ID获取授信日志
func (r *CreditLogRepository) GetByID(ctx context.Context, id int64) (*model.CreditLog, error) {
	var log model.CreditLog
	err := r.db.WithContext(ctx).First(&log, id).Error
	if err != nil {
		return nil, err
	}
	return &log, nil
}

// List 获取授信日志列表
func (r *CreditLogRepository) List(ctx context.Context, req *model.CreditLogListRequest) ([]model.CreditLog, int64, error) {
	var logs []model.CreditLog
	var total int64

	query := r.db.WithContext(ctx).Model(&model.CreditLog{})

	if req.UserID > 0 {
		query = query.Where("user_id = ?", req.UserID)
	}
	if req.Type > 0 {
		query = query.Where("type = ?", req.Type)
	}
	if req.OrderID > 0 {
		query = query.Where("order_id = ?", req.OrderID)
	}

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Offset((req.Current - 1) * req.Size).Limit(req.Size).Order("created_at DESC").Find(&logs).Error
	if err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// GetUserCreditStats 获取用户授信统计
func (r *CreditLogRepository) GetUserCreditStats(ctx context.Context, userID int64) (float64, float64, error) {
	var totalUsed float64
	var totalRestored float64

	// 统计已使用额度
	err := r.db.WithContext(ctx).Model(&model.CreditLog{}).
		Where("user_id = ? AND type = ?", userID, model.CreditTypeUse).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&totalUsed).Error
	if err != nil {
		return 0, 0, err
	}

	// 统计已恢复额度
	err = r.db.WithContext(ctx).Model(&model.CreditLog{}).
		Where("user_id = ? AND type = ?", userID, model.CreditTypeRestore).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&totalRestored).Error
	if err != nil {
		return 0, 0, err
	}

	return totalUsed, totalRestored, nil
}
