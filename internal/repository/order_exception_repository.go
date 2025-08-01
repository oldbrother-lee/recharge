package repository

import (
	"context"
	"fmt"
	"time"

	"recharge-go/internal/model"

	"gorm.io/gorm"
)

// OrderExceptionRepository 订单异常仓库接口
type OrderExceptionRepository interface {
	// Create 创建订单异常记录
	Create(ctx context.Context, exception *model.OrderException) error
	// GetByID 根据ID获取订单异常记录
	GetByID(ctx context.Context, id int64) (*model.OrderException, error)
	// GetByOrderID 根据订单ID获取异常记录列表
	GetByOrderID(ctx context.Context, orderID int64) ([]model.OrderException, error)
	// List 获取订单异常列表
	List(ctx context.Context, req *model.OrderExceptionListRequest) ([]model.OrderException, int64, error)
	// Update 更新订单异常记录
	Update(ctx context.Context, exception *model.OrderException) error
	// UpdateStatus 更新异常状态
	UpdateStatus(ctx context.Context, id int64, status string, resolvedBy *int64, resolvedNote string) error
	// GetPendingCount 获取待处理异常数量
	GetPendingCount(ctx context.Context) (int64, error)
	// GetStatistics 获取异常统计信息
	GetStatistics(ctx context.Context, startDate, endDate time.Time) (map[string]int64, error)
}

type orderExceptionRepository struct {
	db *gorm.DB
}

// NewOrderExceptionRepository 创建订单异常仓库实例
func NewOrderExceptionRepository(db *gorm.DB) OrderExceptionRepository {
	return &orderExceptionRepository{
		db: db,
	}
}

// Create 创建订单异常记录
func (r *orderExceptionRepository) Create(ctx context.Context, exception *model.OrderException) error {
	return r.db.WithContext(ctx).Create(exception).Error
}

// GetByID 根据ID获取订单异常记录
func (r *orderExceptionRepository) GetByID(ctx context.Context, id int64) (*model.OrderException, error) {
	var exception model.OrderException
	err := r.db.WithContext(ctx).Preload("Order").First(&exception, id).Error
	if err != nil {
		return nil, err
	}
	return &exception, nil
}

// GetByOrderID 根据订单ID获取异常记录列表
func (r *orderExceptionRepository) GetByOrderID(ctx context.Context, orderID int64) ([]model.OrderException, error) {
	var exceptions []model.OrderException
	err := r.db.WithContext(ctx).Where("order_id = ?", orderID).Order("created_at DESC").Find(&exceptions).Error
	return exceptions, err
}

// List 获取订单异常列表
func (r *orderExceptionRepository) List(ctx context.Context, req *model.OrderExceptionListRequest) ([]model.OrderException, int64, error) {
	var exceptions []model.OrderException
	var total int64

	// 构建查询条件
	query := r.db.WithContext(ctx).Model(&model.OrderException{})

	if req.OrderNumber != "" {
		query = query.Where("order_number LIKE ?", "%"+req.OrderNumber+"%")
	}

	if req.ExceptionType != "" {
		query = query.Where("exception_type = ?", req.ExceptionType)
	}

	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}

	if req.StartDate != "" {
		startDate, err := time.Parse("2006-01-02", req.StartDate)
		if err == nil {
			query = query.Where("created_at >= ?", startDate)
		}
	}

	if req.EndDate != "" {
		endDate, err := time.Parse("2006-01-02", req.EndDate)
		if err == nil {
			// 结束日期加一天，包含当天的所有记录
			endDate = endDate.AddDate(0, 0, 1)
			query = query.Where("created_at < ?", endDate)
		}
	}

	// 获取总数
	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	// 分页参数
	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	offset := (page - 1) * pageSize

	// 获取列表数据
	err = query.Preload("Order").Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&exceptions).Error
	if err != nil {
		return nil, 0, err
	}

	return exceptions, total, nil
}

// Update 更新订单异常记录
func (r *orderExceptionRepository) Update(ctx context.Context, exception *model.OrderException) error {
	return r.db.WithContext(ctx).Save(exception).Error
}

// UpdateStatus 更新异常状态
func (r *orderExceptionRepository) UpdateStatus(ctx context.Context, id int64, status string, resolvedBy *int64, resolvedNote string) error {
	updates := map[string]interface{}{
		"status":        status,
		"resolved_note": resolvedNote,
	}

	if status == model.ExceptionStatusResolved || status == model.ExceptionStatusIgnored {
		now := time.Now()
		updates["resolved_at"] = &now
		updates["resolved_by"] = resolvedBy
	} else {
		updates["resolved_at"] = nil
		updates["resolved_by"] = nil
	}

	return r.db.WithContext(ctx).Model(&model.OrderException{}).Where("id = ?", id).Updates(updates).Error
}

// GetPendingCount 获取待处理异常数量
func (r *orderExceptionRepository) GetPendingCount(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.OrderException{}).Where("status = ?", model.ExceptionStatusPending).Count(&count).Error
	return count, err
}

// GetStatistics 获取异常统计信息
func (r *orderExceptionRepository) GetStatistics(ctx context.Context, startDate, endDate time.Time) (map[string]int64, error) {
	type StatResult struct {
		ExceptionType string `json:"exception_type"`
		Status        string `json:"status"`
		Count         int64  `json:"count"`
	}

	var results []StatResult
	err := r.db.WithContext(ctx).Model(&model.OrderException{}).
		Select("exception_type, status, COUNT(*) as count").
		Where("created_at >= ? AND created_at < ?", startDate, endDate).
		Group("exception_type, status").
		Find(&results).Error

	if err != nil {
		return nil, err
	}

	stats := make(map[string]int64)
	for _, result := range results {
		key := fmt.Sprintf("%s_%s", result.ExceptionType, result.Status)
		stats[key] = result.Count
	}

	return stats, nil
}