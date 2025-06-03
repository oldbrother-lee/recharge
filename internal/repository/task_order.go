package repository

import (
	"recharge-go/internal/model"
	"recharge-go/pkg/database"

	"gorm.io/gorm"
)

type TaskOrderRepository struct {
	db *gorm.DB
}

func NewTaskOrderRepository() *TaskOrderRepository {
	return &TaskOrderRepository{
		db: database.DB,
	}
}

// Create 创建任务订单
func (r *TaskOrderRepository) Create(order *model.TaskOrder) error {
	return r.db.Create(order).Error
}

// Update 更新任务订单
func (r *TaskOrderRepository) Update(order *model.TaskOrder) error {
	return r.db.Save(order).Error
}

// GetByOrderNumber 根据订单号获取任务订单
func (r *TaskOrderRepository) GetByOrderNumber(orderNumber string) (*model.TaskOrder, error) {
	var order model.TaskOrder
	err := r.db.Where("order_number = ?", orderNumber).First(&order).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

// List 获取任务订单列表
func (r *TaskOrderRepository) List(page, pageSize int) ([]model.TaskOrder, int64, error) {
	var orders []model.TaskOrder
	var total int64

	err := r.db.Model(&model.TaskOrder{}).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = r.db.Offset((page - 1) * pageSize).Limit(pageSize).Find(&orders).Error
	if err != nil {
		return nil, 0, err
	}

	return orders, total, nil
}

// GetPendingOrders 获取待处理的订单
func (r *TaskOrderRepository) GetPendingOrders() ([]model.TaskOrder, error) {
	var orders []model.TaskOrder
	err := r.db.Where("order_status = ? AND settlement_status = ?", 1, 1).Find(&orders).Error
	if err != nil {
		return nil, err
	}
	return orders, nil
}
