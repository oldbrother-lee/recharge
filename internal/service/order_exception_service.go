package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"recharge-go/internal/model"
	"recharge-go/internal/repository"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// OrderExceptionService 订单异常服务接口
type OrderExceptionService interface {
	// CreateBalanceVerificationException 创建余额验证异常记录
	CreateBalanceVerificationException(ctx context.Context, tx *gorm.DB, order *model.Order, exceptionData *model.BalanceVerificationExceptionData) error
	// GetByID 根据ID获取订单异常记录
	GetByID(ctx context.Context, id int64) (*model.OrderException, error)
	// GetByOrderID 根据订单ID获取异常记录列表
	GetByOrderID(ctx context.Context, orderID int64) ([]model.OrderException, error)
	// List 获取订单异常列表
	List(ctx context.Context, req *model.OrderExceptionListRequest) (*model.OrderExceptionListResponse, error)
	// UpdateStatus 更新异常状态
	UpdateStatus(ctx context.Context, id int64, req *model.UpdateOrderExceptionRequest, operatorID int64) error
	// GetPendingCount 获取待处理异常数量
	GetPendingCount(ctx context.Context) (int64, error)
	// GetStatistics 获取异常统计信息
	GetStatistics(ctx context.Context, startDate, endDate time.Time) (map[string]int64, error)
}

type orderExceptionService struct {
	exceptionRepo repository.OrderExceptionRepository
	orderRepo     repository.OrderRepository
	logger        *zap.Logger
}

// NewOrderExceptionService 创建订单异常服务实例
func NewOrderExceptionService(
	exceptionRepo repository.OrderExceptionRepository,
	orderRepo repository.OrderRepository,
	logger *zap.Logger,
) OrderExceptionService {
	return &orderExceptionService{
		exceptionRepo: exceptionRepo,
		orderRepo:     orderRepo,
		logger:        logger,
	}
}

// CreateBalanceVerificationException 创建余额验证异常记录
func (s *orderExceptionService) CreateBalanceVerificationException(ctx context.Context, tx *gorm.DB, order *model.Order, exceptionData *model.BalanceVerificationExceptionData) error {
	// 序列化异常数据
	exceptionDataJSON, err := json.Marshal(exceptionData)
	if err != nil {
		s.logger.Error("序列化异常数据失败", zap.Error(err), zap.Int64("order_id", order.ID))
		return fmt.Errorf("序列化异常数据失败: %v", err)
	}

	// 构建异常原因描述
	exceptionReason := fmt.Sprintf("余额验证失败：充值前余额 %s，充值后余额 %s，预期差额 %.2f，实际差额 %.2f",
		exceptionData.PreBalance,
		exceptionData.PostBalance,
		exceptionData.ExpectedDiff,
		exceptionData.ActualDiff,
	)

	// 创建异常记录
	exception := &model.OrderException{
		OrderID:         order.ID,
		OrderNumber:     order.OrderNumber,
		ExceptionType:   model.ExceptionTypeBalanceVerificationFailed,
		ExceptionReason: exceptionReason,
		ExceptionData:   exceptionDataJSON,
		Status:          model.ExceptionStatusPending,
	}

	// 在事务中创建异常记录
	var createErr error
	if tx != nil {
		createErr = tx.WithContext(ctx).Create(exception).Error
	} else {
		createErr = s.exceptionRepo.Create(ctx, exception)
	}

	if createErr != nil {
		s.logger.Error("创建订单异常记录失败", zap.Error(createErr), zap.Int64("order_id", order.ID))
		return fmt.Errorf("创建订单异常记录失败: %v", createErr)
	}

	// 更新订单的异常标记
	updateOrderErr := s.updateOrderHasException(ctx, order.ID, true, tx)
	if updateOrderErr != nil {
		s.logger.Error("更新订单异常标记失败", zap.Error(updateOrderErr), zap.Int64("order_id", order.ID))
		// 这里不返回错误，因为异常记录已经创建成功
	}

	s.logger.Info("创建余额验证异常记录成功",
		zap.Int64("order_id", order.ID),
		zap.String("order_number", order.OrderNumber),
		zap.Int64("exception_id", exception.ID),
	)

	return nil
}

// updateOrderHasException 更新订单的异常标记
func (s *orderExceptionService) updateOrderHasException(ctx context.Context, orderID int64, hasException bool, tx *gorm.DB) error {
	if tx != nil {
		return tx.WithContext(ctx).Model(&model.Order{}).Where("id = ?", orderID).Update("has_exception", hasException).Error
	}
	return s.orderRepo.UpdateHasException(ctx, orderID, hasException)
}

// GetByID 根据ID获取订单异常记录
func (s *orderExceptionService) GetByID(ctx context.Context, id int64) (*model.OrderException, error) {
	return s.exceptionRepo.GetByID(ctx, id)
}

// GetByOrderID 根据订单ID获取异常记录列表
func (s *orderExceptionService) GetByOrderID(ctx context.Context, orderID int64) ([]model.OrderException, error) {
	return s.exceptionRepo.GetByOrderID(ctx, orderID)
}

// List 获取订单异常列表
func (s *orderExceptionService) List(ctx context.Context, req *model.OrderExceptionListRequest) (*model.OrderExceptionListResponse, error) {
	exceptions, total, err := s.exceptionRepo.List(ctx, req)
	if err != nil {
		return nil, err
	}

	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}

	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	return &model.OrderExceptionListResponse{
		List:       exceptions,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// UpdateStatus 更新异常状态
func (s *orderExceptionService) UpdateStatus(ctx context.Context, id int64, req *model.UpdateOrderExceptionRequest, operatorID int64) error {
	// 获取异常记录
	exception, err := s.exceptionRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("获取异常记录失败: %v", err)
	}

	if exception == nil {
		return fmt.Errorf("异常记录不存在")
	}

	// 更新状态
	err = s.exceptionRepo.UpdateStatus(ctx, id, req.Status, &operatorID, req.ResolvedNote)
	if err != nil {
		return fmt.Errorf("更新异常状态失败: %v", err)
	}

	s.logger.Info("更新订单异常状态成功",
		zap.Int64("exception_id", id),
		zap.Int64("order_id", exception.OrderID),
		zap.String("old_status", exception.Status),
		zap.String("new_status", req.Status),
		zap.Int64("operator_id", operatorID),
	)

	return nil
}

// GetPendingCount 获取待处理异常数量
func (s *orderExceptionService) GetPendingCount(ctx context.Context) (int64, error) {
	return s.exceptionRepo.GetPendingCount(ctx)
}

// GetStatistics 获取异常统计信息
func (s *orderExceptionService) GetStatistics(ctx context.Context, startDate, endDate time.Time) (map[string]int64, error) {
	return s.exceptionRepo.GetStatistics(ctx, startDate, endDate)
}