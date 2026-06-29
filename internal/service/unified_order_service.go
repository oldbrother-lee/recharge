package service

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
	"recharge-go/internal/model"
	"recharge-go/internal/repository"
	notificationRepo "recharge-go/internal/repository/notification"
	"recharge-go/pkg/lock"
	logger "recharge-go/pkg/log"
	"recharge-go/pkg/queue"
)

// UnifiedOrderService 统一订单处理服务
// 提供统一的订单状态更新、余额查询、退款等功能
// 供外部回调和平台回调共同使用
type UnifiedOrderService struct {
	orderRepo              repository.OrderRepository
	balanceQueryRecordRepo repository.BalanceQueryRecordRepository
	phoneQueryService      PhoneQueryService
	balanceService         *BalanceService
	notificationRepo       notificationRepo.Repository
	queue                  queue.Queue
	db                     *gorm.DB
	logger                 *zap.Logger
	systemConfigService    *SystemConfigService
	productRepo            repository.ProductRepository
	retryService           *RetryService
	orderExceptionService  OrderExceptionService
	notificationHelper     *NotificationHelper
	lockManager            *lock.RefundLockManager
}

// NewUnifiedOrderService 创建统一订单处理服务
func NewUnifiedOrderService(
	orderRepo repository.OrderRepository,
	balanceQueryRecordRepo repository.BalanceQueryRecordRepository,
	phoneQueryService PhoneQueryService,
	balanceService *BalanceService,
	notificationRepo notificationRepo.Repository,
	queue queue.Queue,
	db *gorm.DB,
	logger *zap.Logger,
	systemConfigService *SystemConfigService,
	productRepo repository.ProductRepository,
	retryService *RetryService,
	orderExceptionService OrderExceptionService,
	lockManager *lock.RefundLockManager,
) *UnifiedOrderService {
	return &UnifiedOrderService{
		orderRepo:              orderRepo,
		balanceQueryRecordRepo: balanceQueryRecordRepo,
		phoneQueryService:      phoneQueryService,
		balanceService:         balanceService,
		notificationRepo:       notificationRepo,
		queue:                  queue,
		db:                     db,
		logger:                 logger,
		systemConfigService:    systemConfigService,
		productRepo:            productRepo,
		retryService:           retryService,
		orderExceptionService:  orderExceptionService,
		notificationHelper:     NewNotificationHelper(db, notificationRepo, queue),
		lockManager:            lockManager,
	}
}

// OrderStatusUpdateRequest 订单状态更新请求
type OrderStatusUpdateRequest struct {
	OrderID          int64             `json:"order_id"`
	NewStatus        model.OrderStatus `json:"new_status"`
	CallbackSource   string            `json:"callback_source"` // "external" 或 "platform"
	Remark           string            `json:"remark,omitempty"`
	NeedBalanceCheck bool              `json:"need_balance_check"` // 是否需要余额验证
}

// OrderStatusUpdateResponse 订单状态更新响应
type OrderStatusUpdateResponse struct {
	Success         bool   `json:"success"`
	Message         string `json:"message"`
	BalanceChanged  *bool  `json:"balance_changed,omitempty"` // 余额是否有变化（仅在余额验证时返回）
	RefundTriggered bool   `json:"refund_triggered"`          // 是否触发了退款
}

// UpdateOrderStatusUnified 统一的订单状态更新方法
// 支持外部回调和平台回调的统一处理
func (s *UnifiedOrderService) UpdateOrderStatusUnified(ctx context.Context, req *OrderStatusUpdateRequest) (*OrderStatusUpdateResponse, error) {
	lg := logger.WithContextCategory(ctx, "order")
	lg.Info("开始统一订单状态更新",
		logger.Int64V2("order_id", req.OrderID),
		logger.StringV2("new_status", string(req.NewStatus)),
		logger.StringV2("callback_source", req.CallbackSource),
		zap.Bool("need_balance_check", req.NeedBalanceCheck),
	)

	// 失败状态：先只读判定是否还有可重试通道（无副作用，重试任务在事务提交后再推送，避免与重试worker竞态）
	var hasAvailableChannel bool
	if req.NewStatus == model.OrderStatusFailed {
		var snapshot model.Order
		if e := s.db.WithContext(ctx).Where("id = ?", req.OrderID).First(&snapshot).Error; e != nil {
			lg.Error("预读订单信息失败", logger.Int64V2("order_id", req.OrderID), logger.ErrorV2(e))
			return nil, fmt.Errorf("获取订单信息失败: %v", e)
		}
		// 已是失败终态的不再寻找通道，统一按最终失败处理（确保退款补齐）
		if snapshot.Status != model.OrderStatusFailed {
			has, e := s.hasAvailableRetryChannel(ctx, &snapshot)
			if e != nil {
				lg.Error("检查可用重试通道失败，按无通道处理以走退款兜底", logger.Int64V2("order_id", req.OrderID), logger.ErrorV2(e))
				has = false
			}
			hasAvailableChannel = has
		}
	}

	// 最终失败（失败且无可重试通道）：状态置失败与退款必须在同一事务内原子提交
	finalFailure := req.NewStatus == model.OrderStatusFailed && !hasAvailableChannel

	// 最终失败场景获取订单级分布式锁，与行锁共同防止并发重复退款
	if finalFailure && s.lockManager != nil {
		orderLockValue, lockErr := s.lockManager.LockOrderRefund(ctx, req.OrderID)
		if lockErr != nil {
			// 不静默：拿不到锁直接报错，由调用方重新触发，避免漏退
			lg.Error("获取订单退款锁失败", logger.Int64V2("order_id", req.OrderID), logger.ErrorV2(lockErr))
			return nil, fmt.Errorf("获取订单退款锁失败: %v", lockErr)
		}
		defer func() {
			if unlockErr := s.lockManager.UnlockOrderRefund(ctx, req.OrderID, orderLockValue); unlockErr != nil {
				lg.Error("释放订单退款锁失败", logger.Int64V2("order_id", req.OrderID), logger.ErrorV2(unlockErr))
			}
		}()
	}

	// 使用事务确保订单状态更新（及最终失败退款）的原子性
	var result *OrderStatusUpdateResponse
	var order model.Order
	var refundDone bool
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. 获取订单信息（使用行锁）
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("id = ?", req.OrderID).First(&order).Error; err != nil {
			lg.Error("获取订单信息失败", logger.Int64V2("order_id", req.OrderID), logger.ErrorV2(err))
			return fmt.Errorf("获取订单信息失败: %v", err)
		}

		// 2. 检查订单状态是否需要更新
		if order.Status == req.NewStatus {
			lg.Info("订单状态未发生变化",
				logger.Int64V2("order_id", req.OrderID),
				logger.StringV2("status", string(req.NewStatus)),
			)
			result = &OrderStatusUpdateResponse{
				Success: true,
				Message: "订单状态未发生变化",
			}
			// 兜底：订单已是失败终态但可能此前漏退，这里幂等补退（已退则自动跳过）
			if finalFailure {
				if rerr := s.refundFailedOrderWithTx(ctx, tx, &order); rerr != nil {
					return fmt.Errorf("退款失败: %v", rerr)
				}
				refundDone = true
			}
			return nil
		}

		// 3. 更新订单状态
		if err := tx.Model(&model.Order{}).Where("id = ?", req.OrderID).Update("status", req.NewStatus).Error; err != nil {
			lg.Error("更新订单状态失败", logger.Int64V2("order_id", req.OrderID), logger.ErrorV2(err))
			return fmt.Errorf("更新订单状态失败: %v", err)
		}

		lg.Info("订单状态更新成功",
			logger.Int64V2("order_id", req.OrderID),
			logger.StringV2("old_status", string(order.Status)),
			logger.StringV2("new_status", string(req.NewStatus)),
		)

		// 最终失败：在同一事务内执行幂等退款，保证“失败+退款”原子；
		// 退款失败将回滚状态更新，订单不会停留在“失败未退款”的中间态
		if finalFailure {
			if rerr := s.refundFailedOrderWithTx(ctx, tx, &order); rerr != nil {
				return fmt.Errorf("退款失败: %v", rerr)
			}
			refundDone = true
		}

		result = &OrderStatusUpdateResponse{
			Success: true,
			Message: "订单状态更新成功",
		}
		return nil
	})

	if err != nil {
		lg.Error("订单状态更新事务失败（状态与退款已一并回滚）", logger.Int64V2("order_id", req.OrderID), logger.ErrorV2(err))
		return nil, err
	}

	if refundDone {
		result.RefundTriggered = true
		result.Message += ", 退款成功"
	}

	// 4. 在事务外执行余额验证（避免长时间持有数据库锁）
	if req.NewStatus == model.OrderStatusSuccess && req.NeedBalanceCheck && s.phoneQueryService != nil && order.Mobile != "" {
		balanceCheckResult, err := s.performBalanceCheck(ctx, &order)
		if err != nil {
			lg.Error("余额验证失败", logger.Int64V2("order_id", req.OrderID), logger.ErrorV2(err))
			// 余额验证失败不影响订单状态更新的成功
			result.Message = fmt.Sprintf("%s (余额验证失败: %v)", result.Message, err)
		} else {
			result.BalanceChanged = &balanceCheckResult.BalanceChanged
			result.RefundTriggered = balanceCheckResult.RefundTriggered
			result.Message = balanceCheckResult.Message
		}
	}

	// 5. 有可用通道：事务提交后再推送重试任务（此时订单状态已落库，避免与重试worker竞态）
	//    推送失败则兜底转入最终失败退款，绝不留下“失败未退款”
	if req.NewStatus == model.OrderStatusFailed && hasAvailableChannel {
		if perr := s.pushRetryTaskToQueue(ctx, req.OrderID, 2, "统一订单服务失败，切换通道重试"); perr != nil {
			lg.Error("推送重试任务失败，转最终失败兜底退款", logger.Int64V2("order_id", req.OrderID), logger.ErrorV2(perr))
			if rerr := s.ensureFailedOrderRefunded(ctx, req.OrderID); rerr != nil {
				lg.Error("兜底退款失败", logger.Int64V2("order_id", req.OrderID), logger.ErrorV2(rerr))
				return nil, rerr
			}
			hasAvailableChannel = false
			result.RefundTriggered = true
			result.Message += ", 退款成功"
		} else {
			lg.Info("订单已推送重试任务，跳过退款与失败通知", logger.Int64V2("order_id", req.OrderID))
			result.Message += ", 已推送重试任务"
		}
	}

	// 6. 发送订单状态变更通知（使用订单的最终状态）
	// 如果是失败状态且有可用通道重试，则不发送失败通知
	shouldSendNotification := true
	if req.NewStatus == model.OrderStatusFailed && hasAvailableChannel {
		shouldSendNotification = false
		lg.Info("订单有可用通道重试，跳过失败通知发送", logger.Int64V2("order_id", req.OrderID))
	}

	if shouldSendNotification && s.notificationRepo != nil && s.queue != nil {
		// 重新获取订单以确保使用最新的状态
		var finalOrder model.Order
		if err := s.db.WithContext(ctx).Where("id = ?", req.OrderID).First(&finalOrder).Error; err != nil {
			lg.Error("获取订单最终状态失败", logger.Int64V2("order_id", req.OrderID), logger.ErrorV2(err))
		} else {
			// 只有当最终状态与原始请求状态一致时才发送通知
			// 如果状态已在余额验证中被改变，则余额验证逻辑已经发送了通知
			if finalOrder.Status == req.NewStatus {
				if err := s.sendOrderStatusNotification(ctx, &finalOrder, finalOrder.Status); err != nil {
					lg.Error("发送订单状态通知失败", logger.Int64V2("order_id", req.OrderID), logger.ErrorV2(err))
					// 通知失败不影响订单状态更新的成功
				}
			} else {
				lg.Info("订单状态已在处理过程中变更，跳过重复通知",
					logger.Int64V2("order_id", req.OrderID),
					logger.IntV2("original_status", int(req.NewStatus)),
					logger.IntV2("final_status", int(finalOrder.Status)),
				)
			}
		}
	}

	return result, nil
}

// sendOrderStatusNotification 发送订单状态变更通知
func (s *UnifiedOrderService) sendOrderStatusNotification(ctx context.Context, order *model.Order, newStatus model.OrderStatus) error {
	return s.notificationHelper.SendOrderStatusNotification(ctx, order, newStatus)
}

// BalanceCheckResult 余额验证结果
type BalanceCheckResult struct {
	BalanceChanged  bool   `json:"balance_changed"`
	RefundTriggered bool   `json:"refund_triggered"`
	Message         string `json:"message"`
}

// performBalanceCheck 在事务外执行余额验证逻辑
func (s *UnifiedOrderService) performBalanceCheck(ctx context.Context, order *model.Order) (*BalanceCheckResult, error) {
	// 检查余额验证开关
	balanceVerificationEnabled, err := s.systemConfigService.GetBoolValue(ctx, "balance_verification_enabled")
	if err != nil {
		s.logger.Error("获取余额验证开关配置失败", zap.Error(err))
		// 配置获取失败时，默认启用余额验证以保证安全性
		balanceVerificationEnabled = true
	}

	if !balanceVerificationEnabled {
		s.logger.Info("余额验证已关闭，跳过余额验证", zap.Int64("order_id", order.ID))
		return &BalanceCheckResult{
			BalanceChanged:  true, // 假设余额已变化，避免触发退款
			RefundTriggered: false,
			Message:         "余额验证已关闭",
		}, nil
	}

	s.logger.Info("开始余额验证", zap.Int64("order_id", order.ID), zap.String("mobile", order.Mobile))

	// 获取充值前余额记录
	preBalanceRecord, err := s.balanceQueryRecordRepo.GetByOrderIDAndType(ctx, order.ID, "before")
	if err != nil {
		s.logger.Error("获取充值前余额记录失败", zap.Int64("order_id", order.ID), zap.Error(err))
		return nil, fmt.Errorf("获取充值前余额记录失败: %v", err)
	}

	if preBalanceRecord == nil {
		s.logger.Error("未找到充值前余额记录", zap.Int64("order_id", order.ID))
		return nil, fmt.Errorf("未找到充值前余额记录")
	}

	// 确定运营商类型
	ispType := s.getISPTypeFromOrder(order)

	// 实时查询充值后余额
	startTime := time.Now()
	postBalanceResp, err := s.phoneQueryService.QueryBalanceWithRetry(ctx, order.Mobile, ispType, 3)
	queryDuration := time.Since(startTime)

	s.logger.Info("充值后余额查询完成",
		zap.Int64("order_id", order.ID),
		zap.String("mobile", order.Mobile),
		zap.String("post_balance_data", postBalanceResp.Data),
		zap.Duration("query_duration", queryDuration),
		zap.Error(err),
	)

	if err != nil {
		return nil, fmt.Errorf("查询充值后余额失败: %v", err)
	}

	// 保存充值后余额查询记录
	postBalanceRecord := &model.BalanceQueryRecord{
		OrderID:     order.ID,
		OrderNumber: order.OrderNumber,
		Mobile:      order.Mobile,
		ISPType:     ispType,
		QueryType:   "after",
		Balance:     postBalanceResp.Data,
		QueryTime:   time.Now(),
		Success:     err == nil,
		Duration:    int64(queryDuration.Milliseconds()),
	}

	if err := s.balanceQueryRecordRepo.Create(ctx, postBalanceRecord); err != nil {
		s.logger.Error("保存充值后余额记录失败", zap.Int64("order_id", order.ID), zap.Error(err))
		// 不返回错误，继续执行余额对比
	}

	// 解析余额字符串为浮点数
	preBalance, err := strconv.ParseFloat(preBalanceRecord.Balance, 64)
	if err != nil {
		s.logger.Error("解析充值前余额失败", zap.Int64("order_id", order.ID), zap.String("balance", preBalanceRecord.Balance), zap.Error(err))
		return nil, fmt.Errorf("解析充值前余额失败: %v", err)
	}

	postBalance, err := strconv.ParseFloat(postBalanceResp.Data, 64)
	if err != nil {
		s.logger.Error("解析充值后余额失败", zap.Int64("order_id", order.ID), zap.String("balance", postBalanceResp.Data), zap.Error(err))
		return nil, fmt.Errorf("解析充值后余额失败: %v", err)
	}

	// 计算余额变化
	balanceChange := postBalance - preBalance
	expectedChange := order.Denom
	balanceChanged := math.Abs(balanceChange-expectedChange) < 0.01 // 允许0.01的误差

	s.logger.Info("余额对比结果",
		zap.Int64("order_id", order.ID),
		zap.Float64("pre_balance", preBalance),
		zap.Float64("post_balance", postBalance),
		zap.Float64("balance_change", balanceChange),
		zap.Float64("expected_change", expectedChange),
		zap.Bool("balance_changed", balanceChanged),
	)

	result := &BalanceCheckResult{
		BalanceChanged: balanceChanged,
		Message:        "余额验证完成",
	}

	// 如果余额没有变化，创建异常记录，但保持订单成功状态
	if !balanceChanged {
		s.logger.Warn("余额验证失败，创建异常记录", zap.Int64("order_id", order.ID))

		// 构建异常数据
		exceptionData := &model.BalanceVerificationExceptionData{
			PreBalance:    preBalanceRecord.Balance,
			PostBalance:   postBalanceResp.Data,
			ExpectedDiff:  expectedChange,
			ActualDiff:    balanceChange,
			Mobile:        order.Mobile,
			ISPType:       ispType,
			PlatformCode:  order.PlatformCode,
			Amount:        order.Price,
			QueryDuration: queryDuration.Milliseconds(),
		}

		// 创建异常记录
		if s.orderExceptionService != nil {
			if exceptionErr := s.orderExceptionService.CreateBalanceVerificationException(ctx, nil, order, exceptionData); exceptionErr != nil {
				s.logger.Error("创建余额验证异常记录失败", zap.Int64("order_id", order.ID), zap.Error(exceptionErr))
				result.Message = "余额验证失败，但创建异常记录失败"
			} else {
				s.logger.Info("余额验证异常记录创建成功", zap.Int64("order_id", order.ID))
				result.Message = "余额验证失败，已创建异常记录，订单保持成功状态"
			}
		} else {
			s.logger.Error("订单异常服务未初始化", zap.Int64("order_id", order.ID))
			result.Message = "余额验证失败，但异常服务不可用"
		}

		// 注意：这里不再触发退款，也不更新订单状态为失败
		// 订单保持成功状态，等待人工审核
		result.RefundTriggered = false
	}

	return result, nil
}

// performBalanceCheckWithTx 在事务内执行余额验证逻辑
func (s *UnifiedOrderService) performBalanceCheckWithTx(ctx context.Context, tx *gorm.DB, order *model.Order) (*BalanceCheckResult, error) {
	// 检查余额验证开关
	balanceVerificationEnabled, err := s.systemConfigService.GetBoolValue(ctx, "balance_verification_enabled")
	if err != nil {
		s.logger.Error("获取余额验证开关配置失败", zap.Error(err))
		// 配置获取失败时，默认启用余额验证以保证安全性
		balanceVerificationEnabled = true
	}

	if !balanceVerificationEnabled {
		s.logger.Info("余额验证已关闭，跳过余额验证", zap.Int64("order_id", order.ID))
		return &BalanceCheckResult{
			BalanceChanged:  true, // 假设余额已变化，避免触发退款
			RefundTriggered: false,
			Message:         "余额验证已关闭",
		}, nil
	}

	s.logger.Info("开始执行余额验证", zap.Int64("order_id", order.ID), zap.String("mobile", order.Mobile))

	// 获取充值前余额记录
	preRecord, err := s.balanceQueryRecordRepo.GetByOrderIDAndType(ctx, order.ID, "before")
	if err != nil {
		s.logger.Error("获取充值前余额记录失败", zap.Int64("order_id", order.ID), zap.Error(err))
		return &BalanceCheckResult{
			BalanceChanged:  false,
			RefundTriggered: false,
			Message:         "无法获取充值前余额记录",
		}, nil // 不返回错误，继续正常流程
	}

	// 根据订单信息确定运营商类型
	ispType := s.getISPTypeFromOrder(order)

	// 实时查询充值后余额
	start := time.Now()
	postBalance, err := s.phoneQueryService.QueryBalanceWithRetry(ctx, order.Mobile, ispType, 3)
	duration := time.Since(start).Milliseconds()

	if err != nil {
		s.logger.Error("充值后余额查询失败", zap.Int64("order_id", order.ID), zap.String("mobile", order.Mobile), zap.Error(err))

		// 保存失败的查询记录
		balanceRecord := &model.BalanceQueryRecord{
			OrderID:     order.ID,
			OrderNumber: order.OrderNumber,
			Mobile:      order.Mobile,
			ISPType:     ispType,
			QueryType:   "after",
			QueryTime:   time.Now(),
			Success:     false,
			ErrorMsg:    err.Error(),
			Duration:    duration,
		}

		if err := s.balanceQueryRecordRepo.Create(ctx, balanceRecord); err != nil {
			s.logger.Error("保存充值后余额查询记录失败", zap.Int64("order_id", order.ID), zap.Error(err))
		}

		return &BalanceCheckResult{
			BalanceChanged:  false,
			RefundTriggered: false,
			Message:         "余额查询失败",
		}, fmt.Errorf("充值后余额查询失败: %v", err)
	}

	// 处理查询结果，PhoneBalanceResponse结构体包含Data字段
	var balanceData string
	if postBalance != nil {
		balanceData = postBalance.Data
	} else {
		// 如果查询结果为空
		balanceData = "0"
	}

	s.logger.Info("充值后余额查询成功",
		zap.Int64("order_id", order.ID),
		zap.String("mobile", order.Mobile),
		zap.String("pre_balance", preRecord.Balance),
		zap.String("post_balance", balanceData),
	)

	// 比较充值前后余额
	preAmount, err1 := strconv.ParseFloat(preRecord.Balance, 64)
	postAmount, err2 := strconv.ParseFloat(balanceData, 64)

	balanceChanged := false
	if err1 == nil && err2 == nil {
		// 设置最小变化阈值为0.01元，避免浮点数精度问题
		balanceChanged = (postAmount - preAmount) >= 0.01
	}

	// 保存充值后余额查询记录
	balanceRecord := &model.BalanceQueryRecord{
		OrderID:     order.ID,
		OrderNumber: order.OrderNumber,
		Mobile:      order.Mobile,
		ISPType:     ispType,
		QueryType:   "after",
		Balance:     balanceData,
		QueryTime:   time.Now(),
		Success:     true,
		Duration:    duration,
	}

	if err := s.balanceQueryRecordRepo.Create(ctx, balanceRecord); err != nil {
		s.logger.Error("保存充值后余额查询记录失败", zap.Int64("order_id", order.ID), zap.Error(err))
	}

	result := &BalanceCheckResult{
		BalanceChanged:  balanceChanged,
		RefundTriggered: false,
	}

	// 如果余额没有变化，创建异常记录，但保持订单成功状态
	if !balanceChanged {
		s.logger.Warn("余额验证失败，创建异常记录",
			zap.Int64("order_id", order.ID),
			zap.String("pre_balance", preRecord.Balance),
			zap.String("post_balance", balanceData),
		)

		// 构建异常数据
		exceptionData := &model.BalanceVerificationExceptionData{
			PreBalance:    preRecord.Balance,
			PostBalance:   balanceData,
			ExpectedDiff:  0.01, // 最小期望变化
			ActualDiff:    postAmount - preAmount,
			Mobile:        order.Mobile,
			ISPType:       ispType,
			PlatformCode:  order.PlatformCode,
			Amount:        order.Price,
			QueryDuration: duration,
		}

		// 在事务中创建异常记录
		if s.orderExceptionService != nil {
			if exceptionErr := s.orderExceptionService.CreateBalanceVerificationException(ctx, tx, order, exceptionData); exceptionErr != nil {
				s.logger.Error("创建余额验证异常记录失败", zap.Int64("order_id", order.ID), zap.Error(exceptionErr))
				result.Message = "余额验证失败，但创建异常记录失败"
			} else {
				s.logger.Info("余额验证异常记录创建成功", zap.Int64("order_id", order.ID))
				result.Message = "余额验证失败，已创建异常记录，订单保持成功状态"
			}
		} else {
			s.logger.Error("订单异常服务未初始化", zap.Int64("order_id", order.ID))
			result.Message = "余额验证失败，但异常服务不可用"
		}

		// 注意：这里不再触发退款
		// 订单保持成功状态，等待人工审核
		result.RefundTriggered = false
	} else {
		result.Message = "余额验证通过"
	}

	return result, nil
}

// RefundResult 退款结果
type RefundResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// performRefundAsync 在事务外执行退款逻辑
func (s *UnifiedOrderService) performRefundAsync(ctx context.Context, order *model.Order, remark string) (*RefundResult, error) {
	s.logger.Info("开始异步执行退款", zap.Int64("order_id", order.ID), zap.String("remark", remark))

	// 检查余额服务是否可用
	if s.balanceService == nil {
		s.logger.Error("余额服务未初始化", zap.Int64("order_id", order.ID))
		return &RefundResult{
			Success: false,
			Message: "余额服务未初始化",
		}, nil
	}

	// 检查是否已经退款（通过查询余额日志）
	var existingRefundCount int64
	err := s.db.Model(&model.BalanceLog{}).Where("order_id = ? AND style = ?", order.ID, model.BalanceStyleRefund).Count(&existingRefundCount).Error
	if err != nil {
		s.logger.Error("查询退款记录失败", zap.Int64("order_id", order.ID), zap.Error(err))
		return &RefundResult{
			Success: false,
			Message: "查询退款记录失败",
		}, nil
	}

	if existingRefundCount > 0 {
		s.logger.Info("订单已存在退款记录", zap.Int64("order_id", order.ID))
		return &RefundResult{
			Success: true,
			Message: "订单已存在退款记录",
		}, nil
	}

	// 执行退款
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return s.balanceService.RefundUserOrderPaymentWithTx(ctx, tx, order.CustomerID, order.ID, order.Price, remark, "system")
	})
	if err != nil {
		s.logger.Error("退款失败", zap.Int64("order_id", order.ID), zap.Error(err))
		return &RefundResult{
			Success: false,
			Message: fmt.Sprintf("退款失败: %v", err),
		}, nil
	}

	s.logger.Info("退款成功",
		zap.Int64("order_id", order.ID),
		zap.Float64("amount", order.Price),
		zap.Int64("customer_id", order.CustomerID),
	)

	return &RefundResult{
		Success: true,
		Message: "退款成功",
	}, nil
}

// hasAvailableRetryChannel 只读检查失败订单是否还有可用通道（不推送任务、无副作用）
func (s *UnifiedOrderService) hasAvailableRetryChannel(ctx context.Context, order *model.Order) (bool, error) {
	if s.retryService == nil || s.queue == nil {
		return false, nil
	}
	relations, err := s.retryService.GetAvailableAPIRelations(ctx, order.ID, order.ProductID)
	if err != nil {
		return false, err
	}
	return len(relations) > 0, nil
}

// refundFailedOrderWithTx 在给定事务内对失败订单执行幂等退款（与状态更新原子提交）
// 按订单扣款明细退款：余额退回余额、授信恢复额度（幂等）
func (s *UnifiedOrderService) refundFailedOrderWithTx(ctx context.Context, tx *gorm.DB, order *model.Order) error {
	if order.Price <= 0 {
		// 无金额可退（如0元/赠送单），直接视为无需退款
		return nil
	}
	if s.balanceService == nil {
		return fmt.Errorf("余额服务未初始化，无法退款")
	}
	return s.balanceService.RefundUserOrderPaymentWithTx(ctx, tx, order.CustomerID, order.ID, order.Price, "订单失败退还余额", "system")
}

// ensureFailedOrderRefunded 独立事务幂等补退（用于重试推送失败等兜底场景）
func (s *UnifiedOrderService) ensureFailedOrderRefunded(ctx context.Context, orderID int64) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order model.Order
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("id = ?", orderID).First(&order).Error; err != nil {
			return fmt.Errorf("获取订单信息失败: %v", err)
		}
		return s.refundFailedOrderWithTx(ctx, tx, &order)
	})
}

// performRefund 执行退款逻辑（保留原有方法以兼容其他调用）
func (s *UnifiedOrderService) performRefund(ctx context.Context, tx *gorm.DB, order *model.Order, remark string) (*RefundResult, error) {
	s.logger.Info("开始执行退款", zap.Int64("order_id", order.ID), zap.Float64("amount", order.Price), zap.String("reason", remark))

	if s.balanceService == nil {
		return &RefundResult{
			Success: false,
			Message: "余额服务未初始化",
		}, fmt.Errorf("余额服务未初始化")
	}

	// 统一退款到订单的CustomerID
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return s.balanceService.RefundUserOrderPaymentWithTx(ctx, tx, order.CustomerID, order.ID, order.Price, fmt.Sprintf("订单退款: %s", remark), "system")
	})

	if err != nil {
		s.logger.Error("退款失败", zap.Int64("order_id", order.ID), zap.Error(err))
		return &RefundResult{
			Success: false,
			Message: fmt.Sprintf("退款失败: %v", err),
		}, err
	}

	s.logger.Info("退款成功", zap.Int64("order_id", order.ID), zap.Float64("amount", order.Price))
	return &RefundResult{
		Success: true,
		Message: "退款成功",
	}, nil
}

// getISPTypeFromOrder 从订单信息获取运营商类型
func (s *UnifiedOrderService) getISPTypeFromOrder(order *model.Order) string {
	// 根据订单中的 ISP 字段判断运营商类型
	// 1: 移动(yd), 2: 电信(dx), 3: 联通(lt)
	switch order.ISP {
	case 1:
		return "yd" // 中国移动
	case 2:
		return "dx" // 中国电信
	case 3:
		return "lt" // 中国联通
	default:
		// 默认返回移动
		return "yd"
	}
}

// ProcessOrderStatusChange 处理订单状态变更（供外部回调使用）
func (s *UnifiedOrderService) ProcessOrderStatusChange(ctx context.Context, orderID int64, newStatus model.OrderStatus, callbackSource string) error {
	req := &OrderStatusUpdateRequest{
		OrderID:          orderID,
		NewStatus:        newStatus,
		CallbackSource:   callbackSource,
		NeedBalanceCheck: newStatus == model.OrderStatusSuccess, // 成功状态需要余额验证
	}

	response, err := s.UpdateOrderStatusUnified(ctx, req)
	if err != nil {
		return err
	}

	if !response.Success {
		return fmt.Errorf("订单状态更新失败: %s", response.Message)
	}

	return nil
}

// checkAndHandleFailedOrderRetry 检查失败订单是否有可用通道进行重试
// 返回值：hasAvailableChannel bool - 是否有可用通道并已推送重试任务
// CheckAndHandleFailedOrderRetry 检查失败订单是否有可用通道并处理重试
func (s *UnifiedOrderService) CheckAndHandleFailedOrderRetry(ctx context.Context, order *model.Order) (bool, error) {
	return s.checkAndHandleFailedOrderRetry(ctx, order)
}

func (s *UnifiedOrderService) checkAndHandleFailedOrderRetry(ctx context.Context, order *model.Order) (bool, error) {
	s.logger.Info("处理失败订单，检查是否有其他可用通道",
		zap.Int64("order_id", order.ID),
		zap.String("order_number", order.OrderNumber),
		zap.Int64("product_id", order.ProductID),
		zap.String("used_apis", order.UsedAPIs))

	// 检查依赖是否可用
	if s.retryService == nil || s.queue == nil {
		s.logger.Warn("重试逻辑依赖未初始化，跳过重试检查",
			zap.Int64("order_id", order.ID),
			zap.Bool("retry_service_nil", s.retryService == nil),
			zap.Bool("queue_nil", s.queue == nil))
		return false, nil
	}

	// 使用retryService获取可用的API关系，确保API状态检查的一致性
	availableRelations, err := s.retryService.GetAvailableAPIRelations(ctx, order.ID, order.ProductID)
	if err != nil {
		s.logger.Error("获取可用API关系失败", zap.Int64("order_id", order.ID), zap.Error(err))
		return false, err
	}

	hasAvailableAPI := len(availableRelations) > 0
	s.logger.Info("检查可用API结果",
		zap.Int64("order_id", order.ID),
		zap.Int("available_count", len(availableRelations)),
		zap.Bool("has_available", hasAvailableAPI))

	s.logger.Info("API可用性检查结果",
		zap.Int64("order_id", order.ID),
		zap.Bool("has_available_api", hasAvailableAPI))

	if hasAvailableAPI {
		// 有可用通道，推送重试任务到消息队列
		s.logger.Info("发现可用通道，推送重试任务到队列", zap.Int64("order_id", order.ID))
		if err := s.pushRetryTaskToQueue(ctx, order.ID, 2, "统一订单服务失败，切换通道重试"); err != nil {
			s.logger.Error("推送重试任务到队列失败", zap.Int64("order_id", order.ID), zap.Error(err))
			return false, err
		}
		// 推送成功
		s.logger.Info("重试任务推送成功，等待重试处理", zap.Int64("order_id", order.ID))
		return true, nil
	} else {
		// 没有可用通道
		s.logger.Info("没有可用通道，订单最终失败", zap.Int64("order_id", order.ID))
		return false, nil
	}
}

// pushRetryTaskToQueue 推送重试任务到队列
func (s *UnifiedOrderService) pushRetryTaskToQueue(ctx context.Context, orderID int64, retryType int, reason string) error {
	task := model.NewRetryTaskMessage(orderID, retryType, reason)

	// 使用默认队列名称，也可以从配置中读取
	queueName := "retry_queue"
	if err := s.queue.Push(ctx, queueName, task); err != nil {
		return fmt.Errorf("推送重试任务到队列失败: %v", err)
	}

	s.logger.Info("重试任务推送到队列成功",
		zap.Int64("order_id", orderID),
		zap.Int("retry_type", retryType),
		zap.String("queue_name", queueName))
	return nil
}

// SetRetryService 设置重试服务依赖（用于解决循环依赖）
func (s *UnifiedOrderService) SetRetryService(retryService *RetryService) {
	s.retryService = retryService
}

// ProcessOrderStatusChangeWithBalanceCheck 处理订单状态变更（供平台回调使用，包含余额验证）
func (s *UnifiedOrderService) ProcessOrderStatusChangeWithBalanceCheck(ctx context.Context, orderID int64, newStatus model.OrderStatus, callbackSource string, needBalanceCheck bool) error {
	req := &OrderStatusUpdateRequest{
		OrderID:          orderID,
		NewStatus:        newStatus,
		CallbackSource:   callbackSource,
		NeedBalanceCheck: needBalanceCheck && newStatus == model.OrderStatusSuccess,
	}

	response, err := s.UpdateOrderStatusUnified(ctx, req)
	if err != nil {
		return err
	}

	if !response.Success {
		return fmt.Errorf("订单状态更新失败: %s", response.Message)
	}

	return nil
}
