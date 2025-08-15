package repository

import (
	"context"

	"gorm.io/gorm"
	"recharge-go/internal/model"
)

// BalanceQueryRecordRepository 余额查询记录仓储接口
type BalanceQueryRecordRepository interface {
	Create(ctx context.Context, record *model.BalanceQueryRecord) error
	GetByOrderID(ctx context.Context, orderID int64) (*model.BalanceQueryRecord, error)
	GetByOrderIDAndType(ctx context.Context, orderID int64, queryType string) (*model.BalanceQueryRecord, error)
	GetByOrderNumber(ctx context.Context, orderNumber string) (*model.BalanceQueryRecord, error)
	GetByOrderNumberAndType(ctx context.Context, orderNumber string, queryType string) (*model.BalanceQueryRecord, error)
	Update(ctx context.Context, record *model.BalanceQueryRecord) error
	ListByMobile(ctx context.Context, mobile string, offset, limit int) ([]model.BalanceQueryRecord, int64, error)
	ListByMobileAndType(ctx context.Context, mobile string, queryType string, offset, limit int) ([]model.BalanceQueryRecord, int64, error)
	DeleteByOrderIDs(ctx context.Context, orderIDs []int64) error
}

// balanceQueryRecordRepository 余额查询记录仓储实现
type balanceQueryRecordRepository struct {
	db *gorm.DB
}

// NewBalanceQueryRecordRepository 创建余额查询记录仓储
func NewBalanceQueryRecordRepository(db *gorm.DB) BalanceQueryRecordRepository {
	return &balanceQueryRecordRepository{
		db: db,
	}
}

// Create 创建余额查询记录
func (r *balanceQueryRecordRepository) Create(ctx context.Context, record *model.BalanceQueryRecord) error {
	return r.db.WithContext(ctx).Create(record).Error
}

// GetByOrderID 根据订单ID获取余额查询记录
func (r *balanceQueryRecordRepository) GetByOrderID(ctx context.Context, orderID int64) (*model.BalanceQueryRecord, error) {
	var record model.BalanceQueryRecord
	err := r.db.WithContext(ctx).Where("order_id = ?", orderID).First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// GetByOrderIDAndType 根据订单ID和查询类型获取余额查询记录
func (r *balanceQueryRecordRepository) GetByOrderIDAndType(ctx context.Context, orderID int64, queryType string) (*model.BalanceQueryRecord, error) {
	var record model.BalanceQueryRecord
	err := r.db.WithContext(ctx).Where("order_id = ? AND query_type = ?", orderID, queryType).First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// GetByOrderNumber 根据订单号获取余额查询记录
func (r *balanceQueryRecordRepository) GetByOrderNumber(ctx context.Context, orderNumber string) (*model.BalanceQueryRecord, error) {
	var record model.BalanceQueryRecord
	err := r.db.WithContext(ctx).Where("order_number = ?", orderNumber).First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// GetByOrderNumberAndType 根据订单号和查询类型获取余额查询记录
func (r *balanceQueryRecordRepository) GetByOrderNumberAndType(ctx context.Context, orderNumber string, queryType string) (*model.BalanceQueryRecord, error) {
	var record model.BalanceQueryRecord
	err := r.db.WithContext(ctx).Where("order_number = ? AND query_type = ?", orderNumber, queryType).First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// Update 更新余额查询记录
func (r *balanceQueryRecordRepository) Update(ctx context.Context, record *model.BalanceQueryRecord) error {
	return r.db.WithContext(ctx).Save(record).Error
}

// ListByMobile 根据手机号分页查询余额查询记录
func (r *balanceQueryRecordRepository) ListByMobile(ctx context.Context, mobile string, offset, limit int) ([]model.BalanceQueryRecord, int64, error) {
	var records []model.BalanceQueryRecord
	var total int64
	
	db := r.db.WithContext(ctx).Model(&model.BalanceQueryRecord{})
	if mobile != "" {
		db = db.Where("mobile = ?", mobile)
	}
	
	err := db.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	
	err = db.Preload("Order").Order("created_at desc").Offset(offset).Limit(limit).Find(&records).Error
	return records, total, err
}

// DeleteByOrderIDs 根据订单ID列表删除余额查询记录
func (r *balanceQueryRecordRepository) DeleteByOrderIDs(ctx context.Context, orderIDs []int64) error {
	if len(orderIDs) == 0 {
		return nil
	}
	// 使用Unscoped()进行硬删除，避免外键约束问题
	return r.db.WithContext(ctx).Unscoped().Where("order_id IN ?", orderIDs).Delete(&model.BalanceQueryRecord{}).Error
}

// ListByMobileAndType 根据手机号和查询类型分页查询余额查询记录
func (r *balanceQueryRecordRepository) ListByMobileAndType(ctx context.Context, mobile string, queryType string, offset, limit int) ([]model.BalanceQueryRecord, int64, error) {
	var records []model.BalanceQueryRecord
	var total int64
	
	db := r.db.WithContext(ctx).Model(&model.BalanceQueryRecord{})
	if mobile != "" {
		db = db.Where("mobile = ?", mobile)
	}
	if queryType != "" {
		db = db.Where("query_type = ?", queryType)
	}
	
	err := db.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	
	err = db.Preload("Order").Order("created_at desc").Offset(offset).Limit(limit).Find(&records).Error
	return records, total, err
}