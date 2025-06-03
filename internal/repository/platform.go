package repository

import (
	"context"
	"errors"
	"recharge-go/internal/model"

	"gorm.io/gorm"
)

var ErrNoAPIForProduct = errors.New("商品未绑定接口")

// PlatformRepository 平台仓库接口
type PlatformRepository interface {
	// 平台相关方法
	ListPlatforms(req *model.PlatformListRequest) ([]model.Platform, int64, error)
	GetPlatformByID(id int64) (*model.Platform, error)
	GetPlatformByCode(ctx context.Context, code string) (*model.PlatformAPI, error)
	CreatePlatform(platform *model.Platform) error
	UpdatePlatform(platform *model.Platform) error
	Delete(id int64) error

	// 平台账号相关方法
	GetAccountByID(ctx context.Context, id int64) (*model.PlatformAccount, error)
	GetAccountsByPlatformID(ctx context.Context, platformID int64) ([]*model.PlatformAccount, error)
	CreateAccount(ctx context.Context, account *model.PlatformAccount) error
	UpdateAccount(ctx context.Context, account *model.PlatformAccount) error
	DeleteAccount(ctx context.Context, id int64) error
	ListPlatformAccounts(req *model.PlatformAccountListRequest) (*model.PlatformAccountListResponse, error)
	GetPlatformAccountByID(id int64) (*model.PlatformAccount, error)
	CreatePlatformAccount(account *model.PlatformAccount) error
	UpdatePlatformAccount(account *model.PlatformAccount) error
	GetAPIByID(ctx context.Context, apiID int64) (*model.PlatformAPI, error)
	GetAPIParamByID(ctx context.Context, id int64) (*model.PlatformAPIParam, error)
	UpdatePlatformAccountFields(ctx context.Context, id int64, fields map[string]interface{}) error
	GetPlatformAccountByAccountName(accountName string) (*model.PlatformAccount, error)
	// 数据库相关方法
	GetDB() *gorm.DB
}

type PlatformRepositoryImpl struct {
	db *gorm.DB
}

func NewPlatformRepository(db *gorm.DB) *PlatformRepositoryImpl {
	return &PlatformRepositoryImpl{db: db}
}

// List 获取平台列表
func (r *PlatformRepositoryImpl) List(req *model.PlatformListRequest) (*model.PlatformListResponse, error) {
	var total int64
	var items []model.Platform

	query := r.db.Model(&model.Platform{})

	if req.Name != "" {
		query = query.Where("name LIKE ?", "%"+req.Name+"%")
	}
	if req.Code != "" {
		query = query.Where("code = ?", req.Code)
	}
	if req.Status != nil {
		query = query.Where("status = ?", *req.Status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	offset := (req.Page - 1) * req.PageSize
	if err := query.Offset(offset).Limit(req.PageSize).Find(&items).Error; err != nil {
		return nil, err
	}

	return &model.PlatformListResponse{
		Total: total,
		Items: items,
	}, nil
}

// GetByID 根据ID获取平台
func (r *PlatformRepositoryImpl) GetByID(id int64) (*model.Platform, error) {
	var platform model.Platform
	if err := r.db.First(&platform, id).Error; err != nil {
		return nil, err
	}
	return &platform, nil
}

// Create 创建平台
func (r *PlatformRepositoryImpl) Create(platform *model.Platform) error {
	return r.db.Create(platform).Error
}

// Update 更新平台
func (r *PlatformRepositoryImpl) Update(platform *model.Platform) error {
	return r.db.Save(platform).Error
}

// Delete 删除平台
func (r *PlatformRepositoryImpl) Delete(id int64) error {
	return r.db.Delete(&model.Platform{}, id).Error
}

// ListAccounts 获取平台账号列表
func (r *PlatformRepositoryImpl) ListAccounts(req *model.PlatformAccountListRequest) (*model.PlatformAccountListResponse, error) {
	var total int64
	var items []model.PlatformAccount

	query := r.db.Model(&model.PlatformAccount{}).Preload("Platform")

	if req.PlatformID != nil {
		query = query.Where("platform_id = ?", *req.PlatformID)
	}
	if req.Status != nil {
		query = query.Where("status = ?", *req.Status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	offset := (req.Page - 1) * req.PageSize
	if err := query.Offset(offset).Limit(req.PageSize).Find(&items).Error; err != nil {
		return nil, err
	}

	return &model.PlatformAccountListResponse{
		Total: total,
		Items: items,
	}, nil
}

// GetAccountByID 根据ID获取平台账号
func (r *PlatformRepositoryImpl) GetAccountByID(ctx context.Context, id int64) (*model.PlatformAccount, error) {
	var account model.PlatformAccount
	if err := r.db.Preload("Platform").First(&account, id).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

// CreateAccount 创建平台账号
func (r *PlatformRepositoryImpl) CreateAccount(ctx context.Context, account *model.PlatformAccount) error {
	return r.db.Create(account).Error
}

// UpdateAccount 更新平台账号
func (r *PlatformRepositoryImpl) UpdateAccount(ctx context.Context, account *model.PlatformAccount) error {
	return r.db.Save(account).Error
}

// DeleteAccount 删除平台账号
func (r *PlatformRepositoryImpl) DeleteAccount(ctx context.Context, id int64) error {
	return r.db.Delete(&model.PlatformAccount{}, id).Error
}

// CreatePlatform 创建平台
func (r *PlatformRepositoryImpl) CreatePlatform(platform *model.Platform) error {
	return r.db.Create(platform).Error
}

// UpdatePlatform 更新平台
func (r *PlatformRepositoryImpl) UpdatePlatform(platform *model.Platform) error {
	return r.db.Save(platform).Error
}

// GetPlatformByID 根据ID获取平台
func (r *PlatformRepositoryImpl) GetPlatformByID(id int64) (*model.Platform, error) {
	var platform model.Platform
	err := r.db.First(&platform, id).Error
	if err != nil {
		return nil, err
	}
	return &platform, nil
}

// ListPlatforms 获取平台列表
func (r *PlatformRepositoryImpl) ListPlatforms(req *model.PlatformListRequest) ([]model.Platform, int64, error) {
	var platforms []model.Platform
	var total int64

	query := r.db.Model(&model.Platform{})

	if req.Name != "" {
		query = query.Where("name LIKE ?", "%"+req.Name+"%")
	}
	if req.Code != "" {
		query = query.Where("code LIKE ?", "%"+req.Code+"%")
	}
	if req.Status != nil {
		query = query.Where("status = ?", *req.Status)
	}

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	offset := (req.Page - 1) * req.PageSize
	err = query.Offset(offset).Limit(req.PageSize).Find(&platforms).Error
	if err != nil {
		return nil, 0, err
	}

	return platforms, total, nil
}

// CreatePlatformAccount 创建平台账号
func (r *PlatformRepositoryImpl) CreatePlatformAccount(account *model.PlatformAccount) error {
	return r.db.Create(account).Error
}

// UpdatePlatformAccount 更新平台账号
func (r *PlatformRepositoryImpl) UpdatePlatformAccount(account *model.PlatformAccount) error {
	return r.db.Save(account).Error
}

// GetPlatformAccountByID 根据ID获取平台账号
func (r *PlatformRepositoryImpl) GetPlatformAccountByID(id int64) (*model.PlatformAccount, error) {
	var account model.PlatformAccount
	err := r.db.Preload("Platform").First(&account, id).Error
	if err != nil {
		return nil, err
	}
	return &account, nil
}

// ListPlatformAccounts 获取平台账号列表
func (r *PlatformRepositoryImpl) ListPlatformAccounts(req *model.PlatformAccountListRequest) (*model.PlatformAccountListResponse, error) {
	var accounts []model.PlatformAccount
	var total int64

	query := r.db.Model(&model.PlatformAccount{}).Preload("Platform")
	if req.PlatformID != nil {
		query = query.Where("platform_id = ?", *req.PlatformID)
	}
	if req.Status != nil {
		query = query.Where("status = ?", *req.Status)
	}

	err := query.Count(&total).Error
	if err != nil {
		return nil, err
	}

	offset := (req.Page - 1) * req.PageSize
	err = query.Offset(offset).Limit(req.PageSize).Find(&accounts).Error
	if err != nil {
		return nil, err
	}

	return &model.PlatformAccountListResponse{
		Total: total,
		Items: accounts,
	}, nil
}

// GetAccountsByPlatformID 根据平台ID获取账号列表
func (r *PlatformRepositoryImpl) GetAccountsByPlatformID(ctx context.Context, platformID int64) ([]*model.PlatformAccount, error) {
	var accounts []*model.PlatformAccount
	if err := r.db.Where("platform_id = ?", platformID).Find(&accounts).Error; err != nil {
		return nil, err
	}
	return accounts, nil
}

// GetAPIByID 根据API ID获取平台 API 信息
func (r *PlatformRepositoryImpl) GetAPIByID(ctx context.Context, apiID int64) (*model.PlatformAPI, error) {
	var api model.PlatformAPI
	err := r.db.WithContext(ctx).First(&api, apiID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNoAPIForProduct
		}
		return nil, err
	}
	return &api, nil
}

// GetPlatformByCode 根据平台代码获取平台信息
func (r *PlatformRepositoryImpl) GetPlatformByCode(ctx context.Context, code string) (*model.PlatformAPI, error) {
	var platform model.PlatformAPI
	if err := r.db.WithContext(ctx).Where("code = ? AND status = 1", code).First(&platform).Error; err != nil {
		return nil, err
	}
	return &platform, nil
}

// GetAPIParamByID 根据API参数ID获取API参数信息
func (r *PlatformRepositoryImpl) GetAPIParamByID(ctx context.Context, id int64) (*model.PlatformAPIParam, error) {
	var param model.PlatformAPIParam
	err := r.db.WithContext(ctx).First(&param, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNoAPIForProduct
		}
		return nil, err
	}
	return &param, nil
}

// UpdatePlatformAccountFields 更新平台账号字段
func (r *PlatformRepositoryImpl) UpdatePlatformAccountFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&model.PlatformAccount{}).Where("id = ?", id).Updates(fields).Error
}

// GetPlatformAccountByAccountName 通过账号名查找平台账号
func (r *PlatformRepositoryImpl) GetPlatformAccountByAccountName(accountName string) (*model.PlatformAccount, error) {
	var account model.PlatformAccount
	err := r.db.Where("account_name = ?", accountName).First(&account).Error
	if err != nil {
		return nil, err
	}
	return &account, nil
}

// GetDB 获取数据库连接
func (r *PlatformRepositoryImpl) GetDB() *gorm.DB {
	return r.db
}
