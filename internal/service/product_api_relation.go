package service

import (
	"context"
	"recharge-go/internal/model"
	"recharge-go/internal/repository"
)

// ProductAPIRelationService 商品接口关联服务接口
type ProductAPIRelationService interface {
	Create(ctx context.Context, req *model.ProductAPIRelationCreateRequest) error
	Update(ctx context.Context, req *model.ProductAPIRelationUpdateRequest) error
	Delete(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (*model.ProductAPIRelation, error)
	List(ctx context.Context, req *model.ProductAPIRelationListRequest) (*model.ProductAPIRelationListResponse, error)
}

type productAPIRelationService struct {
	repo repository.ProductAPIRelationRepository
}

// NewProductAPIRelationService 创建商品接口关联服务实例
func NewProductAPIRelationService(repo repository.ProductAPIRelationRepository) ProductAPIRelationService {
	return &productAPIRelationService{repo: repo}
}

// Create 创建商品接口关联
func (s *productAPIRelationService) Create(ctx context.Context, req *model.ProductAPIRelationCreateRequest) error {
	relation := &model.ProductAPIRelation{
		ProductID: req.ProductID,
		APIID:     req.APIID,
		ParamID:   req.ParamID,
		Sort:      req.Sort,
		Status:    req.Status,
		RetryNum:  req.RetryNum,
		ISP:       req.Isp,
	}
	return s.repo.Create(ctx, relation)
}

// Update 更新商品接口关联
func (s *productAPIRelationService) Update(ctx context.Context, req *model.ProductAPIRelationUpdateRequest) error {
	relation := &model.ProductAPIRelation{
		ID:        req.ID,
		ProductID: req.ProductID,
		APIID:     req.APIID,
		ParamID:   req.ParamID,
		Sort:      req.Sort,
		Status:    req.Status,
		RetryNum:  req.RetryNum,
		ISP:       req.Isp,
	}
	return s.repo.Update(ctx, relation)
}

// Delete 删除商品接口关联
func (s *productAPIRelationService) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

// GetByID 根据ID获取商品接口关联
func (s *productAPIRelationService) GetByID(ctx context.Context, id int64) (*model.ProductAPIRelation, error) {
	return s.repo.GetByID(ctx, id)
}

// List 获取商品接口关联列表
func (s *productAPIRelationService) List(ctx context.Context, req *model.ProductAPIRelationListRequest) (*model.ProductAPIRelationListResponse, error) {
	var productID, apiID int64
	var status int = -1

	if req.ProductID != nil {
		productID = *req.ProductID
	}
	if req.APIID != nil {
		apiID = *req.APIID
	}
	if req.Status != nil {
		status = *req.Status
	}

	relations, total, err := s.repo.List(ctx, productID, apiID, status, req.Page, req.PageSize)
	if err != nil {
		return nil, err
	}

	// Convert pointer slice to value slice
	items := make([]model.ProductAPIRelation, len(relations))
	for i, r := range relations {
		items[i] = *r
	}

	return &model.ProductAPIRelationListResponse{
		Total: total,
		List:  items,
	}, nil
}
