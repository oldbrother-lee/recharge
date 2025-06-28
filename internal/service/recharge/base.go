package recharge

import (
	"context"
	"encoding/json"
	"fmt"
	"recharge-go/internal/model"
	"recharge-go/internal/repository"
	"recharge-go/pkg/logger"
)

// rechargeService 充值服务实现
type rechargeService struct {
	platformAccountRepo repository.PlatformAccountRepository
	platformRepo        repository.PlatformRepository
	balanceService      interface { // 只用接口，避免循环依赖
		DeductBalance(ctx context.Context, accountID int64, amount float64, orderID int64, remark string) error
		RefundBalance(ctx context.Context, tx interface{}, accountID int64, amount float64, orderID int64, remark string) error
	}
	orderRepo repository.OrderRepository
	manager   *Manager
}

// NewRechargeService 创建充值服务实例
func NewRechargeService(
	platformAccountRepo repository.PlatformAccountRepository,
	platformRepo repository.PlatformRepository,
	balanceService interface {
		DeductBalance(ctx context.Context, accountID int64, amount float64, orderID int64, remark string) error
		RefundBalance(ctx context.Context, tx interface{}, accountID int64, amount float64, orderID int64, remark string) error
	},
	orderRepo repository.OrderRepository,
	manager *Manager,
) *rechargeService {
	return &rechargeService{
		platformAccountRepo: platformAccountRepo,
		platformRepo:        platformRepo,
		balanceService:      balanceService,
		orderRepo:           orderRepo,
		manager:             manager,
	}
}

// ProcessRechargeTask 处理充值任务
func (s *rechargeService) ProcessRechargeTask(ctx context.Context, order *model.Order) error {
	logger.Info("开始处理充值任务！！！",
		"order_id", order.ID,
		"order_number", order.OrderNumber,
		"amount", order.Price)
	// 1. 检查平台账号余额
	account, err := s.platformAccountRepo.GetByID(order.PlatformAccountID)
	if err != nil {
		logger.Error("获取平台账号失败",
			"error", err,
			"account_id", order.PlatformAccountID)
		return err
	}
	logger.Info("获取平台账号成功",
		"account_id", order.PlatformAccountID,
		"current_balance", account.Balance)

	// 2. 扣除平台账号余额
	if err := s.balanceService.DeductBalance(ctx, order.PlatformAccountID, order.Price, order.ID, "订单充值扣除"); err != nil {
		logger.Error("扣除平台账号余额失败",
			"error", err,
			"account_id", order.PlatformAccountID,
			"amount", order.Price)
		return err
	}
	logger.Info("扣除平台账号余额成功",
		"account_id", order.PlatformAccountID,
		"amount", order.Price)

	// 3. 获取平台API信息
	api, apiParam, err := s.getPlatformAPIByOrderID(ctx, order)
	if err != nil {
		logger.Error("获取平台API信息失败",
			"error", err,
			"order_id", order.ID)
		// 如果获取API信息失败，退还余额
		if refundErr := s.balanceService.RefundBalance(ctx, nil, order.PlatformAccountID, order.Price, order.ID, "获取API信息失败退还"); refundErr != nil {
			logger.Error("退还余额失败",
				"error", refundErr,
				"account_id", order.PlatformAccountID,
				"amount", order.Price)
		}
		return err
	}
	logger.Info("获取平台API信息成功",
		"order_id", order.ID,
		"api_name", api.Name)

	// 4. 提交订单到平台
	if err := s.manager.SubmitOrder(ctx, order, api, apiParam); err != nil {
		logger.Error("提交订单到平台失败",
			"error", err,
			"order_id", order.ID)
		// 5. 如果提交失败，退还余额
		if refundErr := s.balanceService.RefundBalance(ctx, nil, order.PlatformAccountID, order.Price, order.ID, "订单提交失败退还"); refundErr != nil {
			logger.Error("退还余额失败",
				"error", refundErr,
				"account_id", order.PlatformAccountID,
				"amount", order.Price)
		}
		return err
	}
	logger.Info("提交订单到平台成功",
		"order_id", order.ID)

	return nil
}

// getPlatformAPIByOrderID 根据订单获取平台API信息
func (s *rechargeService) getPlatformAPIByOrderID(ctx context.Context, order *model.Order) (*model.PlatformAPI, *model.PlatformAPIParam, error) {
	// 这里假设 order 里有 API ID 和 API Param ID 字段
	api, err := s.platformRepo.GetAPIByID(ctx, order.APICurID)
	if err != nil {
		return nil, nil, fmt.Errorf("获取平台API信息失败: %v", err)
	}
	apiParam, err := s.platformRepo.GetAPIParamByID(ctx, order.APICurParamID)
	if err != nil {
		return nil, nil, fmt.Errorf("获取API参数信息失败: %v", err)
	}
	return api, apiParam, nil
}

// HandleCallback 处理回调
func (s *rechargeService) HandleCallback(ctx context.Context, platformName string, data []byte) error {
	// 解析回调数据
	var callbackData struct {
		OrderID string `json:"order_id"`
		Status  string `json:"status"`
	}
	if err := json.Unmarshal(data, &callbackData); err != nil {
		return fmt.Errorf("解析回调数据失败: %v", err)
	}

	// 获取订单信息
	order, err := s.orderRepo.GetByOrderNumber(ctx, callbackData.OrderID)
	if err != nil {
		return fmt.Errorf("获取订单信息失败: %v", err)
	}

	// 如果充值失败，退还余额
	if callbackData.Status == "failed" {
		if err := s.balanceService.RefundBalance(ctx, nil, order.PlatformAccountID, order.Price, order.ID, "充值失败退还"); err != nil {
			logger.Error("退还余额失败",
				"error", err,
				"account_id", order.PlatformAccountID,
				"amount", order.Price)
			// 这里不返回错误，因为回调处理不应该因为余额退还失败而失败
		}
	}

	return nil
}

// ... existing code ...
