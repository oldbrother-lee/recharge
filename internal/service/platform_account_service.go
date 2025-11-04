package service

import (
	"context"
	"errors"
	"recharge-go/internal/model"
	"recharge-go/internal/repository"
)

type PlatformAccountService struct {
	repo *repository.PlatformAccountRepository
}

func NewPlatformAccountService(repo *repository.PlatformAccountRepository) *PlatformAccountService {
	return &PlatformAccountService{repo: repo}
}

// 绑定本地用户
func (s *PlatformAccountService) BindUser(platformAccountID int64, userID int64) error {
	// 校验账号是否存在
	account, err := s.repo.GetByID(platformAccountID)
	if err != nil {
		return errors.New("平台账号不存在")
	}
	// 可选：校验用户是否存在（如有 UserRepository 可加）
	if userID <= 0 {
		return errors.New("用户ID无效")
	}
	return s.repo.BindUser(account.ID, userID)
}

// 查询账号列表（带本地用户名）
func (s *PlatformAccountService) GetListWithUserName(req *model.PlatformAccountListRequest) (int64, []model.PlatformAccount, error) {
	return s.repo.GetListWithUserName(req)
}

// 拉单相关方法

// GetPullOrderAccounts 获取启用拉单的平台账号列表
func (s *PlatformAccountService) GetPullOrderAccounts(ctx context.Context) ([]model.PlatformAccount, error) {
	return s.repo.GetPullOrderAccounts(ctx)
}

// GetPullOrderAccountByID 获取拉单账号详情
func (s *PlatformAccountService) GetPullOrderAccountByID(ctx context.Context, id int64) (*model.PlatformAccount, error) {
	return s.repo.GetPullOrderAccountByID(ctx, id)
}

// UpdatePullOrderConfig 更新拉单配置
func (s *PlatformAccountService) UpdatePullOrderConfig(ctx context.Context, id int64, config *model.PlatformAccountUpdateRequest) error {
	// 验证账号是否存在
	account, err := s.repo.GetByIDWithContext(ctx, id)
	if err != nil {
		return err
	}
	if account == nil {
		return errors.New("平台账号不存在")
	}
	
	return s.repo.UpdatePullOrderConfig(ctx, id, config)
}
