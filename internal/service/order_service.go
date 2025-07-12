package service

import (
	"context"
	"fmt"
	"recharge-go/internal/model"
	notificationModel "recharge-go/internal/model/notification"
	"recharge-go/internal/repository"
	notificationRepo "recharge-go/internal/repository/notification"
	"recharge-go/internal/utils"
	"recharge-go/pkg/lock"
	"recharge-go/pkg/logger"
	"recharge-go/pkg/queue"
	"strconv"
	"time"
	"gorm.io/gorm"
)

// OrderService 订单服务接口
type OrderService interface {
	// CreateOrder 创建订单
	CreateOrder(ctx context.Context, order *model.Order) error
	// CreateExternalOrder 创建外部订单（事务性处理：先扣款再创建订单）
	CreateExternalOrder(ctx context.Context, order *model.Order, platformAccountID int64) error
	// GetOrderByID 根据ID获取订单
	GetOrderByID(ctx context.Context, id int64) (*model.Order, error)
	// GetOrderByOrderNumber 根据订单号获取订单
	GetOrderByOrderNumber(ctx context.Context, orderNumber string) (*model.Order, error)
	// GetOrdersByCustomerID 根据客户ID获取订单列表
	GetOrdersByCustomerID(ctx context.Context, customerID int64, page, pageSize int) ([]*model.Order, int64, error)
	// UpdateOrderStatus 更新订单状态
	UpdateOrderStatus(ctx context.Context, id int64, status model.OrderStatus) error
	// ProcessOrderPayment 处理订单支付
	ProcessOrderPayment(ctx context.Context, orderID int64, payWay int, serialNumber string) error
	// ProcessOrderRecharge 处理订单充值
	ProcessOrderRecharge(ctx context.Context, orderID int64, apiID int64, apiOrderNumber string, apiTradeNum string) error
	// ProcessOrderSuccess 处理订单成功
	ProcessOrderSuccess(ctx context.Context, orderID int64) error
	// ProcessOrderFail 处理订单失败
	ProcessOrderFail(ctx context.Context, orderID int64, remark string) error
	// ProcessOrderRefund 处理订单退款
	ProcessOrderRefund(ctx context.Context, orderID int64, remark string) error
	// ProcessExternalRefund 处理外部订单退款
	ProcessExternalRefund(ctx context.Context, outTradeNum string, reason string) error
	// GetOrderByOutTradeNum 根据外部交易号获取订单
	GetOrderByOutTradeNum(ctx context.Context, outTradeNum string) (*model.Order, error)
	// ProcessOrderCancel 处理订单取消
	ProcessOrderCancel(ctx context.Context, orderID int64, remark string) error
	// ProcessOrderSplit 处理订单拆单
	ProcessOrderSplit(ctx context.Context, orderID int64, remark string) error
	// ProcessOrderPartial 处理订单部分充值
	ProcessOrderPartial(ctx context.Context, orderID int64, remark string) error
	// GetOrders 获取订单列表
	GetOrders(ctx context.Context, params map[string]interface{}, page, pageSize int) ([]*model.Order, int64, error)
	// GetOrdersWithNotification 获取包含通知信息的订单列表
	GetOrdersWithNotification(ctx context.Context, params map[string]interface{}, page, pageSize int) ([]*model.OrderWithNotification, int64, error)
	// SetRechargeService 设置充值服务
	SetRechargeService(rechargeService RechargeService)
	// DeleteOrder 删除订单（软删除）
	DeleteOrder(ctx context.Context, id string) error
	// CleanupOrders 清理指定时间范围的订单及相关日志
	CleanupOrders(ctx context.Context, start, end string) (int64, error)
	// GetProductID 根据价格、ISP和状态获取产品ID
	GetProductID(price float64, isp int, status int) (*model.Product, error)
	// GetProductByNameValue 根据产品名称数字部分、ISP和状态获取产品
	GetProductByNameValue(nameValue float64, isp int, status int) (*model.Product, error)
	// GetOrderStatistics 按 customer_id 统计今日订单总数、成功订单数、失败订单数、今日成交金额（Denom 字段）
	GetOrderStatistics(ctx context.Context, customerID int64) (*OrderStatistics, error)
	// GetOrdersByUserID 根据用户ID获取订单列表
	GetOrdersByUserID(ctx context.Context, userID int64, params map[string]interface{}, page, pageSize int) ([]*model.Order, int64, error)
	// SendNotification 发送订单回调通知
	SendNotification(ctx context.Context, orderID int64) error
}

type OrderStatistics struct {
	TotalCount      int64   `json:"total_count"`
	SuccessCount    int64   `json:"success_count"`
	FailedCount     int64   `json:"failed_count"`
	ProcessingCount int64   `json:"processing_count"`
	SuccessAmount   float64 `json:"success_amount"`
}

// orderService 订单服务实现
type orderService struct {
	orderRepo        repository.OrderRepository
	rechargeService  RechargeService
	notificationRepo notificationRepo.Repository
	queue            queue.Queue
	balanceLogRepo   *repository.BalanceLogRepository
	userRepo         *repository.UserRepository
	productRepo      repository.ProductRepository
	unifiedRefundService *UnifiedRefundService
	lockManager      *lock.RefundLockManager
	db               *gorm.DB
	creditService    *CreditService
}

// NewOrderService 创建订单服务实例
func NewOrderService(
	orderRepo repository.OrderRepository,
	balanceLogRepo *repository.BalanceLogRepository,
	userRepo *repository.UserRepository,
	rechargeService RechargeService,
	unifiedRefundService *UnifiedRefundService,
	lockManager *lock.RefundLockManager,
	notificationRepo notificationRepo.Repository,
	queue queue.Queue,
	db *gorm.DB,
	productRepo repository.ProductRepository,
	creditService *CreditService,
) OrderService {
	return &orderService{
		orderRepo:        orderRepo,
		rechargeService:  rechargeService,
		notificationRepo: notificationRepo,
		queue:            queue,
		balanceLogRepo:   balanceLogRepo,
		userRepo:         userRepo,
		productRepo:      productRepo,
		unifiedRefundService: unifiedRefundService,
		lockManager:      lockManager,
		db:               db,
		creditService:    creditService,
	}
}

// CreateOrder 创建订单
func (s *orderService) CreateOrder(ctx context.Context, order *model.Order) error {
	// 生成订单号
	order.OrderNumber = generateOrderNumber()
	order.CreateTime = time.Now()
	order.UpdatedAt = time.Now()

	// 根据订单来源决定初始状态
	// 如果是自动取单任务创建的订单(client=3)，直接进入待充值状态
	if order.Client == 3 {
		order.Status = model.OrderStatusPendingRecharge
	} else {
		order.Status = model.OrderStatusPendingPayment
	}

	order.IsDel = 0

	if err := s.orderRepo.Create(ctx, order); err != nil {
		return err
	}

	// 创建成功后，将订单推送到充值队列
	if err := s.rechargeService.PushToRechargeQueue(ctx, order.ID); err != nil {
		logger.Error("推送订单到充值队列失败: %v", err)
		// 这里可以选择是否返回错误，因为订单已经创建成功
	}

	return nil
}

// GetOrderByID 根据ID获取订单
func (s *orderService) GetOrderByID(ctx context.Context, id int64) (*model.Order, error) {
	return s.orderRepo.GetByID(ctx, id)
}

// GetOrderByOrderNumber 根据订单号获取订单
func (s *orderService) GetOrderByOrderNumber(ctx context.Context, orderNumber string) (*model.Order, error) {
	return s.orderRepo.GetByOrderNumber(ctx, orderNumber)
}

// GetOrdersByCustomerID 根据客户ID获取订单列表
func (s *orderService) GetOrdersByCustomerID(ctx context.Context, customerID int64, page, pageSize int) ([]*model.Order, int64, error) {
	return s.orderRepo.GetByCustomerID(ctx, customerID, page, pageSize)
}

// 工具函数：判断是否超级管理员
func isSuperAdmin(ctx context.Context) bool {
	roles, ok := ctx.Value("roles").([]string)
	if !ok {
		return false
	}
	for _, r := range roles {
		if r == "SUPER_ADMIN" {
			return true
		}
	}
	return false
}

// UpdateOrderStatus 更新订单状态
func (s *orderService) UpdateOrderStatus(ctx context.Context, id int64, status model.OrderStatus) error {
	// 安全获取用户ID，如果不存在则使用0（系统操作）
	var userID int64
	if uid := ctx.Value("user_id"); uid != nil {
		userID = uid.(int64)
	}
	logger.Info("开始更新订单状态",
		"order_id", id,
		"new_status", status,
		"user_id", userID,
	)

	// 开启事务
	tx := s.orderRepo.(*repository.OrderRepositoryImpl).DB().Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 获取订单信息
	order, err := s.orderRepo.GetByID(ctx, id)
	if err != nil {
		tx.Rollback()
		logger.Error("获取订单信息失败",
			"error", err,
			"order_id", id,
		)
		return fmt.Errorf("get order failed: %v", err)
	}

	// 权限校验：超级管理员、系统操作（user_id为0且无roles）或订单所有者可以操作
	isSystemOperation := userID == 0 && ctx.Value("roles") == nil
	if !isSuperAdmin(ctx) && !isSystemOperation && order.CustomerID != userID {
		tx.Rollback()
		logger.Error("无权限操作该订单",
			"order_id", id,
			"user_id", userID,
			"order_customer_id", order.CustomerID,
		)
		return fmt.Errorf("无权限操作该订单")
	}

	logger.Info("获取到订单信息",
		"order_id", id,
		"current_status", order.Status,
		"new_status", status,
	)

	// 如果状态没有变化，直接返回
	if order.Status == status {
		tx.Rollback()
		logger.Info("订单状态未发生变化，无需更新",
			"order_id", id,
			"status", status,
		)
		return nil
	}

	// 更新订单状态
	if err := tx.Model(&model.Order{}).Where("id = ?", id).Update("status", status).Error; err != nil {
		tx.Rollback()
		logger.Error("更新订单状态失败",
			"error", err,
			"order_id", id,
			"old_status", order.Status,
			"new_status", status,
		)
		return fmt.Errorf("update order status failed: %v", err)
	}

	// 创建通知记录
	notification := &notificationModel.NotificationRecord{
		OrderID:          id,
		PlatformCode:     order.PlatformCode,
		NotificationType: "order_status_changed",
		Content:          fmt.Sprintf("订单状态已更新为: %d", status),
		Status:           1, // 待处理
	}

	// 保存通知记录到数据库
	if err := s.notificationRepo.Create(ctx, notification); err != nil {
		tx.Rollback()
		logger.Error("创建通知记录失败",
			"error", err,
			"order_id", id,
			"platform_code", order.PlatformCode,
			"notification_type", notification.NotificationType,
		)
		return fmt.Errorf("create notification record failed: %v", err)
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		logger.Error("提交事务失败",
			"error", err,
			"order_id", id,
		)
		return fmt.Errorf("commit transaction failed: %v", err)
	}

	logger.Info("订单状态更新成功",
		"order_id", id,
		"old_status", order.Status,
		"new_status", status,
	)

	// 事务提交成功后，重新获取订单信息并推送通知到队列
	updatedOrder, getErr := s.orderRepo.GetByID(ctx, id)
	if getErr != nil {
		logger.Error("获取更新后的订单信息失败", "order_id", id, "error", getErr)
		return nil // 订单状态已更新成功，通知推送失败不影响主流程
	}

	// 重新创建通知记录，确保包含最新的订单信息
	updatedNotification := &notificationModel.NotificationRecord{
		OrderID:          id,
		PlatformCode:     updatedOrder.PlatformCode,
		NotificationType: "order_status_changed",
		Content:          fmt.Sprintf("订单状态已更新为: %d", status),
		Status:           1, // 待处理
	}

	// 保存新的通知记录到数据库
	if createErr := s.notificationRepo.Create(ctx, updatedNotification); createErr != nil {
		logger.Error("创建更新后的通知记录失败", "order_id", id, "error", createErr)
		return nil // 订单状态已更新成功，通知推送失败不影响主流程
	}

	// 推送通知到队列
	logger.Info("准备推送通知到队列", "order_id", id, "new_status", status)
	if pushErr := s.queue.Push(ctx, "notification_queue", updatedNotification); pushErr != nil {
		logger.Error("推送通知到队列失败", "order_id", id, "error", pushErr)
	} else {
		logger.Info("推送通知到队列成功", "order_id", id, "status", status)
	}
	return nil
}

// ProcessOrderPayment 处理订单支付
func (s *orderService) ProcessOrderPayment(ctx context.Context, orderID int64, payWay int, serialNumber string) error {
	// 更新支付信息
	err := s.orderRepo.UpdatePayInfo(ctx, orderID, payWay, serialNumber)
	if err != nil {
		return err
	}

	// 更新订单状态为待充值
	return s.UpdateOrderStatus(ctx, orderID, model.OrderStatusPendingRecharge)
}

// ProcessOrderRecharge 处理订单充值
func (s *orderService) ProcessOrderRecharge(ctx context.Context, orderID int64, apiID int64, apiOrderNumber string, apiTradeNum string) error {
	// 更新API信息
	err := s.orderRepo.UpdateAPIInfo(ctx, orderID, apiID, apiOrderNumber, apiTradeNum)
	if err != nil {
		return err
	}

	// 更新订单状态为充值中
	return s.UpdateOrderStatus(ctx, orderID, model.OrderStatusRecharging)
}

// ProcessOrderSuccess 处理订单成功
func (s *orderService) ProcessOrderSuccess(ctx context.Context, orderID int64) error {
	// 更新完成时间
	err := s.orderRepo.UpdateFinishTime(ctx, orderID)
	if err != nil {
		return err
	}

	// 更新订单状态为成功
	return s.UpdateOrderStatus(ctx, orderID, model.OrderStatusSuccess)
}

// ProcessOrderFail 处理订单失败
func (s *orderService) ProcessOrderFail(ctx context.Context, orderID int64, remark string) error {
	logger.Info("开始处理订单失败", "order_id", orderID, "remark", remark)
	
	// 1. 先获取订单信息以确定用户ID
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		logger.Error("获取订单信息失败", "order_id", orderID, "error", err)
		return fmt.Errorf("获取订单信息失败: %v", err)
	}
	logger.Info("获取订单信息成功", "order_id", orderID, "customer_id", order.CustomerID, "status", order.Status)

	// 2. 获取用户级别的分布式锁
	lockValue, err := s.lockManager.LockUserRefund(ctx, order.CustomerID)
	if err != nil {
		logger.Error("获取用户退款锁失败", "user_id", order.CustomerID, "order_id", orderID, "error", err)
		return fmt.Errorf("获取退款锁失败: %v", err)
	}
	defer func() {
		if unlockErr := s.lockManager.UnlockUserRefund(ctx, order.CustomerID, lockValue); unlockErr != nil {
			logger.Error("释放用户退款锁失败", "user_id", order.CustomerID, "order_id", orderID, "error", unlockErr)
		}
	}()

	// 3. 在锁保护下执行事务
	logger.Info("开始执行事务", "order_id", orderID)
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		logger.Info("事务内部开始执行", "order_id", orderID)
		// 使用行锁防止同一订单的并发处理
		var lockedOrder model.Order
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("id = ?", orderID).First(&lockedOrder).Error; err != nil {
			logger.Error("获取订单行锁失败", "order_id", orderID, "error", err)
			return err
		}
		logger.Info("获取订单行锁成功", "order_id", orderID, "locked_status", lockedOrder.Status)

		// 检查订单状态，防止重复处理
		if lockedOrder.Status == model.OrderStatusFailed {
			// 订单已经是失败状态，跳过处理
			logger.Info("订单已经是失败状态，跳过重复处理", "order_id", orderID)
			return nil
		}

		// 如果订单已经支付，需要退还余额
		if lockedOrder.Status == model.OrderStatusPendingRecharge || lockedOrder.Status == model.OrderStatusRecharging || lockedOrder.Status == model.OrderStatusProcessing {
			// 使用统一退款服务处理退款
			var refundReq *RefundRequest
			if lockedOrder.Client == 2 {
				// 外部订单直接退款到用户余额
				logger.Info("外部订单失败，使用统一退款服务退款到用户余额",
					"order_id", orderID,
					"customer_id", lockedOrder.CustomerID,
					"amount", lockedOrder.Price)

				refundReq = &RefundRequest{
					UserID:   lockedOrder.CustomerID,
					OrderID:  orderID,
					Amount:   lockedOrder.Price,
					Remark:   "外部订单失败退款",
					Operator: "system",
					Type:     RefundTypeUser,
					Tx:       tx,
				}
			} else {
				// 平台订单退款
				logger.Info("平台订单失败，使用统一退款服务退款",
					"order_id", orderID,
					"customer_id", lockedOrder.CustomerID,
					"platform_account_id", lockedOrder.PlatformAccountID,
					"amount", lockedOrder.Price)

				refundReq = &RefundRequest{
					UserID:    lockedOrder.CustomerID,
					OrderID:   orderID,
					Amount:    lockedOrder.Price,
					Remark:    "订单失败退还余额",
					Operator:  "system",
					Type:      RefundTypePlatform,
					AccountID: &lockedOrder.PlatformAccountID,
					Tx:        tx,
				}
			}

			// 执行统一退款
			refundResp, err := s.unifiedRefundService.ProcessRefund(ctx, refundReq)
			if err != nil || !refundResp.Success {
				logger.Error("统一退款服务退款失败",
					"error", err,
					"order_id", orderID,
					"customer_id", lockedOrder.CustomerID,
					"amount", lockedOrder.Price,
					"response", refundResp)
				if err != nil {
					return fmt.Errorf("统一退款失败: %v", err)
				}
				return fmt.Errorf("统一退款失败: %s", refundResp.Message)
			}

			logger.Info("统一退款服务退款成功",
				"order_id", orderID,
				"customer_id", lockedOrder.CustomerID,
				"amount", refundResp.RefundAmount,
				"balance_after", refundResp.BalanceAfter,
				"already_refund", refundResp.AlreadyRefund)
		}

		// 更新备注
		if err := tx.Model(&model.Order{}).Where("id = ?", orderID).Update("remark", remark).Error; err != nil {
			return err
		}

		// 更新订单状态为失败
		if err := tx.Model(&model.Order{}).Where("id = ?", orderID).Update("status", model.OrderStatusFailed).Error; err != nil {
			return err
		}

		logger.Info("订单失败处理完成",
			"order_id", orderID,
			"status", model.OrderStatusFailed)

		// 创建通知记录
		notification := &notificationModel.NotificationRecord{
			OrderID:          orderID,
			PlatformCode:     lockedOrder.PlatformCode,
			NotificationType: "order_status_changed",
			Content:          fmt.Sprintf("订单失败: %s", remark),
			Status:           1, // 待处理
		}

		// 保存通知记录到数据库
		logger.Info("准备创建通知记录", "order_id", orderID, "platform_code", lockedOrder.PlatformCode)
		if err := s.notificationRepo.Create(ctx, notification); err != nil {
			logger.Error("创建通知记录失败",
				"error", err,
				"order_id", orderID,
				"platform_code", lockedOrder.PlatformCode,
				"notification_type", notification.NotificationType)
			return fmt.Errorf("create notification record failed: %v", err)
		}
		logger.Info("创建通知记录成功", "order_id", orderID, "notification_id", notification.ID)

		logger.Info("事务内部执行完成", "order_id", orderID)
		return nil
	})
	
	logger.Info("事务执行结果", "order_id", orderID, "error", err)

	// 事务提交成功后，异步推送通知到队列
	if err == nil {
		logger.Info("事务提交成功，开始推送通知到队列", "order_id", orderID)
		// 重新获取已创建的通知记录
		notification, getErr := s.notificationRepo.GetByOrderID(ctx, orderID)
		if getErr != nil {
			logger.Error("获取通知记录失败", "error", getErr, "order_id", orderID)
		} else {
			logger.Info("成功获取通知记录", "order_id", orderID, "notification_id", notification.ID)
			// 推送通知到队列
			logger.Info("准备推送订单失败通知到队列", "order_id", orderID, "remark", remark)
			if pushErr := s.queue.Push(ctx, "notification_queue", notification); pushErr != nil {
				logger.Error("推送订单失败通知到队列失败", "order_id", orderID, "error", pushErr)
			} else {
				logger.Info("推送订单失败通知到队列成功", "order_id", orderID)
			}
		}
	} else {
		logger.Error("事务执行失败，跳过推送通知", "order_id", orderID, "error", err)
	}

	return err
}

// ProcessOrderRefund 处理订单退款
func (s *orderService) ProcessOrderRefund(ctx context.Context, orderID int64, remark string) error {
	// 使用事务确保订单状态更新和退款操作的原子性
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 使用行锁防止同一订单的并发处理
		var lockedOrder model.Order
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("id = ?", orderID).First(&lockedOrder).Error; err != nil {
			logger.Error("获取订单失败", "error", err, "order_id", orderID)
			return fmt.Errorf("订单不存在")
		}

		// 2. 检查订单状态是否允许退款
		if lockedOrder.Status == model.OrderStatusRefunded {
			logger.Info("订单已退款，跳过处理", "order_id", orderID)
			return fmt.Errorf("订单已退款")
		}

		// 只有成功、失败、待充值状态的订单可以退款
		if lockedOrder.Status != model.OrderStatusSuccess &&
			lockedOrder.Status != model.OrderStatusFailed &&
			lockedOrder.Status != model.OrderStatusPendingRecharge {
			logger.Error("订单状态不允许退款", "order_id", orderID, "status", lockedOrder.Status)
			return fmt.Errorf("订单状态不允许退款")
		}

		// 3. 执行退款逻辑
		if lockedOrder.Client == 2 {
			// 外部订单退款到用户余额（使用当前事务）
			balanceService := NewBalanceService(s.balanceLogRepo, s.userRepo)
			if err := balanceService.RefundWithTx(ctx, tx, lockedOrder.CustomerID, lockedOrder.Price, orderID, fmt.Sprintf("订单退款: %s", remark), "admin"); err != nil {
				logger.Error("外部订单退款失败", "error", err, "order_id", orderID)
				return fmt.Errorf("退款失败: %v", err)
			}
			logger.Info("外部订单退款成功", "order_id", orderID, "amount", lockedOrder.Price)
		} else {
			// 平台订单退款到用户余额
			if err := s.rechargeService.GetUserBalanceService().RefundWithTx(ctx, tx, lockedOrder.CustomerID, lockedOrder.Price, orderID, fmt.Sprintf("订单退款: %s", remark), "system"); err != nil {
				logger.Error("平台订单退款失败", "error", err, "order_id", orderID)
				return fmt.Errorf("退款失败: %v", err)
			}
			logger.Info("平台订单退款成功", "order_id", orderID, "amount", lockedOrder.Price)
		}

		// 4. 更新备注
		if err := tx.Model(&model.Order{}).Where("id = ?", orderID).Update("remark", remark).Error; err != nil {
			return err
		}

		// 5. 更新订单状态为已退款
		if err := tx.Model(&model.Order{}).Where("id = ?", orderID).Update("status", model.OrderStatusRefunded).Error; err != nil {
			return err
		}

		logger.Info("订单退款处理完成",
			"order_id", orderID,
			"status", model.OrderStatusRefunded)
		return nil
	})
}

// ProcessExternalRefund 处理外部订单退款
func (s *orderService) ProcessExternalRefund(ctx context.Context, outTradeNum string, reason string) error {
	logger.Info("开始处理外部订单退款",
		"out_trade_num", outTradeNum,
		"reason", reason)

	// 使用事务确保订单状态更新和退款操作的原子性
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. 根据外部交易号获取订单
		order, err := s.GetOrderByOutTradeNum(ctx, outTradeNum)
		if err != nil {
			logger.Error("获取订单失败",
				"error", err,
				"out_trade_num", outTradeNum)
			return fmt.Errorf("订单不存在")
		}

		// 使用行锁防止同一订单的并发处理
		var lockedOrder model.Order
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("id = ?", order.ID).First(&lockedOrder).Error; err != nil {
			return err
		}

		// 2. 检查订单状态是否允许退款
		if lockedOrder.Status == model.OrderStatusRefunded {
			logger.Info("订单已退款，跳过处理",
				"order_id", lockedOrder.ID,
				"out_trade_num", outTradeNum)
			return fmt.Errorf("订单已退款")
		}

		// 只有成功、失败、待充值状态的订单可以退款
		if lockedOrder.Status != model.OrderStatusSuccess &&
			lockedOrder.Status != model.OrderStatusFailed &&
			lockedOrder.Status != model.OrderStatusPendingRecharge {
			logger.Error("订单状态不允许退款",
				"order_id", lockedOrder.ID,
				"status", lockedOrder.Status,
				"out_trade_num", outTradeNum)
			return fmt.Errorf("订单状态不允许退款")
		}

		// 3. 检查是否为外部订单
		if lockedOrder.Client != 2 {
			logger.Error("非外部订单，不能使用此退款方法",
				"order_id", lockedOrder.ID,
				"client", lockedOrder.Client,
				"out_trade_num", outTradeNum)
			return fmt.Errorf("非外部订单")
		}

		// 4. 直接退款到用户余额（外部订单使用用户余额系统，使用当前事务）
		balanceService := NewBalanceService(s.balanceLogRepo, s.userRepo)
		if err := balanceService.RefundWithTx(ctx, tx, lockedOrder.CustomerID, lockedOrder.Price, lockedOrder.ID, fmt.Sprintf("外部订单退款: %s", reason), "system"); err != nil {
			logger.Error("退款到用户余额失败",
				"error", err,
				"order_id", lockedOrder.ID,
				"customer_id", lockedOrder.CustomerID,
				"amount", lockedOrder.Price)
			return fmt.Errorf("退款失败: %v", err)
		}

		logger.Info("退款到用户余额成功",
			"order_id", lockedOrder.ID,
			"customer_id", lockedOrder.CustomerID,
			"amount", lockedOrder.Price)

		// 5. 更新订单备注
		if err := tx.Model(&model.Order{}).Where("id = ?", lockedOrder.ID).Update("remark", fmt.Sprintf("外部订单退款: %s", reason)).Error; err != nil {
			logger.Error("更新订单备注失败", "error", err, "order_id", lockedOrder.ID)
			return fmt.Errorf("更新订单备注失败: %v", err)
		}

		// 6. 更新订单状态为已退款
		if err := tx.Model(&model.Order{}).Where("id = ?", lockedOrder.ID).Update("status", model.OrderStatusRefunded).Error; err != nil {
			logger.Error("更新订单状态失败", "error", err, "order_id", lockedOrder.ID)
			return fmt.Errorf("更新订单状态失败: %v", err)
		}

		logger.Info("外部订单退款完成",
			"order_id", lockedOrder.ID,
			"order_number", lockedOrder.OrderNumber,
			"out_trade_num", outTradeNum,
			"amount", lockedOrder.Price)

		return nil
	})
}

// GetOrderByOutTradeNum 根据外部交易号获取订单
func (s *orderService) GetOrderByOutTradeNum(ctx context.Context, outTradeNum string) (*model.Order, error) {
	return s.orderRepo.GetByOutTradeNum(ctx, outTradeNum)
}

// ProcessOrderCancel 处理订单取消
func (s *orderService) ProcessOrderCancel(ctx context.Context, orderID int64, remark string) error {
	// 更新备注
	err := s.orderRepo.UpdateRemark(ctx, orderID, remark)
	if err != nil {
		return err
	}

	// 更新订单状态为已取消
	return s.UpdateOrderStatus(ctx, orderID, model.OrderStatusCancelled)
}

// ProcessOrderSplit 处理订单拆单
func (s *orderService) ProcessOrderSplit(ctx context.Context, orderID int64, remark string) error {
	// 更新备注
	err := s.orderRepo.UpdateRemark(ctx, orderID, remark)
	if err != nil {
		return err
	}

	// 更新订单状态为已拆单
	return s.UpdateOrderStatus(ctx, orderID, model.OrderStatusSplit)
}

// ProcessOrderPartial 处理订单部分充值
func (s *orderService) ProcessOrderPartial(ctx context.Context, orderID int64, remark string) error {
	// 更新备注
	err := s.orderRepo.UpdateRemark(ctx, orderID, remark)
	if err != nil {
		return err
	}

	// 更新订单状态为部分充值
	return s.UpdateOrderStatus(ctx, orderID, model.OrderStatusPartial)
}

// GetOrders 获取订单列表
func (s *orderService) GetOrders(ctx context.Context, params map[string]interface{}, page, pageSize int) ([]*model.Order, int64, error) {
	// 如果参数中包含 user_id，说明是代理商查询自己的订单
	if userID, ok := params["user_id"].(int64); ok {
		return s.orderRepo.GetByUserID(ctx, userID, params, page, pageSize)
	}

	// 否则是管理员查询所有订单
	return s.orderRepo.GetOrders(ctx, params, page, pageSize)
}

// GetOrdersWithNotification 获取包含通知信息的订单列表
func (s *orderService) GetOrdersWithNotification(ctx context.Context, params map[string]interface{}, page, pageSize int) ([]*model.OrderWithNotification, int64, error) {
	// 调用仓储层的新方法
	return s.orderRepo.GetOrdersWithNotification(ctx, params, page, pageSize)
}

// CreateExternalOrder 创建外部订单（事务性处理：先验证商品再扣款创建订单）
func (s *orderService) CreateExternalOrder(ctx context.Context, order *model.Order, userID int64) error {
	logger.Info("开始创建外部订单",
		"out_trade_num", order.OutTradeNum,
		"user_id", userID,
		"product_id", order.ProductID)

	// 1. 验证商品是否存在
	product, err := s.productRepo.GetByID(ctx, order.ProductID)
	if err != nil {
		logger.Error("获取商品信息失败",
			"error", err,
			"product_id", order.ProductID)
		return fmt.Errorf("商品不存在: %v", err)
	}

	// 检查商品状态
	if product.Status != 1 {
		logger.Error("商品已下架",
			"product_id", order.ProductID,
			"status", product.Status)
		return fmt.Errorf("商品已下架")
	}

	// 使用商品表的价格
	actualPrice := product.Price
	logger.Info("使用商品表价格",
		"product_id", order.ProductID,
		"product_name", product.Name,
		"actual_price", actualPrice)

	// 开启事务
	tx := s.orderRepo.(*repository.OrderRepositoryImpl).DB().Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			logger.Error("创建外部订单发生panic，事务回滚", "panic", r)
		}
	}()

	if tx.Error != nil {
		logger.Error("开启事务失败", "error", tx.Error)
		return fmt.Errorf("开启事务失败: %v", tx.Error)
	}

	// 2. 智能扣款（优先使用余额，不足时使用授信额度）
	logger.Info("开始智能扣款",
		"user_id", userID,
		"amount", actualPrice)

	// 创建带授信功能的余额服务实例
	balanceService := NewBalanceServiceWithCredit(s.balanceLogRepo, s.userRepo, s.creditService)
	if err := balanceService.SmartDeduct(ctx, userID, actualPrice, model.BalanceStyleOrderDeduct, "外部订单智能扣款", "system"); err != nil {
		tx.Rollback()
		logger.Error("智能扣款失败",
			"error", err,
			"user_id", userID,
			"amount", actualPrice)
		return fmt.Errorf("余额和授信额度均不足: %v", err)
	}

	logger.Info("智能扣款成功",
		"user_id", userID,
		"amount", actualPrice)

	// 3. 创建订单（直接设置为待充值状态，使用商品表价格）
	order.OrderNumber = generateOrderNumber()
	order.CreateTime = time.Now()
	order.UpdatedAt = time.Now()
	order.Status = model.OrderStatusPendingRecharge // 直接设置为待充值状态
	order.CustomerID = userID
	order.IsDel = 0
	order.Price = actualPrice // 使用商品表的价格

	if err := s.orderRepo.Create(ctx, order); err != nil {
		tx.Rollback()
		// 回滚扣款
		if refundErr := balanceService.Refund(ctx, userID, actualPrice, 0, "订单创建失败退款", "system"); refundErr != nil {
			logger.Error("订单创建失败，退款也失败",
				"create_error", err,
				"refund_error", refundErr,
				"user_id", userID,
				"amount", actualPrice)
		} else {
			logger.Info("订单创建失败，已自动退款",
				"user_id", userID,
				"amount", actualPrice)
		}
		return fmt.Errorf("创建订单失败: %v", err)
	}

	logger.Info("订单创建成功",
		"order_id", order.ID,
		"order_number", order.OrderNumber,
		"status", order.Status,
		"actual_price", actualPrice)

	// 4. 更新扣款记录的订单ID（将之前的临时扣款记录关联到具体订单）
	if err := s.updateUserBalanceLogOrderID(ctx, userID, actualPrice, order.ID); err != nil {
		logger.Error("更新扣款记录订单ID失败", "error", err, "order_id", order.ID)
		// 这个错误不影响主流程，只记录日志
	}

	// 5. 推送到充值队列
	if err := s.rechargeService.PushToRechargeQueue(ctx, order.ID); err != nil {
		logger.Error("推送到充值队列失败", "error", err, "order_id", order.ID)
		// 这个错误不影响主流程，只记录日志
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		logger.Error("提交事务失败", "error", err)
		return fmt.Errorf("提交事务失败: %v", err)
	}

	logger.Info("外部订单创建完成",
		"order_id", order.ID,
		"order_number", order.OrderNumber,
		"out_trade_num", order.OutTradeNum,
		"status", order.Status)

	return nil
}

// updateBalanceLogOrderID 更新余额日志的订单ID
func (s *orderService) updateBalanceLogOrderID(ctx context.Context, platformAccountID int64, amount float64, orderID int64) error {
	// 查找最近的扣款记录（订单ID为0的记录）
	db := s.orderRepo.(*repository.OrderRepositoryImpl).DB()
	err := db.Model(&model.BalanceLog{}).
		Where("platform_account_id = ? AND amount = ? AND order_id = ? AND style = ?",
			platformAccountID, amount, 0, model.BalanceStyleOrderDeduct).
		Order("created_at DESC").
		Limit(1).
		Update("order_id", orderID).Error

	if err != nil {
		logger.Error("更新余额日志订单ID失败",
			"error", err,
			"platform_account_id", platformAccountID,
			"amount", amount,
			"order_id", orderID)
		return err
	}

	logger.Info("更新余额日志订单ID成功",
		"platform_account_id", platformAccountID,
		"amount", amount,
		"order_id", orderID)

	return nil
}

// updateUserBalanceLogOrderID 更新用户余额日志的订单ID
func (s *orderService) updateUserBalanceLogOrderID(ctx context.Context, userID int64, amount float64, orderID int64) error {
	// 查找最近的扣款记录（订单ID为0的记录）
	db := s.orderRepo.(*repository.OrderRepositoryImpl).DB()
	err := db.Model(&model.BalanceLog{}).
		Where("user_id = ? AND amount = ? AND order_id = ? AND style = ?",
			userID, -amount, 0, model.BalanceStyleOrderDeduct).
		Order("created_at DESC").
		Limit(1).
		Update("order_id", orderID).Error

	if err != nil {
		logger.Error("更新用户余额日志订单ID失败",
			"error", err,
			"user_id", userID,
			"amount", amount,
			"order_id", orderID)
		return err
	}

	logger.Info("更新用户余额日志订单ID成功",
		"user_id", userID,
		"amount", amount,
		"order_id", orderID)

	return nil
}

// generateOrderNumber 生成订单号
func generateOrderNumber() string {
	return "P" + time.Now().Format("20060102150405") + utils.RandString(6)
}

// SetRechargeService 设置充值服务
func (s *orderService) SetRechargeService(rechargeService RechargeService) {
	s.rechargeService = rechargeService
}

// DeleteOrder 删除订单（软删除）
func (s *orderService) DeleteOrder(ctx context.Context, id string) error {
	logger.Info("开始软删除订单", "order_id", id)
	orderID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		logger.Error("订单ID格式错误", "order_id", id, "error", err)
		return fmt.Errorf("订单ID格式错误: %v", err)
	}
	// 查询订单信息
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		logger.Error("订单不存在", "order_id", id, "error", err)
		return fmt.Errorf("订单不存在: %v", err)
	}
	userID := ctx.Value("user_id").(int64)
	if !isSuperAdmin(ctx) && order.CustomerID != userID {
		logger.Error("无权限删除该订单", "order_id", id, "user_id", userID, "order_customer_id", order.CustomerID)
		return fmt.Errorf("无权限删除该订单")
	}
	if err := s.orderRepo.SoftDeleteByID(ctx, orderID); err != nil {
		logger.Error("软删除订单失败", "order_id", id, "error", err)
		return fmt.Errorf("软删除订单失败: %v", err)
	}
	logger.Info("软删除订单成功", "order_id", id)
	return nil
}

// CleanupOrders 清理指定时间范围的订单及相关日志
func (s *orderService) CleanupOrders(ctx context.Context, start, end string) (int64, error) {
	// 1. 查询要删除的订单ID
	orderIDs, err := s.orderRepo.GetIDsByTimeRange(ctx, start, end)
	if err != nil {
		return 0, err
	}
	if len(orderIDs) == 0 {
		return 0, nil
	}
	// 2. 删除 balance_logs
	if err := s.rechargeService.GetBalanceService().DeleteByOrderIDs(ctx, orderIDs); err != nil {
		return 0, err
	}
	// 3. 删除 orders
	count, err := s.orderRepo.DeleteByIDs(ctx, orderIDs)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// GetProductID 根据价格、ISP和状态获取产品ID
// 支持价格误差容忍（0.01），并输出详细日志
func (s *orderService) GetProductID(price float64, isp int, status int) (*model.Product, error) {
	logger.Info("GetProductID called",
		"price", price,
		"isp", isp,
		"status", status,
	)
	product, err := s.orderRepo.FindProductByPriceAndISPWithTolerance(price, isp, status, 0.01)
	if err != nil {
		logger.Error("未找到匹配的产品",
			"price", price,
			"isp", isp,
			"status", status,
			"error", err,
		)
		return nil, fmt.Errorf("未找到匹配的产品: price=%.2f, isp=%d, status=%d", price, isp, status)
	}
	logger.Info("匹配到产品", "product_id", product.ID, "price", product.Price, "isp", product.ISP, "status", product.Status)
	return product, nil
}

// GetProductByNameValue 根据产品名称数字部分、ISP和状态获取产品
func (s *orderService) GetProductByNameValue(nameValue float64, isp int, status int) (*model.Product, error) {
	logger.Info("GetProductByNameValue called",
		"nameValue", nameValue,
		"isp", isp,
		"status", status,
	)
	
	product, err := s.orderRepo.FindProductByNameValueAndISP(int(nameValue), isp, status)
	if err != nil {
		logger.Error("未找到匹配的产品",
			"nameValue", nameValue,
			"isp", isp,
			"status", status,
			"error", err,
		)
		return nil, fmt.Errorf("未找到匹配的产品: nameValue=%.0f, isp=%d, status=%d", nameValue, isp, status)
	}
	
	logger.Info("找到匹配的产品",
		"productID", product.ID,
		"productName", product.Name,
		"nameValue", nameValue,
	)
	
	return product, nil
}

// GetOrderStatistics 按 customer_id 统计今日订单总数、成功订单数、失败订单数、今日成交金额（Denom 字段）
func (s *orderService) GetOrderStatistics(ctx context.Context, customerID int64) (*OrderStatistics, error) {
	today := time.Now().Format("2006-01-02")
	// loc := time.Local
	// startTime, _ := time.ParseInLocation("2006-01-02", today, loc)
	// endTime := startTime.Add(24 * time.Hour)

	var (
		totalCount      int64
		successCount    int64
		failedCount     int64
		processingCount int64
		successAmount   float64
	)

	db := s.orderRepo.DB().WithContext(ctx).Model(&model.Order{})
	db = db.Where("platform_account_id = ? AND DATE(created_at) = ?", customerID, today).Debug()
	db.Count(&totalCount)
	db.Where("status = ?", model.OrderStatusSuccess).Count(&successCount)
	db.Where("status = ?", model.OrderStatusFailed).Count(&failedCount)
	db.Where("status = ?", model.OrderStatusRecharging).Count(&processingCount)
	db.Select("SUM(price)").Where("status = ?", model.OrderStatusSuccess).Scan(&successAmount)

	return &OrderStatistics{
		TotalCount:      totalCount,
		SuccessCount:    successCount,
		FailedCount:     failedCount,
		ProcessingCount: processingCount,
		SuccessAmount:   successAmount,
	}, nil
}

// SendNotification 发送订单回调通知
func (s *orderService) SendNotification(ctx context.Context, orderID int64) error {
	// 获取订单信息
	order, err := s.GetOrderByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("获取订单失败: %w", err)
	}

	// 创建通知任务
	notification := &notificationModel.NotificationRecord{
		OrderID:          orderID,
		PlatformCode:     "system",
		NotificationType: "order_callback",
		Content:          fmt.Sprintf("订单 %s 回调通知", order.OrderNumber),
		Status:           1, // 待处理
		RetryCount:       0,
		NextRetryTime:    time.Now().Add(5 * time.Minute),
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	// 保存通知记录
	if err := s.notificationRepo.Create(ctx, notification); err != nil {
		return fmt.Errorf("创建通知记录失败: %w", err)
	}

	// 推送到通知队列
	if err := s.queue.Push(ctx, "notification_queue", notification); err != nil {
		return fmt.Errorf("推送到通知队列失败: %w", err)
	}

	logger.Info("订单回调通知已推送到队列", "order_id", orderID, "order_number", order.OrderNumber)
	return nil
}

// GetOrdersByUserID 根据用户ID获取订单列表
func (s *orderService) GetOrdersByUserID(ctx context.Context, userID int64, params map[string]interface{}, page, pageSize int) ([]*model.Order, int64, error) {
	return s.orderRepo.GetByUserID(ctx, userID, params, page, pageSize)
}
