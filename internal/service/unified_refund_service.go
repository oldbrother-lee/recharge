package service

import (
	"context"
	"errors"
	"recharge-go/internal/model"
	"recharge-go/internal/repository"
	"recharge-go/pkg/lock"
	"recharge-go/pkg/logger"

	"gorm.io/gorm"
)

// RefundType 退款类型
type RefundType int

const (
	// RefundTypeUser 用户余额退款
	RefundTypeUser RefundType = iota + 1
	// RefundTypePlatform 平台账号退款
	RefundTypePlatform
)

// RefundRequest 退款请求
type RefundRequest struct {
	UserID    int64       `json:"user_id"`    // 用户ID
	OrderID   int64       `json:"order_id"`   // 订单ID
	Amount    float64     `json:"amount"`     // 退款金额
	Remark    string      `json:"remark"`     // 退款备注
	Operator  string      `json:"operator"`   // 操作员
	Type      RefundType  `json:"type"`       // 退款类型
	AccountID *int64      `json:"account_id"` // 平台账号ID（平台退款时必填）
	Tx        *gorm.DB    `json:"-"`          // 事务（可选）
}

// RefundResponse 退款响应
type RefundResponse struct {
	Success       bool    `json:"success"`        // 是否成功
	Message       string  `json:"message"`        // 消息
	RefundAmount  float64 `json:"refund_amount"`  // 实际退款金额
	BalanceAfter  float64 `json:"balance_after"`  // 退款后余额
	AlreadyRefund bool    `json:"already_refund"` // 是否已退款（幂等性）
}

// UnifiedRefundService 统一退款服务
type UnifiedRefundService struct {
	db                            *gorm.DB
	balanceService                *BalanceService
	platformAccountBalanceService *PlatformAccountBalanceService
	lockManager                   *lock.RefundLockManager
	userRepo                      *repository.UserRepository
	orderRepo                     repository.OrderRepository
	balanceLogRepo                *repository.BalanceLogRepository
}

// NewUnifiedRefundService 创建统一退款服务
func NewUnifiedRefundService(
	db *gorm.DB,
	userRepo *repository.UserRepository,
	orderRepo repository.OrderRepository,
	balanceLogRepo *repository.BalanceLogRepository,
	lockManager *lock.RefundLockManager,
	balanceService *BalanceService,
	platformAccountBalanceService *PlatformAccountBalanceService,
) *UnifiedRefundService {
	return &UnifiedRefundService{
		db:                            db,
		userRepo:                      userRepo,
		orderRepo:                     orderRepo,
		balanceLogRepo:                balanceLogRepo,
		lockManager:                   lockManager,
		balanceService:                balanceService,
		platformAccountBalanceService: platformAccountBalanceService,
	}
}

// ProcessRefund 处理退款（统一入口）
func (s *UnifiedRefundService) ProcessRefund(ctx context.Context, req *RefundRequest) (*RefundResponse, error) {
	logger.Info("开始处理统一退款请求",
		"user_id", req.UserID,
		"order_id", req.OrderID,
		"amount", req.Amount,
		"type", req.Type,
		"account_id", req.AccountID)

	// 参数验证
	if err := s.validateRefundRequest(req); err != nil {
		return &RefundResponse{
			Success: false,
			Message: err.Error(),
		}, err
	}

	// 注意：分布式锁现在由调用方管理，确保锁的作用域覆盖整个业务流程
	// 根据退款类型调用相应的退款方法
	switch req.Type {
	case RefundTypeUser:
		return s.processUserRefund(ctx, req)
	case RefundTypePlatform:
		return s.processPlatformRefund(ctx, req)
	default:
		return &RefundResponse{
			Success: false,
			Message: "不支持的退款类型",
		}, errors.New("不支持的退款类型")
	}
}

// validateRefundRequest 验证退款请求
func (s *UnifiedRefundService) validateRefundRequest(req *RefundRequest) error {
	if req.UserID <= 0 {
		return errors.New("用户ID不能为空")
	}
	if req.OrderID <= 0 {
		return errors.New("订单ID不能为空")
	}
	if req.Amount <= 0 {
		return errors.New("退款金额必须大于0")
	}
	if req.Type == RefundTypePlatform && (req.AccountID == nil || *req.AccountID <= 0) {
		return errors.New("平台退款时账号ID不能为空")
	}
	return nil
}

// processUserRefund 处理用户余额退款
func (s *UnifiedRefundService) processUserRefund(ctx context.Context, req *RefundRequest) (*RefundResponse, error) {
	logger.Info("处理用户余额退款", "user_id", req.UserID, "order_id", req.OrderID, "amount", req.Amount)

	// 检查是否已退款（幂等性）
	alreadyRefund, err := s.checkAlreadyRefund(ctx, req.UserID, req.OrderID, model.BalanceStyleRefund)
	if err != nil {
		return &RefundResponse{
			Success: false,
			Message: "检查退款状态失败",
		}, err
	}
	if alreadyRefund {
		logger.Info("订单已退款，跳过重复操作", "user_id", req.UserID, "order_id", req.OrderID)
		return &RefundResponse{
			Success:       true,
			Message:       "订单已退款",
			AlreadyRefund: true,
		}, nil
	}

	// 执行退款
	var refundErr error
	if req.Tx != nil {
		// 使用传入的事务
		refundErr = s.balanceService.RefundWithTx(ctx, req.Tx, req.UserID, req.Amount, req.OrderID, req.Remark, req.Operator)
	} else {
		// 创建新事务
		refundErr = s.balanceService.Refund(ctx, req.UserID, req.Amount, req.OrderID, req.Remark, req.Operator)
	}

	if refundErr != nil {
		logger.Error("用户余额退款失败", "user_id", req.UserID, "order_id", req.OrderID, "error", refundErr)
		return &RefundResponse{
			Success: false,
			Message: "退款失败: " + refundErr.Error(),
		}, refundErr
	}

	// 获取退款后余额 - 在事务内查询确保数据一致性
	var balanceAfter float64
	if req.Tx != nil {
		// 使用传入的事务查询最新余额
		var user model.User
		if err := req.Tx.Where("id = ?", req.UserID).First(&user).Error; err != nil {
			logger.Error("在事务内获取用户信息失败", "user_id", req.UserID, "error", err)
			// 退款成功但获取余额失败，不影响退款结果
		} else {
			balanceAfter = user.Balance
		}
	} else {
		// 没有事务时使用普通查询
		user, err := s.userRepo.GetByID(ctx, req.UserID)
		if err != nil {
			logger.Error("获取用户信息失败", "user_id", req.UserID, "error", err)
			// 退款成功但获取余额失败，不影响退款结果
		} else {
			balanceAfter = user.Balance
		}
	}

	logger.Info("用户余额退款成功", "user_id", req.UserID, "order_id", req.OrderID, "amount", req.Amount, "balance_after", balanceAfter)
	return &RefundResponse{
		Success:      true,
		Message:      "退款成功",
		RefundAmount: req.Amount,
		BalanceAfter: balanceAfter,
	}, nil
}

// processPlatformRefund 处理平台账号退款
func (s *UnifiedRefundService) processPlatformRefund(ctx context.Context, req *RefundRequest) (*RefundResponse, error) {
	logger.Info("处理平台账号退款", "user_id", req.UserID, "order_id", req.OrderID, "amount", req.Amount, "account_id", *req.AccountID)

	// 检查必要的服务是否已初始化
	if s.platformAccountBalanceService == nil {
		logger.Error("platformAccountBalanceService 未初始化")
		return &RefundResponse{
			Success: false,
			Message: "平台账户余额服务未初始化",
		}, errors.New("平台账户余额服务未初始化")
	}

	// 检查是否已退款（幂等性）
	alreadyRefund, err := s.checkAlreadyRefund(ctx, req.UserID, req.OrderID, model.BalanceStyleRefund)
	if err != nil {
		return &RefundResponse{
			Success: false,
			Message: "检查退款状态失败",
		}, err
	}
	if alreadyRefund {
		logger.Info("订单已退款，跳过重复操作", "user_id", req.UserID, "order_id", req.OrderID)
		return &RefundResponse{
			Success:       true,
			Message:       "订单已退款",
			AlreadyRefund: true,
		}, nil
	}

	// 执行退款
	var refundErr error
	// 注意：RefundBalance方法现在不再接受事务参数，内部自己管理事务
	refundErr = s.platformAccountBalanceService.RefundBalance(ctx, req.UserID, req.Amount, req.OrderID, req.Remark)

	if refundErr != nil {
		logger.Error("平台账号退款失败", "user_id", req.UserID, "order_id", req.OrderID, "account_id", *req.AccountID, "error", refundErr)
		return &RefundResponse{
			Success: false,
			Message: "退款失败: " + refundErr.Error(),
		}, refundErr
	}

	// 获取退款后余额 - 在事务内查询确保数据一致性
	var balanceAfter float64
	if req.Tx != nil {
		// 使用传入的事务查询最新余额
		var user model.User
		if err := req.Tx.Where("id = ?", req.UserID).First(&user).Error; err != nil {
			logger.Error("在事务内获取用户信息失败", "user_id", req.UserID, "error", err)
			// 退款成功但获取余额失败，不影响退款结果
		} else {
			balanceAfter = user.Balance
		}
	} else {
		// 没有事务时使用普通查询
		user, err := s.userRepo.GetByID(ctx, req.UserID)
		if err != nil {
			logger.Error("获取用户信息失败", "user_id", req.UserID, "error", err)
			// 退款成功但获取余额失败，不影响退款结果
		} else {
			balanceAfter = user.Balance
		}
	}

	logger.Info("平台账号退款成功", "user_id", req.UserID, "order_id", req.OrderID, "amount", req.Amount, "account_id", *req.AccountID, "balance_after", balanceAfter)
	return &RefundResponse{
		Success:      true,
		Message:      "退款成功",
		RefundAmount: req.Amount,
		BalanceAfter: balanceAfter,
	}, nil
}

// checkAlreadyRefund 检查是否已退款
func (s *UnifiedRefundService) checkAlreadyRefund(ctx context.Context, userID, orderID int64, style int) (bool, error) {
	var count int64
	err := s.db.Model(&model.BalanceLog{}).
		Where("user_id = ? AND order_id = ? AND style = ?", userID, orderID, style).
		Count(&count).Error
	if err != nil {
		logger.Error("检查退款记录失败", "user_id", userID, "order_id", orderID, "error", err)
		return false, err
	}
	return count > 0, nil
}

// BatchRefund 批量退款
func (s *UnifiedRefundService) BatchRefund(ctx context.Context, requests []*RefundRequest) ([]*RefundResponse, error) {
	logger.Info("开始批量退款", "count", len(requests))
	
	responses := make([]*RefundResponse, len(requests))
	
	for i, req := range requests {
		resp, err := s.ProcessRefund(ctx, req)
		if err != nil {
			logger.Error("批量退款中单个退款失败", "index", i, "user_id", req.UserID, "order_id", req.OrderID, "error", err)
		}
		responses[i] = resp
	}
	
	logger.Info("批量退款完成", "count", len(requests))
	return responses, nil
}

// GetRefundHistory 获取退款历史
func (s *UnifiedRefundService) GetRefundHistory(ctx context.Context, userID int64, limit, offset int) ([]*model.BalanceLog, error) {
	var logs []*model.BalanceLog
	err := s.db.Where("user_id = ? AND style = ?", userID, model.BalanceStyleRefund).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&logs).Error
	if err != nil {
		logger.Error("获取退款历史失败", "user_id", userID, "error", err)
		return nil, err
	}
	return logs, nil
}