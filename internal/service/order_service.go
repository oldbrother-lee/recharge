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
	// ProcessOrderCancel 处理订单取消
	ProcessOrderCancel(ctx context.Context, orderID int64, remark string) error
	// ProcessOrderSplit 处理订单拆单
	ProcessOrderSplit(ctx context.Context, orderID int64, remark string) error
	// ProcessOrderPartial 处理订单部分充值
	ProcessOrderPartial(ctx context.Context, orderID int64, remark string) error
	// GetOrderByOutTradeNum 根据外部交易号获取订单
	GetOrderByOutTradeNum(ctx context.Context, outTradeNum string) (*model.Order, error)
	// GetOrders 获取订单列表
	GetOrders(ctx context.Context, params map[string]interface{}, page, pageSize int) ([]*model.Order, int64, error)
	// SetRechargeService 设置充值服务
	SetRechargeService(rechargeService RechargeService)
	// DeleteOrder 删除订单（软删除）
	DeleteOrder(ctx context.Context, id string) error
	// CleanupOrders 清理指定时间范围的订单及相关日志
	CleanupOrders(ctx context.Context, start, end string) (int64, error)
	// GetProductID 根据价格、ISP和状态获取产品ID
	GetProductID(price float64, isp int, status int) (int64, error)
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
}

// NewOrderService 创建订单服务实例
func NewOrderService(
	orderRepo repository.OrderRepository,
	rechargeService RechargeService,
	notificationRepo notificationRepo.Repository,
	queue queue.Queue,
) OrderService {
	return &orderService{
		orderRepo:        orderRepo,
		rechargeService:  rechargeService,
		notificationRepo: notificationRepo,
		queue:            queue,
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
	userID := ctx.Value("user_id").(int64)
	logger.Info("开始更新订单状态",
		"order_id", id,
		"new_status", status,
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

	// 权限校验：只有超级管理员才能操作
	if !isSuperAdmin(ctx) && order.CustomerID != userID {
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
	if order.Status == model.OrderStatusPendingRecharge || order.Status == model.OrderStatusRecharging {
		// 退还余额
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

	// 更新备注
	if err := s.orderRepo.UpdateRemark(ctx, orderID, remark); err != nil {
		return err
	}

	// 更新订单状态为失败
	return s.UpdateOrderStatus(ctx, orderID, model.OrderStatusFailed)
}

// ProcessOrderRefund 处理订单退款
func (s *orderService) ProcessOrderRefund(ctx context.Context, orderID int64, remark string) error {
	// 更新备注
	err := s.orderRepo.UpdateRemark(ctx, orderID, remark)
	if err != nil {
		return err
	}

	// 更新订单状态为已退款
	return s.UpdateOrderStatus(ctx, orderID, model.OrderStatusRefunded)
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

// GetOrderByOutTradeNum 根据外部交易号获取订单
func (s *orderService) GetOrderByOutTradeNum(ctx context.Context, outTradeNum string) (*model.Order, error) {
	return s.orderRepo.GetOrderByOutTradeNum(ctx, outTradeNum)
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
func (s *orderService) GetProductID(price float64, isp int, status int) (int64, error) {
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
		return 0, fmt.Errorf("未找到匹配的产品: price=%.2f, isp=%d, status=%d", price, isp, status)
	}
	logger.Info("匹配到产品", "product_id", product.ID, "price", product.Price, "isp", product.ISP, "status", product.Status)
	return product.ID, nil
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
