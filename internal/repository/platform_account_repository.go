package repository

import (
	"context"
	"errors"
	"recharge-go/internal/model"

	"gorm.io/gorm"
)

// PlatformAccountRepository 平台账号仓储
type PlatformAccountRepository struct {
	db *gorm.DB
}

func NewPlatformAccountRepository(db *gorm.DB) *PlatformAccountRepository {
	return &PlatformAccountRepository{db: db}
}

// 绑定本地用户
func (r *PlatformAccountRepository) BindUser(platformAccountID int64, userID int64) error {
	return r.db.Model(&model.PlatformAccount{}).
		Where("id = ?", platformAccountID).
		Update("bind_user_id", userID).Error
}

// 查询平台账号（带本地用户名）
func (r *PlatformAccountRepository) GetListWithUserName(req *model.PlatformAccountListRequest) (total int64, list []model.PlatformAccount, err error) {
	db := r.db.Model(&model.PlatformAccount{}).Where("deleted_at IS NULL")

	if req.PlatformID != nil {
		db = db.Where("platform_id = ?", *req.PlatformID)
	}
	if req.Status != nil {
		db = db.Where("status = ?", *req.Status)
	}

	// 统计总数
	err = db.Count(&total).Error
	if err != nil {
		return
	}

	// 分页查询并 join 用户表
	err = db.
		Select("platform_accounts.*, u.username as bind_user_name").
		Joins("LEFT JOIN users u ON platform_accounts.bind_user_id = u.id").
		Order("platform_accounts.id DESC").
		Limit(req.PageSize).
		Offset((req.Page - 1) * req.PageSize).
		Find(&list).Error
	return
}

// 查询单个账号
func (r *PlatformAccountRepository) GetByID(id int64) (*model.PlatformAccount, error) {
	var account model.PlatformAccount
	err := r.db.Preload("Platform").Where("id = ?", id).First(&account).Error
	if err != nil {
		return nil, err
	}
	return &account, nil
}

// GetByAccount 根据账号获取平台账号
func (r *PlatformAccountRepository) GetByAccount(ctx context.Context, account string) (*model.PlatformAccount, error) {
	var platformAccount model.PlatformAccount
	if err := r.db.Where("account = ?", account).First(&platformAccount).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &platformAccount, nil
}

// UpdateBalance 更新平台账号余额
func (r *PlatformAccountRepository) UpdateBalance(ctx context.Context, id int64, balance float64) error {
	return r.db.Model(&model.PlatformAccount{}).Where("id = ?", id).
		Update("balance", balance).Error
}

// Create 创建平台账号
func (r *PlatformAccountRepository) Create(ctx context.Context, account *model.PlatformAccount) error {
	return r.db.Create(account).Error
}

// List 获取平台账号列表
func (r *PlatformAccountRepository) List(ctx context.Context, offset, limit int) ([]*model.PlatformAccount, int64, error) {
	var accounts []*model.PlatformAccount
	var total int64

	if err := r.db.Model(&model.PlatformAccount{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := r.db.Offset(offset).Limit(limit).Find(&accounts).Error; err != nil {
		return nil, 0, err
	}

	return accounts, total, nil
}

// GetDB 获取数据库连接
func (r *PlatformAccountRepository) GetDB() *gorm.DB {
	return r.db
}
