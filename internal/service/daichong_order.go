package service

import (
	"context"
	"recharge-go/internal/model"
	"recharge-go/internal/repository"
)

type DaichongOrderService struct {
	repo *repository.DaichongOrderRepository
}

func NewDaichongOrderService(repo *repository.DaichongOrderRepository) *DaichongOrderService {
	return &DaichongOrderService{repo: repo}
}

// Create 新增订单
func (s *DaichongOrderService) Create(ctx context.Context, order *model.DaichongOrder) error {

	return s.repo.Create(order)
}

// GetByID 根据ID查询订单
func (s *DaichongOrderService) GetByID(ctx context.Context, id int64) (*model.DaichongOrder, error) {
	return s.repo.GetByID(id)
}

// Update 更新订单
func (s *DaichongOrderService) Update(ctx context.Context, order *model.DaichongOrder) error {
	return s.repo.Update(order)
}

// Delete 删除订单
func (s *DaichongOrderService) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(id)
}

// List 分页查询订单列表
func (s *DaichongOrderService) List(query map[string]interface{}) ([]*model.DaichongOrder, int64, error) {
	page := query["page"].(int)
	pageSize := query["page_size"].(int)

	// 构建查询条件
	conditions := make(map[string]interface{})
	if account, ok := query["account"]; ok && account != "" {
		conditions["account"] = account
	}
	if YrOrderID, ok := query["yr_order_id"]; ok && YrOrderID != "" {
		conditions["yr_order_id"] = YrOrderID
	}
	if status, ok := query["status"]; ok && status.(int) > 0 {
		conditions["status"] = status
	}
	if startTime, ok := query["start_time"]; ok && startTime.(int64) > 0 {
		conditions["create_time >= ?"] = startTime
	}
	if endTime, ok := query["end_time"]; ok && endTime.(int64) > 0 {
		conditions["create_time <= ?"] = endTime
	}

	return s.repo.List(page, pageSize, conditions)
}
