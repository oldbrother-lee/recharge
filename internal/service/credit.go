package service

import (
	"context"
	"errors"
	"recharge-go/internal/model"
	"recharge-go/internal/repository"
	"time"
)

// CreditService 授信服务
type CreditService struct {
	userRepo      *repository.UserRepository
	creditLogRepo *repository.CreditLogRepository
}

// NewCreditService 创建授信服务
func NewCreditService(userRepo *repository.UserRepository, creditLogRepo *repository.CreditLogRepository) *CreditService {
	return &CreditService{
		userRepo:      userRepo,
		creditLogRepo: creditLogRepo,
	}
}

// SetCredit 设置授信额度
func (s *CreditService) SetCredit(ctx context.Context, req *model.CreditLogRequest) error {
	// 获取用户信息
	user, err := s.userRepo.GetByID(ctx, req.UserID)
	if err != nil {
		return err
	}

	// 如果 type 为空，设置为设置授信额度
	if req.Type == 0 {
		req.Type = model.CreditTypeSet
	}

	// 如果 operator 为空，从上下文中获取当前登录用户
	if req.Operator == "" {
		// 从上下文中获取当前登录用户
		if username, ok := ctx.Value("username").(string); ok {
			req.Operator = username
		} else {
			req.Operator = "system"
		}
	}

	// 创建授信日志
	log := &model.CreditLog{
		UserID:       req.UserID,
		Amount:       req.Amount,
		Type:         req.Type,
		CreditBefore: user.Credit,
		CreditAfter:  req.Amount,
		Remark:       req.Remark,
		Operator:     req.Operator,
		CreatedAt:    time.Now(),
	}

	// 开启事务
	tx := s.userRepo.DB().Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 更新用户授信额度
	if err := tx.Model(user).Update("credit", req.Amount).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 创建授信日志
	if err := s.creditLogRepo.Create(ctx, log); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

// UseCredit 使用授信额度
func (s *CreditService) UseCredit(ctx context.Context, userID int64, amount float64, orderID int64, remark string) error {
	// 获取用户信息
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	// 检查授信额度是否足够
	if user.Credit < amount {
		return errors.New("授信额度不足")
	}

	// 创建授信日志
	log := &model.CreditLog{
		UserID:       userID,
		Amount:       amount,
		Type:         model.CreditTypeUse,
		CreditBefore: user.Credit,
		CreditAfter:  user.Credit - amount,
		OrderID:      orderID,
		Remark:       remark,
		Operator:     "system",
		CreatedAt:    time.Now(),
	}

	// 开启事务
	tx := s.userRepo.DB().Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 更新用户授信额度
	if err := tx.Model(user).Update("credit", user.Credit-amount).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 创建授信日志
	if err := s.creditLogRepo.Create(ctx, log); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

// RestoreCredit 恢复授信额度
func (s *CreditService) RestoreCredit(ctx context.Context, userID int64, amount float64, orderID int64, remark string) error {
	// 获取用户信息
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	// 创建授信日志
	log := &model.CreditLog{
		UserID:       userID,
		Amount:       amount,
		Type:         model.CreditTypeRestore,
		CreditBefore: user.Credit,
		CreditAfter:  user.Credit + amount,
		OrderID:      orderID,
		Remark:       remark,
		Operator:     "system",
		CreatedAt:    time.Now(),
	}

	// 开启事务
	tx := s.userRepo.DB().Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 更新用户授信额度
	if err := tx.Model(user).Update("credit", user.Credit+amount).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 创建授信日志
	if err := s.creditLogRepo.Create(ctx, log); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

// GetCreditLogs 获取授信日志列表
func (s *CreditService) GetCreditLogs(ctx context.Context, req *model.CreditLogListRequest) (*model.CreditLogListResponse, error) {
	logs, total, err := s.creditLogRepo.List(ctx, req)
	if err != nil {
		return nil, err
	}

	var responses []model.CreditLogResponse
	for _, log := range logs {
		responses = append(responses, model.CreditLogResponse{
			ID:           log.ID,
			UserID:       log.UserID,
			Amount:       log.Amount,
			Type:         log.Type,
			CreditBefore: log.CreditBefore,
			CreditAfter:  log.CreditAfter,
			OrderID:      log.OrderID,
			Remark:       log.Remark,
			Operator:     log.Operator,
			CreatedAt:    log.CreatedAt,
		})
	}

	return &model.CreditLogListResponse{
		List:  responses,
		Total: total,
	}, nil
}

// GetUserCreditStats 获取用户授信统计
func (s *CreditService) GetUserCreditStats(ctx context.Context, userID int64) (float64, float64, error) {
	return s.creditLogRepo.GetUserCreditStats(ctx, userID)
}
