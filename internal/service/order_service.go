package service

import (
	"context"
	"encoding/json"
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
	"strings"
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
	// GetSuccessStatsByIspDenom 按运营商与面值统计成功订单数与金额
	GetSuccessStatsByIspDenom(ctx context.Context, params map[string]interface{}) ([]model.IspDenomSuccessStat, error)
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
	orderRepo              repository.OrderRepository
	rechargeService        RechargeService
	notificationRepo       notificationRepo.Repository
	queue                  queue.Queue
	balanceLogRepo         *repository.BalanceLogRepository
	userRepo               *repository.UserRepository
	productRepo            repository.ProductRepository
	unifiedRefundService   *UnifiedRefundService
	lockManager            *lock.RefundLockManager
	db                     *gorm.DB
	creditService          *CreditService
	balanceQueryRecordRepo repository.BalanceQueryRecordRepository
	notificationHelper     *NotificationHelper
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
	balanceQueryRecordRepo repository.BalanceQueryRecordRepository,
) OrderService {
	return &orderService{
		orderRepo:              orderRepo,
		rechargeService:        rechargeService,
		notificationRepo:       notificationRepo,
		queue:                  queue,
		balanceLogRepo:         balanceLogRepo,
		userRepo:               userRepo,
		productRepo:            productRepo,
		unifiedRefundService:   unifiedRefundService,
		lockManager:            lockManager,
		db:                     db,
		creditService:          creditService,
		balanceQueryRecordRepo: balanceQueryRecordRepo,
		notificationHelper:     NewNotificationHelper(db, notificationRepo, queue),
	}
}

// CreateOrder 创建订单
func (s *orderService) CreateOrder(ctx context.Context, order *model.Order) error {
	// 生成订单号
	order.OrderNumber = generateOrderNumber()
	order.CreateTime = time.Now()
	order.UpdatedAt = time.Now()

	// 注入订单号到上下文，贯穿全链路日志
	ctx = logger.InjectOrderNumber(ctx, order.OrderNumber)
	logger.WithContextCategory(ctx, "order").Info(fmt.Sprintf("【创建订单】生成订单号并初始化状态 | order_number=%s client=%d", order.OrderNumber, order.Client))

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

	// 自动取单订单创建后，发送“接单/处理中”预上报通知
	if order.Client == 3 {
		if notifyErr := s.notificationHelper.SendOrderStatusNotification(ctx, order, model.OrderStatusProcessing); notifyErr != nil {
			logger.WithContextCategory(ctx, "order").Error("【预上报通知创建失败】", logger.ErrorV2(notifyErr))
		} else {
			logger.WithContextCategory(ctx, "order").Info("【已创建预上报通知】", logger.Int64V2("order_id", order.ID))
		}
	}

	// 创建成功后，将订单推送到充值队列
	if err := s.rechargeService.PushToRechargeQueue(ctx, order.ID); err != nil {
		logger.WithContextCategory(ctx, "order").Error(fmt.Sprintf("【推送订单到充值队列失败】order_id=%d error=%v", order.ID, err))
		// 这里可以选择是否返回错误，因为订单已经创建成功
	} else {
		logger.WithContextCategory(ctx, "order").Info(fmt.Sprintf("【已推送订单到充值队列】order_id=%d", order.ID))
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
	logger.WithContextCategory(ctx, "order").Info("开始更新订单状态",
		logger.Int64V2("order_id", id),
		logger.IntV2("new_status", int(status)),
		logger.Int64V2("user_id", userID),
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
		logger.WithContextCategory(ctx, "order").Error("获取订单信息失败",
			logger.ErrorV2(err),
			logger.Int64V2("order_id", id),
		)
		return fmt.Errorf("get order failed: %v", err)
	}

	// 权限校验：超级管理员、系统操作（user_id为0且无roles）或订单所有者可以操作
	isSystemOperation := userID == 0 && ctx.Value("roles") == nil
	if !isSuperAdmin(ctx) && !isSystemOperation && order.CustomerID != userID {
		tx.Rollback()
		logger.WithContextCategory(ctx, "order").Error("无权限操作该订单",
			logger.Int64V2("order_id", id),
			logger.Int64V2("user_id", userID),
			logger.Int64V2("order_customer_id", order.CustomerID),
		)
		return fmt.Errorf("无权限操作该订单")
	}

	logger.WithContextCategory(ctx, "order").Info("获取到订单信息",
		logger.Int64V2("order_id", id),
		logger.IntV2("current_status", int(order.Status)),
		logger.IntV2("new_status", int(status)),
	)

	// 如果状态没有变化，也需要触发通知逻辑（幂等）
	if order.Status == status {
		tx.Rollback()
		logger.WithContextCategory(ctx, "order").Info("订单状态未发生变化，触发幂等通知逻辑",
			logger.Int64V2("order_id", id),
			logger.IntV2("status", int(status)),
		)

		// 幂等性检查：是否已存在相同(order_id, type, target_status)的通知记录
		existing, _, listErr := s.notificationRepo.List(ctx, map[string]interface{}{
			"order_id":          id,
			"notification_type": "order_status_changed",
			"target_status":     int(status),
		}, 1, 10)
		if listErr == nil && len(existing) > 0 {
			for _, n := range existing {
				switch n.Status {
				case 3: // 成功
					logger.WithContextCategory(ctx, "order").Info("已存在成功通知，跳过创建",
						logger.Int64V2("order_id", id),
						logger.Int64V2("notification_id", n.ID),
					)
					return nil
				case 1, 2: // 待处理或处理中
					logger.WithContextCategory(ctx, "order").Info("已存在待处理/处理中通知，尝试重新推送到队列",
						logger.Int64V2("order_id", id),
						logger.Int64V2("notification_id", n.ID),
					)
					if pushErr := s.queue.Push(ctx, "notification_queue", n); pushErr != nil {
						logger.WithContextCategory(ctx, "order").Error("重新推送通知失败",
							logger.Int64V2("order_id", id),
							logger.Int64V2("notification_id", n.ID),
							logger.ErrorV2(pushErr),
						)
					}
					return nil
				case 4: // 失败，重置后重推
					logger.WithContextCategory(ctx, "order").Info("存在失败通知，重置为待处理并重新推送",
						logger.Int64V2("order_id", id),
						logger.Int64V2("notification_id", n.ID),
					)
					if upErr := s.notificationRepo.UpdateStatus(ctx, n.ID, 1); upErr != nil {
						logger.WithContextCategory(ctx, "order").Error("重置通知状态失败",
							logger.Int64V2("order_id", id),
							logger.Int64V2("notification_id", n.ID),
							logger.ErrorV2(upErr),
						)
					} else {
						n.Status = 1
						if pushErr := s.queue.Push(ctx, "notification_queue", n); pushErr != nil {
							logger.WithContextCategory(ctx, "order").Error("重新推送通知失败",
								logger.Int64V2("order_id", id),
								logger.Int64V2("notification_id", n.ID),
								logger.ErrorV2(pushErr),
							)
						}
					}
					return nil
				}
			}
		} else if listErr != nil {
			logger.WithContextCategory(ctx, "order").Warn("幂等检查查询通知记录失败，继续创建",
				logger.Int64V2("order_id", id),
				logger.ErrorV2(listErr),
			)
		}

		// 序列化订单快照
		orderData, err := json.Marshal(order)
		if err != nil {
			logger.WithContextCategory(ctx, "order").Error("序列化订单快照失败",
				logger.Int64V2("order_id", id),
				logger.ErrorV2(err),
			)
			return nil // 订单状态已是目标状态，通知推送失败不影响主流程
		}

		// 创建通知记录（包含订单快照）
		notification := &notificationModel.NotificationRecord{
			OrderID:          id,
			PlatformCode:     order.PlatformCode,
			NotificationType: "order_status_changed",
			Content:          fmt.Sprintf("订单状态已更新为: %d", status),
			OrderSnapshot:    string(orderData), // 保存完整订单快照
			TargetStatus:     int(status),       // 保存目标状态
			Status:           1,                 // 待处理
		}

		// 保存通知记录到数据库（容错处理唯一键冲突）
		if createErr := s.notificationRepo.Create(ctx, notification); createErr != nil {
			if strings.Contains(createErr.Error(), "Duplicate entry") || strings.Contains(createErr.Error(), "UNIQUE constraint") {
				logger.WithContextCategory(ctx, "order").Warn("通知记录已存在（唯一键冲突），可能并发创建",
					logger.Int64V2("order_id", id),
					logger.IntV2("target_status", int(status)),
				)
				return nil
			}
			logger.WithContextCategory(ctx, "order").Error("创建通知记录失败",
				logger.Int64V2("order_id", id),
				logger.ErrorV2(createErr),
			)
			return nil // 订单状态已是目标状态，通知推送失败不影响主流程
		}

		// 推送通知到队列
		logger.WithContextCategory(ctx, "order").Info("准备推送通知到队列",
			logger.Int64V2("order_id", id),
			logger.IntV2("new_status", int(status)),
			logger.Int64V2("notification_id", notification.ID),
		)
		if pushErr := s.queue.Push(ctx, "notification_queue", notification); pushErr != nil {
			logger.WithContextCategory(ctx, "order").Error("推送通知到队列失败",
				logger.Int64V2("order_id", id),
				logger.Int64V2("notification_id", notification.ID),
				logger.ErrorV2(pushErr),
			)
		} else {
			logger.WithContextCategory(ctx, "order").Info("推送通知到队列成功",
				logger.Int64V2("order_id", id),
				logger.Int64V2("notification_id", notification.ID),
				logger.IntV2("status", int(status)),
			)
		}
		return nil
	}

	// 更新订单状态
	if err := tx.Model(&model.Order{}).Where("id = ?", id).Update("status", status).Error; err != nil {
		tx.Rollback()
		logger.WithContextCategory(ctx, "order").Error("更新订单状态失败",
			logger.ErrorV2(err),
			logger.Int64V2("order_id", id),
			logger.IntV2("old_status", int(order.Status)),
			logger.IntV2("new_status", int(status)),
		)
		return fmt.Errorf("update order status failed: %v", err)
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		logger.WithContextCategory(ctx, "order").Error("提交事务失败",
			logger.ErrorV2(err),
			logger.Int64V2("order_id", id),
		)
		return fmt.Errorf("commit transaction failed: %v", err)
	}

	logger.WithContextCategory(ctx, "order").Info("订单状态更新成功",
		logger.Int64V2("order_id", id),
		logger.IntV2("old_status", int(order.Status)),
		logger.IntV2("new_status", int(status)),
	)

	// 事务提交成功后，获取更新后的订单信息并创建通知记录
	updatedOrder, getErr := s.orderRepo.GetByID(ctx, id)
	if getErr != nil {
		logger.WithContextCategory(ctx, "order").Error("获取更新后的订单信息失败",
			logger.Int64V2("order_id", id),
			logger.ErrorV2(getErr),
		)
		return nil // 订单状态已更新成功，通知推送失败不影响主流程
	}

	// 幂等性检查：是否已存在相同(order_id, type, target_status)的通知记录
	existing, _, listErr := s.notificationRepo.List(ctx, map[string]interface{}{
		"order_id":          id,
		"notification_type": "order_status_changed",
		"target_status":     int(status),
	}, 1, 10)
	if listErr == nil && len(existing) > 0 {
		for _, n := range existing {
			switch n.Status {
			case 3: // 成功
				logger.WithContextCategory(ctx, "order").Info("已存在成功通知，跳过创建",
					logger.Int64V2("order_id", id),
					logger.Int64V2("notification_id", n.ID),
				)
				return nil
			case 1, 2: // 待处理或处理中
				logger.WithContextCategory(ctx, "order").Info("已存在待处理/处理中通知，尝试重新推送到队列",
					logger.Int64V2("order_id", id),
					logger.Int64V2("notification_id", n.ID),
				)
				if pushErr := s.queue.Push(ctx, "notification_queue", n); pushErr != nil {
					logger.WithContextCategory(ctx, "order").Error("重新推送通知失败",
						logger.Int64V2("order_id", id),
						logger.Int64V2("notification_id", n.ID),
						logger.ErrorV2(pushErr),
					)
				}
				return nil
			case 4: // 失败，重置后重推
				logger.WithContextCategory(ctx, "order").Info("存在失败通知，重置为待处理并重新推送",
					logger.Int64V2("order_id", id),
					logger.Int64V2("notification_id", n.ID),
				)
				if upErr := s.notificationRepo.UpdateStatus(ctx, n.ID, 1); upErr != nil {
					logger.WithContextCategory(ctx, "order").Error("重置通知状态失败",
						logger.Int64V2("order_id", id),
						logger.Int64V2("notification_id", n.ID),
						logger.ErrorV2(upErr),
					)
				} else {
					n.Status = 1
					if pushErr := s.queue.Push(ctx, "notification_queue", n); pushErr != nil {
						logger.WithContextCategory(ctx, "order").Error("重新推送通知失败",
							logger.Int64V2("order_id", id),
							logger.Int64V2("notification_id", n.ID),
							logger.ErrorV2(pushErr),
						)
					}
				}
				return nil
			}
		}
	} else if listErr != nil {
		logger.WithContextCategory(ctx, "order").Warn("幂等检查查询通知记录失败，继续创建",
			logger.Int64V2("order_id", id),
			logger.ErrorV2(listErr),
		)
	}

	// 序列化订单快照
	orderData, err := json.Marshal(updatedOrder)
	if err != nil {
		logger.WithContextCategory(ctx, "order").Error("序列化订单快照失败",
			logger.Int64V2("order_id", id),
			logger.ErrorV2(err),
		)
		return nil // 订单状态已更新成功，通知推送失败不影响主流程
	}

	// 创建通知记录（包含订单快照）
	notification := &notificationModel.NotificationRecord{
		OrderID:          id,
		PlatformCode:     updatedOrder.PlatformCode,
		NotificationType: "order_status_changed",
		Content:          fmt.Sprintf("订单状态已更新为: %d", status),
		OrderSnapshot:    string(orderData), // 保存完整订单快照
		TargetStatus:     int(status),       // 保存目标状态
		Status:           1,                 // 待处理
	}

	// 保存通知记录到数据库（容错处理唯一键冲突）
	if createErr := s.notificationRepo.Create(ctx, notification); createErr != nil {
		if strings.Contains(createErr.Error(), "Duplicate entry") || strings.Contains(createErr.Error(), "UNIQUE constraint") {
			logger.WithContextCategory(ctx, "order").Warn("通知记录已存在（唯一键冲突），可能并发创建",
				logger.Int64V2("order_id", id),
				logger.IntV2("target_status", int(status)),
			)
			return nil
		}
		logger.WithContextCategory(ctx, "order").Error("创建通知记录失败",
			logger.Int64V2("order_id", id),
			logger.ErrorV2(createErr),
		)
		return nil // 订单状态已更新成功，通知推送失败不影响主流程
	}

	// 推送通知到队列
	logger.WithContextCategory(ctx, "order").Info("准备推送通知到队列",
		logger.Int64V2("order_id", id),
		logger.IntV2("new_status", int(status)),
		logger.Int64V2("notification_id", notification.ID),
	)
	if pushErr := s.queue.Push(ctx, "notification_queue", notification); pushErr != nil {
		logger.WithContextCategory(ctx, "order").Error("推送通知到队列失败",
			logger.Int64V2("order_id", id),
			logger.Int64V2("notification_id", notification.ID),
			logger.ErrorV2(pushErr),
		)
	} else {
		logger.WithContextCategory(ctx, "order").Info("推送通知到队列成功",
			logger.Int64V2("order_id", id),
			logger.Int64V2("notification_id", notification.ID),
			logger.IntV2("status", int(status)),
		)
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
	logger.WithContextCategory(ctx, "order").Info("开始处理订单失败", logger.Int64V2("order_id", orderID), logger.StringV2("remark", remark))

	// 1. 先获取订单信息以确定用户ID
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		logger.WithContextCategory(ctx, "order").Error("获取订单信息失败", logger.Int64V2("order_id", orderID), logger.ErrorV2(err))
		return fmt.Errorf("获取订单信息失败: %v", err)
	}
	logger.WithContextCategory(ctx, "order").Info("获取订单信息成功", logger.Int64V2("order_id", orderID), logger.Int64V2("customer_id", order.CustomerID), logger.IntV2("status", int(order.Status)))

	// 2. 如果订单已经是失败状态，直接调用 UpdateOrderStatus 确保通知发送的幂等性
	if order.Status == model.OrderStatusFailed {
		logger.WithContextCategory(ctx, "order").Info("订单已经是失败状态，调用 UpdateOrderStatus 确保通知幂等性", logger.Int64V2("order_id", orderID))
		return s.UpdateOrderStatus(ctx, orderID, model.OrderStatusFailed)
	}

	// 3. 获取用户级别的分布式锁
	lockValue, err := s.lockManager.LockUserRefund(ctx, order.CustomerID)
	if err != nil {
		logger.WithContextCategory(ctx, "order").Error("获取用户退款锁失败", logger.Int64V2("user_id", order.CustomerID), logger.Int64V2("order_id", orderID), logger.ErrorV2(err))
		return fmt.Errorf("获取退款锁失败: %v", err)
	}
	defer func() {
		if unlockErr := s.lockManager.UnlockUserRefund(ctx, order.CustomerID, lockValue); unlockErr != nil {
			logger.WithContextCategory(ctx, "order").Error("释放用户退款锁失败", logger.Int64V2("user_id", order.CustomerID), logger.Int64V2("order_id", orderID), logger.ErrorV2(unlockErr))
		}
	}()

	// 4. 在锁保护下执行事务（只处理业务逻辑，不创建通知）
	logger.WithContextCategory(ctx, "order").Info("开始执行事务", logger.Int64V2("order_id", orderID))
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		logger.WithContextCategory(ctx, "order").Info("事务内部开始执行", logger.Int64V2("order_id", orderID))
		// 使用行锁防止同一订单的并发处理
		var lockedOrder model.Order
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("id = ?", orderID).First(&lockedOrder).Error; err != nil {
			logger.WithContextCategory(ctx, "order").Error("获取订单行锁失败", logger.Int64V2("order_id", orderID), logger.ErrorV2(err))
			return err
		}
		logger.WithContextCategory(ctx, "order").Info("获取订单行锁成功", logger.Int64V2("order_id", orderID), logger.IntV2("locked_status", int(lockedOrder.Status)))

		// 检查订单状态，防止重复处理
		if lockedOrder.Status == model.OrderStatusFailed {
			// 订单已经是失败状态，跳过处理
			logger.WithContextCategory(ctx, "order").Info("订单已经是失败状态，跳过重复处理", logger.Int64V2("order_id", orderID))
			return nil
		}

		// 如果订单已经支付，需要退还余额
		if lockedOrder.Status == model.OrderStatusPendingRecharge || lockedOrder.Status == model.OrderStatusRecharging || lockedOrder.Status == model.OrderStatusProcessing {
			// 使用统一退款服务处理退款
			var refundReq *RefundRequest
			if lockedOrder.Client == 2 {
				// 外部订单直接退款到用户余额
				logger.WithContextCategory(ctx, "order").Info("外部订单失败，使用统一退款服务退款到用户余额",
					logger.Int64V2("order_id", orderID),
					logger.Int64V2("customer_id", lockedOrder.CustomerID),
					logger.Float64V2("amount", lockedOrder.Price))

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
				logger.WithContextCategory(ctx, "order").Info("平台订单失败，使用统一退款服务退款",
					logger.Int64V2("order_id", orderID),
					logger.Int64V2("customer_id", lockedOrder.CustomerID),
					logger.Int64V2("platform_account_id", lockedOrder.PlatformAccountID),
					logger.Float64V2("amount", lockedOrder.Price))

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
				logger.WithContextCategory(ctx, "order").Error("统一退款服务退款失败",
					logger.ErrorV2(err),
					logger.Int64V2("order_id", orderID),
					logger.Int64V2("customer_id", lockedOrder.CustomerID),
					logger.Float64V2("amount", lockedOrder.Price),
					logger.AnyV2("response", refundResp),
				)
				if err != nil {
					return fmt.Errorf("统一退款失败: %v", err)
				}
				return fmt.Errorf("统一退款失败: %s", refundResp.Message)
			}

			logger.WithContextCategory(ctx, "order").Info("统一退款服务退款成功",
				logger.Int64V2("order_id", orderID),
				logger.Int64V2("customer_id", lockedOrder.CustomerID),
				logger.Float64V2("amount", refundResp.RefundAmount),
				logger.Float64V2("balance_after", refundResp.BalanceAfter),
				logger.BoolV2("already_refund", refundResp.AlreadyRefund),
			)
		}

		// 更新备注
		if err := tx.Model(&model.Order{}).Where("id = ?", orderID).Update("remark", remark).Error; err != nil {
			return err
		}

		// 更新订单状态为失败
		if err := tx.Model(&model.Order{}).Where("id = ?", orderID).Update("status", model.OrderStatusFailed).Error; err != nil {
			return err
		}

		logger.WithContextCategory(ctx, "order").Info("订单失败处理完成",
			logger.Int64V2("order_id", orderID),
			logger.IntV2("status", int(model.OrderStatusFailed)))

		logger.WithContextCategory(ctx, "order").Info("事务内部执行完成", logger.Int64V2("order_id", orderID))
		return nil
	})

	logger.WithContextCategory(ctx, "order").Info("事务执行结果", logger.Int64V2("order_id", orderID), logger.ErrorV2(err))

	// 5. 事务提交成功后，使用 UpdateOrderStatus 统一处理状态变更通知（含幂等保护）
	if err == nil {
		logger.WithContextCategory(ctx, "order").Info("事务提交成功，调用 UpdateOrderStatus 发送通知", logger.Int64V2("order_id", orderID))
		// 使用 UpdateOrderStatus 方法统一处理状态变更通知，该方法已包含完善的幂等保护
		if notifyErr := s.UpdateOrderStatus(ctx, orderID, model.OrderStatusFailed); notifyErr != nil {
			logger.WithContextCategory(ctx, "order").Error("调用 UpdateOrderStatus 发送通知失败", logger.Int64V2("order_id", orderID), logger.ErrorV2(notifyErr))
			// 通知失败不影响订单状态已成功更新的结果
		} else {
			logger.WithContextCategory(ctx, "order").Info("调用 UpdateOrderStatus 发送通知成功", logger.Int64V2("order_id", orderID))
		}
	} else {
		logger.WithContextCategory(ctx, "order").Error("事务执行失败，跳过通知发送", logger.Int64V2("order_id", orderID), logger.ErrorV2(err))
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
			logger.WithContextCategory(ctx, "order").Error("获取订单失败", logger.ErrorV2(err), logger.Int64V2("order_id", orderID))
			return fmt.Errorf("订单不存在")
		}

		// 2. 检查订单状态是否允许退款
		if lockedOrder.Status == model.OrderStatusRefunded {
			logger.WithContextCategory(ctx, "order").Info("订单已退款，跳过处理", logger.Int64V2("order_id", orderID))
			return fmt.Errorf("订单已退款")
		}

		// 只有成功、失败、待充值状态的订单可以退款
		if lockedOrder.Status != model.OrderStatusSuccess &&
			lockedOrder.Status != model.OrderStatusFailed &&
			lockedOrder.Status != model.OrderStatusPendingRecharge {
			logger.WithContextCategory(ctx, "order").Error("订单状态不允许退款", logger.Int64V2("order_id", orderID), logger.IntV2("status", int(lockedOrder.Status)))
			return fmt.Errorf("订单状态不允许退款")
		}

		// 3. 执行退款逻辑
		if lockedOrder.Client == 2 {
			// 外部订单退款到用户余额（使用当前事务）
			balanceService := NewBalanceService(s.balanceLogRepo, s.userRepo)
			if err := balanceService.RefundWithTx(ctx, tx, lockedOrder.CustomerID, lockedOrder.Price, orderID, fmt.Sprintf("订单退款: %s", remark), "admin"); err != nil {
				logger.WithContextCategory(ctx, "order").Error("外部订单退款失败", logger.ErrorV2(err), logger.Int64V2("order_id", orderID))
				return fmt.Errorf("退款失败: %v", err)
			}
			logger.WithContextCategory(ctx, "order").Info("外部订单退款成功", logger.Int64V2("order_id", orderID), logger.Float64V2("amount", lockedOrder.Price))
		} else {
			// 平台订单退款到用户余额
			if err := s.rechargeService.GetUserBalanceService().RefundWithTx(ctx, tx, lockedOrder.CustomerID, lockedOrder.Price, orderID, fmt.Sprintf("订单退款: %s", remark), "system"); err != nil {
				logger.WithContextCategory(ctx, "order").Error("平台订单退款失败", logger.ErrorV2(err), logger.Int64V2("order_id", orderID))
				return fmt.Errorf("退款失败: %v", err)
			}
			logger.WithContextCategory(ctx, "order").Info("平台订单退款成功", logger.Int64V2("order_id", orderID), logger.Float64V2("amount", lockedOrder.Price))
		}

		// 4. 更新备注
		if err := tx.Model(&model.Order{}).Where("id = ?", orderID).Update("remark", remark).Error; err != nil {
			return err
		}

		// 5. 更新订单状态为已退款
		if err := tx.Model(&model.Order{}).Where("id = ?", orderID).Update("status", model.OrderStatusRefunded).Error; err != nil {
			return err
		}

		logger.WithContextCategory(ctx, "order").Info("订单退款处理完成",
			logger.Int64V2("order_id", orderID),
			logger.IntV2("status", int(model.OrderStatusRefunded)))
		return nil
	})
}

// ProcessExternalRefund 处理外部订单退款
func (s *orderService) ProcessExternalRefund(ctx context.Context, outTradeNum string, reason string) error {
	logger.WithContextCategory(ctx, "order").Info("开始处理外部订单退款",
		logger.StringV2("out_trade_num", outTradeNum),
		logger.StringV2("reason", reason))

	// 使用事务确保订单状态更新和退款操作的原子性
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. 根据外部交易号获取订单
		order, err := s.GetOrderByOutTradeNum(ctx, outTradeNum)
		if err != nil {
			logger.WithContextCategory(ctx, "order").Error("获取订单失败",
				logger.ErrorV2(err),
				logger.StringV2("out_trade_num", outTradeNum))
			return fmt.Errorf("订单不存在")
		}

		// 使用行锁防止同一订单的并发处理
		var lockedOrder model.Order
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("id = ?", order.ID).First(&lockedOrder).Error; err != nil {
			return err
		}

		// 2. 检查订单状态是否允许退款
		if lockedOrder.Status == model.OrderStatusRefunded {
			logger.WithContextCategory(ctx, "order").Info("订单已退款，跳过处理",
				logger.Int64V2("order_id", lockedOrder.ID),
				logger.StringV2("out_trade_num", outTradeNum))
			return fmt.Errorf("订单已退款")
		}

		// 只有成功、失败、待充值状态的订单可以退款
		if lockedOrder.Status != model.OrderStatusSuccess &&
			lockedOrder.Status != model.OrderStatusFailed &&
			lockedOrder.Status != model.OrderStatusPendingRecharge {
			logger.WithContextCategory(ctx, "order").Error("订单状态不允许退款",
				logger.Int64V2("order_id", lockedOrder.ID),
				logger.IntV2("status", int(lockedOrder.Status)),
				logger.StringV2("out_trade_num", outTradeNum))
			return fmt.Errorf("订单状态不允许退款")
		}

		// 3. 检查是否为外部订单
		if lockedOrder.Client != 2 {
			logger.WithContextCategory(ctx, "order").Error("非外部订单，不能使用此退款方法",
				logger.Int64V2("order_id", lockedOrder.ID),
				logger.IntV2("client", lockedOrder.Client),
				logger.StringV2("out_trade_num", outTradeNum))
			return fmt.Errorf("非外部订单")
		}

		// 4. 直接退款到用户余额（外部订单使用用户余额系统，使用当前事务）
		balanceService := NewBalanceService(s.balanceLogRepo, s.userRepo)
		if err := balanceService.RefundWithTx(ctx, tx, lockedOrder.CustomerID, lockedOrder.Price, lockedOrder.ID, fmt.Sprintf("外部订单退款: %s", reason), "system"); err != nil {
			logger.WithContextCategory(ctx, "order").Error("退款到用户余额失败",
				logger.ErrorV2(err),
				logger.Int64V2("order_id", lockedOrder.ID),
				logger.Int64V2("customer_id", lockedOrder.CustomerID),
				logger.Float64V2("amount", lockedOrder.Price))
			return fmt.Errorf("退款失败: %v", err)
		}

		logger.WithContextCategory(ctx, "order").Info("退款到用户余额成功",
			logger.Int64V2("order_id", lockedOrder.ID),
			logger.Int64V2("customer_id", lockedOrder.CustomerID),
			logger.Float64V2("amount", lockedOrder.Price))

		// 5. 更新订单备注
		if err := tx.Model(&model.Order{}).Where("id = ?", lockedOrder.ID).Update("remark", fmt.Sprintf("外部订单退款: %s", reason)).Error; err != nil {
			logger.WithContextCategory(ctx, "order").Error("更新订单备注失败", logger.ErrorV2(err), logger.Int64V2("order_id", lockedOrder.ID))
			return fmt.Errorf("更新订单备注失败: %v", err)
		}

		// 6. 更新订单状态为已退款
		if err := tx.Model(&model.Order{}).Where("id = ?", lockedOrder.ID).Update("status", model.OrderStatusRefunded).Error; err != nil {
			logger.WithContextCategory(ctx, "order").Error("更新订单状态失败", logger.ErrorV2(err), logger.Int64V2("order_id", lockedOrder.ID))
			return fmt.Errorf("更新订单状态失败: %v", err)
		}

		logger.WithContextCategory(ctx, "order").Info("外部订单退款完成",
			logger.Int64V2("order_id", lockedOrder.ID),
			logger.StringV2("order_number", lockedOrder.OrderNumber))

		return nil
	})
}

// GetOrderByOutTradeNum 根据外部交易号获取订单
func (s *orderService) GetOrderByOutTradeNum(ctx context.Context, outTradeNum string) (*model.Order, error) {
	// 使用仓储层带有兼容查询的实现：优先 out_trade_num，其次 order_number，最后通过 active_out_trade_num 映射
	return s.orderRepo.GetOrderByOutTradeNum(ctx, outTradeNum)
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

// GetOrdersByUserID 根据用户ID获取订单列表
func (s *orderService) GetOrdersByUserID(ctx context.Context, userID int64, params map[string]interface{}, page, pageSize int) ([]*model.Order, int64, error) {
	return s.orderRepo.GetByUserID(ctx, userID, params, page, pageSize)
}

// 修复 List 返回值赋值数量
func (s *orderService) GetOrdersWithNotification(ctx context.Context, params map[string]interface{}, page, pageSize int) ([]*model.OrderWithNotification, int64, error) {
	// 诊断日志：记录是否携带 user_id 以及参数
	var hasUserID bool
	var userIDVal interface{}
	if v, ok := params["user_id"]; ok {
		hasUserID = true
		userIDVal = v
	}
	logger.WithContextCategory(ctx, "order").Info("OrderService.GetOrdersWithNotification",
		logger.BoolV2("has_user_id", hasUserID),
		logger.AnyV2("user_id_val", userIDVal),
		logger.IntV2("page", page),
		logger.IntV2("page_size", pageSize),
		logger.AnyV2("params", params),
	)
	// 调用仓储层的新方法
	return s.orderRepo.GetOrdersWithNotification(ctx, params, page, pageSize)
}

// GetSuccessStatsByIspDenom 按运营商与面值统计成功订单数与金额
func (s *orderService) GetSuccessStatsByIspDenom(ctx context.Context, params map[string]interface{}) ([]model.IspDenomSuccessStat, error) {
	logger.WithContextCategory(ctx, "order").Info("OrderService.GetSuccessStatsByIspDenom", logger.AnyV2("params", params))
	return s.orderRepo.GetSuccessStatsByIspDenom(ctx, params)
}

// CreateExternalOrder 创建外部订单（事务性处理：先验证商品再扣款创建订单）
func (s *orderService) CreateExternalOrder(ctx context.Context, order *model.Order, userID int64) error {
	logger.WithContextCategory(ctx, "order").Info("开始创建外部订单",
		logger.StringV2("out_trade_num", order.OutTradeNum),
		logger.Int64V2("user_id", userID),
		logger.Int64V2("product_id", order.ProductID))

	// 1. 验证商品是否存在
	product, err := s.productRepo.GetByID(ctx, order.ProductID)
	if err != nil {
		logger.WithContextCategory(ctx, "order").Error("获取商品信息失败",
			logger.ErrorV2(err),
			logger.Int64V2("product_id", order.ProductID))
		return fmt.Errorf("商品不存在: %v", err)
	}

	// 检查商品状态
	if product.Status != 1 {
		logger.WithContextCategory(ctx, "order").Error("商品已下架",
			logger.Int64V2("product_id", order.ProductID),
			logger.IntV2("status", product.Status))
		return fmt.Errorf("商品已下架")
	}

	// 使用商品表的价格
	actualPrice := product.Price
	logger.WithContextCategory(ctx, "order").Info("使用商品表价格",
		logger.Int64V2("product_id", order.ProductID),
		logger.StringV2("product_name", product.Name),
		logger.Float64V2("actual_price", actualPrice))

	// 开启事务
	tx := s.orderRepo.(*repository.OrderRepositoryImpl).DB().Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			logger.WithContextCategory(ctx, "order").Error("创建外部订单发生panic，事务回滚",
				logger.AnyV2("panic", r))
		}
	}()

	if tx.Error != nil {
		logger.WithContextCategory(ctx, "order").Error("开启事务失败",
			logger.ErrorV2(tx.Error))
		return fmt.Errorf("开启事务失败: %v", tx.Error)
	}

	// 2. 智能扣款（优先使用余额，不足时使用授信额度）
	logger.WithContextCategory(ctx, "order").Info("开始智能扣款",
		logger.Int64V2("user_id", userID),
		logger.Float64V2("amount", actualPrice))

	// 创建带授信功能的余额服务实例
	balanceService := NewBalanceServiceWithCredit(s.balanceLogRepo, s.userRepo, s.creditService)
	if err := balanceService.SmartDeduct(ctx, userID, actualPrice, model.BalanceStyleOrderDeduct, "外部订单智能扣款", "system"); err != nil {
		tx.Rollback()
		logger.WithContextCategory(ctx, "order").Error("智能扣款失败",
			logger.ErrorV2(err),
			logger.Int64V2("user_id", userID),
			logger.Float64V2("amount", actualPrice))
		return fmt.Errorf("余额和授信额度均不足: %v", err)
	}

	logger.WithContextCategory(ctx, "order").Info("智能扣款成功",
		logger.Int64V2("user_id", userID),
		logger.Float64V2("amount", actualPrice))

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
			logger.WithContextCategory(ctx, "order").Error("订单创建失败，退款也失败",
				logger.ErrorV2(err),
				logger.ErrorV2(refundErr),
				logger.Int64V2("user_id", userID),
				logger.Float64V2("amount", actualPrice))
		} else {
			logger.WithContextCategory(ctx, "order").Info("订单创建失败，已自动退款",
				logger.Int64V2("user_id", userID),
				logger.Float64V2("amount", actualPrice))
		}
		return fmt.Errorf("创建订单失败: %v", err)
	}

	logger.WithContextCategory(ctx, "order").Info("订单创建成功",
		logger.Int64V2("order_id", order.ID),
		logger.StringV2("order_number", order.OrderNumber),
		logger.IntV2("status", int(order.Status)),
		logger.Float64V2("actual_price", actualPrice))

	// 4. 更新扣款记录的订单ID（将之前的临时扣款记录关联到具体订单）
	if err := s.updateUserBalanceLogOrderID(ctx, userID, actualPrice, order.ID); err != nil {
		logger.WithContextCategory(ctx, "order").Error("更新扣款记录订单ID失败",
			logger.ErrorV2(err),
			logger.Int64V2("order_id", order.ID))
		// 这个错误不影响主流程，只记录日志
	}

	// 5. 推送到充值队列
	if err := s.rechargeService.PushToRechargeQueue(ctx, order.ID); err != nil {
		logger.WithContextCategory(ctx, "order").Error("推送到充值队列失败",
			logger.ErrorV2(err),
			logger.Int64V2("order_id", order.ID))
		// 这个错误不影响主流程，只记录日志
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		logger.WithContextCategory(ctx, "order").Error("提交事务失败",
			logger.ErrorV2(err),
			logger.Int64V2("order_id", order.ID))
		return fmt.Errorf("提交事务失败: %v", err)
	}

	logger.WithContextCategory(ctx, "order").Info("外部订单创建完成",
		logger.Int64V2("order_id", order.ID),
		logger.StringV2("order_number", order.OrderNumber),
		logger.IntV2("status", int(order.Status)))

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
		logger.WithContextCategory(ctx, "order").Error("更新余额日志订单ID失败",
			logger.ErrorV2(err),
			logger.Int64V2("platform_account_id", platformAccountID),
			logger.Float64V2("amount", amount),
			logger.Int64V2("order_id", orderID))
		return err
	}

	logger.WithContextCategory(ctx, "order").Info("更新余额日志订单ID成功",
		logger.Int64V2("platform_account_id", platformAccountID),
		logger.Float64V2("amount", amount),
		logger.Int64V2("order_id", orderID))

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
		logger.WithContextCategory(ctx, "order").Error("更新用户余额日志订单ID失败",
			logger.ErrorV2(err),
			logger.Int64V2("user_id", userID),
			logger.Float64V2("amount", amount),
			logger.Int64V2("order_id", orderID))
		return err
	}

	logger.WithContextCategory(ctx, "order").Info("更新用户余额日志订单ID成功",
		logger.Int64V2("user_id", userID),
		logger.Float64V2("amount", amount),
		logger.Int64V2("order_id", orderID))

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
	logger.WithContextCategory(ctx, "order").Info("开始软删除订单", logger.StringV2("order_id", id))
	orderID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		logger.WithContextCategory(ctx, "order").Error("订单ID格式错误", logger.StringV2("order_id", id), logger.ErrorV2(err))
		return fmt.Errorf("订单ID格式错误: %v", err)
	}
	// 查询订单信息
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		logger.WithContextCategory(ctx, "order").Error("订单不存在", logger.StringV2("order_id", id), logger.ErrorV2(err))
		return fmt.Errorf("订单不存在: %v", err)
	}
	userID := ctx.Value("user_id").(int64)
	if !isSuperAdmin(ctx) && order.CustomerID != userID {
		logger.WithContextCategory(ctx, "order").Error("无权限删除该订单", logger.StringV2("order_id", id), logger.Int64V2("user_id", userID), logger.Int64V2("order_customer_id", order.CustomerID))
		return fmt.Errorf("无权限删除该订单")
	}
	if err := s.orderRepo.SoftDeleteByID(ctx, orderID); err != nil {
		logger.WithContextCategory(ctx, "order").Error("软删除订单失败", logger.StringV2("order_id", id), logger.ErrorV2(err))
		return fmt.Errorf("软删除订单失败: %v", err)
	}
	logger.WithContextCategory(ctx, "order").Info("软删除订单成功", logger.StringV2("order_id", id))
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
	// 2. 删除 balance_query_records
	if err := s.balanceQueryRecordRepo.DeleteByOrderIDs(ctx, orderIDs); err != nil {
		return 0, fmt.Errorf("删除余额查询记录失败: %w", err)
	}

	// 3. 删除 notification_records（仅删除已完成或失败的通知记录，避免删除待处理的记录）
	if err := s.notificationRepo.DeleteCompletedByOrderIDs(ctx, orderIDs); err != nil {
		return 0, fmt.Errorf("删除通知记录失败: %w", err)
	}
	// 4. 删除 balance_logs
	if err := s.rechargeService.GetBalanceService().DeleteByOrderIDs(ctx, orderIDs); err != nil {
		return 0, err
	}
	// 5. 删除 orders
	count, err := s.orderRepo.DeleteByIDs(ctx, orderIDs)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// GetProductID 根据价格、ISP和状态获取产品ID
func (s *orderService) GetProductID(price float64, isp int, status int) (*model.Product, error) {
	logger.GetCategoryLogger("order").Info("GetProductID called",
		logger.Float64V2("price", price),
		logger.IntV2("isp", isp),
		logger.IntV2("status", status),
	)
	product, err := s.orderRepo.FindProductByPriceAndISPWithTolerance(price, isp, status, 0.01)
	if err != nil {
		logger.GetCategoryLogger("order").Error("未找到匹配的产品",
			logger.Float64V2("price", price),
			logger.IntV2("isp", isp),
			logger.IntV2("status", status),
			logger.ErrorV2(err),
		)
		return nil, fmt.Errorf("未找到匹配的产品: price=%.2f, isp=%d, status=%d", price, isp, status)
	}
	logger.GetCategoryLogger("order").Info("匹配到产品",
		logger.Int64V2("product_id", product.ID),
		logger.Float64V2("price", product.Price),
		logger.StringV2("isp", product.ISP),
		logger.IntV2("status", product.Status),
	)
	return product, nil
}

// GetProductByNameValue 根据产品名称数字部分、ISP和状态获取产品
func (s *orderService) GetProductByNameValue(nameValue float64, isp int, status int) (*model.Product, error) {
	logger.GetCategoryLogger("order").Info("GetProductByNameValue called",
		logger.Float64V2("name_value", nameValue),
		logger.IntV2("isp", isp),
		logger.IntV2("status", status),
	)

	product, err := s.orderRepo.FindProductByNameValueAndISP(int(nameValue), isp, status)
	if err != nil {
		logger.GetCategoryLogger("order").Error("未找到匹配的产品",
			logger.Float64V2("name_value", nameValue),
			logger.IntV2("isp", isp),
			logger.IntV2("status", status),
			logger.ErrorV2(err),
		)
		return nil, fmt.Errorf("未找到匹配的产品: nameValue=%.0f, isp=%d, status=%d", nameValue, isp, status)
	}

	logger.GetCategoryLogger("order").Info("找到匹配的产品",
		logger.Int64V2("product_id", product.ID),
		logger.StringV2("product_name", product.Name),
		logger.Float64V2("name_value", nameValue),
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

	// 注入订单号上下文并记录发送动作
	ctx = logger.InjectOrderNumber(ctx, order.OrderNumber)
	logger.WithContextCategory(ctx, "order").Info("【发送订单回调通知】开始",
		logger.Int64V2("order_id", order.ID),
		logger.StringV2("order_number", order.OrderNumber),
		logger.IntV2("status", int(order.Status)),
	)

	err = s.notificationHelper.SendOrderCallbackNotification(ctx, orderID, order)
	if err != nil {
		logger.WithContextCategory(ctx, "order").Error("【发送订单回调通知失败】", logger.ErrorV2(err))
		return err
	}
	logger.WithContextCategory(ctx, "order").Info("【发送订单回调通知成功】")
	return nil
}

// GetOrdersByUserID 根据用户ID获取订单列表
// 删除：
