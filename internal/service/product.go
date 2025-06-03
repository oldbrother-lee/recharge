package service

import (
	"context"
	"fmt"
	"recharge-go/internal/model"
	"recharge-go/internal/repository"
)

type ProductService struct {
	productRepo repository.ProductRepository
}

func NewProductService(productRepo repository.ProductRepository) *ProductService {
	return &ProductService{
		productRepo: productRepo,
	}
}

// List 获取商品列表
func (s *ProductService) List(ctx context.Context, req *model.ProductListRequest) (*model.ProductListResponse, error) {
	params := map[string]interface{}{
		"type":     req.Type,
		"category": req.Category,
		"isp":      req.ISP,
		"status":   req.Status,
	}
	products, total, err := s.productRepo.List(ctx, params, req.Page, req.PageSize)
	if err != nil {
		return nil, err
	}

	return &model.ProductListResponse{
		Total:   total,
		Records: products,
	}, nil
}

// GetByID 获取商品详情
func (s *ProductService) GetByID(ctx context.Context, id int64) (*model.ProductDetailResponse, error) {
	product, err := s.productRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	specs, err := s.productRepo.GetSpecs(ctx, id)
	if err != nil {
		return nil, err
	}

	gradePrices, err := s.productRepo.GetGradePrices(ctx, id)
	if err != nil {
		return nil, err
	}

	category, err := s.productRepo.GetCategory(ctx, product.CategoryID)
	if err != nil {
		return nil, err
	}

	return &model.ProductDetailResponse{
		Product:     *product,
		Specs:       specs,
		GradePrices: gradePrices,
		Category:    *category,
	}, nil
}

// Create 创建商品
func (s *ProductService) Create(ctx context.Context, req *model.ProductCreateRequest) (*model.Product, error) {
	product := &model.Product{
		Name:            req.Name,
		Description:     req.Description,
		Price:           req.Price,
		Type:            int64(req.Type),
		ISP:             req.ISP,
		Status:          req.Status,
		Sort:            req.Sort,
		APIEEnabled:     req.APIEnabled,
		Remark:          req.Remark,
		CategoryID:      req.CategoryID,
		OperatorTag:     req.OperatorTag,
		MaxPrice:        req.MaxPrice,
		VoucherPrice:    req.VoucherPrice,
		VoucherName:     req.VoucherName,
		ShowStyle:       req.ShowStyle,
		APIFailStyle:    req.APIFailStyle,
		AllowProvinces:  req.AllowProvinces,
		AllowCities:     req.AllowCities,
		ForbidProvinces: req.ForbidProvinces,
		ForbidCities:    req.ForbidCities,
		APIDelay:        req.APIDelay,
		GradeIDs:        req.GradeIDs,
		APIID:           req.APIID,
		APIParamID:      req.APIParamID,
		IsApi:           req.IsApi,
	}

	err := s.productRepo.Create(ctx, product)
	if err != nil {
		return nil, err
	}

	return product, nil
}

// Update 更新商品
func (s *ProductService) Update(ctx context.Context, req *model.ProductUpdateRequest) (*model.Product, error) {
	product, err := s.productRepo.GetByID(ctx, req.ID)
	if err != nil {
		return nil, err
	}

	product.Name = req.Name
	product.Description = req.Description
	product.Price = req.Price
	product.Type = int64(req.Type)
	product.ISP = req.ISP
	product.Status = req.Status
	product.Sort = req.Sort
	product.APIEEnabled = req.APIEnabled
	product.Remark = req.Remark
	product.CategoryID = req.CategoryID
	product.OperatorTag = req.OperatorTag
	product.MaxPrice = req.MaxPrice
	product.VoucherPrice = req.VoucherPrice
	product.VoucherName = req.VoucherName
	product.ShowStyle = req.ShowStyle
	product.APIFailStyle = req.APIFailStyle
	product.AllowProvinces = req.AllowProvinces
	product.AllowCities = req.AllowCities
	product.ForbidProvinces = req.ForbidProvinces
	product.ForbidCities = req.ForbidCities
	product.APIDelay = req.APIDelay
	product.GradeIDs = req.GradeIDs
	product.APIID = req.APIID
	product.APIParamID = req.APIParamID
	product.IsApi = req.IsApi

	fmt.Println("更新商品分类ID：", req.CategoryID)

	err = s.productRepo.Update(ctx, product)
	if err != nil {
		return nil, err
	}

	return product, nil
}

// Delete 删除商品
func (s *ProductService) Delete(ctx context.Context, id int64) error {
	return s.productRepo.Delete(ctx, id)
}

// ListCategories 获取商品分类列表
func (s *ProductService) ListCategories(ctx context.Context) (*model.ProductCategoryListResponse, error) {
	categories, err := s.productRepo.ListCategories(ctx)
	if err != nil {
		return nil, err
	}

	return &model.ProductCategoryListResponse{
		Total: int64(len(categories)),
		List:  categories,
	}, nil
}

// CreateCategory 创建商品分类
func (s *ProductService) CreateCategory(ctx context.Context, category *model.ProductCategory) error {
	return s.productRepo.CreateCategory(ctx, category)
}

// UpdateCategory 更新商品分类
func (s *ProductService) UpdateCategory(ctx context.Context, category *model.ProductCategory) error {
	return s.productRepo.UpdateCategory(ctx, category)
}

// DeleteCategory 删除商品分类
func (s *ProductService) DeleteCategory(ctx context.Context, id int64) error {
	return s.productRepo.DeleteCategory(ctx, id)
}

// ListTypes 获取商品类型列表
func (s *ProductService) ListTypes(ctx context.Context) ([]model.ProductType, error) {
	return s.productRepo.ListTypes(ctx)
}
