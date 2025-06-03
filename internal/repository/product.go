package repository

import (
	"context"
	"fmt"
	"recharge-go/internal/model"
	"strconv"

	"gorm.io/gorm"
)

// ProductRepository 商品仓库接口
type ProductRepository interface {
	GetByID(ctx context.Context, id int64) (*model.Product, error)
	GetByCode(ctx context.Context, code string) (*model.Product, error)
	List(ctx context.Context, params map[string]interface{}, page, pageSize int) ([]*model.Product, int64, error)
	Create(ctx context.Context, product *model.Product) error
	Update(ctx context.Context, product *model.Product) error
	Delete(ctx context.Context, id int64) error
	GetSpecs(ctx context.Context, productID int64) ([]model.ProductSpec, error)
	GetGradePrices(ctx context.Context, productID int64) ([]model.ProductGradePrice, error)
	GetCategory(ctx context.Context, id int64) (*model.ProductCategory, error)
	ListCategories(ctx context.Context) ([]model.ProductCategory, error)
	CreateCategory(ctx context.Context, category *model.ProductCategory) error
	UpdateCategory(ctx context.Context, category *model.ProductCategory) error
	DeleteCategory(ctx context.Context, id int64) error
	ListTypes(ctx context.Context) ([]model.ProductType, error)
	GetAPIRelationsByProductID(ctx context.Context, productID int64) ([]*model.ProductAPIRelation, error)
}

type productRepository struct {
	db *gorm.DB
}

func NewProductRepository(db *gorm.DB) ProductRepository {
	return &productRepository{db: db}
}

// List 获取商品列表
func (r *productRepository) List(ctx context.Context, params map[string]interface{}, page, pageSize int) ([]*model.Product, int64, error) {
	var products []*model.Product
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Product{}).
		Preload("Category").    // 预加载分类
		Preload("ProductType"). // 预加载商品类型
		Where("status = ?", 1)

	// 参数白名单判断
	if v, ok := params["type"]; ok {
		if typeInt, err := strconv.Atoi(fmt.Sprint(v)); err == nil && typeInt > 0 {
			query = query.Where("type = ?", typeInt)
		}
	}
	if v, ok := params["category"]; ok {
		if catInt, err := strconv.Atoi(fmt.Sprint(v)); err == nil && catInt > 0 {
			query = query.Where("category_id = ?", catInt)
		}
	}
	if v, ok := params["isp"]; ok {
		ispStr := fmt.Sprint(v)
		if ispStr != "" {
			query = query.Where("isp LIKE ?", "%"+ispStr+"%")
		}
	}
	if v, ok := params["status"]; ok {
		if statusInt, err := strconv.Atoi(fmt.Sprint(v)); err == nil && statusInt > 0 {
			query = query.Where("status = ?", statusInt)
		}
	}

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Order("sort asc, id asc").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&products).Error

	return products, total, err
}

// GetByID 根据ID获取商品
func (r *productRepository) GetByID(ctx context.Context, id int64) (*model.Product, error) {
	var product model.Product
	err := r.db.WithContext(ctx).Preload("Category").
		Preload("ProductType").
		First(&product, id).Error
	if err != nil {
		return nil, err
	}
	return &product, nil
}

// GetByCode 根据编码获取商品
func (r *productRepository) GetByCode(ctx context.Context, code string) (*model.Product, error) {
	var product model.Product
	err := r.db.WithContext(ctx).Where("code = ?", code).First(&product).Error
	if err != nil {
		return nil, err
	}
	return &product, nil
}

// Create 创建商品
func (r *productRepository) Create(ctx context.Context, product *model.Product) error {
	return r.db.WithContext(ctx).Create(product).Error
}

// Update 更新商品
func (r *productRepository) Update(ctx context.Context, product *model.Product) error {
	return r.db.WithContext(ctx).Save(product).Error
}

// Delete 删除商品
func (r *productRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.Product{}, id).Error
}

// GetSpecs 获取商品规格
func (r *productRepository) GetSpecs(ctx context.Context, productID int64) ([]model.ProductSpec, error) {
	var specs []model.ProductSpec
	err := r.db.WithContext(ctx).Where("product_id = ?", productID).Order("sort asc").Find(&specs).Error
	return specs, err
}

// GetGradePrices 获取商品会员价格
func (r *productRepository) GetGradePrices(ctx context.Context, productID int64) ([]model.ProductGradePrice, error) {
	var prices []model.ProductGradePrice
	err := r.db.WithContext(ctx).Where("product_id = ?", productID).Find(&prices).Error
	return prices, err
}

// GetCategory 获取商品分类
func (r *productRepository) GetCategory(ctx context.Context, id int64) (*model.ProductCategory, error) {
	var category model.ProductCategory
	err := r.db.WithContext(ctx).First(&category, id).Error
	if err != nil {
		return nil, err
	}
	return &category, nil
}

// ListCategories 获取商品分类列表
func (r *productRepository) ListCategories(ctx context.Context) ([]model.ProductCategory, error) {
	var categories []model.ProductCategory
	err := r.db.WithContext(ctx).Order("sort asc").Find(&categories).Error
	return categories, err
}

// CreateCategory 创建商品分类
func (r *productRepository) CreateCategory(ctx context.Context, category *model.ProductCategory) error {
	return r.db.WithContext(ctx).Create(category).Error
}

// UpdateCategory 更新商品分类
func (r *productRepository) UpdateCategory(ctx context.Context, category *model.ProductCategory) error {
	return r.db.WithContext(ctx).Save(category).Error
}

// DeleteCategory 删除商品分类
func (r *productRepository) DeleteCategory(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.ProductCategory{}, id).Error
}

// ListTypes 获取商品类型列表
func (r *productRepository) ListTypes(ctx context.Context) ([]model.ProductType, error) {
	var types []model.ProductType
	err := r.db.WithContext(ctx).Order("sort asc").Find(&types).Error
	return types, err
}

// GetAPIRelationsByProductID 获取商品API关系列表
func (r *productRepository) GetAPIRelationsByProductID(ctx context.Context, productID int64) ([]*model.ProductAPIRelation, error) {
	var relations []*model.ProductAPIRelation
	err := r.db.WithContext(ctx).
		Where("product_id = ? AND status = 1", productID).
		Order("priority DESC").
		Find(&relations).Error
	return relations, err
}
