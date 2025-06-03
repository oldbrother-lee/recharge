package repository

import (
	"recharge-go/internal/model"

	"gorm.io/gorm"
)

// ProductTypeRepository 产品类型仓储
type ProductTypeRepository struct {
	db *gorm.DB
}

// NewProductTypeRepository 创建产品类型仓储实例
func NewProductTypeRepository(db *gorm.DB) *ProductTypeRepository {
	return &ProductTypeRepository{
		db: db,
	}
}

// Create 创建产品类型
func (r *ProductTypeRepository) Create(productType *model.ProductType) error {
	return r.db.Create(productType).Error
}

// Update 更新产品类型
func (r *ProductTypeRepository) Update(productType *model.ProductType) error {
	return r.db.Save(productType).Error
}

// Delete 删除产品类型
func (r *ProductTypeRepository) Delete(id int64) error {
	return r.db.Delete(&model.ProductType{}, id).Error
}

// GetByID 根据ID获取产品类型
func (r *ProductTypeRepository) GetByID(id int64) (*model.ProductType, error) {
	var productType model.ProductType
	err := r.db.Preload("Category").First(&productType, id).Error
	if err != nil {
		return nil, err
	}
	return &productType, nil
}

// GetByTypeName 根据类型名称获取产品类型
func (r *ProductTypeRepository) GetByTypeName(typeName string) (*model.ProductType, error) {
	var productType model.ProductType
	err := r.db.Where("type_name = ?", typeName).First(&productType).Error
	if err != nil {
		return nil, err
	}
	return &productType, nil
}

// List 获取产品类型列表
func (r *ProductTypeRepository) List(req *model.ProductTypeListRequest) (*model.ProductTypeListResponse, error) {
	var total int64
	var types []model.ProductType

	// 构建查询条件
	query := r.db.Model(&model.ProductType{}).Preload("Category")
	if req.TypeName != "" {
		query = query.Where("type_name LIKE ?", "%"+req.TypeName+"%")
	}
	if req.TypecID != nil {
		query = query.Where("typec_id = ?", *req.TypecID)
	}
	if req.Status != nil {
		query = query.Where("status = ?", *req.Status)
	}
	if req.AccountType != nil {
		query = query.Where("account_type = ?", *req.AccountType)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// 分页查询
	offset := (req.Page - 1) * req.PageSize
	if err := query.Order("sort ASC").Offset(offset).Limit(req.PageSize).Find(&types).Error; err != nil {
		return nil, err
	}

	return &model.ProductTypeListResponse{
		Total: total,
		Items: types,
	}, nil
}

// ProductTypeCategoryRepository 产品类型分类仓储
type ProductTypeCategoryRepository struct {
	db *gorm.DB
}

// NewProductTypeCategoryRepository 创建产品类型分类仓储实例
func NewProductTypeCategoryRepository(db *gorm.DB) *ProductTypeCategoryRepository {
	return &ProductTypeCategoryRepository{
		db: db,
	}
}

// GetByID 根据ID获取产品类型分类
func (r *ProductTypeCategoryRepository) GetByID(id int64) (*model.ProductTypeCategory, error) {
	var category model.ProductTypeCategory
	err := r.db.First(&category, id).Error
	if err != nil {
		return nil, err
	}
	return &category, nil
}

// List 获取产品类型分类列表
func (r *ProductTypeCategoryRepository) List(req *model.ProductTypeCategoryListRequest) (*model.ProductTypeCategoryListResponse, error) {
	var total int64
	var categories []model.ProductTypeCategory

	// 构建查询条件
	query := r.db.Model(&model.ProductTypeCategory{})
	if req.Name != "" {
		query = query.Where("cname LIKE ?", "%"+req.Name+"%")
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// 分页查询
	offset := (req.Page - 1) * req.PageSize
	if err := query.Order("id DESC").Offset(offset).Limit(req.PageSize).Find(&categories).Error; err != nil {
		return nil, err
	}

	return &model.ProductTypeCategoryListResponse{
		Total: total,
		Items: categories,
	}, nil
}
