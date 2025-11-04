package service

import (
	"context"
	"errors"
	"recharge-go/internal/model"
	"recharge-go/internal/repository"
)

type PlatformAccountVariantService struct {
	repo        repository.PlatformAccountVariantRepository
	accountRepo *repository.PlatformAccountRepository
}

func NewPlatformAccountVariantService(
	repo repository.PlatformAccountVariantRepository,
	accountRepo *repository.PlatformAccountRepository,
) *PlatformAccountVariantService {
	return &PlatformAccountVariantService{
		repo:        repo,
		accountRepo: accountRepo,
	}
}

// Create 创建变体
func (s *PlatformAccountVariantService) Create(ctx context.Context, variant *model.PlatformAccountVariant) error {
	// 验证平台账号是否存在
	account, err := s.accountRepo.GetByIDWithContext(ctx, variant.PlatformAccountID)
	if err != nil {
		return err
	}
	if account == nil {
		return errors.New("平台账号不存在")
	}
	
	// 检查是否已存在相同的变体
	existing, err := s.repo.GetByISPAndFaceValue(ctx, variant.PlatformAccountID, variant.ISP, variant.FaceValue)
	if err != nil {
		return err
	}
	if existing != nil {
		return errors.New("该运营商和面值的变体已存在")
	}
	
	return s.repo.Create(ctx, variant)
}

// Update 更新变体
func (s *PlatformAccountVariantService) Update(ctx context.Context, variant *model.PlatformAccountVariant) error {
	// 验证变体是否存在
	existing, err := s.repo.GetByID(ctx, variant.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return errors.New("变体不存在")
	}
	
	// 如果修改了运营商或面值，检查是否与其他变体冲突
	if existing.ISP != variant.ISP || existing.FaceValue != variant.FaceValue {
		conflicting, err := s.repo.GetByISPAndFaceValue(ctx, variant.PlatformAccountID, variant.ISP, variant.FaceValue)
		if err != nil {
			return err
		}
		if conflicting != nil && conflicting.ID != variant.ID {
			return errors.New("该运营商和面值的变体已存在")
		}
	}
	
	return s.repo.Update(ctx, variant)
}

// Delete 删除变体
func (s *PlatformAccountVariantService) Delete(ctx context.Context, id int64) error {
	// 验证变体是否存在
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return errors.New("变体不存在")
	}
	
	return s.repo.Delete(ctx, id)
}

// GetByID 获取变体详情
func (s *PlatformAccountVariantService) GetByID(ctx context.Context, id int64) (*model.PlatformAccountVariant, error) {
	return s.repo.GetByID(ctx, id)
}

// GetByPlatformAccountID 根据平台账号ID获取变体列表
func (s *PlatformAccountVariantService) GetByPlatformAccountID(ctx context.Context, platformAccountID int64) ([]model.PlatformAccountVariant, error) {
	return s.repo.GetByPlatformAccountID(ctx, platformAccountID)
}

// GetEnabledVariants 获取启用的变体列表
func (s *PlatformAccountVariantService) GetEnabledVariants(ctx context.Context, platformAccountID int64) ([]model.PlatformAccountVariant, error) {
	return s.repo.GetEnabledVariants(ctx, platformAccountID)
}

// List 获取变体列表
func (s *PlatformAccountVariantService) List(ctx context.Context, platformAccountID *int64, isp *int, enabled *bool, offset, limit int) ([]model.PlatformAccountVariant, int64, error) {
	return s.repo.List(ctx, platformAccountID, isp, enabled, offset, limit)
}

// UpdateCursor 更新拉取游标
func (s *PlatformAccountVariantService) UpdateCursor(ctx context.Context, variantID int64, cursor string) error {
	return s.repo.UpdateCursor(ctx, variantID, cursor)
}

// UpdateLastPullAt 更新最后拉取时间
func (s *PlatformAccountVariantService) UpdateLastPullAt(ctx context.Context, variantID int64) error {
	return s.repo.UpdateLastPullAt(ctx, variantID)
}

// IncrementFailCount 增加失败计数
func (s *PlatformAccountVariantService) IncrementFailCount(ctx context.Context, variantID int64) error {
	return s.repo.IncrementFailCount(ctx, variantID)
}

// ResetFailCount 重置失败计数
func (s *PlatformAccountVariantService) ResetFailCount(ctx context.Context, variantID int64) error {
	return s.repo.ResetFailCount(ctx, variantID)
}