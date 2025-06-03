package service

import (
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
