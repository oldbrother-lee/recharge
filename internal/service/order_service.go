package service

import (
	"context"
	"fmt"
	"recharge-go/internal/model"
	notificationModel "recharge-go/internal/model/notification"
	"recharge-go/internal/repository"
	notificationRepo "recharge-go/internal/repository/notification"
	"recharge-go/internal/utils"
	"recharge-go/pkg/logger"
	"recharge-go/pkg/queue"
	"strconv"
	"time"
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
}

// NewOrderService 创建订单服务实例
func NewOrderService(
	orderRepo repository.OrderRepository,
	rechargeService RechargeService,
	notificationRepo notificationRepo.Repository,
	queue queue.Queue,
	balanceLogRepo *repository.BalanceLogRepository,
	userRepo *repository.UserRepository,
	productRepo repository.ProductRepository,
) OrderService {
	return &orderService{
		orderRepo:        orderRepo,
		rechargeService:  rechargeService,
		notificationRepo: notificationRepo,
		queue:            queue,
		balanceLogRepo:   balanceLogRepo,
		userRepo:         userRepo,
		productRepo:      productRepo,
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

	// 在订单状态更新后，添加日志记录通知推送的流程
	logger.Info("准备推送通知到队列", "order_id", order.ID, "status", order.Status)
	err = s.queue.Push(ctx, "notification_queue", notification)
	if err != nil {
		logger.Error("推送通知到队列失败", "order_id", order.ID, "error", err)
	} else {
		logger.Info("推送通知到队列成功", "order_id", order.ID)
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
	// 获取订单信息
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return err
	}

	// 如果订单已经支付，需要退还余额
	if order.Status == model.OrderStatusPendingRecharge || order.Status == model.OrderStatusRecharging || order.Status == model.OrderStatusProcessing {
		// 检查是否为外部订单
		if order.Client == 2 {
			// 外部订单直接退款到用户余额
			logger.Info("外部订单失败，退款到用户余额",
				"order_id", orderID,
				"customer_id", order.CustomerID,
				"amount", order.Price)

			// 使用用户余额服务进行退款
			balanceService := s.rechargeService.GetUserBalanceService()
			if err := balanceService.Refund(ctx, order.CustomerID, order.Price, orderID, "外部订单失败退款", "system"); err != nil {
				logger.Error("外部订单退款失败",
					"error", err,
					"order_id", orderID,
					"customer_id", order.CustomerID,
					"amount", order.Price)
				return fmt.Errorf("外部订单退款失败: %v", err)
			}
			logger.Info("外部订单退款成功",
				"order_id", orderID,
				"customer_id", order.CustomerID,
				"amount", order.Price)
		} else {
			// 平台订单使用原有的退款方法
			if err := s.rechargeService.GetBalanceService().RefundBalance(ctx, nil, order.PlatformAccountID, order.Price, orderID, "订单失败退还余额"); err != nil {
				logger.Error("退还余额失败",
					"error", err,
					"order_id", orderID,
					"amount", order.Price)
				return fmt.Errorf("refund balance failed: %v", err)
			}
			logger.Info("退还余额成功",
				"order_id", orderID,
				"amount", order.Price)
		}
	}

	// 更新备注
	if err := s.orderRepo.UpdateRemark(ctx, orderID, remark); err != nil {
		return err
	}

	// 更新订单状态为失败
	return s.UpdateOrderStatus(ctx, orderID, model.OrderStatusFailed)
}

// ProcessOrderRefund 处理订单退款
func (s *orderService) ProcessOrderRefund(ctx context.Context, orderID int64, remark string) error {


	// 1. 获取订单信息
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		logger.Error("获取订单失败", "error", err, "order_id", orderID)
		return fmt.Errorf("订单不存在")
	}

	// 2. 检查订单状态是否允许退款
	if order.Status == model.OrderStatusRefunded {
		logger.Info("订单已退款，跳过处理", "order_id", orderID)
		return fmt.Errorf("订单已退款")
	}

	// 只有成功、失败、待充值状态的订单可以退款
	if order.Status != model.OrderStatusSuccess &&
		order.Status != model.OrderStatusFailed &&
		order.Status != model.OrderStatusPendingRecharge {
		logger.Error("订单状态不允许退款", "order_id", orderID, "status", order.Status)
		return fmt.Errorf("订单状态不允许退款")
	}

	// 3. 执行退款逻辑
	if order.Client == 2 {
		// 外部订单退款到用户余额
		balanceService := NewBalanceService(s.balanceLogRepo, s.userRepo)
		if err := balanceService.Refund(ctx, order.CustomerID, order.Price, orderID, fmt.Sprintf("订单退款: %s", remark), "admin"); err != nil {
			logger.Error("外部订单退款失败", "error", err, "order_id", orderID)
			return fmt.Errorf("退款失败: %v", err)
		}
		logger.Info("外部订单退款成功", "order_id", orderID, "amount", order.Price)
	} else {
		// 平台订单退款到平台账户
		if err := s.rechargeService.GetBalanceService().RefundBalance(ctx, nil, order.PlatformAccountID, order.Price, orderID, fmt.Sprintf("订单退款: %s", remark)); err != nil {
			logger.Error("平台订单退款失败", "error", err, "order_id", orderID)
			return fmt.Errorf("退款失败: %v", err)
		}
		logger.Info("平台订单退款成功", "order_id", orderID, "amount", order.Price)
	}

	// 4. 更新备注
	if err := s.orderRepo.UpdateRemark(ctx, orderID, remark); err != nil {
		return err
	}

	// 5. 更新订单状态为已退款
	return s.UpdateOrderStatus(ctx, orderID, model.OrderStatusRefunded)
}

// ProcessExternalRefund 处理外部订单退款
func (s *orderService) ProcessExternalRefund(ctx context.Context, outTradeNum string, reason string) error {
	logger.Info("开始处理外部订单退款",
		"out_trade_num", outTradeNum,
		"reason", reason)

	// 1. 根据外部交易号获取订单
	order, err := s.GetOrderByOutTradeNum(ctx, outTradeNum)
	if err != nil {
		logger.Error("获取订单失败",
			"error", err,
			"out_trade_num", outTradeNum)
		return fmt.Errorf("订单不存在")
	}

	// 2. 检查订单状态是否允许退款
	if order.Status == model.OrderStatusRefunded {
		logger.Info("订单已退款，跳过处理",
			"order_id", order.ID,
			"out_trade_num", outTradeNum)
		return fmt.Errorf("订单已退款")
	}

	// 只有成功、失败、待充值状态的订单可以退款
	if order.Status != model.OrderStatusSuccess &&
		order.Status != model.OrderStatusFailed &&
		order.Status != model.OrderStatusPendingRecharge {
		logger.Error("订单状态不允许退款",
			"order_id", order.ID,
			"status", order.Status,
			"out_trade_num", outTradeNum)
		return fmt.Errorf("订单状态不允许退款")
	}

	// 3. 检查是否为外部订单
	if order.Client != 2 {
		logger.Error("非外部订单，不能使用此退款方法",
			"order_id", order.ID,
			"client", order.Client,
			"out_trade_num", outTradeNum)
		return fmt.Errorf("非外部订单")
	}

	// 开启事务
	tx := s.orderRepo.(*repository.OrderRepositoryImpl).DB().Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			logger.Error("外部订单退款发生panic，事务回滚", "panic", r)
		}
	}()

	if tx.Error != nil {
		logger.Error("开启事务失败", "error", tx.Error)
		return fmt.Errorf("开启事务失败: %v", tx.Error)
	}

	// 4. 直接退款到用户余额（外部订单使用用户余额系统）
	balanceService := NewBalanceService(s.balanceLogRepo, s.userRepo)
	if err := balanceService.Refund(ctx, order.CustomerID, order.Price, order.ID, fmt.Sprintf("外部订单退款: %s", reason), "system"); err != nil {
		tx.Rollback()
		logger.Error("退款到用户余额失败",
			"error", err,
			"order_id", order.ID,
			"customer_id", order.CustomerID,
			"amount", order.Price)
		return fmt.Errorf("退款失败: %v", err)
	}

	logger.Info("退款到用户余额成功",
		"order_id", order.ID,
		"customer_id", order.CustomerID,
		"amount", order.Price)

	// 5. 更新订单备注
	if err := s.orderRepo.UpdateRemark(ctx, order.ID, fmt.Sprintf("外部订单退款: %s", reason)); err != nil {
		tx.Rollback()
		logger.Error("更新订单备注失败", "error", err, "order_id", order.ID)
		return fmt.Errorf("更新订单备注失败: %v", err)
	}

	// 6. 更新订单状态为已退款
	if err := s.orderRepo.UpdateStatus(ctx, order.ID, model.OrderStatusRefunded); err != nil {
		tx.Rollback()
		logger.Error("更新订单状态失败", "error", err, "order_id", order.ID)
		return fmt.Errorf("更新订单状态失败: %v", err)
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		logger.Error("提交事务失败", "error", err)
		return fmt.Errorf("提交事务失败: %v", err)
	}

	logger.Info("外部订单退款完成",
		"order_id", order.ID,
		"order_number", order.OrderNumber,
		"out_trade_num", outTradeNum,
		"amount", order.Price)

	return nil
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

	// 2. 从用户余额扣款（使用商品表的价格）
	logger.Info("开始扣除用户余额",
		"user_id", userID,
		"amount", actualPrice)

	// 创建余额服务实例
	balanceService := NewBalanceService(s.balanceLogRepo, s.userRepo)
	if err := balanceService.Deduct(ctx, userID, actualPrice, model.BalanceStyleOrderDeduct, "外部订单预扣款", "system"); err != nil {
		tx.Rollback()
		logger.Error("扣除用户余额失败",
			"error", err,
			"user_id", userID,
			"amount", actualPrice)
		return fmt.Errorf("余额不足: %v", err)
	}

	logger.Info("扣除用户余额成功",
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

// GetOrdersByUserID 根据用户ID获取订单列表
func (s *orderService) GetOrdersByUserID(ctx context.Context, userID int64, params map[string]interface{}, page, pageSize int) ([]*model.Order, int64, error) {
	return s.orderRepo.GetByUserID(ctx, userID, params, page, pageSize)
}
