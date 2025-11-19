package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"recharge-go/internal/model"
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
	phoneQueryService      PhoneQueryService                       // 新增手机查询服务
	balanceQueryRecordRepo repository.BalanceQueryRecordRepository // 余额查询记录仓库
	unifiedOrderService    *UnifiedOrderService                    // 统一订单处理服务
	systemConfigService    *SystemConfigService                    // 系统配置服务
	manager                *recharge.Manager
	redisClient            *redisV8.Client
	processingOrders       map[int64]bool
	processingOrdersMu     sync.Mutex
	notificationRepo       notificationRepo.Repository
	queue                  queue.Queue
	orderService           OrderService
	notificationHelper     *NotificationHelper
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
	phoneQueryService PhoneQueryService, // 新增手机查询服务参数
	balanceQueryRecordRepo repository.BalanceQueryRecordRepository, // 余额查询记录仓库参数
	unifiedOrderService *UnifiedOrderService, // 统一订单处理服务参数
	systemConfigService *SystemConfigService, // 系统配置服务参数
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
		phoneQueryService:      phoneQueryService,      // 新增手机查询服务初始化
		balanceQueryRecordRepo: balanceQueryRecordRepo, // 余额查询记录仓库初始化
		unifiedOrderService:    unifiedOrderService,    // 统一订单处理服务初始化
		systemConfigService:    systemConfigService,    // 系统配置服务初始化
		manager:                recharge.NewManager(db),
		redisClient:            redis.GetClient(),
		processingOrders:       make(map[int64]bool),
		notificationRepo:       notificationRepo,
		queue:                  queue,
		notificationHelper:     NewNotificationHelper(db, notificationRepo, queue),
	}
}

// Recharge 执行充值
func (s *rechargeService) Recharge(ctx context.Context, orderID int64) error {
	logger.WithContextCategory(ctx, "recharge").Info("【开始执行充值】", logger.Int64V2("order_id", orderID))

	// 1. 获取订单信息
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		logger.WithContextCategory(ctx, "recharge").Error("【获取订单信息失败】", logger.Int64V2("order_id", orderID), logger.ErrorV2(err))
		return fmt.Errorf("get order failed: %v", err)
	}
	logger.WithContextCategory(ctx, "recharge").Info("【获取订单信息成功】", logger.Int64V2("order_id", orderID), logger.IntV2("status", int(order.Status)), logger.String("order_number", order.OrderNumber))

	// 检查订单状态，如果已经是充值中或已完成，则不再处理
	if order.Status == model.OrderStatusRecharging || order.Status == model.OrderStatusSuccess {
		logger.WithContextCategory(ctx, "recharge").Info("【订单状态异常，跳过处理】", logger.Int64V2("order_id", orderID), logger.IntV2("status", int(order.Status)))
		// 从处理中队列移除
		_ = s.RemoveFromProcessingQueue(ctx, orderID)
		return nil
	}

	// 2. 获取平台API信息
	api, apiParam, err := s.GetPlatformAPIByOrderID(ctx, order.OrderNumber)
	if err != nil {
		logger.WithContextCategory(ctx, "recharge").Error("【获取平台API信息失败】", logger.Int64V2("order_id", orderID), logger.ErrorV2(err))
		return fmt.Errorf("get platform api failed: %v", err)
	}
	logger.WithContextCategory(ctx, "recharge").Info("【获取平台API信息成功】",
		logger.StringV2("stage", "route"),
		logger.Int64V2("order_id", orderID),
		logger.Int64V2("api_id", api.ID),
		logger.Int64V2("platform_id", api.PlatformID))
	logger.WithContextCategory(ctx, "recharge").Info("平台API参数获取成功",
		logger.StringV2("stage", "route"),
		logger.Int64V2("order_id", orderID),
		logger.Int64V2("param_id", apiParam.ID))
	// 3. 提交订单到平台
	logger.WithContextCategory(ctx, "recharge").Info("【开始提交订单到平台】",
		logger.StringV2("stage", "submit"),
		logger.Int64V2("order_id", orderID),
		logger.Int64V2("platform_id", api.PlatformID))
	if err := s.manager.SubmitOrder(ctx, order, api, apiParam); err != nil {
		logger.WithContextCategory(ctx, "recharge").Error("【提交订单到平台失败】",
			logger.StringV2("stage", "submit"),
			logger.Int64V2("order_id", orderID),
			logger.ErrorV2(err))

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
			logger.WithContextCategory(ctx, "recharge").Error("【获取重试记录失败】", logger.Int64V2("order_id", orderID), logger.ErrorV2(err))
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
			logger.WithContextCategory(ctx, "recharge").Error("【创建重试记录失败】", logger.Int64V2("order_id", orderID), logger.ErrorV2(err))
		} else {
			logger.WithContextCategory(ctx, "recharge").Info("【创建重试记录成功】", logger.Int64V2("order_id", orderID), logger.Int64V2("retry_id", retryRecord.ID))
		}

		return fmt.Errorf("submit order failed: %v", err)
	}
	logger.WithContextCategory(ctx, "recharge").Info("【提交订单到平台成功】", logger.Int64V2("order_id", orderID))

	// 3.1 记录使用的API（成功提交后立即记录）
	usedAPIs := []map[string]interface{}{
		{
			"api_id": api.ID,
		},
	}
	usedAPIsJSON, _ := json.Marshal(usedAPIs)

	retryRecord := &model.OrderRetryRecord{
		OrderID:       orderID,
		APIID:         api.ID,
		ParamID:       apiParam.ID,
		RetryType:     1,  // 1: 平台切换
		RetryCount:    0,  // 首次提交
		LastError:     "", // 成功提交，无错误
		RetryParams:   "{}",
		UsedAPIs:      string(usedAPIsJSON),
		Status:        1, // 1: 已处理（成功提交）
		NextRetryTime: time.Now(),
	}

	if err := s.retryRepo.Create(ctx, retryRecord); err != nil {
		logger.WithContextCategory(ctx, "recharge").Error("【记录使用API失败】", logger.Int64V2("order_id", orderID), logger.ErrorV2(err))
		// 不影响主流程，继续执行
	} else {
		logger.WithContextCategory(ctx, "recharge").Info("【记录使用API成功】", logger.Int64V2("order_id", orderID), logger.Int64V2("api_id", api.ID))
	}

	// 4. 开启事务
	logger.WithContextCategory(ctx, "recharge").Info("【开始更新订单状态和平台信息】", logger.Int64V2("order_id", orderID))
	tx := s.orderRepo.(*repository.OrderRepositoryImpl).DB().Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			logger.WithContextCategory(ctx, "recharge").Error("【事务回滚】", logger.Int64V2("order_id", orderID), logger.AnyV2("panic", r))
		}
	}()

	// 5. 更新订单状态
	logger.WithContextCategory(ctx, "recharge").Info("【开始更新订单状态】", logger.Int64V2("order_id", orderID), logger.IntV2("old_status", int(order.Status)), logger.IntV2("new_status", int(model.OrderStatusRecharging)))
	result := tx.Model(&model.Order{}).Where("id = ?", orderID).Update("status", model.OrderStatusRecharging)
	if result.Error != nil {
		tx.Rollback()
		logger.WithContextCategory(ctx, "recharge").Error("【更新订单状态失败】", logger.Int64V2("order_id", orderID), logger.ErrorV2(result.Error))
		return fmt.Errorf("update order status failed: %v", result.Error)
	}
	if result.RowsAffected == 0 {
		tx.Rollback()
		logger.WithContextCategory(ctx, "recharge").Error("【更新订单状态失败】", logger.Int64V2("order_id", orderID), logger.String("detail", "没有记录被更新"))
		return fmt.Errorf("no record updated")
	}
	logger.WithContextCategory(ctx, "recharge").Info("【更新订单状态成功】", logger.Int64V2("order_id", orderID), logger.Int64V2("rows_affected", result.RowsAffected))

	// 6. 更新平台信息（包含已使用通道记录，用于回调阶段判定可用通道）
	logger.WithContextCategory(ctx, "recharge").Info("【开始更新平台信息】", logger.Int64V2("order_id", orderID), logger.Int64V2("platform_id", api.PlatformID), logger.Int64V2("api_id", api.ID), logger.Int64V2("param_id", apiParam.ID))
	result = tx.Model(&model.Order{}).Where("id = ?", orderID).Updates(map[string]interface{}{
		"api_cur_id":       api.ID,
		"api_cur_param_id": apiParam.ID,
		"used_apis":        string(usedAPIsJSON),
	})
	if result.Error != nil {
		tx.Rollback()
		logger.WithContextCategory(ctx, "recharge").Error("【更新平台信息失败】", logger.Int64V2("order_id", orderID), logger.ErrorV2(result.Error))
		return fmt.Errorf("update platform info failed: %v", result.Error)
	}
	if result.RowsAffected == 0 {
		tx.Rollback()
		logger.WithContextCategory(ctx, "recharge").Error("【更新平台信息失败】", logger.Int64V2("order_id", orderID), logger.String("detail", "没有记录被更新"))
		return fmt.Errorf("no record updated")
	}
	logger.WithContextCategory(ctx, "recharge").Info("【更新平台信息成功】", logger.Int64V2("order_id", orderID), logger.Int64V2("rows_affected", result.RowsAffected))

	// 7. 提交事务
	if err := tx.Commit().Error; err != nil {
		logger.WithContextCategory(ctx, "recharge").Error("【提交事务失败】", logger.Int64V2("order_id", orderID), logger.ErrorV2(err))
		return fmt.Errorf("commit transaction failed: %v", err)
	}
	logger.WithContextCategory(ctx, "recharge").Info("【提交事务成功】", logger.Int64V2("order_id", orderID))

	// 8. 从处理中队列移除
	logger.WithContextCategory(ctx, "recharge").Info("【从处理中队列移除】", logger.Int64V2("order_id", orderID))
	if err := s.RemoveFromProcessingQueue(ctx, orderID); err != nil {
		logger.WithContextCategory(ctx, "recharge").Error("【从处理中队列移除失败】", logger.Int64V2("order_id", orderID), logger.ErrorV2(err))
	}

	// 9. 验证更新结果
	updatedOrder, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		logger.WithContextCategory(ctx, "recharge").Error("【验证更新结果失败】", logger.Int64V2("order_id", orderID), logger.ErrorV2(err))
	} else {
		logger.WithContextCategory(ctx, "recharge").Info("【验证更新结果】", logger.Int64V2("order_id", orderID), logger.IntV2("status", int(updatedOrder.Status)), logger.Int64V2("platform_id", updatedOrder.PlatformId))
	}

	// 提交成功后，更新订单的 const_price 字段为 apiParam.Price
	err = s.orderRepo.DB().Model(&model.Order{}).
		Where("id = ?", order.ID).
		Update("const_price", apiParam.Price).Error
	if err != nil {
		logger.WithContextCategory(ctx, "recharge").Error("【更新订单成本价失败】", logger.Int64V2("order_id", order.ID), logger.ErrorV2(err))
		// 新增：将订单状态设置为失败，并写入备注
		_ = s.orderRepo.UpdateStatus(ctx, order.ID, model.OrderStatusFailed)
		_ = s.orderRepo.UpdateRemark(ctx, order.ID, "余额不足，订单失败")
		// 发送状态变更通知（幂等）
		if nErr := s.sendOrderStatusNotificationWithIdempotency(ctx, order, model.OrderStatusFailed); nErr != nil {
			logger.WithContextCategory(ctx, "recharge").Error("发送订单失败通知失败", logger.Int64V2("order_id", order.ID), logger.ErrorV2(nErr))
		}
	} else {
		logger.WithContextCategory(ctx, "recharge").Info("【更新订单成本价成功】", logger.Int64V2("order_id", order.ID), logger.Float64V2("const_price", apiParam.Price))
	}

	logger.WithContextCategory(ctx, "recharge").Info("【充值流程完成】", logger.Int64V2("order_id", orderID))
	return nil
}

// sendOrderStatusNotificationWithIdempotency 发送订单状态变更通知（委托给NotificationHelper）
func (s *rechargeService) sendOrderStatusNotificationWithIdempotency(ctx context.Context, order *model.Order, newStatus model.OrderStatus) error {
	return s.notificationHelper.SendOrderStatusNotification(ctx, order, newStatus)
}

// getISPTypeFromOrder 根据订单信息确定运营商类型
func (s *rechargeService) getISPTypeFromOrder(order *model.Order) string {
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

// HandleCallback 处理平台回调
func (s *rechargeService) HandleCallback(ctx context.Context, platformName string, data []byte) error {
	// 1. 解析回调数据
	callbackData, err := s.manager.ParseCallbackData(platformName, data)
	if err != nil {
		logger.WithContextCategory(ctx, "recharge").Error("解析回调数据失败", logger.ErrorV2(err))
		return fmt.Errorf("parse callback data failed service 层: %v", err)
	}
	// 注入订单号至上下文，便于全链路日志
	if callbackData.OrderNumber != "" {
		ctx = logger.InjectOrderNumber(ctx, callbackData.OrderNumber)
	}
	logger.WithContextCategory(ctx, "recharge").Info("收到回调，解析成功",
		logger.StringV2("platform", platformName),
		logger.StringV2("callback_type", callbackData.CallbackType),
	)

	// 2. 检查是否已处理过该回调（使用订单号、回调类型和平台交易ID进行精确匹配）
	// 优先使用平台交易ID进行重复检查，避免换通道重试时的误判
	var exists *model.CallbackLog

	// 只有当OrderNumber不为空时才进行重复检查，避免空字符串匹配到错误的记录
	if callbackData.OrderNumber != "" {
		if callbackData.TransactionID != "" {
			// 如果有平台交易ID，使用更精确的检查
			exists, err = s.callbackLogRepo.GetByOrderIDTypeAndPlatformID(ctx, callbackData.OrderNumber, callbackData.CallbackType, callbackData.TransactionID)
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				logger.WithContextCategory(ctx, "recharge").Error("检查回调记录失败", logger.ErrorV2(err))
				return err
			}
			if exists != nil {
				logger.WithContextCategory(ctx, "recharge").Info("回调已处理过",
					logger.StringV2("order_number", callbackData.OrderNumber),
					logger.StringV2("callback_type", callbackData.CallbackType),
					logger.StringV2("platform_id", callbackData.TransactionID),
				)
			}
		} else {
			// 如果没有平台交易ID，使用原有的检查方式
			exists, err = s.callbackLogRepo.GetByOrderIDAndType(ctx, callbackData.OrderNumber, callbackData.CallbackType)
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				logger.WithContextCategory(ctx, "recharge").Error("检查回调记录失败", logger.ErrorV2(err))
				return err
			}
			if exists != nil {
				logger.WithContextCategory(ctx, "recharge").Info("回调已处理过",
					logger.StringV2("order_number", callbackData.OrderNumber),
					logger.StringV2("callback_type", callbackData.CallbackType),
				)
			}
		}
	} else {
		logger.WithContextCategory(ctx, "recharge").Warn("回调数据中OrderNumber为空，跳过重复检查")
	}

	// 3. 开启事务
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 4. 处理回调
	logger.WithContextCategory(ctx, "recharge").Info("开始处理平台回调",
		logger.StringV2("platform", platformName),
		logger.StringV2("raw_body", string(data)),
	)
	if err := s.manager.HandleCallback(ctx, platformName, data); err != nil {
		tx.Rollback()
		logger.WithContextCategory(ctx, "recharge").Error("处理回调失败", logger.ErrorV2(err))
		return fmt.Errorf("handle callback failed: %v", err)
	}

	// 4.1 更新订单状态
	var orderState model.OrderStatus
	logger.WithContextCategory(ctx, "recharge").Info("回调数据解析结果",
		logger.StringV2("order_number", callbackData.OrderNumber),
		logger.StringV2("order_id", callbackData.OrderID),
		logger.StringV2("transaction_id", callbackData.TransactionID),
		logger.StringV2("status_raw", callbackData.Status),
	)
	switch callbackData.Status {
	case "success":
		orderState = model.OrderStatusSuccess
	case "4":
		orderState = model.OrderStatusSuccess
	case "failed":
		orderState = model.OrderStatusFailed
	case "3": // kekebang平台充值中状态
		orderState = model.OrderStatusRecharging
	case "5":
		orderState = model.OrderStatusFailed
	case "6":
		orderState = model.OrderStatusFailed
	case "7":
		orderState = model.OrderStatusFailed
	case "processing":
		orderState = model.OrderStatusRecharging
	default:
		tx.Rollback()
		logger.WithContextCategory(ctx, "recharge").Error("未知的订单状态", logger.StringV2("status", callbackData.Status))
		return fmt.Errorf("unknown order status: %s", callbackData.Status)
	}

	// 获取订单信息
	logger.WithContextCategory(ctx, "recharge").Info("查询订单", logger.StringV2("callback_order_number", callbackData.OrderNumber))
	order, err := s.orderRepo.GetByOrderID(ctx, callbackData.OrderNumber)
	if err != nil {
		// 如果按订单号找不到，尝试通过ActiveOutTradeNum反向查找原始订单
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.WithContextCategory(ctx, "recharge").Info("按订单号未找到，尝试通过ActiveOutTradeNum查找重试记录",
				logger.StringV2("callback_order_number", callbackData.OrderNumber),
			)

			// 通过ActiveOutTradeNum查找重试记录
			retryRecord, retryErr := s.retryRepo.GetByActiveOutTradeNum(ctx, callbackData.OrderNumber)
			if retryErr != nil {
				tx.Rollback()
				logger.WithContextCategory(ctx, "recharge").Error("通过ActiveOutTradeNum查找重试记录失败", logger.ErrorV2(retryErr))
				return fmt.Errorf("get retry record by active_out_trade_num failed: %v", retryErr)
			}

			// 通过重试记录的OrderID查找原始订单
			order, err = s.orderRepo.GetByID(ctx, retryRecord.OrderID)
			if err != nil {
				tx.Rollback()
				logger.WithContextCategory(ctx, "recharge").Error("通过重试记录OrderID查找原始订单失败", logger.ErrorV2(err))
				return fmt.Errorf("get original order by retry record failed: %v", err)
			}

			// 覆盖注入原始订单号
			ctx = logger.InjectOrderNumber(ctx, order.OrderNumber)
			logger.WithContextCategory(ctx, "recharge").Info("通过ActiveOutTradeNum成功找到原始订单",
				logger.StringV2("callback_order_number", callbackData.OrderNumber),
				logger.StringV2("original_order_number", order.OrderNumber),
				logger.Int64V2("order_id", order.ID),
				logger.Int64V2("retry_record_id", retryRecord.ID),
			)
		} else {
			tx.Rollback()
			logger.WithContextCategory(ctx, "recharge").Error("获取订单信息失败", logger.ErrorV2(err))
			return fmt.Errorf("get order failed: %v", err)
		}
	}
	logger.WithContextCategory(ctx, "recharge").Info("查询订单结果",
		logger.Int64V2("order_id", order.ID),
		logger.StringV2("order_number", order.OrderNumber),
		logger.StringV2("status", fmt.Sprintf("%v", order.Status)),
		logger.StringV2("mapped_state", fmt.Sprintf("%v", orderState)),
		logger.Int64V2("product_id", order.ProductID),
	)

	// 4.2 终态门槛：订单已终态(成功/失败)则忽略本次回调，不修改订单状态
	if order.Status == model.OrderStatusSuccess || order.Status == model.OrderStatusFailed {
		platformID := callbackData.TransactionID
		if platformID == "" {
			platformID = callbackData.OrderID
		}
		log := &model.CallbackLog{
			OrderID:      callbackData.OrderNumber,
			PlatformID:   platformID,
			CallbackType: callbackData.CallbackType,
			Status:       2,
			RequestData:  string(data),
			ResponseData: "ignored_terminal",
			CreateTime:   time.Now(),
			UpdateTime:   time.Now(),
		}
		if err := s.callbackLogRepo.Create(ctx, log); err != nil {
			logger.WithContextCategory(ctx, "recharge").Error("记录终态忽略回调失败", logger.ErrorV2(err))
		}
		if err := tx.Commit().Error; err != nil {
			logger.WithContextCategory(ctx, "recharge").Error("提交事务失败(终态忽略)", logger.ErrorV2(err))
		}
		logger.WithContextCategory(ctx, "recharge").Info("订单已终态，忽略回调",
			logger.Int64V2("order_id", order.ID),
			logger.StringV2("order_number", order.OrderNumber),
		)
		return nil
	}

	// 4.3 非终态回调：仅记录，不修改订单状态（如 processing/取消 等）
	if orderState != model.OrderStatusSuccess && orderState != model.OrderStatusFailed {
		platformID := callbackData.TransactionID
		if platformID == "" {
			platformID = callbackData.OrderID
		}
		log := &model.CallbackLog{
			OrderID:      callbackData.OrderNumber,
			PlatformID:   platformID,
			CallbackType: callbackData.CallbackType,
			Status:       2,
			RequestData:  string(data),
			ResponseData: "accepted_non_terminal",
			CreateTime:   time.Now(),
			UpdateTime:   time.Now(),
		}
		if err := s.callbackLogRepo.Create(ctx, log); err != nil {
			logger.WithContextCategory(ctx, "recharge").Error("记录非终态回调失败", logger.ErrorV2(err))
		}
		if err := tx.Commit().Error; err != nil {
			logger.WithContextCategory(ctx, "recharge").Error("提交事务失败(非终态)", logger.ErrorV2(err))
			return fmt.Errorf("commit transaction failed: %v", err)
		}
		logger.WithContextCategory(ctx, "recharge").Info("已接受非终态回调，不修改订单",
			logger.Int64V2("order_id", order.ID),
			logger.StringV2("order_number", order.OrderNumber),
			logger.StringV2("mapped_state", fmt.Sprintf("%v", orderState)),
		)
		return nil
	}

	// 检查订单当前状态，如果已经是最终状态，忽略后续回调
	// if order.Status == model.OrderStatusSuccess || order.Status == model.OrderStatusFailed {
	// 	tx.Rollback()
	// 	logger.Info(fmt.Sprintf("订单已处于最终状态，忽略回调: 订单号%s, 当前状态%s, 回调状态%s, 平台%s",
	// 		order.OrderNumber, order.Status, orderState, platformName))
	// 	return nil
	// }

	// === 处理失败回调：检查是否还有其他通道可用 ===
	// 根据订单状态分别处理
	if orderState == model.OrderStatusFailed {
		// 失败订单处理逻辑
		return s.handleFailedOrderCallback(ctx, tx, order, callbackData, data, platformName)
	} else {
		// 成功订单处理逻辑（仅终态更新）
		return s.handleSuccessOrderCallback(ctx, tx, order, callbackData, data, platformName, model.OrderStatus(orderState))
	}

}

// handleFailedOrderCallback 处理失败订单回调
func (s *rechargeService) handleFailedOrderCallback(ctx context.Context, tx *gorm.DB, order *model.Order, callbackData *model.CallbackData, data []byte, platformName string) error {
	// 处理失败订单回调日志

	// 检查是否还有其他通道可以重试
	// 失败回调不走重试，直接进行最终失败处理

	// 没有可用通道或检查失败，进行最终失败处理
	logger.WithContextCategory(ctx, "recharge").Info("没有可用通道，进行最终失败处理",
		logger.Int64V2("order_id", order.ID),
		logger.StringV2("order_number", order.OrderNumber),
		logger.StringV2("platform", platformName))

	// 更新订单状态为失败
	if s.unifiedOrderService != nil {
		logger.WithContextCategory(ctx, "recharge").Info("使用统一订单服务处理失败状态更新", logger.Int64V2("order_id", order.ID))
		if err := s.unifiedOrderService.ProcessOrderStatusChangeWithBalanceCheck(ctx, order.ID, model.OrderStatusFailed, "平台", true); err != nil {
			tx.Rollback()
			logger.WithContextCategory(ctx, "recharge").Error("统一订单状态更新失败", logger.Int64V2("order_id", order.ID), logger.ErrorV2(err))
			return fmt.Errorf("unified order status update failed: %v", err)
		}
	} else {
		// 降级处理：手动退款和更新状态
		logger.WithContextCategory(ctx, "recharge").Warn("统一订单服务未初始化，使用手动退款和状态更新", logger.Int64V2("order_id", order.ID))

		// 退款
		err := s.balanceService.RefundBalance(ctx, order.CustomerID, order.Price, order.ID, "订单失败退还余额")
		if err != nil {
			tx.Rollback()
			logger.WithContextCategory(ctx, "recharge").Error("订单失败退款失败", logger.Int64V2("order_id", order.ID), logger.ErrorV2(err))
			return fmt.Errorf("订单失败退款失败: %v", err)
		}

		// 更新订单状态
		if err := s.orderRepo.UpdateStatus(ctx, order.ID, model.OrderStatusFailed); err != nil {
			tx.Rollback()
			logger.WithContextCategory(ctx, "recharge").Error("更新订单状态失败", logger.Int64V2("order_id", order.ID), logger.ErrorV2(err))
			return fmt.Errorf("update order status failed: %v", err)
		}
	}

	// 发送状态变更通知（幂等）- 仅在未使用统一订单服务时
	if s.unifiedOrderService == nil {
		if nErr := s.sendOrderStatusNotificationWithIdempotency(ctx, order, model.OrderStatusFailed); nErr != nil {
			tx.Rollback()
			logger.WithContextCategory(ctx, "recharge").Error("发送订单失败通知失败", logger.Int64V2("order_id", order.ID), logger.ErrorV2(nErr))
			return fmt.Errorf("push notification to queue failed: %v", nErr)
		}
		logger.WithContextCategory(ctx, "recharge").Info("订单失败推送通知到队列成功", logger.Int64V2("order_id", order.ID))
	}

	// 记录回调日志
	platformID := callbackData.TransactionID
	if platformID == "" {
		platformID = callbackData.OrderID
	}

	log := &model.CallbackLog{
		OrderID:      callbackData.OrderNumber,
		PlatformID:   platformID,
		CallbackType: callbackData.CallbackType,
		Status:       1,
		RequestData:  string(data),
		ResponseData: "failed_processed",
		CreateTime:   time.Now(),
		UpdateTime:   time.Now(),
	}
	if err := s.callbackLogRepo.Create(ctx, log); err != nil {
		tx.Rollback()
		logger.WithContextCategory(ctx, "recharge").Error("记录回调日志失败", logger.Int64V2("order_id", order.ID), logger.ErrorV2(err))
		return fmt.Errorf("create callback log failed: %v", err)
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		logger.WithContextCategory(ctx, "recharge").Error("提交事务失败", logger.Int64V2("order_id", order.ID), logger.ErrorV2(err))
		return fmt.Errorf("commit transaction failed: %v", err)
	}

	logger.WithContextCategory(ctx, "recharge").Info("失败订单回调处理完成", logger.Int64V2("order_id", order.ID), logger.StringV2("order_number", order.OrderNumber))
	return nil
}

// handleSuccessOrderCallback 处理成功订单回调
func (s *rechargeService) handleSuccessOrderCallback(ctx context.Context, tx *gorm.DB, order *model.Order, callbackData *model.CallbackData, data []byte, platformName string, orderState model.OrderStatus) error {
	// 处理成功订单回调日志

	// 使用统一订单处理服务更新订单状态
	logger.WithContextCategory(ctx, "recharge").Info("准备更新订单状态",
		logger.StringV2("order_number", order.OrderNumber),
		logger.Int64V2("order_id", order.ID),
		logger.StringV2("from_to", fmt.Sprintf("%v->%v", order.Status, orderState)),
	)
	if s.unifiedOrderService != nil {
		logger.WithContextCategory(ctx, "recharge").Info("使用统一订单服务处理状态更新",
			logger.Int64V2("order_id", order.ID),
			logger.StringV2("target_status", fmt.Sprintf("%v", orderState)),
		)
		if err := s.unifiedOrderService.ProcessOrderStatusChangeWithBalanceCheck(ctx, order.ID, orderState, "平台", true); err != nil {
			tx.Rollback()
			logger.WithContextCategory(ctx, "recharge").Error("统一订单状态更新失败", logger.Int64V2("order_id", order.ID), logger.ErrorV2(err))
			return fmt.Errorf("unified order status update failed: %v", err)
		}
		logger.WithContextCategory(ctx, "recharge").Info("统一订单回调更新订单状态成功",
			logger.StringV2("order_number", order.OrderNumber),
			logger.Int64V2("order_id", order.ID),
			logger.StringV2("status", fmt.Sprintf("%v", orderState)),
		)
	} else {
		// 降级到原有的简单状态更新
		logger.WithContextCategory(ctx, "recharge").Warn("统一订单服务未初始化，使用原有的简单状态更新", logger.Int64V2("order_id", order.ID))
		if err := s.orderRepo.UpdateStatus(ctx, order.ID, orderState); err != nil {
			tx.Rollback()
			logger.WithContextCategory(ctx, "recharge").Error("更新订单状态失败", logger.Int64V2("order_id", order.ID), logger.ErrorV2(err))
			return fmt.Errorf("update order status failed: %v", err)
		}
		logger.WithContextCategory(ctx, "recharge").Info("订单回调更新订单状态成功",
			logger.StringV2("order_number", order.OrderNumber),
			logger.Int64V2("order_id", order.ID),
			logger.StringV2("status", fmt.Sprintf("%v", orderState)),
		)
	}

	// 发送状态变更通知（幂等）- 仅在未使用统一订单服务时
	if s.unifiedOrderService == nil {
		if nErr := s.sendOrderStatusNotificationWithIdempotency(ctx, order, orderState); nErr != nil {
			tx.Rollback()
			logger.WithContextCategory(ctx, "recharge").Error("发送订单状态通知失败", logger.Int64V2("order_id", order.ID), logger.StringV2("status", fmt.Sprintf("%v", orderState)), logger.ErrorV2(nErr))
			return fmt.Errorf("push notification to queue failed: %v", nErr)
		}
		logger.WithContextCategory(ctx, "recharge").Info("订单推送通知到队列成功", logger.Int64V2("order_id", order.ID), logger.StringV2("status", fmt.Sprintf("%v", orderState)))
	}

	// 记录回调日志
	platformID := callbackData.TransactionID
	if platformID == "" {
		platformID = callbackData.OrderID
	}

	log := &model.CallbackLog{
		OrderID:      callbackData.OrderNumber,
		PlatformID:   platformID,
		CallbackType: callbackData.CallbackType,
		Status:       1,
		RequestData:  string(data),
		ResponseData: "success",
		CreateTime:   time.Now(),
		UpdateTime:   time.Now(),
	}
	if err := s.callbackLogRepo.Create(ctx, log); err != nil {
		tx.Rollback()
		logger.WithContextCategory(ctx, "recharge").Error("记录回调日志失败", logger.Int64V2("order_id", order.ID), logger.ErrorV2(err))
		return fmt.Errorf("create callback log failed: %v", err)
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		logger.WithContextCategory(ctx, "recharge").Error("提交事务失败", logger.Int64V2("order_id", order.ID), logger.ErrorV2(err))
		return fmt.Errorf("commit transaction failed: %v", err)
	}

	logger.WithContextCategory(ctx, "recharge").Info("成功订单回调处理完成",
		logger.Int64V2("order_id", order.ID),
		logger.StringV2("order_number", order.OrderNumber),
		logger.StringV2("status", fmt.Sprintf("%v", orderState)),
	)
	return nil
}

// GetPendingTasks 获取待处理的充值任务
func (s *rechargeService) GetPendingTasks(ctx context.Context, limit int) ([]*model.Order, error) {
	// 从Redis队列中获取待处理的订单ID
	if s.redisClient == nil {
		logger.WithContextCategory(ctx, "recharge").Error("【Redis客户端为空】")
		return nil, fmt.Errorf("redis client is nil")
	}

	// 获取队列中的订单ID列表
	orderIDs, err := s.redisClient.LRange(ctx, "recharge_queue", 0, int64(limit-1)).Result()
	if err != nil {
		logger.WithContextCategory(ctx, "recharge").Error("【从Redis队列获取订单ID失败】", logger.ErrorV2(err))
		return nil, fmt.Errorf("get order IDs from queue failed: %v", err)
	}

	logger.WithContextCategory(ctx, "recharge").Info("【调试：从Redis获取的订单ID列表】", logger.AnyV2("order_ids", orderIDs), logger.IntV2("limit", limit))

	if len(orderIDs) == 0 {
		logger.WithContextCategory(ctx, "recharge").Info("【Redis队列中没有待处理订单】")
		return []*model.Order{}, nil
	}

	// 将字符串ID转换为int64并查询订单信息
	var orders []*model.Order
	now := time.Now()
	for _, orderIDStr := range orderIDs {
		orderID, err := strconv.ParseInt(orderIDStr, 10, 64)
		if err != nil {
			logger.WithContextCategory(ctx, "recharge").Error("【解析订单ID失败】", logger.StringV2("order_id_str", orderIDStr), logger.ErrorV2(err))
			continue
		}

		// 获取订单信息
		order, err := s.orderRepo.GetByID(ctx, orderID)
		if err != nil {
			logger.WithContextCategory(ctx, "recharge").Error("【获取订单信息失败】", logger.Int64V2("order_id", orderID), logger.ErrorV2(err))
			// 从Redis队列中移除该订单
			if removeErr := s.redisClient.LRem(ctx, "recharge_queue", 0, orderIDStr).Err(); removeErr != nil {
				logger.WithContextCategory(ctx, "recharge").Error("【从队列移除失效订单失败】", logger.Int64V2("order_id", orderID), logger.ErrorV2(removeErr))
			} else {
				logger.WithContextCategory(ctx, "recharge").Info("【成功从队列移除失效订单】", logger.Int64V2("order_id", orderID))
			}
			continue
		}

		logger.WithContextCategory(ctx, "recharge").Info("【调试：检查订单】",
			logger.Int64V2("order_id", orderID),
			logger.StringV2("status", fmt.Sprintf("%v", order.Status)),
			logger.TimeV2("created_at", order.CreatedAt),
			logger.TimeV2("updated_at", order.UpdatedAt),
		)

		// 检查订单状态和时间过滤条件
		if order.Status != model.OrderStatusPendingRecharge {
			logger.WithContextCategory(ctx, "recharge").Info("【订单状态不是待充值，从队列中移除】",
				logger.Int64V2("order_id", orderID),
				logger.StringV2("status", fmt.Sprintf("%v", order.Status)),
			)
			// 从Redis队列中移除该订单
			if err := s.redisClient.LRem(ctx, "recharge_queue", 0, orderIDStr).Err(); err != nil {
				logger.WithContextCategory(ctx, "recharge").Error("【从队列移除订单失败】", logger.Int64V2("order_id", orderID), logger.ErrorV2(err))
			} else {
				logger.WithContextCategory(ctx, "recharge").Info("【成功从队列移除订单】", logger.Int64V2("order_id", orderID))
			}
			continue
		}

		// 只对非新订单应用1分钟冷却机制
		// 新订单的创建时间和更新时间相差很小（通常在几毫秒内），不应该被冷却机制拦截
		timeDiff := order.UpdatedAt.Sub(order.CreatedAt)
		isNewOrder := timeDiff < 5*time.Second // 5秒内的时间差认为是新订单

		logger.WithContextCategory(ctx, "recharge").Info("【调试：时间检查】",
			logger.Int64V2("order_id", orderID),
			logger.DurationV2("time_diff", timeDiff),
			logger.BoolV2("is_new_order", isNewOrder),
		)

		if !isNewOrder && order.UpdatedAt.Add(1*time.Minute).After(now) {
			logger.WithContextCategory(ctx, "recharge").Info("【订单最近1分钟内被处理过，跳过】", logger.Int64V2("order_id", orderID))
			continue
		}

		// 如果订单创建时间超过24小时，跳过
		// 优先使用CreateTime字段，如果为空则使用CreatedAt字段
		createTime := order.CreateTime
		if createTime.IsZero() && !order.CreatedAt.IsZero() {
			createTime = order.CreatedAt
		}

		if !createTime.IsZero() && createTime.Add(24*time.Hour).Before(now) {
			logger.WithContextCategory(ctx, "recharge").Info("【订单创建时间超过24小时，跳过】",
				logger.Int64V2("order_id", orderID),
				logger.TimeV2("create_time", createTime),
			)
			continue
		}

		orders = append(orders, order)
	}

	logger.WithContextCategory(ctx, "recharge").Info("【获取到待处理订单】", logger.IntV2("count", len(orders)))
	return orders, nil
}

// ProcessRechargeTask 处理充值任务
func (s *rechargeService) ProcessRechargeTask(ctx context.Context, order *model.Order) error {
	logger.WithContextCategory(ctx, "recharge").Info("【开始处理充值任务】",
		logger.Int64V2("order_id", order.ID),
		logger.StringV2("order_number", order.OrderNumber),
		logger.StringV2("mobile", order.Mobile))

	// 1. 获取商品信息，检查是否开启重复检查
	product, err := s.productRepo.GetByID(ctx, order.ProductID)
	if err != nil {
		logger.WithContextCategory(ctx, "recharge").Error("【获取商品信息失败】",
			logger.Int64V2("order_id", order.ID),
			logger.Int64V2("product_id", order.ProductID),
			logger.ErrorV2(err))
		return fmt.Errorf("获取商品信息失败: %v", err)
	}

	// 2. 防重复充值检查（仅在商品开启重复检查时执行）
	if product.DuplicateCheck {
		logger.WithContextCategory(ctx, "recharge").Info("【商品已开启重复检查，开始执行重复订单检查】",
			logger.Int64V2("order_id", order.ID),
			logger.Int64V2("product_id", order.ProductID))

		// 查询相同手机号、金额、运营商、商品ID的进行中订单（排除当前订单）
		processingStatuses := []model.OrderStatus{
			model.OrderStatusPendingPayment,
			model.OrderStatusPendingRecharge,
			model.OrderStatusRecharging,
			model.OrderStatusProcessing,
		}

		existingOrder, err := s.orderRepo.FindDuplicateOrder(ctx, order.Mobile, order.Denom, order.ISP, order.ProductID, processingStatuses)
		if err != nil && err != gorm.ErrRecordNotFound {
			logger.WithContextCategory(ctx, "recharge").Error("【检查重复订单失败】",
				logger.Int64V2("order_id", order.ID),
				logger.ErrorV2(err))
			return fmt.Errorf("检查重复订单失败: %v", err)
		}

		if existingOrder != nil && existingOrder.ID != order.ID {
			logger.WithContextCategory(ctx, "recharge").Error("【检测到重复充值，设置订单为失败】",
				logger.Int64V2("order_id", order.ID),
				logger.StringV2("mobile", order.Mobile),
				logger.Float64V2("denom", order.Denom),
				logger.IntV2("isp", order.ISP),
				logger.Int64V2("product_id", order.ProductID),
				logger.Int64V2("existing_order_id", existingOrder.ID),
				logger.StringV2("existing_order_number", existingOrder.OrderNumber))

			// 直接设置订单为失败
			errorMsg := fmt.Sprintf("检测到重复充值：手机号 %s，金额 %.2f，运营商 %d，商品ID %d 存在进行中的订单 %s",
				order.Mobile, order.Denom, order.ISP, order.ProductID, existingOrder.OrderNumber)
			if failErr := s.orderService.ProcessOrderFail(ctx, order.ID, errorMsg); failErr != nil {
				logger.WithContextCategory(ctx, "recharge").Error("【设置订单失败状态失败】",
					logger.Int64V2("order_id", order.ID),
					logger.ErrorV2(failErr))
				return failErr
			}
			return nil // 返回nil表示任务处理完成，不需要重试
		}
	} else {
		logger.WithContextCategory(ctx, "recharge").Info("【商品未开启重复检查，跳过重复订单检查】",
			logger.Int64V2("order_id", order.ID),
			logger.Int64V2("product_id", order.ProductID))
	}

	// 3. 充值前查询余额并记录
	// 检查余额验证开关
	balanceVerificationEnabled, err := s.systemConfigService.GetBoolValue(ctx, "balance_verification_enabled")
	if err != nil {
		logger.WithContextCategory(ctx, "recharge").Error("【获取余额验证开关配置失败】",
			logger.Int64V2("order_id", order.ID),
			logger.ErrorV2(err))
		// 配置获取失败时，默认启用余额验证以保证安全性
		balanceVerificationEnabled = true
	}

	if !balanceVerificationEnabled {
		logger.WithContextCategory(ctx, "recharge").Info("【余额验证已关闭，跳过充值前余额查询】", logger.Int64V2("order_id", order.ID))
	} else {
		logger.WithContextCategory(ctx, "recharge").Info("【调试：余额查询条件检查】",
			logger.Int64V2("order_id", order.ID),
			logger.BoolV2("phoneQueryService_nil", s.phoneQueryService == nil),
			logger.StringV2("mobile", order.Mobile))
		if s.phoneQueryService != nil && order.Mobile != "" {
			logger.WithContextCategory(ctx, "recharge").Info("【充值前查询余额】",
				logger.Int64V2("order_id", order.ID),
				logger.StringV2("mobile", order.Mobile))

			// 根据订单信息确定运营商类型
			ispType := s.getISPTypeFromOrder(order)

			// 查询充值前余额
			preBalance, err := s.phoneQueryService.QueryBalanceWithRetry(ctx, order.Mobile, ispType, 3)
			if err != nil {
				logger.WithContextCategory(ctx, "recharge").Error("【充值前余额查询失败】",
					logger.Int64V2("order_id", order.ID),
					logger.StringV2("mobile", order.Mobile),
					logger.ErrorV2(err))
				// 余额查询失败不阻断充值流程，只记录日志
			} else {
				logger.WithContextCategory(ctx, "recharge").Info("【充值前余额查询成功】",
					logger.Int64V2("order_id", order.ID),
					logger.StringV2("mobile", order.Mobile),
					logger.StringV2("balance", preBalance.Data))

				// 保存余额查询记录到独立表
				balanceRecord := &model.BalanceQueryRecord{
					OrderID:     order.ID,
					OrderNumber: order.OrderNumber,
					Mobile:      order.Mobile,
					ISPType:     ispType,
					QueryType:   "before", // 充值前查询
					Balance:     preBalance.Data,
					QueryTime:   time.Now(),
					Success:     true,
				}

				// 保存到余额查询记录表
				if err := s.balanceQueryRecordRepo.Create(ctx, balanceRecord); err != nil {
					logger.WithContextCategory(ctx, "recharge").Error("【保存余额查询记录失败】",
						logger.Int64V2("order_id", order.ID),
						logger.ErrorV2(err))
				} else {
					logger.WithContextCategory(ctx, "recharge").Info("【余额查询记录保存成功】",
						logger.Int64V2("order_id", order.ID),
						logger.StringV2("balance", preBalance.Data))
				}
			}
		}
	}

	api, apiParam, err := s.getPlatformAPIByOrder(ctx, order)
	if err != nil {
		logger.WithContextCategory(ctx, "recharge").Error("【获取API信息失败】",
			logger.ErrorV2(err),
			logger.Int64V2("order_id", order.ID))
		return fmt.Errorf("get platform API failed: %v", err)
	}
	logger.WithContextCategory(ctx, "recharge").Info("【获取API信息成功】",
		logger.Int64V2("order_id", order.ID),
		logger.Int64V2("api_id", api.ID),
		logger.StringV2("api_name", api.Name))

	// 原子性锁定订单，防止并发重复处理
	// 只有当前 api_id 对应的订单状态为 'pending_recharge' 时，才允许锁定
	locked, err := s.orderRepo.UpdateStatusCAS(ctx, order.ID, model.OrderStatusPendingRecharge, model.OrderStatusProcessing, api.ID)
	if err != nil {
		logger.WithContextCategory(ctx, "recharge").Error("【订单状态原子更新失败】",
			logger.ErrorV2(err),
			logger.Int64V2("order_id", order.ID))
		return err
	}
	if !locked {
		logger.WithContextCategory(ctx, "recharge").Info("【订单已被其他worker处理，跳过】", logger.Int64V2("order_id", order.ID))
		return nil
	}

	// 获取订单信息
	order, err = s.orderRepo.GetByID(ctx, order.ID)
	if err != nil {
		logger.WithContextCategory(ctx, "recharge").Error("【获取订单信息失败】",
			logger.ErrorV2(err),
			logger.Int64V2("order_id", order.ID))
		return fmt.Errorf("get order failed: %v", err)
	}
	logger.WithContextCategory(ctx, "recharge").Info("【获取订单信息成功】",
		logger.Int64V2("order_id", order.ID),
		logger.StringV2("status", fmt.Sprintf("%v", order.Status)))

	// 检查订单状态
	if order.Status == model.OrderStatusRecharging || order.Status == model.OrderStatusSuccess {
		return nil
	}

	// 检查是否需要扣款（外部订单在创建时已扣款）
	if order.Client == 2 { // 外部API订单
		logger.WithContextCategory(ctx, "recharge").Info("【外部订单已预扣款，跳过扣款步骤】",
			logger.Int64V2("order_id", order.ID),
			logger.IntV2("client", order.Client))
	} else {
		// 平台订单：从平台账号扣款（支持授信额度）
		logger.WithContextCategory(ctx, "recharge").Info("【平台订单开始扣款】",
			logger.Int64V2("order_id", order.ID),
			logger.Int64V2("platform_account_id", order.PlatformAccountID),
			logger.Float64V2("amount", order.Price))

		// 使用平台账号余额服务进行扣款（支持授信额度）
		balanceService := s.GetBalanceService()
		if err := balanceService.DeductBalance(ctx, order.PlatformAccountID, order.Price, order.ID, "订单充值扣除"); err != nil {
			logger.WithContextCategory(ctx, "recharge").Error("【扣除平台账号余额失败】",
				logger.ErrorV2(err),
				logger.Int64V2("platform_account_id", order.PlatformAccountID),
				logger.Float64V2("amount", order.Price))

			// 使用事务处理订单状态更新
			txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
				// 将订单状态设置为失败，并写入备注
				if err := tx.Model(&model.Order{}).Where("id = ?", order.ID).Update("status", model.OrderStatusFailed).Error; err != nil {
					return err
				}
				if err := tx.Model(&model.Order{}).Where("id = ?", order.ID).Update("remark", "平台账号余额和授信额度均不足，订单失败").Error; err != nil {
					return err
				}
				return nil
			})

			if txErr != nil {
				logger.WithContextCategory(ctx, "recharge").Error("更新订单状态失败", logger.ErrorV2(txErr), logger.Int64V2("order_id", order.ID))
			} else {
				// 事务提交成功后，发送状态变更通知（幂等）
				if nErr := s.sendOrderStatusNotificationWithIdempotency(ctx, order, model.OrderStatusFailed); nErr != nil {
					logger.WithContextCategory(ctx, "recharge").Error("发送扣款失败通知失败", logger.Int64V2("order_id", order.ID), logger.ErrorV2(nErr))
				} else {
					logger.WithContextCategory(ctx, "recharge").Info("推送扣款失败通知到队列成功", logger.Int64V2("order_id", order.ID))
				}
			}

			return fmt.Errorf("deduct platform account balance failed: %v", err)
		}
		logger.WithContextCategory(ctx, "recharge").Info("【扣除平台账号余额成功】",
			logger.Int64V2("platform_account_id", order.PlatformAccountID),
			logger.Float64V2("amount", order.Price))
	}

	// 提交订单到平台
	if err := s.SubmitOrder(ctx, order, api, apiParam); err != nil {
		logger.WithContextCategory(ctx, "recharge").Error("【提交订单到平台失败1】",
			logger.ErrorV2(err),
			logger.Int64V2("order_id", order.ID),
			logger.StringV2("order_number", order.OrderNumber),
			logger.Int64V2("product_id", order.ProductID),
			logger.Int64V2("api_id", api.ID),
			logger.Int64V2("param_id", apiParam.ID),
			logger.Int64V2("platform_id", api.PlatformID),
			logger.StringV2("api_code", api.Code),
			logger.Int64V2("api_account_id", api.AccountID))

		// 获取所有可用的API关系
		relations, err2 := s.productRepo.GetAPIRelationsByProductID(ctx, order.ProductID)
		if err2 != nil {
			logger.WithContextCategory(ctx, "recharge").Error("【获取API关系失败】",
				logger.ErrorV2(err2),
				logger.Int64V2("order_id", order.ID))
			return fmt.Errorf("get API relations failed: %v", err2)
		}

		// 解析已使用的API列表
		var usedAPIs []map[string]interface{}
		if order.UsedAPIs != "" {
			if err := json.Unmarshal([]byte(order.UsedAPIs), &usedAPIs); err != nil {
				logger.WithContextCategory(ctx, "recharge").Error("【解析已使用API列表失败】",
					logger.ErrorV2(err),
					logger.Int64V2("order_id", order.ID))
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
			logger.WithContextCategory(ctx, "recharge").Error("【没有可用的API】", logger.Int64V2("order_id", order.ID))
			// 调用订单失败处理方法，会自动退还余额和创建通知
			if err := s.orderService.ProcessOrderFail(ctx, order.ID, "无可用API"); err != nil {
				logger.WithContextCategory(ctx, "recharge").Error("处理订单失败时出错", logger.ErrorV2(err), logger.Int64V2("order_id", order.ID))
			}
			return fmt.Errorf("no available API")
		}

		logger.WithContextCategory(ctx, "recharge").Info("【准备更新订单状态与切换API】",
			logger.Int64V2("order_id", order.ID),
			logger.Int64V2("from_api_id", api.ID),
			logger.Int64V2("to_api_id", nextAPIID),
			logger.IntV2("used_apis_count", len(usedAPIs)))
		if err2 := s.orderRepo.UpdateStatusAndAPIID(ctx, order.ID, model.OrderStatusPendingRecharge, nextAPIID, string(usedAPIsJSON)); err2 != nil {
			logger.WithContextCategory(ctx, "recharge").Error("【更新订单状态和API ID失败】", logger.ErrorV2(err2), logger.Int64V2("order_id", order.ID), logger.Int64V2("to_api_id", nextAPIID))
			return fmt.Errorf("update order status and API ID failed: %v", err2)
		}
		logger.WithContextCategory(ctx, "recharge").Info("【更新订单状态与API成功】", logger.Int64V2("order_id", order.ID), logger.Int64V2("new_api_id", nextAPIID))

		logger.WithContextCategory(ctx, "recharge").Info("【准备创建重试记录】", logger.Int64V2("order_id", order.ID))
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
		logger.WithContextCategory(ctx, "recharge").Info("重试参数预览", logger.Int64V2("order_id", order.ID), logger.Int64V2("api_id", nextAPIID), logger.Int64V2("param_id", nextParamID))
		retryParamsJSON, _ := json.Marshal(retryParams)

		logger.WithContextCategory(ctx, "recharge").Info("【重试参数已构建】", logger.Int64V2("order_id", order.ID), logger.AnyV2("params_keys", func() []string {
			keys := make([]string, 0, len(retryParams))
			for k := range retryParams {
				keys = append(keys, k)
			}
			return keys
		}()))
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
		logger.WithContextCategory(ctx, "recharge").Info("重试记录构建完成", logger.Int64V2("order_id", retryRecord.OrderID), logger.IntV2("retry_count", retryRecord.RetryCount), logger.Int64V2("api_id", retryRecord.APIID), logger.Int64V2("param_id", retryRecord.ParamID))

		if s.retryRepo == nil {
			logger.WithContextCategory(ctx, "recharge").Error("【严重错误】retryRecordRepo 为空！",
				logger.Int64V2("order_id", order.ID))
			return fmt.Errorf("retry repository is nil")
		}

		logger.WithContextCategory(ctx, "recharge").Info("【准备调用Create方法】",
			logger.Int64V2("order_id", order.ID))
		if err := s.retryRepo.Create(ctx, retryRecord); err != nil {
			logger.WithContextCategory(ctx, "recharge").Error("【创建重试记录失败】",
				logger.ErrorV2(err),
				logger.Int64V2("order_id", order.ID))
			return fmt.Errorf("create retry record failed: %v", err)
		}
		logger.WithContextCategory(ctx, "recharge").Info("【创建重试记录成功】",
			logger.Int64V2("order_id", order.ID),
			logger.Int64V2("retry_id", retryRecord.ID))

		// 从处理队列中移除
		if err := s.RemoveFromProcessingQueue(ctx, order.ID); err != nil {
			logger.WithContextCategory(ctx, "recharge").Error("【从处理队列移除失败】",
				logger.ErrorV2(err),
				logger.Int64V2("order_id", order.ID))
		}

		logger.WithContextCategory(ctx, "recharge").Info("【充值任务处理完成】",
			logger.Int64V2("order_id", order.ID),
			logger.StringV2("order_number", order.OrderNumber))
		return fmt.Errorf("submit order failed: %v", err)
	}

	// 更新订单状态为充值中
	if err := s.orderRepo.UpdateStatus(ctx, order.ID, model.OrderStatusRecharging); err != nil {
		logger.WithContextCategory(ctx, "recharge").Error("【更新订单状态失败】",
			logger.ErrorV2(err),
			logger.Int64V2("order_id", order.ID))
		return fmt.Errorf("update order status failed: %v", err)
	}
	logger.WithContextCategory(ctx, "recharge").Info("【订单状态更新成功】",
		logger.Int64V2("order_id", order.ID),
		logger.StringV2("status", fmt.Sprintf("%v", model.OrderStatusRecharging)))

	// 更新订单成本价
	err = s.orderRepo.DB().Model(&model.Order{}).
		Where("id = ?", order.ID).
		Update("const_price", apiParam.Price).Error
	if err != nil {
		logger.WithContextCategory(ctx, "recharge").Error("【更新订单成本价失败】", logger.Int64V2("order_id", order.ID), logger.ErrorV2(err))
	} else {
		logger.WithContextCategory(ctx, "recharge").Info("【更新订单成本价成功】", logger.Int64V2("order_id", order.ID), logger.Float64V2("const_price", apiParam.Price))
	}

	// 从处理队列中移除
	if err := s.RemoveFromProcessingQueue(ctx, order.ID); err != nil {
		logger.WithContextCategory(ctx, "recharge").Error("【从处理队列移除失败】",
			logger.ErrorV2(err),
			logger.Int64V2("order_id", order.ID))
	}

	logger.WithContextCategory(ctx, "recharge").Info("【充值任务处理完成】",
		logger.Int64V2("order_id", order.ID),
		logger.StringV2("order_number", order.OrderNumber))
	return nil
}

// CreateRechargeTask 创建充值任务
func (s *rechargeService) CreateRechargeTask(ctx context.Context, orderID int64) error {
	logger.WithContextCategory(ctx, "recharge").Info("【开始创建充值任务】",
		logger.Int64V2("order_id", orderID))

	// 获取订单信息
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		logger.WithContextCategory(ctx, "recharge").Error("【获取订单信息失败】",
			logger.ErrorV2(err),
			logger.Int64V2("order_id", orderID))
		return fmt.Errorf("get order failed: %v", err)
	}
	logger.WithContextCategory(ctx, "recharge").Info("【获取订单信息成功】",
		logger.Int64V2("order_id", orderID),
		logger.StringV2("order_number", order.OrderNumber),
		logger.IntV2("status", int(order.Status)))

	// 检查订单状态是否为待充值
	if order.Status != model.OrderStatusPendingRecharge {
		logger.WithContextCategory(ctx, "recharge").Warn("【订单状态不是待充值，跳过创建充值任务】",
			logger.Int64V2("order_id", orderID),
			logger.IntV2("current_status", int(order.Status)),
			logger.IntV2("expected_status", int(model.OrderStatusPendingRecharge)))
		return fmt.Errorf("order status is not pending recharge, current status: %d", order.Status)
	}

	logger.WithContextCategory(ctx, "recharge").Info("【订单状态验证通过，状态为待充值】",
		logger.Int64V2("order_id", orderID),
		logger.IntV2("status", int(order.Status)))

	// 将订单推送到充值队列
	if err := s.PushToRechargeQueue(ctx, orderID); err != nil {
		logger.WithContextCategory(ctx, "recharge").Error("【推送订单到充值队列失败】",
			logger.ErrorV2(err),
			logger.Int64V2("order_id", orderID))
		return fmt.Errorf("push to recharge queue failed: %v", err)
	}
	logger.WithContextCategory(ctx, "recharge").Info("【推送订单到充值队列成功】",
		logger.Int64V2("order_id", orderID))

	logger.WithContextCategory(ctx, "recharge").Info("【充值任务创建完成】",
		logger.Int64V2("order_id", orderID),
		logger.StringV2("order_number", order.OrderNumber))
	return nil
}

// GetPlatformAPIByOrderID 根据订单ID获取平台API信息
// getPlatformAPIByOrder 直接使用订单对象获取平台API信息
func (s *rechargeService) getPlatformAPIByOrder(ctx context.Context, order *model.Order) (*model.PlatformAPI, *model.PlatformAPIParam, error) {

	logger.WithContextCategory(ctx, "recharge").Info("【获取平台API信息】",
		logger.StringV2("stage", "route"),
		logger.Int64V2("order_id", order.ID),
		logger.Int64V2("product_id", order.ProductID))

	//product_api_relations
	r, err := s.productAPIRelationRepo.GetByProductID(ctx, order.ProductID)
	if err != nil {
		// 将订单设置为失败状态
		if err := s.orderRepo.UpdateStatus(ctx, order.ID, model.OrderStatusFailed); err != nil {
			logger.WithContextCategory(ctx, "recharge").Error("【更新订单状态失败】",
				logger.ErrorV2(err),
				logger.Int64V2("order_id", order.ID))
		}
		// 更新订单备注
		if err := s.orderRepo.UpdateRemark(ctx, order.ID, "商品未绑定接口"); err != nil {
			logger.WithContextCategory(ctx, "recharge").Error("【更新订单备注失败】",
				logger.ErrorV2(err),
				logger.Int64V2("order_id", order.ID))
		}
		return nil, nil, fmt.Errorf("商品未绑定接口: %v", err)
	}

	//获取api套餐 platform_api_params
	apiParam, err := s.platformAPIParamRepo.GetByID(ctx, r.ParamID)
	if err != nil {
		if errors.Is(err, repository.ErrNoAPIForProduct) {
			// 将订单设置为失败状态
			if err := s.orderRepo.UpdateStatus(ctx, order.ID, model.OrderStatusFailed); err != nil {
				logger.WithContextCategory(ctx, "recharge").Error("【更新订单状态失败】",
					logger.ErrorV2(err),
					logger.Int64V2("order_id", order.ID))
			}
			// 更新订单备注
			if err := s.orderRepo.UpdateRemark(ctx, order.ID, "商品未绑定接口"); err != nil {
				logger.WithContextCategory(ctx, "recharge").Error("【更新订单备注失败】",
					logger.ErrorV2(err),
					logger.Int64V2("order_id", order.ID))
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
				logger.WithContextCategory(ctx, "recharge").Error("【更新订单状态失败】",
					logger.ErrorV2(err),
					logger.Int64V2("order_id", order.ID))
			}
			// 更新订单备注
			if err := s.orderRepo.UpdateRemark(ctx, order.ID, "商品未绑定接口"); err != nil {
				logger.WithContextCategory(ctx, "recharge").Error("【更新订单备注失败】",
					logger.ErrorV2(err),
					logger.Int64V2("order_id", order.ID))
			}
			return nil, nil, fmt.Errorf("商品未绑定接口")
		}
		return nil, nil, fmt.Errorf("获取平台API信息失败: %v", err)
	}

	// 检查平台API状态
	if api.Status != 1 {
		logger.WithContextCategory(ctx, "recharge").Error("【平台API已禁用】",
			logger.Int64V2("api_id", api.ID),
			logger.StringV2("api_code", api.Code),
			logger.IntV2("status", api.Status),
			logger.Int64V2("order_id", order.ID))
		// 将订单设置为失败状态
		if err := s.orderRepo.UpdateStatus(ctx, order.ID, model.OrderStatusFailed); err != nil {
			logger.WithContextCategory(ctx, "recharge").Error("【更新订单状态失败】",
				logger.ErrorV2(err),
				logger.Int64V2("order_id", order.ID))
		}
		// 更新订单备注
		if err := s.orderRepo.UpdateRemark(ctx, order.ID, "平台API已禁用"); err != nil {
			logger.WithContextCategory(ctx, "recharge").Error("【更新订单备注失败】",
				logger.ErrorV2(err),
				logger.Int64V2("order_id", order.ID))
		}
		return nil, nil, fmt.Errorf("平台API已禁用: api_id=%d, status=%d", api.ID, api.Status)
	}

	logger.WithContextCategory(ctx, "recharge").Info("【通道路由成功】",
		logger.StringV2("stage", "route"),
		logger.Int64V2("order_id", order.ID),
		logger.Int64V2("api_id", api.ID),
		logger.Int64V2("param_id", apiParam.ID),
		logger.StringV2("platform_code", api.Code),
	)
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
	logger.WithContextCategory(ctx, "recharge").Info("【准备推送订单到充值队列】",
		logger.StringV2("stage", "enqueue"),
		logger.Int64V2("order_id", orderID))

	if s.redisClient == nil {
		logger.WithContextCategory(ctx, "recharge").Error("【Redis客户端为空】",
			logger.Int64V2("order_id", orderID))
		return fmt.Errorf("redis client is nil")
	}

	// 检查订单是否已经在队列中，避免重复推送
	orderIDStr := strconv.FormatInt(orderID, 10)
	exists, err := s.redisClient.LPos(ctx, "recharge_queue", orderIDStr, redisV8.LPosArgs{}).Result()
	if err == nil {
		logger.WithContextCategory(ctx, "recharge").Info("【订单已在充值队列中，跳过推送】",
			logger.Int64V2("order_id", orderID),
			logger.Int64V2("position", exists))
		return nil
	}
	// 如果是redis.Nil错误，说明订单不在队列中，可以继续推送
	if err != redisV8.Nil {
		logger.WithContextCategory(ctx, "recharge").Error("【检查订单是否在队列中失败】",
			logger.ErrorV2(err),
			logger.Int64V2("order_id", orderID))
		// 即使检查失败，也继续推送，避免丢失订单
	}

	err = s.redisClient.LPush(ctx, "recharge_queue", orderID).Err()
	if err != nil {
		logger.WithContextCategory(ctx, "recharge").Error("【推送订单到充值队列失败】",
			logger.ErrorV2(err),
			logger.Int64V2("order_id", orderID))
		return err
	}

	logger.WithContextCategory(ctx, "recharge").Info("【推送订单到充值队列成功】",
		logger.StringV2("stage", "enqueue"),
		logger.Int64V2("order_id", orderID))
	return nil
}

// PopFromRechargeQueue 从充值队列获取订单
func (s *rechargeService) PopFromRechargeQueue(ctx context.Context) (int64, error) {
	logger.WithContextCategory(ctx, "recharge").Debug("【准备从充值队列获取订单】",
		logger.StringV2("stage", "dequeue"))

	if s.redisClient == nil {
		logger.WithContextCategory(ctx, "recharge").Error("【Redis客户端为空】")
		return 0, fmt.Errorf("redis client is nil")
	}

	// 使用 BRPOPLPUSH 命令，将任务从队列中移除并放入处理中队列
	result, err := s.redisClient.BRPopLPush(ctx, "recharge_queue", "recharge_processing", 0).Result()
	if err != nil {
		if err == redisV8.Nil {
			logger.WithContextCategory(ctx, "recharge").Debug("【充值队列为空】",
				logger.StringV2("stage", "dequeue"))
			return 0, nil
		}
		logger.WithContextCategory(ctx, "recharge").Error("【从充值队列获取订单失败】", logger.ErrorV2(err))
		return 0, err
	}

	orderID, err := strconv.ParseInt(result, 10, 64)
	if err != nil {
		logger.WithContextCategory(ctx, "recharge").Error("【解析订单ID失败】", logger.ErrorV2(err), logger.StringV2("result", result))
		return 0, fmt.Errorf("parse order id failed: %v", err)
	}

	logger.WithContextCategory(ctx, "recharge").Info("【从充值队列获取订单成功】",
		logger.StringV2("stage", "dequeue"),
		logger.Int64V2("order_id", orderID))
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
	logger.WithContextCategory(ctx, "recharge").Info("【开始检查充值中订单】开始执行定时检查任务")

	// 获取所有充值中的订单
	orders, err := s.orderRepo.GetByStatus(ctx, model.OrderStatusRecharging)
	if err != nil {
		logger.WithContextCategory(ctx, "recharge").Error("【获取充值中订单失败】", logger.ErrorV2(err))
		return fmt.Errorf("get recharging orders failed: %v", err)
	}

	logger.WithContextCategory(ctx, "recharge").Info("【获取充值中订单成功】", logger.IntV2("total", len(orders)))

	now := time.Now()
	checkedCount := 0
	for _, order := range orders {
		// 检查订单是否超过5分钟
		if order.UpdatedAt.Add(5 * time.Minute).Before(now) {
			logger.WithContextCategory(ctx, "recharge").Info("【发现超时订单】",
				logger.Int64V2("order_id", order.ID),
				logger.StringV2("order_number", order.OrderNumber),
				logger.StringV2("last_update", order.UpdatedAt.Format("2006-01-02 15:04:05")),
				logger.StringV2("overdue", now.Sub(order.UpdatedAt).String()))

			// 查询订单状态
			if err := s.manager.QueryOrderStatus(ctx, order); err != nil {
				logger.WithContextCategory(ctx, "recharge").Error("【查询订单状态失败】",
					logger.Int64V2("order_id", order.ID),
					logger.StringV2("order_number", order.OrderNumber),
					logger.ErrorV2(err))
				continue
			}

			logger.WithContextCategory(ctx, "recharge").Info("【订单状态查询完成】",
				logger.Int64V2("order_id", order.ID),
				logger.StringV2("order_number", order.OrderNumber))
			checkedCount++
		}
	}

	logger.WithContextCategory(ctx, "recharge").Info("【充值中订单检查完成】",
		logger.IntV2("total", len(orders)),
		logger.IntV2("checked_count", checkedCount))
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
			logger.WithContextCategory(ctx, "recharge").Error("【事务回滚】", logger.Int64V2("order_id", order.ID), logger.AnyV2("panic", r))
		}
	}()

	// 更新订单状态和成本价
	result := tx.Model(&model.Order{}).Where("id = ?", order.ID).Updates(map[string]interface{}{
		"status":      model.OrderStatusRecharging,
		"const_price": apiParam.Price,
	})
	if result.Error != nil {
		tx.Rollback()
		logger.WithContextCategory(ctx, "recharge").Error("【更新订单状态和成本价失败】", logger.Int64V2("order_id", order.ID), logger.ErrorV2(result.Error))
		return fmt.Errorf("update order status and cost price failed: %v", result.Error)
	}
	if result.RowsAffected == 0 {
		tx.Rollback()
		logger.WithContextCategory(ctx, "recharge").Error("【更新订单状态和成本价失败】", logger.Int64V2("order_id", order.ID), logger.StringV2("reason", "没有记录被更新"))
		return fmt.Errorf("no record updated")
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		logger.WithContextCategory(ctx, "recharge").Error("【提交事务失败】", logger.Int64V2("order_id", order.ID), logger.ErrorV2(err))
		return fmt.Errorf("commit transaction failed: %v", err)
	}

	logger.WithContextCategory(ctx, "recharge").Info("【订单状态和成本价更新成功】", logger.Int64V2("order_id", order.ID), logger.IntV2("status", int(model.OrderStatusRecharging)), logger.Float64V2("const_price", apiParam.Price))
	return nil
}

// ProcessRetryTask 处理重试任务
func (s *rechargeService) ProcessRetryTask(ctx context.Context, retryRecord *model.OrderRetryRecord) error {
	logger.WithContextCategory(ctx, "recharge").Info("【开始处理重试任务】", logger.Int64V2("retry_id", retryRecord.ID), logger.Int64V2("order_id", retryRecord.OrderID))

	// 1. 获取订单信息
	order, err := s.orderRepo.GetByID(ctx, retryRecord.OrderID)
	if err != nil {
		logger.WithContextCategory(ctx, "recharge").Error("【获取订单信息失败】", logger.Int64V2("retry_id", retryRecord.ID), logger.Int64V2("order_id", retryRecord.OrderID), logger.ErrorV2(err))
		return fmt.Errorf("get order failed: %v", err)
	}
	logger.WithContextCategory(ctx, "recharge").Info("【获取订单信息成功】", logger.Int64V2("retry_id", retryRecord.ID), logger.Int64V2("order_id", retryRecord.OrderID), logger.IntV2("status", int(order.Status)), logger.StringV2("order_number", order.OrderNumber))

	// 2. 获取平台API信息
	api, err := s.platformRepo.GetAPIByID(ctx, retryRecord.APIID)
	if err != nil {
		logger.WithContextCategory(ctx, "recharge").Error("【获取平台API信息失败】", logger.Int64V2("retry_id", retryRecord.ID), logger.Int64V2("order_id", retryRecord.OrderID), logger.ErrorV2(err))
		return fmt.Errorf("get platform api failed: %v", err)
	}
	logger.WithContextCategory(ctx, "recharge").Info("【获取平台API信息成功】", logger.Int64V2("retry_id", retryRecord.ID), logger.Int64V2("order_id", retryRecord.OrderID), logger.Int64V2("api_id", api.ID), logger.StringV2("api_name", api.Name))

	// 3. 获取API参数信息
	apiParam, err := s.platformAPIParamRepo.GetByID(ctx, retryRecord.ParamID)
	if err != nil {
		logger.WithContextCategory(ctx, "recharge").Error("【获取API参数信息失败】", logger.Int64V2("retry_id", retryRecord.ID), logger.Int64V2("order_id", retryRecord.OrderID), logger.ErrorV2(err))
		return err
	}
	logger.WithContextCategory(ctx, "recharge").Info("【获取API参数信息成功】", logger.Int64V2("retry_id", retryRecord.ID), logger.Int64V2("order_id", retryRecord.OrderID), logger.Int64V2("param_id", apiParam.ID))

	// 4. 提交订单到平台
	logger.WithContextCategory(ctx, "recharge").Info("【开始提交订单到平台】", logger.Int64V2("retry_id", retryRecord.ID), logger.Int64V2("order_id", retryRecord.OrderID), logger.StringV2("order_number", order.OrderNumber))
	if err := s.manager.SubmitOrder(ctx, order, api, apiParam); err != nil {
		logger.WithContextCategory(ctx, "recharge").Error("【提交订单到平台失败】", logger.Int64V2("retry_id", retryRecord.ID), logger.Int64V2("order_id", retryRecord.OrderID), logger.ErrorV2(err))
		return fmt.Errorf("submit order failed: %v", err)
	}
	logger.WithContextCategory(ctx, "recharge").Info("【提交订单到平台成功】", logger.Int64V2("retry_id", retryRecord.ID), logger.Int64V2("order_id", retryRecord.OrderID), logger.StringV2("order_number", order.OrderNumber))

	// 5. 开启事务
	logger.WithContextCategory(ctx, "recharge").Info("【开始更新订单状态和平台信息】", logger.Int64V2("retry_id", retryRecord.ID), logger.Int64V2("order_id", retryRecord.OrderID), logger.StringV2("order_number", order.OrderNumber))
	tx := s.orderRepo.(*repository.OrderRepositoryImpl).DB().Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			logger.WithContextCategory(ctx, "recharge").Error("【事务回滚】", logger.Int64V2("retry_id", retryRecord.ID), logger.Int64V2("order_id", retryRecord.OrderID), logger.AnyV2("panic", r))
		}
	}()

	// 6. 更新订单状态
	logger.WithContextCategory(ctx, "recharge").Info("【开始更新订单状态】", logger.Int64V2("retry_id", retryRecord.ID), logger.Int64V2("order_id", retryRecord.OrderID), logger.StringV2("order_number", order.OrderNumber), logger.IntV2("old_status", int(order.Status)), logger.IntV2("new_status", int(model.OrderStatusRecharging)))
	result := tx.Model(&model.Order{}).Where("id = ?", retryRecord.OrderID).Updates(map[string]interface{}{
		"status":      model.OrderStatusRecharging,
		"const_price": apiParam.Price,
	})
	if result.Error != nil {
		tx.Rollback()
		logger.WithContextCategory(ctx, "recharge").Error("【更新订单状态失败】", logger.Int64V2("retry_id", retryRecord.ID), logger.Int64V2("order_id", retryRecord.OrderID), logger.ErrorV2(result.Error))
		return fmt.Errorf("update order status failed: %v", result.Error)
	}
	if result.RowsAffected == 0 {
		tx.Rollback()
		logger.WithContextCategory(ctx, "recharge").Error("【更新订单状态失败】", logger.Int64V2("retry_id", retryRecord.ID), logger.Int64V2("order_id", retryRecord.OrderID), logger.StringV2("reason", "没有记录被更新"))
		return fmt.Errorf("no record updated")
	}
	logger.WithContextCategory(ctx, "recharge").Info("【更新订单状态和成本价成功】", logger.Int64V2("retry_id", retryRecord.ID), logger.Int64V2("order_id", retryRecord.OrderID), logger.Int64V2("rows_affected", result.RowsAffected), logger.Float64V2("const_price", apiParam.Price))

	// 7. 更新平台信息
	logger.WithContextCategory(ctx, "recharge").Info("【开始更新平台信息】", logger.Int64V2("retry_id", retryRecord.ID), logger.Int64V2("order_id", retryRecord.OrderID), logger.Int64V2("platform_id", api.ID), logger.Int64V2("api_id", api.ID), logger.Int64V2("param_id", apiParam.ID))
	result = tx.Model(&model.Order{}).Where("id = ?", retryRecord.OrderID).Updates(map[string]interface{}{
		"platform_id":      api.ID,
		"api_cur_id":       api.ID,
		"api_cur_param_id": apiParam.ID,
		"platform_name":    api.Name,
		"platform_code":    api.Code,
	})
	if result.Error != nil {
		tx.Rollback()
		logger.WithContextCategory(ctx, "recharge").Error("【更新平台信息失败】", logger.Int64V2("retry_id", retryRecord.ID), logger.Int64V2("order_id", retryRecord.OrderID), logger.ErrorV2(result.Error))
		return fmt.Errorf("update platform info failed: %v", result.Error)
	}
	if result.RowsAffected == 0 {
		tx.Rollback()
		logger.WithContextCategory(ctx, "recharge").Error("【更新平台信息失败】", logger.Int64V2("retry_id", retryRecord.ID), logger.Int64V2("order_id", retryRecord.OrderID), logger.StringV2("reason", "没有记录被更新"))
		return fmt.Errorf("no record updated")
	}
	logger.WithContextCategory(ctx, "recharge").Info("【更新平台信息成功】", logger.Int64V2("retry_id", retryRecord.ID), logger.Int64V2("order_id", retryRecord.OrderID), logger.Int64V2("rows_affected", result.RowsAffected))

	// 8. 更新重试记录状态
	logger.WithContextCategory(ctx, "recharge").Info("【开始更新重试记录状态】", logger.Int64V2("retry_id", retryRecord.ID), logger.Int64V2("order_id", retryRecord.OrderID))
	if err := tx.Model(&model.OrderRetryRecord{}).Where("id = ?", retryRecord.ID).Updates(map[string]interface{}{
		"status":      1, // 1: 处理成功
		"retry_count": retryRecord.RetryCount + 1,
	}).Error; err != nil {
		tx.Rollback()
		logger.WithContextCategory(ctx, "recharge").Error("【更新重试记录状态失败】", logger.Int64V2("retry_id", retryRecord.ID), logger.Int64V2("order_id", retryRecord.OrderID), logger.ErrorV2(err))
		return fmt.Errorf("update retry record failed: %v", err)
	}
	logger.WithContextCategory(ctx, "recharge").Info("【更新重试记录状态成功】", logger.Int64V2("retry_id", retryRecord.ID), logger.Int64V2("order_id", retryRecord.OrderID))

	// 9. 提交事务
	if err := tx.Commit().Error; err != nil {
		logger.WithContextCategory(ctx, "recharge").Error("【提交事务失败】", logger.Int64V2("retry_id", retryRecord.ID), logger.Int64V2("order_id", retryRecord.OrderID), logger.ErrorV2(err))
		return fmt.Errorf("commit transaction failed: %v", err)
	}
	logger.WithContextCategory(ctx, "recharge").Info("【提交事务成功】", logger.Int64V2("retry_id", retryRecord.ID), logger.Int64V2("order_id", retryRecord.OrderID))

	// 9.1 更新订单成本价
	logger.WithContextCategory(ctx, "recharge").Info("【开始更新订单成本价】", logger.Int64V2("retry_id", retryRecord.ID), logger.Int64V2("order_id", retryRecord.OrderID), logger.Float64V2("const_price", apiParam.Price))
	err = s.orderRepo.DB().Model(&model.Order{}).
		Where("id = ?", retryRecord.OrderID).
		Update("const_price", apiParam.Price).Error
	if err != nil {
		logger.WithContextCategory(ctx, "recharge").Error("【更新订单成本价失败】", logger.Int64V2("retry_id", retryRecord.ID), logger.Int64V2("order_id", retryRecord.OrderID), logger.ErrorV2(err))
	} else {
		logger.WithContextCategory(ctx, "recharge").Info("【更新订单成本价成功】", logger.Int64V2("retry_id", retryRecord.ID), logger.Int64V2("order_id", retryRecord.OrderID), logger.Float64V2("const_price", apiParam.Price))
	}

	// 10. 验证更新结果
	updatedOrder, err := s.orderRepo.GetByID(ctx, retryRecord.OrderID)
	if err != nil {
		logger.WithContextCategory(ctx, "recharge").Error("【验证更新结果失败】", logger.Int64V2("retry_id", retryRecord.ID), logger.Int64V2("order_id", retryRecord.OrderID), logger.ErrorV2(err))
	} else {
		logger.WithContextCategory(ctx, "recharge").Info("【验证更新结果】", logger.Int64V2("retry_id", retryRecord.ID), logger.Int64V2("order_id", retryRecord.OrderID), logger.StringV2("order_number", updatedOrder.OrderNumber), logger.IntV2("status", int(updatedOrder.Status)), logger.Int64V2("platform_id", updatedOrder.PlatformId))
	}

	logger.WithContextCategory(ctx, "recharge").Info("【重试任务处理完成】", logger.Int64V2("retry_id", retryRecord.ID), logger.Int64V2("order_id", retryRecord.OrderID), logger.StringV2("order_number", order.OrderNumber))
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
