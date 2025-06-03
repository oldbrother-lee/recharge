package repository

import (
	"context"
	"errors"
	"fmt"
	"recharge-go/internal/model"

	"gorm.io/gorm"
)

// ProductAPIRelationRepository 商品接口关联仓库接口
type ProductAPIRelationRepository interface {
	Create(ctx context.Context, relation *model.ProductAPIRelation) error
	Update(ctx context.Context, relation *model.ProductAPIRelation) error
	Delete(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (*model.ProductAPIRelation, error)
	List(ctx context.Context, productID, apiID int64, status int, page, pageSize int) ([]*model.ProductAPIRelation, int64, error)
	GetByProductID(ctx context.Context, productID int64) (*model.ProductAPIRelation, error)
}

type productAPIRelationRepository struct {
	db *gorm.DB
}

// NewProductAPIRelationRepository 创建商品接口关联仓库实例
func NewProductAPIRelationRepository(db *gorm.DB) ProductAPIRelationRepository {
	return &productAPIRelationRepository{db: db}
}

// Create 创建商品接口关联
func (r *productAPIRelationRepository) Create(ctx context.Context, relation *model.ProductAPIRelation) error {
	return r.db.WithContext(ctx).Create(relation).Error
}

// Update 更新商品接口关联
func (r *productAPIRelationRepository) Update(ctx context.Context, relation *model.ProductAPIRelation) error {
	return r.db.WithContext(ctx).Save(relation).Error
}

// Delete 删除商品接口关联
func (r *productAPIRelationRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.ProductAPIRelation{}, id).Error
}

// GetByID 根据ID获取商品接口关联
func (r *productAPIRelationRepository) GetByID(ctx context.Context, id int64) (*model.ProductAPIRelation, error) {
	var relation model.ProductAPIRelation
	err := r.db.WithContext(ctx).First(&relation, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &relation, nil
}

// List 获取商品接口关联列表
func (r *productAPIRelationRepository) List(ctx context.Context, productID, apiID int64, status int, page, pageSize int) ([]*model.ProductAPIRelation, int64, error) {
	var relations []*model.ProductAPIRelation
	var total int64

	query := r.db.WithContext(ctx).Table("product_api_relations").
		Select("product_api_relations.*, platform_apis.name as product_name, platform_api_params.name as api_name").
		Joins("LEFT JOIN platform_apis ON product_api_relations.api_id = platform_apis.id").
		Joins("LEFT JOIN platform_api_params ON product_api_relations.param_id = platform_api_params.id").
		Debug()

	// 打印执行的 sql 语句
	sql := query.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return tx
	})
	fmt.Println("SQL:", sql)

	if productID > 0 {
		query = query.Where("product_api_relations.product_id = ?", productID)
	}
	if apiID > 0 {
		query = query.Where("product_api_relations.api_id = ?", apiID)
	}
	if status >= 0 {
		query = query.Where("product_api_relations.status = ?", status)
	}

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Offset((page - 1) * pageSize).Limit(pageSize).Find(&relations).Error
	if err != nil {
		return nil, 0, err
	}

	// 打印调试信息
	for _, relation := range relations {
		fmt.Printf("Relation: %+v\n", relation)
	}

	return relations, total, nil
}

func (r *productAPIRelationRepository) GetByProductID(ctx context.Context, productID int64) (*model.ProductAPIRelation, error) {
	var relation model.ProductAPIRelation
	if err := r.db.Where("product_id = ?", productID).First(&relation).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("product %d has no API relation", productID)
		}
		return nil, err
	}
	return &relation, nil
}
