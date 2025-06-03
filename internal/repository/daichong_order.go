package repository

import (
	"recharge-go/internal/model"
	"recharge-go/internal/utils"
	"strings"
	"time"

	"gorm.io/gorm"
)

type DaichongOrderRepository struct {
	db *gorm.DB
}

func NewDaichongOrderRepository(db *gorm.DB) *DaichongOrderRepository {
	return &DaichongOrderRepository{db: db}
}

// Create 新增订单
func (r *DaichongOrderRepository) Create(order *model.DaichongOrder) error {
	order.YrOrderID = generateOrderNumber()
	return r.db.Create(order).Error
}

// GetByID 根据ID查询订单
func (r *DaichongOrderRepository) GetByID(id int64) (*model.DaichongOrder, error) {
	var order model.DaichongOrder
	if err := r.db.First(&order, id).Error; err != nil {
		return nil, err
	}
	return &order, nil
}

// Update 更新订单
func (r *DaichongOrderRepository) Update(order *model.DaichongOrder) error {
	return r.db.Save(order).Error
}

// Delete 删除订单
func (r *DaichongOrderRepository) Delete(id int64) error {
	return r.db.Delete(&model.DaichongOrder{}, id).Error
}

// List 分页查询订单列表
func (r *DaichongOrderRepository) List(page, pageSize int, conditions map[string]interface{}) ([]*model.DaichongOrder, int64, error) {
	var orders []*model.DaichongOrder
	var total int64

	query := r.db.Model(&model.DaichongOrder{})

	// 应用过滤条件
	for key, value := range conditions {
		if strings.Contains(key, "?") {
			// 处理带占位符的条件
			query = query.Where(key, value)
		} else {
			query = query.Where(key, value)
		}
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("create_time DESC").Find(&orders).Error; err != nil {
		return nil, 0, err
	}

	return orders, total, nil
}

// generateOrderNumber 生成订单号
func generateOrderNumber() string {
	return "DC" + time.Now().Format("20060102150405") + utils.RandString(6)
}
