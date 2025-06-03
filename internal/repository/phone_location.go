package repository

import (
	"recharge-go/internal/model"

	"gorm.io/gorm"
)

type PhoneLocationRepository struct {
	db *gorm.DB
}

func NewPhoneLocationRepository(db *gorm.DB) *PhoneLocationRepository {
	return &PhoneLocationRepository{
		db: db,
	}
}

// Create 创建手机归属地记录
func (r *PhoneLocationRepository) Create(phoneLocation *model.PhoneLocation) error {
	return r.db.Create(phoneLocation).Error
}

// Update 更新手机归属地记录
func (r *PhoneLocationRepository) Update(phoneLocation *model.PhoneLocation) error {
	return r.db.Save(phoneLocation).Error
}

// Delete 删除手机归属地记录
func (r *PhoneLocationRepository) Delete(id int64) error {
	return r.db.Delete(&model.PhoneLocation{}, id).Error
}

// GetByID 根据ID获取手机归属地记录
func (r *PhoneLocationRepository) GetByID(id int64) (*model.PhoneLocation, error) {
	var phoneLocation model.PhoneLocation
	err := r.db.First(&phoneLocation, id).Error
	if err != nil {
		return nil, err
	}
	return &phoneLocation, nil
}

// GetByPhoneNumber 根据手机号获取归属地记录
func (r *PhoneLocationRepository) GetByPhoneNumber(phoneNumber string) (*model.PhoneLocation, error) {
	var phoneLocation model.PhoneLocation
	err := r.db.Where("phone_number = ?", phoneNumber).First(&phoneLocation).Error
	if err != nil {
		return nil, err
	}
	return &phoneLocation, nil
}

// List 获取手机归属地列表
func (r *PhoneLocationRepository) List(req *model.PhoneLocationListRequest) (*model.PhoneLocationListResponse, error) {
	var total int64
	var locations []model.PhoneLocation

	// 构建查询条件
	query := r.db.Model(&model.PhoneLocation{})
	if req.Phone != "" {
		query = query.Where("phone_number LIKE ?", "%"+req.Phone+"%")
	}
	if req.Province != "" {
		query = query.Where("province LIKE ?", "%"+req.Province+"%")
	}
	if req.City != "" {
		query = query.Where("city LIKE ?", "%"+req.City+"%")
	}
	if req.ISP != "" {
		query = query.Where("isp LIKE ?", "%"+req.ISP+"%")
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// 分页查询
	offset := (req.Page - 1) * req.PageSize
	if err := query.Offset(offset).Limit(req.PageSize).Find(&locations).Error; err != nil {
		return nil, err
	}

	return &model.PhoneLocationListResponse{
		Total: total,
		Items: locations,
	}, nil
}
