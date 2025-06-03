package service

import (
	"errors"
	"recharge-go/internal/model"
	"recharge-go/internal/repository"
)

var (
	ErrProductTypeNotFound         = errors.New("product type not found")
	ErrProductTypeNameExists       = errors.New("product type name already exists")
	ErrProductTypeCategoryNotFound = errors.New("product type category not found")
)

// ProductTypeService 产品类型服务
type ProductTypeService struct {
	productTypeRepo     *repository.ProductTypeRepository
	productTypeCateRepo *repository.ProductTypeCategoryRepository
}

// NewProductTypeService 创建产品类型服务实例
func NewProductTypeService(
	productTypeRepo *repository.ProductTypeRepository,
	productTypeCateRepo *repository.ProductTypeCategoryRepository,
) *ProductTypeService {
	return &ProductTypeService{
		productTypeRepo:     productTypeRepo,
		productTypeCateRepo: productTypeCateRepo,
	}
}

// Create 创建产品类型
func (s *ProductTypeService) Create(req *model.ProductTypeCreateRequest) error {
	// 检查类型分类是否存在
	category, err := s.productTypeCateRepo.GetByID(req.TypecID)
	if err != nil {
		return ErrProductTypeCategoryNotFound
	}

	// 检查类型名称是否已存在
	existingType, err := s.productTypeRepo.GetByTypeName(req.TypeName)
	if err == nil && existingType != nil {
		return ErrProductTypeNameExists
	}

	// 创建产品类型
	productType := &model.ProductType{
		TypeName:    req.TypeName,
		TypecID:     req.TypecID,
		Status:      req.Status,
		Sort:        req.Sort,
		AccountType: req.AccountType,
		TishiDoc:    req.TishiDoc,
		Icon:        req.Icon,
		Category:    category,
	}

	return s.productTypeRepo.Create(productType)
}

// Update 更新产品类型
func (s *ProductTypeService) Update(req *model.ProductTypeUpdateRequest) error {
	// 检查产品类型是否存在
	productType, err := s.productTypeRepo.GetByID(req.ID)
	if err != nil {
		return ErrProductTypeNotFound
	}

	// 检查类型分类是否存在
	category, err := s.productTypeCateRepo.GetByID(req.TypecID)
	if err != nil {
		return ErrProductTypeCategoryNotFound
	}

	// 如果类型名称有变更，检查是否与其他记录重复
	if productType.TypeName != req.TypeName {
		existingType, err := s.productTypeRepo.GetByTypeName(req.TypeName)
		if err == nil && existingType != nil && existingType.ID != req.ID {
			return ErrProductTypeNameExists
		}
	}

	// 更新产品类型
	productType.TypeName = req.TypeName
	productType.TypecID = req.TypecID
	productType.Status = req.Status
	productType.Sort = req.Sort
	productType.AccountType = req.AccountType
	productType.TishiDoc = req.TishiDoc
	productType.Icon = req.Icon
	productType.Category = category

	return s.productTypeRepo.Update(productType)
}

// Delete 删除产品类型
func (s *ProductTypeService) Delete(id int64) error {
	// 检查产品类型是否存在
	_, err := s.productTypeRepo.GetByID(id)
	if err != nil {
		return ErrProductTypeNotFound
	}

	return s.productTypeRepo.Delete(id)
}

// GetByID 根据ID获取产品类型
func (s *ProductTypeService) GetByID(id int64) (*model.ProductType, error) {
	productType, err := s.productTypeRepo.GetByID(id)
	if err != nil {
		return nil, ErrProductTypeNotFound
	}
	return productType, nil
}

// List 获取产品类型列表
func (s *ProductTypeService) List(req *model.ProductTypeListRequest) (*model.ProductTypeListResponse, error) {
	return s.productTypeRepo.List(req)
}

// ListCategories 获取产品类型分类列表
func (s *ProductTypeService) ListCategories(req *model.ProductTypeCategoryListRequest) (*model.ProductTypeCategoryListResponse, error) {
	return s.productTypeCateRepo.List(req)
}
