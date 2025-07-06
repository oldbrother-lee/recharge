package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"recharge-go/internal/model"
	notificationModel "recharge-go/internal/model/notification"
	"recharge-go/internal/repository"
	notificationRepo "recharge-go/internal/repository/notification"
	"recharge-go/internal/service/recharge"
	"recharge-go/pkg/logger"
	"recharge-go/pkg/queue"
	"recharge-go/pkg/redis"
	"strconv"
	"sync"
	"time"

	redisV8 "github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

// RechargeService 充值服务接口
type RechargeService interface {
	// Recharge 执行充值
	Recharge(ctx context.Context, orderID int64) error
	// HandleCallback 处理平台回调
	HandleCallback(ctx context.Context, platformName string, data []byte) error
	// GetPendingTasks 获取待处理的充值任务
	GetPendingTasks(ctx context.Context, limit int) ([]*model.Order, error)
	// ProcessRechargeTask 处理充值任务
	ProcessRechargeTask(ctx context.Context, order *model.Order) error
	// CreateRechargeTask 创建充值任务
	CreateRechargeTask(ctx context.Context, orderID int64) error
	// GetPlatformAPIByOrderID 根据订单ID获取平台API信息
	GetPlatformAPIByOrderID(ctx context.Context, orderID string) (*model.PlatformAPI, *model.PlatformAPIParam, error)
	// PushToRechargeQueue 将订单推送到充值队列
	PushToRechargeQueue(ctx context.Context, orderID int64) error
	// PopFromRechargeQueue 从充值队列获取订单
	PopFromRechargeQueue(ctx context.Context) (int64, error)
	// GetOrderByID 根据ID获取订单
	GetOrderByID(ctx context.Context, orderID int64) (*model.Order, error)
	// RemoveFromProcessingQueue 从处理中队列移除任务
	RemoveFromProcessingQueue(ctx context.Context, orderID int64) error
	// CheckRechargingOrders 检查充值中订单
	CheckRechargingOrders(ctx context.Context) error
	// SubmitOrder 提交订单到平台
	SubmitOrder(ctx context.Context, order *model.Order, api *model.PlatformAPI, apiParam *model.PlatformAPIParam) error
	// ProcessRetryTask 处理重试任务
	ProcessRetryTask(ctx context.Context, retryRecord *model.OrderRetryRecord) error
	// GetBalanceService 获取余额服务
	GetBalanceService() *PlatformAccountBalanceService
	// GetUserBalanceService 获取用户余额服务
	GetUserBalanceService() *BalanceService
	// SetOrderService 设置订单服务
	SetOrderService(orderService OrderService)
}

// rechargeService 充值服务
type rechargeService struct {
	db                     *gorm.DB
	orderRepo              repository.OrderRepository
	platformRepo           repository.PlatformRepository
	platformAPIRepo        repository.PlatformAPIRepository
	retryRepo              repository.RetryRepository
	callbackLogRepo        repository.CallbackLogRepository
	productAPIRelationRepo repository.ProductAPIRelationRepository
	productRepo            repository.ProductRepository
	platformAPIParamRepo   repository.PlatformAPIParamRepository
	balanceService         *PlatformAccountBalanceService
	userBalanceService     *BalanceService
	manager                *recharge.Manager
	redisClient            *redisV8.Client
	processingOrders       map[int64]bool
	processingOrdersMu     sync.Mutex
	notificationRepo       notificationRepo.Repository
	queue                  queue.Queue
	orderService           OrderService
}

// NewRechargeService 创建充值服务实例
func NewRechargeService(
	db *gorm.DB,
	orderRepo repository.OrderRepository,
	platformRepo repository.PlatformRepository,
	platformAPIRepo repository.PlatformAPIRepository,
	retryRepo repository.RetryRepository,
	callbackLogRepo repository.CallbackLogRepository,
	productAPIRelationRepo repository.ProductAPIRelationRepository,
	productRepo repository.ProductRepository,
	platformAPIParamRepo repository.PlatformAPIParamRepository,
	balanceService *PlatformAccountBalanceService,
	userBalanceService *BalanceService,
	notificationRepo notificationRepo.Repository,
	queue queue.Queue,
) *rechargeService {
	return &rechargeService{
		db:                     db,
		orderRepo:              orderRepo,
		platformRepo:           platformRepo,
		platformAPIRepo:        platformAPIRepo,
		retryRepo:              retryRepo,
		callbackLogRepo:        callbackLogRepo,
		productAPIRelationRepo: productAPIRelationRepo,
		productRepo:            productRepo,
		platformAPIParamRepo:   platformAPIParamRepo,
		balanceService:         balanceService,
		userBalanceService:     userBalanceService,
		manager:                recharge.NewManager(db),
		redisClient:            redis.GetClient(),
		processingOrders:       make(map[int64]bool),
		notificationRepo:       notificationRepo,
		queue:                  queue,
	}
}

// Recharge 执行充值
func (s *rechargeService) Recharge(ctx context.Context, orderID int64) error {
	logger.Info(fmt.Sprintf("【开始执行充值】order_id: %d", orderID))

	// 1. 获取订单信息
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		logger.Error(fmt.Sprintf("【获取订单信息失败】order_id: %d, error: %v", orderID, err))
		return fmt.Errorf("get order failed: %v", err)
	}
	logger.Info(fmt.Sprintf("【获取订单信息成功】order_id: %d, status: %d, order_number: %s",
		orderID, order.Status, order.OrderNumber))

	// 检查订单状态，如果已经是充值中或已完成，则不再处理
	if order.Status == model.OrderStatusRecharging || order.Status == model.OrderStatusSuccess {
		logger.Info(fmt.Sprintf("【订单状态异常，跳过处理】order_id: %d, status: %d", orderID, order.Status))
		// 从处理中队列移除
		_ = s.RemoveFromProcessingQueue(ctx, orderID)
		return nil
	}

	// 2. 获取平台API信息
	api, apiParam, err := s.GetPlatformAPIByOrderID(ctx, order.OrderNumber)
	if err != nil {
		logger.Error(fmt.Sprintf("【获取平台API信息失败】order_id: %d, error: %v", orderID, err))
		return fmt.Errorf("get platform api failed: %v", err)
	}
	logger.Info(fmt.Sprintf("【获取平台API信息成功】api %+v: \n", api))
	fmt.Println(apiParam, "apiParam+!!!!!!!!!!!!!!!!!!1+++++++")
	// 3. 提交订单到平台
	logger.Info(fmt.Sprintf("【开始提交订单到平台】order_id: %d, platform: %d", orderID, api.PlatformID))
	if err := s.manager.SubmitOrder(ctx, order, api, apiParam); err != nil {
		logger.Error(fmt.Sprintf("【提交订单到平台失败】order_id: %d, error: %v", orderID, err))

		// 创建重试记录
		retryParams := map[string]interface{}{
			"order_id": orderID,
			"amount":   order.TotalPrice,
			"mobile":   order.Mobile,
		}
		retryParamsJSON, _ := json.Marshal(retryParams)

		usedAPIs := map[string]interface{}{
			"api_id":   api.ID,
			"param_id": apiParam.ID,
		}
		usedAPIsJSON, _ := json.Marshal(usedAPIs)

		// 获取已存在的重试记录数量
		records, err := s.retryRepo.GetByOrderID(ctx, orderID)
		if err != nil {
			logger.Error("【获取重试记录失败】order_id: %d, error: %v", orderID, err)
			return fmt.Errorf("get retry records failed: %v", err)
		}

		retryCount := len(records)
		// 计算重试时间：首次切换平台立即重试，后续重试延迟5分钟
		nextRetryTime := time.Now()
		if retryCount > 1 {
			nextRetryTime = time.Now().Add(5 * time.Minute)
		}

		retryRecord := &model.OrderRetryRecord{
			OrderID:       orderID,
			APIID:         api.ID,
			ParamID:       apiParam.ID,
			RetryType:     1, // 1: 平台切换
			RetryCount:    retryCount,
			LastError:     err.Error(),
			RetryParams:   string(retryParamsJSON),
			UsedAPIs:      string(usedAPIsJSON),
			Status:        0, // 0: 待处理
			NextRetryTime: nextRetryTime,
		}

		if err := s.retryRepo.Create(ctx, retryRecord); err != nil {
			logger.Error("【创建重试记录失败】order_id: %d, error: %v", orderID, err)
		} else {
			logger.Info("【创建重试记录成功】order_id: %d, retry_id: %d", orderID, retryRecord.ID)
		}

		return fmt.Errorf("submit order failed: %v", err)
	}
	logger.Info("【提交订单到平台成功】order_id: %d", orderID)

	// 4. 开启事务
	logger.Info("【开始更新订单状态和平台信息】order_id: %d", orderID)
	tx := s.orderRepo.(*repository.OrderRepositoryImpl).DB().Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			logger.Error("【事务回滚】order_id: %d, panic: %v", orderID, r)
		}
	}()

	// 5. 更新订单状态
	logger.Info("【开始更新订单状态】order_id: %d, old_status: %d, new_status: %d",
		orderID, order.Status, model.OrderStatusRecharging)
	result := tx.Model(&model.Order{}).Where("id = ?", orderID).Update("status", model.OrderStatusRecharging)
	if result.Error != nil {
		tx.Rollback()
		logger.Error("【更新订单状态失败】order_id: %d, error: %v", orderID, result.Error)
		return fmt.Errorf("update order status failed: %v", result.Error)
	}
	if result.RowsAffected == 0 {
		tx.Rollback()
		logger.Error("【更新订单状态失败】order_id: %d, 没有记录被更新", orderID)
		return fmt.Errorf("no record updated")
	}
	logger.Info("【更新订单状态成功】order_id: %d, rows_affected: %d", orderID, result.RowsAffected)

	// 6. 更新平台信息
	logger.Info("【开始更新平台信息】order_id: %d, platform_id: %d, api_id: %d, param_id: %d",
		orderID, api.ID, api.ID, apiParam.ID)
	result = tx.Model(&model.Order{}).Where("id = ?", orderID).Updates(map[string]interface{}{
		"api_cur_id":       api.ID,
		"api_cur_param_id": apiParam.ID,
	})
	if result.Error != nil {
		tx.Rollback()
		logger.Error("【更新平台信息失败】order_id: %d, error: %v", orderID, result.Error)
		return fmt.Errorf("update platform info failed: %v", result.Error)
	}
	if result.RowsAffected == 0 {
		tx.Rollback()
		logger.Error("【更新平台信息失败】order_id: %d, 没有记录被更新", orderID)
		return fmt.Errorf("no record updated")
	}
	logger.Info("【更新平台信息成功】order_id: %d, rows_affected: %d", orderID, result.RowsAffected)

	// 7. 提交事务
	if err := tx.Commit().Error; err != nil {
		logger.Error("【提交事务失败】order_id: %d, error: %v", orderID, err)
		return fmt.Errorf("commit transaction failed: %v", err)
	}
	logger.Info("【提交事务成功】order_id: %d", orderID)

	// 8. 从处理中队列移除
	logger.Info("【从处理中队列移除】order_id: %d", orderID)
	if err := s.RemoveFromProcessingQueue(ctx, orderID); err != nil {
		logger.Error("【从处理中队列移除失败】order_id: %d, error: %v", orderID, err)
	}

	// 9. 验证更新结果
	updatedOrder, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		logger.Error("【验证更新结果失败】order_id: %d, error: %v", orderID, err)
	} else {
		logger.Info("【验证更新结果】order_id: %d, status: %d, platform_id: %d",
			orderID, updatedOrder.Status, updatedOrder.PlatformId)
	}

	// 提交成功后，更新订单的 const_price 字段为 apiParam.Price
	err = s.orderRepo.DB().Model(&model.Order{}).
		Where("id = ?", order.ID).
		Update("const_price", apiParam.Price).Error
	if err != nil {
		logger.Error("【更新订单成本价失败】", "order_id", order.ID, "error", err)
		// 新增：将订单状态设置为失败，并写入备注
		_ = s.orderRepo.UpdateStatus(ctx, order.ID, model.OrderStatusFailed)
		_ = s.orderRepo.UpdateRemark(ctx, order.ID, "余额不足，订单失败")
		// 新增：推送订单失败通知
		notification := &notificationModel.NotificationRecord{
			OrderID:          order.ID,
			PlatformCode:     order.PlatformCode,
			NotificationType: "order_status_changed",
			Content:          "订单失败：余额不足",
			Status:           1, // 待处理
		}
		_ = s.notificationRepo.Create(ctx, notification)
		_ = s.queue.Push(ctx, "notification_queue", notification)
	} else {
		logger.Info("【更新订单成本价成功】", "order_id", order.ID, "const_price", apiParam.Price)
	}

	logger.Info("【充值流程完成】order_id: %d", orderID)
	return nil
}

// HandleCallback 处理平台回调
func (s *rechargeService) HandleCallback(ctx context.Context, platformName string, data []byte) error {
	// 1. 解析回调数据
	callbackData, err := s.manager.ParseCallbackData(platformName, data)
	if err != nil {
		logger.Error(fmt.Sprintf("解析回调数据失败: %v", err))
		return fmt.Errorf("parse callback data failed service 层: %v", err)
	}
	logger.Info(fmt.Sprintf("收到秘史回调，解析回调数据成功: %+v", callbackData))

	// 2. 检查是否已处理过该回调
	exists, err := s.callbackLogRepo.GetByOrderIDAndType(ctx, callbackData.OrderID, callbackData.CallbackType)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		logger.Error(fmt.Sprintf("检查回调记录失败: %v", err))
		tx := s.db.Begin()
		defer func() {
			if r := recover(); r != nil {
				tx.Rollback()
			}
		}()
		return err
	}
	if exists != nil {
		logger.Info(fmt.Sprintf("回调已处理过: order_id: %s, callback_type: %s", callbackData.OrderID, callbackData.CallbackType))
		return nil
	}

	// 3. 开启事务
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 4. 处理回调
	if err := s.manager.HandleCallback(ctx, platformName, data); err != nil {
		tx.Rollback()
		logger.Error("处理回调失败: %v", err)
		return fmt.Errorf("handle callback failed: %v", err)
	}

	// 4.1 更新订单状态
	orderState, err := strconv.Atoi(callbackData.Status)
	if err != nil {
		tx.Rollback()
		logger.Error("解析订单状态失败1111: %v", err)
		return fmt.Errorf("parse order status failed: %v", err)
	}

	// 获取订单信息
	order, err := s.orderRepo.GetByOrderID(ctx, callbackData.OrderNumber)
	if err != nil {
		tx.Rollback()
		logger.Error("获取订单信息失败: %v", err)
		return fmt.Errorf("get order failed: %v", err)
	}

	// === 新增：订单失败自动退款 ===
	if model.OrderStatus(orderState) == model.OrderStatusFailed {
		// 检查是否为外部订单
		if order.Client == 2 {
			// 外部订单直接退款到用户余额
			// 注意：这里需要通过依赖注入获取余额服务，暂时记录日志
			logger.Info("外部订单失败，需要退款到用户余额",
				"order_id", order.ID,
				"customer_id", order.CustomerID,
				"amount", order.Price)
			// TODO: 实现外部订单退款逻辑
		} else {
			// 平台订单使用原有的退款方法
			err := s.balanceService.RefundBalance(ctx, order.CustomerID, order.Price, order.ID, "订单失败退还余额")
			if err != nil {
				tx.Rollback()
				logger.Error("订单失败退款失败: %v", err)
				return fmt.Errorf("订单失败退款失败: %v", err)
			}
		}
	}
	// === 新增 END ===

	// 更新订单状态
	if err := s.orderRepo.UpdateStatus(ctx, order.ID, model.OrderStatus(orderState)); err != nil {
		tx.Rollback()
		logger.Error("更新订单状态失败: %v", err)
		return fmt.Errorf("update order status failed: %v", err)
	}
	logger.Info(fmt.Sprintf("订单回调更新订单状态成功: 订单号%s, 订单id%d, 状态%s", order.OrderNumber, order.ID, orderState))

	// 创建通知记录
	notification := &notificationModel.NotificationRecord{
		OrderID:          order.ID,
		PlatformCode:     order.PlatformCode,
		NotificationType: "order_status_changed",
		Content:          fmt.Sprintf("订单状态已更新为: %d", orderState),
		Status:           1, // 待处理
	}

	// 保存通知记录到数据库
	if err := s.notificationRepo.Create(ctx, notification); err != nil {
		tx.Rollback()
		logger.Error("创建通知记录失败",
			"error", err,
			"order_id", order.ID,
			"platform_code", order.PlatformCode,
			"notification_type", notification.NotificationType,
		)
		return fmt.Errorf("create notification record failed: %v", err)
	}
	logger.Info(fmt.Sprintf("秘史创建通知记录成功: 订单号%s, 订单id%d, 状态%s", order.OrderNumber, order.ID, orderState))

	// 推送通知到队列
	logger.Info("准备推送通知到队列", "order_id", order.ID, "status", orderState)
	err = s.queue.Push(ctx, "notification_queue", notification)
	if err != nil {
		tx.Rollback()
		logger.Error("推送通知到队列失败", "order_id", order.ID, "error", err)
		return fmt.Errorf("push notification to queue failed: %v", err)
	}
	logger.Info("订单推送通知到队列成功", "order_id", order.ID)

	// 5. 记录回调日志
	log := &model.CallbackLog{
		OrderID:      callbackData.OrderID,
		PlatformID:   callbackData.OrderNumber,
		CallbackType: callbackData.CallbackType,
		Status:       1,
		RequestData:  string(data),
		ResponseData: "success",
		CreateTime:   time.Now(),
		UpdateTime:   time.Now(),
	}
	if err := s.callbackLogRepo.Create(ctx, log); err != nil {
		tx.Rollback()
		logger.Error("记录回调日志失败: %v", err)
		return err
	}

	// 6. 提交事务
	if err := tx.Commit().Error; err != nil {
		logger.Error("提交事务失败: %v", err)
		return fmt.Errorf("commit transaction failed: %v", err)
	}

	return nil
}

// GetPendingTasks 获取待处理的充值任务
func (s *rechargeService) GetPendingTasks(ctx context.Context, limit int) ([]*model.Order, error) {
	// 从Redis队列中获取待处理的订单ID
	if s.redisClient == nil {
		logger.Error("【Redis客户端为空】")
		return nil, fmt.Errorf("redis client is nil")
	}

	// 获取队列中的订单ID列表
	orderIDs, err := s.redisClient.LRange(ctx, "recharge_queue", 0, int64(limit-1)).Result()
	if err != nil {
		logger.Error("【从Redis队列获取订单ID失败】", "error", err)
		return nil, fmt.Errorf("get order IDs from queue failed: %v", err)
	}

	logger.Info("【调试：从Redis获取的订单ID列表】", "order_ids", orderIDs, "limit", limit)

	if len(orderIDs) == 0 {
		logger.Info("【Redis队列中没有待处理订单】")
		return []*model.Order{}, nil
	}

	// 将字符串ID转换为int64并查询订单信息
	var orders []*model.Order
	now := time.Now()
	for _, orderIDStr := range orderIDs {
		orderID, err := strconv.ParseInt(orderIDStr, 10, 64)
		if err != nil {
			logger.Error("【解析订单ID失败】", "order_id_str", orderIDStr, "error", err)
			continue
		}

		// 获取订单信息
		order, err := s.orderRepo.GetByID(ctx, orderID)
		if err != nil {
			logger.Error("【获取订单信息失败】", "order_id", orderID, "error", err)
			// 从Redis队列中移除该订单
			if removeErr := s.redisClient.LRem(ctx, "recharge_queue", 0, orderIDStr).Err(); removeErr != nil {
				logger.Error("【从队列移除失效订单失败】", "order_id", orderID, "error", removeErr)
			} else {
				logger.Info("【成功从队列移除失效订单】", "order_id", orderID)
			}
			continue
		}

		logger.Info("【调试：检查订单】", "order_id", orderID, "status", order.Status, "created_at", order.CreatedAt, "updated_at", order.UpdatedAt)

		// 检查订单状态和时间过滤条件
		if order.Status != model.OrderStatusPendingRecharge {
			logger.Info("【订单状态不是待充值，从队列中移除】", "order_id", orderID, "status", order.Status)
			// 从Redis队列中移除该订单
			if err := s.redisClient.LRem(ctx, "recharge_queue", 0, orderIDStr).Err(); err != nil {
				logger.Error("【从队列移除订单失败】", "order_id", orderID, "error", err)
			} else {
				logger.Info("【成功从队列移除订单】", "order_id", orderID)
			}
			continue
		}

		// 只对非新订单应用1分钟冷却机制
		// 新订单的创建时间和更新时间相差很小（通常在几毫秒内），不应该被冷却机制拦截
		timeDiff := order.UpdatedAt.Sub(order.CreatedAt)
		isNewOrder := timeDiff < 5*time.Second // 5秒内的时间差认为是新订单

		logger.Info("【调试：时间检查】", "order_id", orderID, "time_diff", timeDiff, "is_new_order", isNewOrder)

		if !isNewOrder && order.UpdatedAt.Add(1*time.Minute).After(now) {
			logger.Info("【订单最近1分钟内被处理过，跳过】", "order_id", orderID)
			continue
		}

		// 如果订单创建时间超过24小时，跳过
		// 优先使用CreateTime字段，如果为空则使用CreatedAt字段
		createTime := order.CreateTime
		if createTime.IsZero() && !order.CreatedAt.IsZero() {
			createTime = order.CreatedAt
		}

		if !createTime.IsZero() && createTime.Add(24*time.Hour).Before(now) {
			logger.Info("【订单创建时间超过24小时，跳过】", "order_id", orderID, "create_time", createTime)
			continue
		}

		orders = append(orders, order)
	}

	logger.Info("【获取到待处理订单】", "count", len(orders))
	return orders, nil
}

// ProcessRechargeTask 处理充值任务
func (s *rechargeService) ProcessRechargeTask(ctx context.Context, order *model.Order) error {
	logger.Info("【开始处理充值任务】",
		"order_id", order.ID,
		"order_number", order.OrderNumber,
		"mobile", order.Mobile)

	// 获取平台API信息 - 直接使用传入的订单对象，避免通过order_number重新查询
	api, apiParam, err := s.getPlatformAPIByOrder(ctx, order)
	if err != nil {
		logger.Error("【获取API信息失败】",
			"error", err,
			"order_id", order.ID)
		return fmt.Errorf("get platform API failed: %v", err)
	}
	fmt.Println(apiParam, "apiParam+$$$$$$$$$$$$$$$$$$$$$$$$$$")
	logger.Info("【获取API信息成功】",
		"order_id", order.ID,
		"api_id", api.ID,
		"api_name", api.Name)

	// 原子性锁定订单，防止并发重复处理
	// 只有当前 api_id 对应的订单状态为 'pending_recharge' 时，才允许锁定
	locked, err := s.orderRepo.UpdateStatusCAS(ctx, order.ID, model.OrderStatusPendingRecharge, model.OrderStatusProcessing, api.ID)
	if err != nil {
		logger.Error("【订单状态原子更新失败】",
			"error", err,
			"order_id", order.ID)
		return err
	}
	if !locked {
		logger.Info("【订单已被其他worker处理，跳过】", "order_id", order.ID)
		return nil
	}

	// 获取订单信息
	order, err = s.orderRepo.GetByID(ctx, order.ID)
	if err != nil {
		logger.Error("【获取订单信息失败】",
			"error", err,
			"order_id", order.ID)
		return fmt.Errorf("get order failed: %v", err)
	}
	logger.Info("【获取订单信息成功】",
		"order_id", order.ID,
		"status", order.Status)

	// 检查订单状态
	if order.Status == model.OrderStatusRecharging || order.Status == model.OrderStatusSuccess {
		return nil
	}

	// 检查是否需要扣款（外部订单在创建时已扣款）
	if order.Client == 2 { // 外部API订单
		logger.Info("【外部订单已预扣款，跳过扣款步骤】",
			"order_id", order.ID,
			"client", order.Client)
	} else {
		// 平台订单：从平台账号扣款（支持授信额度）
		logger.Info("【平台订单开始扣款】",
			"order_id", order.ID,
			"platform_account_id", order.PlatformAccountID,
			"amount", order.Price)

		// 使用平台账号余额服务进行扣款（支持授信额度）
		balanceService := s.GetBalanceService()
		if err := balanceService.DeductBalance(ctx, order.PlatformAccountID, order.Price, order.ID, "订单充值扣除"); err != nil {
			logger.Error("【扣除平台账号余额失败】",
				"error", err,
				"platform_account_id", order.PlatformAccountID,
				"amount", order.Price)
			// 将订单状态设置为失败，并写入备注
			_ = s.orderRepo.UpdateStatus(ctx, order.ID, model.OrderStatusFailed)
			_ = s.orderRepo.UpdateRemark(ctx, order.ID, "平台账号余额和授信额度均不足，订单失败")
			// 推送订单失败通知
			notification := &notificationModel.NotificationRecord{
				OrderID:          order.ID,
				PlatformCode:     order.PlatformCode,
				NotificationType: "order_status_changed",
				Content:          "订单失败：平台账号余额和授信额度均不足",
				Status:           1, // 待处理
			}
			_ = s.notificationRepo.Create(ctx, notification)
			_ = s.queue.Push(ctx, "notification_queue", notification)
			return fmt.Errorf("deduct platform account balance failed: %v", err)
		}
		logger.Info("【扣除平台账号余额成功】",
			"platform_account_id", order.PlatformAccountID,
			"amount", order.Price)
	}

	// 提交订单到平台
	if err := s.SubmitOrder(ctx, order, api, apiParam); err != nil {
		logger.Error("【提交订单到平台失败1】",
			"error", err,
			"order_id", order.ID)

		// 获取所有可用的API关系
		relations, err2 := s.productRepo.GetAPIRelationsByProductID(ctx, order.ProductID)
		if err2 != nil {
			logger.Error("【获取API关系失败】",
				"error", err2,
				"order_id", order.ID)
			return fmt.Errorf("get API relations failed: %v", err2)
		}

		// 解析已使用的API列表
		var usedAPIs []map[string]interface{}
		if order.UsedAPIs != "" {
			if err := json.Unmarshal([]byte(order.UsedAPIs), &usedAPIs); err != nil {
				logger.Error("【解析已使用API列表失败】",
					"error", err,
					"order_id", order.ID)
				usedAPIs = []map[string]interface{}{}
			}
		}

		// 添加当前API到已使用列表
		usedAPIs = append(usedAPIs, map[string]interface{}{
			"api_id": api.ID,
		})
		usedAPIsJSON, _ := json.Marshal(usedAPIs)

		// 找到下一个可用的API
		var nextAPIID, nextParamID int64
		for _, relation := range relations {
			// 检查API是否已使用
			alreadyUsed := false
			for _, usedAPI := range usedAPIs {
				if usedAPI["api_id"] == relation.APIID {
					alreadyUsed = true
					break
				}
			}
			if !alreadyUsed {
				nextAPIID = relation.APIID
				nextParamID = relation.ParamID
				break
			}
		}

		if nextAPIID == 0 {
			logger.Error("【没有可用的API】",
				"order_id", order.ID)
			// 调用订单失败处理方法，会自动退还余额和创建通知
			if err := s.orderService.ProcessOrderFail(ctx, order.ID, "无可用API"); err != nil {
				logger.Error("处理订单失败时出错", "error", err, "order_id", order.ID)
			}
			return fmt.Errorf("no available API")
		}

		fmt.Println("调用 UpdateStatusAndAPIID 之前")
		if err2 := s.orderRepo.UpdateStatusAndAPIID(ctx, order.ID, model.OrderStatusPendingRecharge, nextAPIID, string(usedAPIsJSON)); err2 != nil {
			fmt.Println("UpdateStatusAndAPIID 执行出错，err =", err2)
			logger.Error("【更新订单状态和API ID失败】",
				"error", err2,
				"order_id", order.ID)
			return fmt.Errorf("update order status and API ID failed: %v", err2)
		}
		fmt.Println("UpdateStatusAndAPIID 执行完毕")

		fmt.Println("准备创建重试记录 retryParams")
		submitErr := err // 保存 SubmitOrder 的错误
		retryParams := map[string]interface{}{
			"order_id":  order.ID,
			"api_id":    nextAPIID,
			"param_id":  nextParamID,
			"platform":  api.PlatformID,
			"retry_at":  time.Now(),
			"next_at":   time.Now().Add(5 * time.Minute),
			"error_msg": submitErr.Error(),
		}
		fmt.Println("retryParams =", retryParams)
		retryParamsJSON, _ := json.Marshal(retryParams)

		fmt.Println("准备创建 retryRecord")
		// 计算重试时间：首次切换平台立即重试，后续重试延迟5分钟
		nextRetryTime := time.Now()
		if len(usedAPIs) > 1 {
			nextRetryTime = time.Now().Add(5 * time.Minute)
		}

		retryRecord := &model.OrderRetryRecord{
			OrderID:       order.ID,
			APIID:         nextAPIID,
			ParamID:       nextParamID,
			RetryType:     1, // 1: 平台切换
			RetryCount:    len(usedAPIs),
			LastError:     submitErr.Error(),
			RetryParams:   string(retryParamsJSON),
			UsedAPIs:      string(usedAPIsJSON),
			Status:        0, // 0: 待处理
			NextRetryTime: nextRetryTime,
		}
		fmt.Println("retryRecord =", retryRecord)

		if s.retryRepo == nil {
			logger.Error("【严重错误】retryRecordRepo 为空！",
				"order_id", order.ID)
			return fmt.Errorf("retry repository is nil")
		}

		logger.Info("【准备调用Create方法】",
			"order_id", order.ID)
		if err := s.retryRepo.Create(ctx, retryRecord); err != nil {
			logger.Error("【创建重试记录失败】",
				"error", err,
				"order_id", order.ID)
			return fmt.Errorf("create retry record failed: %v", err)
		}
		logger.Info("【创建重试记录成功】",
			"order_id", order.ID,
			"retry_id", retryRecord.ID)

		// 从处理队列中移除
		if err := s.RemoveFromProcessingQueue(ctx, order.ID); err != nil {
			logger.Error("【从处理队列移除失败】",
				"error", err,
				"order_id", order.ID)
		}

		logger.Info("【充值任务处理完成】",
			"order_id", order.ID,
			"order_number", order.OrderNumber)
		return fmt.Errorf("submit order failed: %v", err)
	}

	// 更新订单状态为充值中
	if err := s.orderRepo.UpdateStatus(ctx, order.ID, model.OrderStatusRecharging); err != nil {
		logger.Error("【更新订单状态失败】",
			"error", err,
			"order_id", order.ID)
		return fmt.Errorf("update order status failed: %v", err)
	}
	logger.Info("【订单状态更新成功】",
		"order_id", order.ID,
		"status", model.OrderStatusRecharging)

	// 更新订单成本价
	err = s.orderRepo.DB().Model(&model.Order{}).
		Where("id = ?", order.ID).
		Update("const_price", apiParam.Price).Error
	if err != nil {
		logger.Error("【更新订单成本价失败】", "order_id", order.ID, "error", err)
	} else {
		logger.Info("【更新订单成本价成功】", "order_id", order.ID, "const_price", apiParam.Price)
	}

	// 从处理队列中移除
	if err := s.RemoveFromProcessingQueue(ctx, order.ID); err != nil {
		logger.Error("【从处理队列移除失败】",
			"error", err,
			"order_id", order.ID)
	}

	logger.Info("【充值任务处理完成】",
		"order_id", order.ID,
		"order_number", order.OrderNumber)
	return nil
}

// CreateRechargeTask 创建充值任务
func (s *rechargeService) CreateRechargeTask(ctx context.Context, orderID int64) error {
	logger.Info("【开始创建充值任务】",
		"order_id", orderID)

	// 获取订单信息
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		logger.Error("【获取订单信息失败】",
			"error", err,
			"order_id", orderID)
		return fmt.Errorf("get order failed: %v", err)
	}
	logger.Info("【获取订单信息成功】",
		"order_id", orderID,
		"order_number", order.OrderNumber,
		"status", order.Status)

	// 检查订单状态是否为待充值
	if order.Status != model.OrderStatusPendingRecharge {
		logger.Warn("【订单状态不是待充值，跳过创建充值任务】",
			"order_id", orderID,
			"current_status", order.Status,
			"expected_status", model.OrderStatusPendingRecharge)
		return fmt.Errorf("order status is not pending recharge, current status: %d", order.Status)
	}

	logger.Info("【订单状态验证通过，状态为待充值】",
		"order_id", orderID,
		"status", order.Status)

	// 将订单推送到充值队列
	if err := s.PushToRechargeQueue(ctx, orderID); err != nil {
		logger.Error("【推送订单到充值队列失败】",
			"error", err,
			"order_id", orderID)
		return fmt.Errorf("push to recharge queue failed: %v", err)
	}
	logger.Info("【推送订单到充值队列成功】",
		"order_id", orderID)

	logger.Info("【充值任务创建完成】",
		"order_id", orderID,
		"order_number", order.OrderNumber)
	return nil
}

// GetPlatformAPIByOrderID 根据订单ID获取平台API信息
// getPlatformAPIByOrder 直接使用订单对象获取平台API信息
func (s *rechargeService) getPlatformAPIByOrder(ctx context.Context, order *model.Order) (*model.PlatformAPI, *model.PlatformAPIParam, error) {
	// 直接使用传入的订单对象，无需重新查询
	logger.Info("【获取平台API信息】", "order_id", order.ID, "product_id", order.ProductID)

	//product_api_relations
	r, err := s.productAPIRelationRepo.GetByProductID(ctx, order.ProductID)
	if err != nil {
		// 将订单设置为失败状态
		if err := s.orderRepo.UpdateStatus(ctx, order.ID, model.OrderStatusFailed); err != nil {
			logger.Error("【更新订单状态失败】",
				"error", err,
				"order_id", order.ID)
		}
		// 更新订单备注
		if err := s.orderRepo.UpdateRemark(ctx, order.ID, "商品未绑定接口"); err != nil {
			logger.Error("【更新订单备注失败】",
				"error", err,
				"order_id", order.ID)
		}
		return nil, nil, fmt.Errorf("商品未绑定接口: %v", err)
	}

	//获取api套餐 platform_api_params
	apiParam, err := s.platformAPIParamRepo.GetByID(ctx, r.ParamID)
	if err != nil {
		if errors.Is(err, repository.ErrNoAPIForProduct) {
			// 将订单设置为失败状态
			if err := s.orderRepo.UpdateStatus(ctx, order.ID, model.OrderStatusFailed); err != nil {
				logger.Error("【更新订单状态失败】",
					"error", err,
					"order_id", order.ID)
			}
			// 更新订单备注
			if err := s.orderRepo.UpdateRemark(ctx, order.ID, "商品未绑定接口"); err != nil {
				logger.Error("【更新订单备注失败】",
					"error", err,
					"order_id", order.ID)
			}
			return nil, nil, fmt.Errorf("商品未绑定接口")
		}
		return nil, nil, fmt.Errorf("获取API参数信息失败: %v", err)
	}

	// 获取平台API信息 PlatformAPI
	api, err := s.platformRepo.GetAPIByID(ctx, r.APIID)
	if err != nil {
		if errors.Is(err, repository.ErrNoAPIForProduct) {
			// 将订单设置为失败状态
			if err := s.orderRepo.UpdateStatus(ctx, order.ID, model.OrderStatusFailed); err != nil {
				logger.Error("【更新订单状态失败】",
					"error", err,
					"order_id", order.ID)
			}
			// 更新订单备注
			if err := s.orderRepo.UpdateRemark(ctx, order.ID, "商品未绑定接口"); err != nil {
				logger.Error("【更新订单备注失败】",
					"error", err,
					"order_id", order.ID)
			}
			return nil, nil, fmt.Errorf("商品未绑定接口")
		}
		return nil, nil, fmt.Errorf("获取平台API信息失败: %v", err)
	}

	return api, apiParam, nil
}

func (s *rechargeService) GetPlatformAPIByOrderID(ctx context.Context, orderID string) (*model.PlatformAPI, *model.PlatformAPIParam, error) {
	// 获取订单信息
	order, err := s.orderRepo.GetByOrderID(ctx, orderID)
	if err != nil {
		return nil, nil, fmt.Errorf("获取订单信息失败: %v", err)
	}
	// 调用新的方法
	return s.getPlatformAPIByOrder(ctx, order)
}

// PushToRechargeQueue 将订单推送到充值队列
func (s *rechargeService) PushToRechargeQueue(ctx context.Context, orderID int64) error {
	logger.Info("【准备推送订单到充值队列】",
		"order_id", orderID)

	if s.redisClient == nil {
		logger.Error("【Redis客户端为空】",
			"order_id", orderID)
		return fmt.Errorf("redis client is nil")
	}

	// 检查订单是否已经在队列中，避免重复推送
	orderIDStr := strconv.FormatInt(orderID, 10)
	exists, err := s.redisClient.LPos(ctx, "recharge_queue", orderIDStr, redisV8.LPosArgs{}).Result()
	if err == nil {
		logger.Info("【订单已在充值队列中，跳过推送】",
			"order_id", orderID,
			"position", exists)
		return nil
	}
	// 如果是redis.Nil错误，说明订单不在队列中，可以继续推送
	if err != redisV8.Nil {
		logger.Error("【检查订单是否在队列中失败】",
			"error", err,
			"order_id", orderID)
		// 即使检查失败，也继续推送，避免丢失订单
	}

	err = s.redisClient.LPush(ctx, "recharge_queue", orderID).Err()
	if err != nil {
		logger.Error("【推送订单到充值队列失败】",
			"error", err,
			"order_id", orderID)
		return err
	}

	logger.Info("【推送订单到充值队列成功】",
		"order_id", orderID)
	return nil
}

// PopFromRechargeQueue 从充值队列获取订单
func (s *rechargeService) PopFromRechargeQueue(ctx context.Context) (int64, error) {
	logger.Debug("【准备从充值队列获取订单】")

	if s.redisClient == nil {
		logger.Error("【Redis客户端为空】")
		return 0, fmt.Errorf("redis client is nil")
	}

	// 使用 BRPOPLPUSH 命令，将任务从队列中移除并放入处理中队列
	result, err := s.redisClient.BRPopLPush(ctx, "recharge_queue", "recharge_processing", 0).Result()
	if err != nil {
		if err == redisV8.Nil {
			logger.Debug("【充值队列为空】")
			return 0, nil
		}
		logger.Error("【从充值队列获取订单失败】",
			"error", err)
		return 0, err
	}

	orderID, err := strconv.ParseInt(result, 10, 64)
	if err != nil {
		logger.Error("【解析订单ID失败】",
			"error", err,
			"result", result)
		return 0, fmt.Errorf("parse order id failed: %v", err)
	}

	logger.Info("【从充值队列获取订单成功】",
		"order_id", orderID)
	return orderID, nil
}

// RemoveFromProcessingQueue 从处理中队列移除任务
func (s *rechargeService) RemoveFromProcessingQueue(ctx context.Context, orderID int64) error {
	return s.redisClient.LRem(ctx, "recharge_processing", 0, orderID).Err()
}

// GetOrderByID 根据ID获取订单
func (s *rechargeService) GetOrderByID(ctx context.Context, orderID int64) (*model.Order, error) {
	return s.orderRepo.GetByID(ctx, orderID)
}

// CheckRechargingOrders 检查充值中订单
func (s *rechargeService) CheckRechargingOrders(ctx context.Context) error {
	logger.Info("【开始检查充值中订单】开始执行定时检查任务")

	// 获取所有充值中的订单
	orders, err := s.orderRepo.GetByStatus(ctx, model.OrderStatusRecharging)
	if err != nil {
		logger.Error("【获取充值中订单失败】error: %v", err)
		return fmt.Errorf("get recharging orders failed: %v", err)
	}

	logger.Info("【获取充值中订单成功】共获取到 %d 个订单", len(orders))

	now := time.Now()
	checkedCount := 0
	for _, order := range orders {
		// 检查订单是否超过5分钟
		if order.UpdatedAt.Add(5 * time.Minute).Before(now) {
			logger.Info("【发现超时订单】order_id: %d, order_number: %s, 最后更新时间: %s, 已超时: %v",
				order.ID, order.OrderNumber, order.UpdatedAt.Format("2006-01-02 15:04:05"), now.Sub(order.UpdatedAt))

			// 查询订单状态
			if err := s.manager.QueryOrderStatus(ctx, order); err != nil {
				logger.Error("【查询订单状态失败】order_id: %d, order_number: %s, error: %v",
					order.ID, order.OrderNumber, err)
				continue
			}

			logger.Info("【订单状态查询完成】order_id: %d, order_number: %s",
				order.ID, order.OrderNumber)
			checkedCount++
		}
	}

	logger.Info("【充值中订单检查完成】共检查 %d 个订单，其中 %d 个订单需要查询状态",
		len(orders), checkedCount)
	return nil
}

// SubmitOrder 提交订单到平台
func (s *rechargeService) SubmitOrder(ctx context.Context, order *model.Order, api *model.PlatformAPI, apiParam *model.PlatformAPIParam) error {
	// 获取平台实例
	platform, err := s.manager.GetPlatform(api.Code)
	if err != nil {
		return fmt.Errorf("通过 %s 获取平台失败: %v", api.Code, err)
	}
	// 提交订单到平台
	err = platform.SubmitOrder(ctx, order, api, apiParam)
	if err != nil {
		return fmt.Errorf("submit order failed: %v", err)
	}

	// 开启事务
	tx := s.orderRepo.(*repository.OrderRepositoryImpl).DB().Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			logger.Error("【事务回滚】order_id: %d, panic: %v", order.ID, r)
		}
	}()

	// 更新订单状态和成本价
	result := tx.Model(&model.Order{}).Where("id = ?", order.ID).Updates(map[string]interface{}{
		"status":      model.OrderStatusRecharging,
		"const_price": apiParam.Price,
	})
	if result.Error != nil {
		tx.Rollback()
		logger.Error("【更新订单状态和成本价失败】order_id: %d, error: %v", order.ID, result.Error)
		return fmt.Errorf("update order status and cost price failed: %v", result.Error)
	}
	if result.RowsAffected == 0 {
		tx.Rollback()
		logger.Error("【更新订单状态和成本价失败】order_id: %d, 没有记录被更新", order.ID)
		return fmt.Errorf("no record updated")
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		logger.Error("【提交事务失败】order_id: %d, error: %v", order.ID, err)
		return fmt.Errorf("commit transaction failed: %v", err)
	}

	logger.Info(fmt.Sprintf("【订单状态和成本价更新成功】order_id: %d, status: %d, const_price: %f",
		order.ID, model.OrderStatusRecharging, apiParam.Price))
	return nil
}

// ProcessRetryTask 处理重试任务
func (s *rechargeService) ProcessRetryTask(ctx context.Context, retryRecord *model.OrderRetryRecord) error {
	logger.Info("【开始处理重试任务】retry_id: %d, order_id: %d", retryRecord.ID, retryRecord.OrderID)

	// 1. 获取订单信息
	order, err := s.orderRepo.GetByID(ctx, retryRecord.OrderID)
	if err != nil {
		logger.Error("【获取订单信息失败】retry_id: %d, order_id: %d, error: %v",
			retryRecord.ID, retryRecord.OrderID, err)
		return fmt.Errorf("get order failed: %v", err)
	}
	logger.Info("【获取订单信息成功】retry_id: %d, order_id: %d, status: %d, order_number: %s",
		retryRecord.ID, retryRecord.OrderID, order.Status, order.OrderNumber)

	// 2. 获取平台API信息
	api, err := s.platformRepo.GetAPIByID(ctx, retryRecord.APIID)
	if err != nil {
		logger.Error("【获取平台API信息失败】retry_id: %d, order_id: %d, error: %v",
			retryRecord.ID, retryRecord.OrderID, err)
		return fmt.Errorf("get platform api failed: %v", err)
	}
	logger.Info("【获取平台API信息成功】retry_id: %d, order_id: %d, api_id: %d, api_name: %s",
		retryRecord.ID, retryRecord.OrderID, api.ID, api.Name)

	// 3. 获取API参数信息
	apiParam, err := s.platformAPIParamRepo.GetByID(ctx, retryRecord.ParamID)
	if err != nil {
		logger.Error("【获取API参数信息失败】retry_id: %d, order_id: %d, error: %v",
			retryRecord.ID, retryRecord.OrderID, err)
		return err
	}
	logger.Info("【获取API参数信息成功】retry_id: %d, order_id: %d, param_id: %d",
		retryRecord.ID, retryRecord.OrderID, apiParam.ID)

	// 4. 提交订单到平台
	logger.Info("【开始提交订单到平台】retry_id: %d, order_id: %d, order_number: %s",
		retryRecord.ID, retryRecord.OrderID, order.OrderNumber)
	if err := s.manager.SubmitOrder(ctx, order, api, apiParam); err != nil {
		logger.Error("【提交订单到平台失败】retry_id: %d, order_id: %d, error: %v",
			retryRecord.ID, retryRecord.OrderID, err)
		return fmt.Errorf("submit order failed: %v", err)
	}
	logger.Info("【提交订单到平台成功】retry_id: %d, order_id: %d, order_number: %s",
		retryRecord.ID, retryRecord.OrderID, order.OrderNumber)

	// 5. 开启事务
	logger.Info("【开始更新订单状态和平台信息】retry_id: %d, order_id: %d, order_number: %s",
		retryRecord.ID, retryRecord.OrderID, order.OrderNumber)
	tx := s.orderRepo.(*repository.OrderRepositoryImpl).DB().Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			logger.Error("【事务回滚】retry_id: %d, order_id: %d, panic: %v",
				retryRecord.ID, retryRecord.OrderID, r)
		}
	}()

	// 6. 更新订单状态
	logger.Info("【开始更新订单状态】retry_id: %d, order_id: %d, order_number: %s, old_status: %d, new_status: %d",
		retryRecord.ID, retryRecord.OrderID, order.OrderNumber, order.Status, model.OrderStatusRecharging)
	result := tx.Model(&model.Order{}).Where("id = ?", retryRecord.OrderID).Updates(map[string]interface{}{
		"status":      model.OrderStatusRecharging,
		"const_price": apiParam.Price,
	})
	if result.Error != nil {
		tx.Rollback()
		logger.Error("【更新订单状态失败】retry_id: %d, order_id: %d, error: %v",
			retryRecord.ID, retryRecord.OrderID, result.Error)
		return fmt.Errorf("update order status failed: %v", result.Error)
	}
	if result.RowsAffected == 0 {
		tx.Rollback()
		logger.Error("【更新订单状态失败】retry_id: %d, order_id: %d, 没有记录被更新",
			retryRecord.ID, retryRecord.OrderID)
		return fmt.Errorf("no record updated")
	}
	logger.Info("【更新订单状态和成本价成功】retry_id: %d, order_id: %d, rows_affected: %d, const_price: %f",
		retryRecord.ID, retryRecord.OrderID, result.RowsAffected, apiParam.Price)

	// 7. 更新平台信息
	logger.Info("【开始更新平台信息】retry_id: %d, order_id: %d, platform_id: %d, api_id: %d, param_id: %d",
		retryRecord.ID, retryRecord.OrderID, api.ID, api.ID, apiParam.ID)
	result = tx.Model(&model.Order{}).Where("id = ?", retryRecord.OrderID).Updates(map[string]interface{}{
		"platform_id":      api.ID,
		"api_cur_id":       api.ID,
		"api_cur_param_id": apiParam.ID,
		"platform_name":    api.Name,
		"platform_code":    api.Code,
	})
	if result.Error != nil {
		tx.Rollback()
		logger.Error("【更新平台信息失败】retry_id: %d, order_id: %d, error: %v",
			retryRecord.ID, retryRecord.OrderID, result.Error)
		return fmt.Errorf("update platform info failed: %v", result.Error)
	}
	if result.RowsAffected == 0 {
		tx.Rollback()
		logger.Error("【更新平台信息失败】retry_id: %d, order_id: %d, 没有记录被更新",
			retryRecord.ID, retryRecord.OrderID)
		return fmt.Errorf("no record updated")
	}
	logger.Info("【更新平台信息成功】retry_id: %d, order_id: %d, rows_affected: %d",
		retryRecord.ID, retryRecord.OrderID, result.RowsAffected)

	// 8. 更新重试记录状态
	logger.Info("【开始更新重试记录状态】retry_id: %d, order_id: %d", retryRecord.ID, retryRecord.OrderID)
	if err := tx.Model(&model.OrderRetryRecord{}).Where("id = ?", retryRecord.ID).Updates(map[string]interface{}{
		"status":      1, // 1: 处理成功
		"retry_count": retryRecord.RetryCount + 1,
	}).Error; err != nil {
		tx.Rollback()
		logger.Error("【更新重试记录状态失败】retry_id: %d, order_id: %d, error: %v",
			retryRecord.ID, retryRecord.OrderID, err)
		return fmt.Errorf("update retry record failed: %v", err)
	}
	logger.Info("【更新重试记录状态成功】retry_id: %d, order_id: %d", retryRecord.ID, retryRecord.OrderID)

	// 9. 提交事务
	if err := tx.Commit().Error; err != nil {
		logger.Error("【提交事务失败】retry_id: %d, order_id: %d, error: %v",
			retryRecord.ID, retryRecord.OrderID, err)
		return fmt.Errorf("commit transaction failed: %v", err)
	}
	logger.Info("【提交事务成功】retry_id: %d, order_id: %d", retryRecord.ID, retryRecord.OrderID)

	// 9.1 更新订单成本价
	logger.Info("【开始更新订单成本价】retry_id: %d, order_id: %d, const_price: %f", retryRecord.ID, retryRecord.OrderID, apiParam.Price)
	err = s.orderRepo.DB().Model(&model.Order{}).
		Where("id = ?", retryRecord.OrderID).
		Update("const_price", apiParam.Price).Error
	if err != nil {
		logger.Error("【更新订单成本价失败】retry_id: %d, order_id: %d, error: %v", retryRecord.ID, retryRecord.OrderID, err)
	} else {
		logger.Info("【更新订单成本价成功】retry_id: %d, order_id: %d, const_price: %f", retryRecord.ID, retryRecord.OrderID, apiParam.Price)
	}

	// 10. 验证更新结果
	updatedOrder, err := s.orderRepo.GetByID(ctx, retryRecord.OrderID)
	if err != nil {
		logger.Error("【验证更新结果失败】retry_id: %d, order_id: %d, error: %v",
			retryRecord.ID, retryRecord.OrderID, err)
	} else {
		logger.Info("【验证更新结果】retry_id: %d, order_id: %d, order_number: %s, status: %d, platform_id: %d",
			retryRecord.ID, retryRecord.OrderID, updatedOrder.OrderNumber, updatedOrder.Status, updatedOrder.PlatformId)
	}

	logger.Info("【重试任务处理完成】retry_id: %d, order_id: %d, order_number: %s",
		retryRecord.ID, retryRecord.OrderID, order.OrderNumber)
	return nil
}

// SetOrderService 设置订单服务
func (s *rechargeService) SetOrderService(orderService OrderService) {
	s.orderService = orderService
}

// GetBalanceService 获取余额服务
func (s *rechargeService) GetBalanceService() *PlatformAccountBalanceService {
	return s.balanceService
}

// GetUserBalanceService 获取用户余额服务
func (s *rechargeService) GetUserBalanceService() *BalanceService {
	return s.userBalanceService
}
