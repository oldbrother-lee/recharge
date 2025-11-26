package service

import (
	"context"
	"fmt"
	"recharge-go/configs"
	"recharge-go/internal/model"
	"recharge-go/internal/repository"
    "recharge-go/internal/service/platform"
    "recharge-go/internal/utils"
    logger "recharge-go/pkg/log"
	"strings"
	"sync"
	"time"
)

type TaskService struct {
	taskConfigRepo      *repository.TaskConfigRepository
	taskOrderRepo       *repository.TaskOrderRepository
	orderRepo           repository.OrderRepository
	daichongOrderRepo   *repository.DaichongOrderRepository
	platformSvc         *platform.Service
	orderService        OrderService
	config              *configs.TaskConfig
	ctx                 context.Context
	cancel              context.CancelFunc
	wg                  sync.WaitGroup
	mu                  sync.Mutex
	isRunning           bool
	platformAccountRepo *repository.PlatformAccountRepository
	// 任务上下文管理
	taskContexts map[int64]*TaskContext // 任务ID -> 任务上下文
	taskMutex    sync.RWMutex           // 保护taskContexts的读写锁
	// 配置监听器
	configListener *TaskConfigListener
	// 订单数量监控相关字段
	isPullingSuspended bool            // 是否暂停拉单
	suspendMutex       sync.RWMutex    // 保护暂停状态的读写锁
	orderThresholds    OrderThresholds // 订单数量阈值配置
}

// TaskContext 任务上下文信息
type TaskContext struct {
	Ctx    context.Context
	Cancel context.CancelFunc
}

// OrderThresholds 订单数量阈值配置
type OrderThresholds struct {
	SuspendThreshold int // 暂停拉单阈值（充值中+待充值订单数量）
	ResumeThreshold  int // 恢复拉单阈值（处理中订单数量）
}

func NewTaskService(
	taskConfigRepo *repository.TaskConfigRepository,
	taskOrderRepo *repository.TaskOrderRepository,
	orderRepo repository.OrderRepository,
	daichongOrderRepo *repository.DaichongOrderRepository,
	platformSvc *platform.Service,
	orderService OrderService,
	config *configs.TaskConfig,
	platformAccountRepo *repository.PlatformAccountRepository,
) *TaskService {
	// 这里初始化一个可取消的背景上下文，保证即使在Start之前调用Stop也不会panic
	ctx, cancel := context.WithCancel(context.Background())
	return &TaskService{
		taskConfigRepo:      taskConfigRepo,
		taskOrderRepo:       taskOrderRepo,
		orderRepo:           orderRepo,
		daichongOrderRepo:   daichongOrderRepo,
		platformSvc:         platformSvc,
		orderService:        orderService,
		config:              config,
		ctx:                 ctx,
		cancel:              cancel,
		platformAccountRepo: platformAccountRepo,
		taskContexts:        make(map[int64]*TaskContext),
		isPullingSuspended:  false,
		orderThresholds: OrderThresholds{
			SuspendThreshold: config.SuspendThreshold,
			ResumeThreshold:  config.ResumeThreshold,
		},
	}
}

// SetConfigListener 设置配置监听器（仅在Task进程中调用）
func (s *TaskService) SetConfigListener(listener *TaskConfigListener) {
	s.configListener = listener
}

// StartConfigListener 启动配置监听器（仅在Task进程中调用）
func (s *TaskService) StartConfigListener() {
	if s.configListener != nil {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			if err := s.configListener.Start(); err != nil {
				logger.ErrorLogV2("配置监听器启动失败", logger.ErrorV2(err))
			}
		}()
		logger.InfoV2("任务配置监听器已启动")
	}
}

// StartTask 启动自动取单任务
func (s *TaskService) StartTask(ctx context.Context) {
	s.mu.Lock()
	if s.isRunning {
		s.mu.Unlock()
		return
	}
	s.isRunning = true
	s.mu.Unlock()

	// 从传入的父上下文派生可取消上下文，确保退出时能够正确传播
	s.ctx, s.cancel = context.WithCancel(ctx)

	// 启动主要的取单任务处理
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(time.Duration(s.config.Interval) * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-s.ctx.Done():
				return
			case <-ticker.C:
				s.processTask()
			}
		}
	}()

	// 启动配置检查定时器（每30秒检查一次配置变更）
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		configTicker := time.NewTicker(30 * time.Second)
		defer configTicker.Stop()

		logger.InfoV2("启动配置检查定时器，检查间隔: 30秒")

		for {
			select {
			case <-s.ctx.Done():
				return
			case <-configTicker.C:
				logger.DebugV2("开始定时检查配置变更")
				err := s.ReloadTaskConfig()
				if err != nil {
					logger.ErrorLogV2("定时配置检查失败", logger.ErrorV2(err))
				}
			}
		}
	}()
}

// StartOrderDetailsTask 启动订单详情查询任务
func (s *TaskService) StartOrderDetailsTask(ctx context.Context) {
	logger.InfoV2("启动订单详情查询任务",
		logger.Int64V2("interval_seconds", int64(s.config.OrderDetailsInterval)))
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(time.Duration(s.config.OrderDetailsInterval) * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-s.ctx.Done():
				return
			case <-ticker.C:
				s.processOrderDetails()
			}
		}
	}()
}

// StopTask 停止自动取单任务
func (s *TaskService) StopTask() {
	s.mu.Lock()
	if !s.isRunning {
		s.mu.Unlock()
		return
	}
	s.isRunning = false
	s.mu.Unlock()

	// 停止配置监听器
	if s.configListener != nil {
		s.configListener.Stop()
	}

	s.cancel()
	s.wg.Wait()
}

// StopTaskByID 主动停止特定任务
func (s *TaskService) StopTaskByID(taskID int64) {
	s.taskMutex.RLock()
	taskCtx, exists := s.taskContexts[taskID]
	s.taskMutex.RUnlock()

	if exists {
		logger.InfoV2(fmt.Sprintf("主动停止任务: TaskID=%d", taskID))
		taskCtx.Cancel() // 触发context取消，会导致processTaskConfig中的defer清理逻辑执行
	} else {
		logger.DebugV2(fmt.Sprintf("任务不存在或已停止: TaskID=%d", taskID))
	}
}

// ReloadTaskConfig 重新加载任务配置（用于API调用时主动触发热更新）
func (s *TaskService) ReloadTaskConfig() error {
	logger.DebugV2("开始重新加载任务配置")

	// 服务未运行则跳过重载，防止停止过程中被误触发启动新任务
	s.mu.Lock()
	if !s.isRunning {
		s.mu.Unlock()
		logger.DebugV2("服务已停止，跳过任务配置重载")
		return nil
	}
	s.mu.Unlock()

	// 获取最新的任务配置
	configs, err := s.taskConfigRepo.GetEnabledConfigs()
	if err != nil {
		logger.ErrorLogV2("重新加载任务配置失败", logger.ErrorV2(err))
		return err
	}

	// 获取当前运行的任务数量
	s.taskMutex.RLock()
	runningTaskCount := len(s.taskContexts)
	s.taskMutex.RUnlock()

	logger.DebugV2(fmt.Sprintf("配置检查: 数据库中启用配置=%d个, 当前运行任务=%d个", len(configs), runningTaskCount))

	// 检查并停止过时的任务
	s.checkAndStopObsoleteTasks(configs)

	// 启动新启用的任务
	s.startNewEnabledTasks(configs)

	// 获取更新后的运行任务数量
	s.taskMutex.RLock()
	newRunningTaskCount := len(s.taskContexts)
	s.taskMutex.RUnlock()

	if newRunningTaskCount != runningTaskCount {
		logger.InfoV2(fmt.Sprintf("任务配置已更新: 启用配置=%d个, 运行任务数量: %d -> %d", len(configs), runningTaskCount, newRunningTaskCount))
	} else {
		logger.DebugV2(fmt.Sprintf("任务配置无变化: 启用配置=%d个, 运行任务=%d个", len(configs), newRunningTaskCount))
	}

	return nil
}

// processOrderDetails 处理订单详情查询
func (s *TaskService) processOrderDetails() {
	logger.InfoV2("开始执行订单详情查询任务")
	configs, err := s.taskConfigRepo.GetEnabledConfigs()
	if err != nil {
		logger.ErrorLogV2("获取任务配置失败", logger.ErrorV2(err))
		return
	}

	for i, cfg := range configs {
		// 如果不是第一个配置，添加2秒间隔
		if i > 0 {
			select {
			case <-s.ctx.Done():
				return
			case <-time.After(2 * time.Second):
			}
		}
		s.processOrderDetailsForConfig(&cfg)
	}
}

// processOrderDetailsForConfig 为指定配置处理订单详情查询
func (s *TaskService) processOrderDetailsForConfig(cfg *model.TaskConfig) {
	logger.InfoV2(fmt.Sprintf("开始为配置处理订单详情查询: TaskID=%d, ChannelID=%d, ProductID=%s", cfg.ID, cfg.ChannelID, cfg.ProductID))
	// 获取平台账号信息
	platformAccount, err := s.platformAccountRepo.GetByIDWithContext(s.ctx, cfg.PlatformAccountID)
	if err != nil {
		logger.ErrorLogV2("获取平台账号失败", logger.Int64V2("platform_account_id", cfg.PlatformAccountID), logger.ErrorV2(err))
		return
	}

	// 获取平台信息
	_, platform, _, err := s.platformSvc.GetAPIKeyAndSecret(cfg.PlatformAccountID)
	if err != nil {
		logger.ErrorLogV2("获取平台信息失败", logger.ErrorV2(err))
		return
	}

	// 查询订单列表 - 获取第一页数据，查询状态为1的订单
	ctx, cancel := context.WithTimeout(s.ctx, 15*time.Second)
	defer cancel()
	orderList, pageResult, err := s.platformSvc.GetOrderList(ctx, "", 4, 0, 1, 100, platform.ApiURL, platformAccount) // 查询订单状态为1的订单
	if err != nil {
		logger.ErrorLogV2("查询订单列表失败", logger.ErrorV2(err))
		return
	}

	// 处理查询到的订单
	for _, order := range orderList {
		s.processOrderIfNotExists(&order, cfg, platformAccount, platform)
	}

	// 记录查询结果
	logger.InfoV2("查询到状态为4的订单",
		logger.Int64V2("count", int64(len(orderList))),
		logger.Int64V2("pages", int64(pageResult.Pages)))
}

// processOrderIfNotExists 检查订单是否存在，如果不存在则创建
func (s *TaskService) processOrderIfNotExists(order *platform.PlatformOrder, cfg *model.TaskConfig, platformAccount *model.PlatformAccount, platformInfo *model.Platform) {
	// 检查任务订单表中是否已存在
	existingTaskOrder, err := func() (*model.TaskOrder, error) {
		ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
		defer cancel()
		return s.taskOrderRepo.GetByOrderNumberWithContext(ctx, order.OrderNumber)
	}()
	if err == nil && existingTaskOrder != nil {
		// 订单已存在，忽略
		return
	}

	// 检查主订单表中是否已存在
	existingOrder, err := s.orderService.GetOrderByOrderNumber(s.ctx, order.OrderNumber)
	if err == nil && existingOrder != nil {
		// 订单已存在，忽略
		return
	}

	// 订单不存在，创建新订单
	s.createNewOrder(order, cfg, platformAccount, platformInfo)
}

// createNewOrder 创建新订单
func (s *TaskService) createNewOrder(order *platform.PlatformOrder, cfg *model.TaskConfig, platformAccount *model.PlatformAccount, platformInfo *model.Platform) {
	// 创建任务订单
	// taskOrder := &model.TaskOrder{
	// 	OrderNumber:      order.OrderNumber,
	// 	ChannelID:        order.ChannelId,
	// 	ProductID:        fmt.Sprintf("%d", order.ProductId),
	// 	AccountNum:       order.AccountNum,
	// 	AccountLocation:  order.AccountLocation,
	// 	SettlementAmount: order.SettlementAmount,
	// 	OrderStatus:      order.OrderStatus,
	// 	FaceValue:        order.FaceValue,
	// 	SettlementStatus: 1, // 待结算
	// 	CreateTime:       order.CreateTime.UnixMilli(),
	// 	ExpirationTime:   order.ExpirationTime.UnixMilli(),
	// }

	// if err := s.taskOrderRepo.Create(taskOrder); err != nil {
	// 	logger.Error(fmt.Sprintf("保存任务订单失败: OrderNumber=%s, error=%v", order.OrderNumber, err))
	// 	return
	// }

	// 保存订单到 order 订单表
	// 使用订单面值与商品表products的name字段中的数字进行匹配
	logger.InfoV2("开始匹配产品",
		logger.StringV2("order_number", order.OrderNumber),
		logger.Float64V2("face_value", order.FaceValue),
		logger.StringV2("product_name", order.ProductName))

	// 使用订单面值匹配商品表中name字段包含该数字的产品
	productObject, err := s.orderService.GetProductByNameValue(order.FaceValue, utils.ISPNameToCode(order.ProductName), 1)
	if err != nil {
		logger.ErrorLogV2("获取产品id失败",
			logger.StringV2("order_number", order.OrderNumber),
			logger.ErrorV2(err))
		return
	}

	var customerID int64
	if platformAccount.BindUserID != nil {
		customerID = *platformAccount.BindUserID
	} else {
		// 如果没有绑定用户，跳过订单创建
		logger.WarnV2("平台账号未绑定用户，跳过订单创建",
			logger.Int64V2("platform_account_id", cfg.PlatformAccountID),
			logger.StringV2("order_number", order.OrderNumber))
		return
	}

	// 根据订单状态设置初始状态和备注
	var initialStatus model.OrderStatus
	var remark string
	if order.OrderStatus == 4 {
		// 状态为4的订单直接设置为失败状态
		initialStatus = model.OrderStatusFailed
		remark = "系统检测到第三方平台订单状态为失败，自动创建失败订单"
		logger.InfoV2("检测到状态为4的订单，将创建为失败状态",
			logger.StringV2("order_number", order.OrderNumber))
	} else {
		initialStatus = model.OrderStatusPendingRecharge
		remark = ""
	}

	orderRecord := &model.Order{
		Mobile:            order.AccountNum,
		ProductID:         productObject.ID,
		Denom:             order.FaceValue,
		OfficialPayment:   order.SettlementAmount,
		UserQuotePayment:  order.SettlementAmount,
		UserPayment:       order.SettlementAmount,
		Price:             productObject.Price,
		Status:            initialStatus,
		IsDel:             0,
		Client:            3,
		ISP:               utils.ISPNameToCode(order.ProductName),
		Param1:            strings.Replace(order.ProductName, "中国", "", -1),
		AccountLocation:   order.AccountLocation,
		Param3:            order.ProductName,
		CreateTime:        order.CreateTime.Time,
		OutTradeNum:       order.OrderNumber,
		PlatformAccountID: cfg.PlatformAccountID,
		CustomerID:        customerID,
		PlatformName:      platformInfo.Name,
		PlatformCode:      platformInfo.Code,
		Remark:            remark,
	}

	if err := s.orderService.CreateOrder(s.ctx, orderRecord); err != nil {
		logger.ErrorLogV2("保存订单失败",
			logger.StringV2("order_number", order.OrderNumber),
			logger.ErrorV2(err))
		return
	}

	// 如果是状态为4的订单，创建后需要处理失败逻辑（包括发送通知）
	if order.OrderStatus == 4 {
		if err := s.orderService.ProcessOrderFail(s.ctx, orderRecord.ID, "第三方平台订单状态为失败，自动处理"); err != nil {
			logger.ErrorLogV2("处理失败订单失败",
				logger.StringV2("order_number", order.OrderNumber),
				logger.Int64V2("order_id", orderRecord.ID),
				logger.ErrorV2(err))
		} else {
			logger.InfoV2("失败订单处理完成，已发送通知",
				logger.StringV2("order_number", order.OrderNumber),
				logger.Int64V2("order_id", orderRecord.ID))
		}
	}

	logger.InfoV2("通过查单创建新订单成功",
		logger.StringV2("order_number", order.OrderNumber),
		logger.StringV2("status", initialStatus.String()))
}

// handleMatchedOrder 处理匹配到的订单
func (s *TaskService) handleMatchedOrder(order *platform.PlatformOrder, cfg *model.TaskConfig, channelID int, productID string, platformAccount *model.PlatformAccount, platformInfo *model.Platform) {
	// taskOrder := &model.TaskOrder{
	// 	OrderNumber:      order.OrderNumber,
	// 	ChannelID:        channelID,
	// 	ProductID:        productID,
	// 	AccountNum:       order.AccountNum,
	// 	AccountLocation:  order.AccountLocation,
	// 	SettlementAmount: order.SettlementAmount,
	// 	OrderStatus:      order.OrderStatus,
	// 	FaceValue:        order.FaceValue,
	// 	SettlementStatus: 1, // 待结算
	// 	CreateTime:       order.CreateTime.UnixMilli(),
	// 	ExpirationTime:   order.ExpirationTime.UnixMilli(),
	// }

	// if err := s.taskOrderRepo.Create(taskOrder); err != nil {
	// 	logger.Error(fmt.Sprintf("保存任务订单失败: OrderNumber=%s, error=%v", order.OrderNumber, err))
	// 	return
	// }

	// 保存订单到 order 订单表
	// 使用订单面值与商品表products的name字段中的数字进行匹配
	logger.InfoV2("开始匹配产品",
		logger.StringV2("order_number", order.OrderNumber),
		logger.Float64V2("face_value", order.FaceValue),
		logger.StringV2("product_name", order.ProductName))

	// 使用订单面值匹配商品表中name字段包含该数字的产品
	productObject, err := s.orderService.GetProductByNameValue(order.FaceValue, utils.ISPNameToCode(order.ProductName), 1)
	if err != nil {
		logger.ErrorLogV2("获取产品id失败",
			logger.StringV2("order_number", order.OrderNumber),
			logger.ErrorV2(err))
		return
	}

	var customerID int64
	if platformAccount.BindUserID != nil {
		customerID = *platformAccount.BindUserID
	} else {
		// 如果没有绑定用户，跳过订单创建
		logger.WarnV2("平台账号未绑定用户，跳过订单创建",
			logger.Int64V2("platform_account_id", cfg.PlatformAccountID),
			logger.StringV2("order_number", order.OrderNumber))
		return
	}

	orderRecord := &model.Order{
		Mobile:            order.AccountNum,
		ProductID:         productObject.ID,
		Denom:             order.FaceValue,
		OfficialPayment:   order.SettlementAmount,
		UserQuotePayment:  order.SettlementAmount,
		UserPayment:       order.SettlementAmount,
		Price:             productObject.Price,
		Status:            model.OrderStatusPendingRecharge,
		IsDel:             0,
		Client:            3,
		ISP:               utils.ISPNameToCode(order.ProductName),
		Param1:            strings.Replace(order.ProductName, "中国", "", -1),
		AccountLocation:   order.AccountLocation,
		Param3:            order.ProductName,
		CreateTime:        order.CreateTime.Time,
		OutTradeNum:       order.OrderNumber,
		PlatformAccountID: cfg.PlatformAccountID,
		CustomerID:        customerID,
		PlatformName:      platformInfo.Name,
		PlatformCode:      platformInfo.Code,
	}

	if err := s.orderService.CreateOrder(s.ctx, orderRecord); err != nil {
		logger.ErrorLogV2("保存订单失败",
			logger.StringV2("order_number", order.OrderNumber),
			logger.ErrorV2(err))
		return
	}

	logger.InfoV2("保存任务订单成功",
		logger.StringV2("order_number", order.OrderNumber))
}

// checkOrderThresholds 检查订单数量阈值并决定是否暂停或恢复拉单
func (s *TaskService) checkOrderThresholds(ctx context.Context) error {
	s.suspendMutex.RLock()
	currentSuspended := s.isPullingSuspended
	s.suspendMutex.RUnlock()

	if currentSuspended {
		// 当前已暂停，检查是否可以恢复
		processingCount, err := s.orderRepo.CountProcessingOrders(ctx)
		if err != nil {
			logger.ErrorLogV2("统计处理中订单数量失败",
				logger.ErrorV2(err))
			return err
		}

		if processingCount < int64(s.orderThresholds.ResumeThreshold) {
			s.resumePulling()
			logger.InfoV2("恢复拉单，处理中订单数量低于恢复阈值",
				logger.Int64V2("processing_count", processingCount),
				logger.Int64V2("resume_threshold", int64(s.orderThresholds.ResumeThreshold)))
		}
	} else {
		// 当前未暂停，检查是否需要暂停
		rechargeStatuses := []model.OrderStatus{
			model.OrderStatusPendingRecharge, // 待充值 (2)
			model.OrderStatusRecharging,      // 充值中 (3)
		}
		rechargeCount, err := s.orderRepo.CountByStatuses(ctx, rechargeStatuses)
		if err != nil {
			logger.ErrorLogV2("统计充值中和待充值订单数量失败",
				logger.ErrorV2(err))
			return err
		}

		if rechargeCount >= int64(s.orderThresholds.SuspendThreshold) {
			s.suspendPulling()
			logger.WarnV2("暂停拉单，订单数量达到暂停阈值",
				logger.Int64V2("recharge_count", rechargeCount),
				logger.Int64V2("suspend_threshold", int64(s.orderThresholds.SuspendThreshold)))
		}
	}

	return nil
}

// suspendPulling 暂停拉单
func (s *TaskService) suspendPulling() {
	s.suspendMutex.Lock()
	defer s.suspendMutex.Unlock()
	s.isPullingSuspended = true
}

// resumePulling 恢复拉单
func (s *TaskService) resumePulling() {
	s.suspendMutex.Lock()
	defer s.suspendMutex.Unlock()
	s.isPullingSuspended = false
}

// isPullingAllowed 检查是否允许拉单
func (s *TaskService) isPullingAllowed() bool {
	s.suspendMutex.RLock()
	defer s.suspendMutex.RUnlock()
	return !s.isPullingSuspended
}

// processTask 处理取单任务
func (s *TaskService) processTask() {
	logger.InfoV2("开始执行定时任务")

	// 若服务已停止，跳过处理，避免停止过程中继续拉起任务
	s.mu.Lock()
	if !s.isRunning {
		s.mu.Unlock()
		logger.DebugV2("服务已停止，跳过定时任务执行")
		return
	}
	s.mu.Unlock()

	// 获取所有启用的任务配置
	configs, err := s.taskConfigRepo.GetEnabledConfigs()
	if err != nil {
		logger.ErrorLogV2("获取任务配置失败",
			logger.ErrorV2(err))
		return
	}
	logger.InfoV2("获取到启用的任务配置",
		logger.Int64V2("count", int64(len(configs))))

	// 检查配置变更，停止已删除或禁用的任务
	s.checkAndStopObsoleteTasks(configs)

	maxConcurrent := s.config.MaxConcurrent
	logger.InfoV2("最大并发数",
		logger.Int64V2("max_concurrent", int64(maxConcurrent)))
	if maxConcurrent <= 0 {
		maxConcurrent = 20 // 默认最大并发数
	}
	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup

	for _, config := range configs {
		// 检查任务是否已在运行（使用双重检查确保准确性）
		s.taskMutex.Lock()
		taskCtx, isRunning := s.taskContexts[config.ID]
		if isRunning {
			// 验证上下文是否仍然有效
			select {
			case <-taskCtx.Ctx.Done():
				// 上下文已取消，删除无效的映射
				delete(s.taskContexts, config.ID)
				isRunning = false
				logger.DebugV2("清理无效的任务上下文",
					logger.Int64V2("task_id", config.ID))
			default:
				// 上下文仍然有效，任务确实在运行
				logger.DebugV2("任务已在运行，跳过",
					logger.Int64V2("task_id", config.ID),
					logger.Int64V2("channel_id", int64(config.ChannelID)),
					logger.StringV2("product_id", config.ProductID))
			}
		}
		s.taskMutex.Unlock()

		if isRunning {
			continue
		}

		logger.InfoV2("启动新任务",
			logger.Int64V2("task_id", config.ID),
			logger.Int64V2("channel_id", int64(config.ChannelID)),
			logger.StringV2("product_id", config.ProductID))

		sem <- struct{}{} // 占用一个并发槽
		wg.Add(1)
		go func(cfg *model.TaskConfig) {
			defer func() {
				<-sem // 释放并发槽
				wg.Done()
			}()

			s.processTaskConfig(cfg)
		}(&config)
	}
	wg.Wait()
}

// checkAndStopObsoleteTasks 检查并停止已删除或禁用的任务
func (s *TaskService) checkAndStopObsoleteTasks(currentConfigs []model.TaskConfig) {
	// 构建当前启用的任务ID集合
	currentTaskIDs := make(map[int64]bool)
	for _, cfg := range currentConfigs {
		currentTaskIDs[cfg.ID] = true
	}

	// 检查正在运行的任务，停止不在当前配置中的任务
	s.taskMutex.RLock()
	var tasksToStop []int64
	for taskID := range s.taskContexts {
		if !currentTaskIDs[taskID] {
			tasksToStop = append(tasksToStop, taskID)
		}
	}
	s.taskMutex.RUnlock()

	// 停止过时的任务
	for _, taskID := range tasksToStop {
		logger.InfoV2("正在停止过时任务",
			logger.Int64V2("task_id", int64(taskID)))
		s.StopTaskByID(taskID)
		// 注意：不需要手动删除任务上下文，StopTaskByID会触发defer清理逻辑
	}
}

// startNewEnabledTasks 启动新启用的任务
func (s *TaskService) startNewEnabledTasks(configs []model.TaskConfig) {
	// 若服务已停止，直接返回，避免停止过程中启动新任务
	s.mu.Lock()
	if !s.isRunning {
		s.mu.Unlock()
		logger.DebugV2("服务已停止，跳过启动新任务")
		return
	}
	s.mu.Unlock()

	maxConcurrent := s.config.MaxConcurrent
	if maxConcurrent <= 0 {
		maxConcurrent = 20 // 默认最大并发数
	}
	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup
	newTaskCount := 0

	for _, config := range configs {
		// 检查任务是否已在运行（使用双重检查确保准确性）
		s.taskMutex.Lock()
		taskCtx, isRunning := s.taskContexts[config.ID]
		if isRunning {
			// 验证上下文是否仍然有效
			select {
			case <-taskCtx.Ctx.Done():
				// 上下文已取消，删除无效的映射
				delete(s.taskContexts, config.ID)
				isRunning = false
				logger.DebugV2("清理无效的任务上下文",
					logger.Int64V2("task_id", config.ID))
			default:
				// 上下文仍然有效，任务确实在运行
				logger.DebugV2("任务已在运行，跳过",
					logger.Int64V2("task_id", config.ID),
					logger.Int64V2("channel_id", int64(config.ChannelID)),
					logger.StringV2("product_id", config.ProductID))
			}
		}
		s.taskMutex.Unlock()

		if isRunning {
			continue
		}

		newTaskCount++
		logger.InfoV2("启动新任务",
			logger.Int64V2("task_id", config.ID),
			logger.Int64V2("channel_id", int64(config.ChannelID)),
			logger.StringV2("product_id", config.ProductID))

		sem <- struct{}{} // 占用一个并发槽
		wg.Add(1)
		go func(cfg *model.TaskConfig) {
			defer func() {
				<-sem // 释放并发槽
				wg.Done()
			}()

			s.processTaskConfig(cfg)
		}(&config)
	}
	wg.Wait()

	if newTaskCount > 0 {
		logger.InfoV2("新任务启动完成",
			logger.Int64V2("new_task_count", int64(newTaskCount)))
	}
}

// checkTaskConfigChanged 检查任务配置是否发生变更
func (s *TaskService) checkTaskConfigChanged(oldCfg *model.TaskConfig) bool {
	// 从数据库获取最新配置
	newCfg, err := s.taskConfigRepo.GetByID(oldCfg.ID)
	if err != nil {
		logger.ErrorLogV2("获取任务配置失败",
			logger.Int64V2("task_id", oldCfg.ID),
			logger.ErrorV2(err))
		return false
	}

	// 检查任务是否被禁用
	if newCfg.Status != 1 {
		logger.InfoV2("任务配置已被禁用",
			logger.Int64V2("task_id", oldCfg.ID))
		return true
	}

	// 检查关键配置是否发生变更
	if oldCfg.PlatformAccountID != newCfg.PlatformAccountID ||
		oldCfg.ChannelID != newCfg.ChannelID ||
		oldCfg.ProductID != newCfg.ProductID ||
		oldCfg.FaceValues != newCfg.FaceValues ||
		oldCfg.MinSettleAmounts != newCfg.MinSettleAmounts ||
		oldCfg.Provinces != newCfg.Provinces {
		logger.InfoV2("任务配置发生变更",
			logger.Int64V2("task_id", oldCfg.ID))
		return true
	}

	return false
}

// processTaskConfig 处理单个任务配置
func (s *TaskService) processTaskConfig(cfg *model.TaskConfig) {
	// 创建任务专用的上下文
	taskCtx, taskCancel := context.WithCancel(s.ctx)
	taskID := cfg.ID

	logger.InfoV2("开始处理任务配置",
		logger.Int64V2("task_id", cfg.ID),
		logger.Int64V2("platform_account_id", cfg.PlatformAccountID),
		logger.Int64V2("channel_id", int64(cfg.ChannelID)),
		logger.StringV2("product_id", cfg.ProductID))

	// 获取任务配置信息
	channelID := int(cfg.ChannelID)
	productID := cfg.ProductID
	provinces := cfg.Provinces
	faceValues := cfg.FaceValues
	minSettleAmounts := cfg.MinSettleAmounts

	// 注册任务上下文（确保原子性操作）
	s.taskMutex.Lock()
	if existingTaskCtx, exists := s.taskContexts[taskID]; exists {
		// 如果任务已存在，先取消旧任务
		logger.WarnV2("检测到重复任务，取消旧任务",
			logger.Int64V2("task_id", taskID))
		existingTaskCtx.Cancel()
		// 立即删除旧的上下文映射
		delete(s.taskContexts, taskID)
	}
	// 注册新的任务上下文
	s.taskContexts[taskID] = &TaskContext{
		Ctx:    taskCtx,
		Cancel: taskCancel,
	}
	s.taskMutex.Unlock()

	logger.InfoV2("任务上下文已注册",
		logger.Int64V2("task_id", taskID),
		logger.Int64V2("channel_id", int64(channelID)),
		logger.StringV2("product_id", productID))

	// 定义查询间隔
	queryInterval := s.config.Interval

	// 确保任务结束时清理上下文
	defer func() {
		// 先取消上下文
		taskCancel()

		// 再清理任务上下文映射
		s.taskMutex.Lock()
		defer s.taskMutex.Unlock()

		// 只有当前上下文仍然存在时才删除（避免重复删除）
		if currentTaskCtx, exists := s.taskContexts[taskID]; exists && currentTaskCtx.Ctx == taskCtx {
			delete(s.taskContexts, taskID)
			logger.InfoV2("任务上下文已清理",
				logger.Int64V2("task_id", taskID),
				logger.Int64V2("channel_id", int64(channelID)),
				logger.StringV2("product_id", productID))
		} else {
			logger.DebugV2("任务上下文已被其他实例清理或替换",
				logger.Int64V2("task_id", taskID))
		}
	}()

	// 获取账号信息，失败时重试而不是退出
	var appkey, accountName string
	var platform *model.Platform
	for {
		var err error
		appkey, platform, accountName, err = s.platformSvc.GetAPIKeyAndSecret(cfg.PlatformAccountID)
		if err != nil {
			logger.ErrorLogV2("获取账号信息失败，稍后重试",
				logger.Int64V2("retry_seconds", int64(queryInterval)),
				logger.Int64V2("task_id", taskID),
				logger.Int64V2("platform_account_id", cfg.PlatformAccountID),
				logger.ErrorV2(err))
			select {
			case <-taskCtx.Done():
				logger.InfoV2("任务在获取账号信息重试中被停止",
					logger.Int64V2("task_id", taskID))
				return
			case <-time.After(time.Duration(queryInterval) * time.Second):
				continue
			}
		}
		break
	}
	//获取平台账号信息
	// 获取平台账号信息（失败或未绑定用户时重试）
	var platformAccount *model.PlatformAccount
	for {
		var err error
		platformAccount, err = s.platformAccountRepo.GetByIDWithContext(taskCtx, cfg.PlatformAccountID)
		if err != nil {
			logger.ErrorLogV2("获取平台账号信息失败，稍后重试",
				logger.Int64V2("retry_seconds", int64(queryInterval)),
				logger.Int64V2("platform_account_id", cfg.PlatformAccountID),
				logger.ErrorV2(err))
			select {
			case <-taskCtx.Done():
				logger.InfoV2("任务在获取平台账号信息重试中被停止",
					logger.Int64V2("task_id", taskID))
				return
			case <-time.After(time.Duration(queryInterval) * time.Second):
				continue
			}
		}

		if platformAccount.BindUserID == nil {
			logger.WarnV2("平台账号未绑定用户，稍后重试",
				logger.Int64V2("retry_seconds", int64(queryInterval)),
				logger.Int64V2("platform_account_id", cfg.PlatformAccountID),
				logger.Int64V2("channel_id", int64(channelID)),
				logger.StringV2("product_id", productID))
			select {
			case <-taskCtx.Done():
				logger.InfoV2("任务在等待绑定用户时被停止",
					logger.Int64V2("task_id", taskID))
				return
			case <-time.After(time.Duration(queryInterval) * time.Second):
				continue
			}
		}
		break
	}
	logger.InfoV2("开始处理任务配置",
		logger.Int64V2("channel_id", int64(channelID)),
		logger.StringV2("product_id", productID),
		logger.StringV2("account_name", accountName),
		logger.StringV2("provinces", provinces),
		logger.StringV2("face_values", faceValues),
		logger.StringV2("min_settle_amounts", minSettleAmounts))

	// 获取或申请token
	logger.InfoV2("开始申请token",
		logger.Int64V2("channel_id", int64(channelID)),
		logger.StringV2("product_id", productID),
		logger.StringV2("account_name", accountName),
		logger.StringV2("provinces", provinces),
		logger.StringV2("face_values", faceValues),
		logger.StringV2("min_settle_amounts", minSettleAmounts))
	var token string
	for {
		tokenApplyStartTime := time.Now()
		var err error
		token, err = s.platformSvc.GetTokenWithContext(taskCtx, cfg.ID, channelID, productID, "", cfg.FaceValues, cfg.MinSettleAmounts, appkey, accountName, platform.ApiURL)
		if err != nil {
			logger.ErrorLogV2("申请token失败",
				logger.Int64V2("channel_id", int64(channelID)),
				logger.StringV2("product_id", productID),
				logger.StringV2("account_name", accountName),
				logger.DurationV2("duration", time.Since(tokenApplyStartTime)),
				logger.ErrorV2(err))
			select {
			case <-taskCtx.Done():
				logger.InfoV2("任务在申请token重试中被停止",
					logger.Int64V2("task_id", taskID))
				return
			case <-time.After(time.Duration(queryInterval) * time.Second):
				continue
			}
		}
		logger.InfoV2("申请token成功",
			logger.Int64V2("channel_id", int64(channelID)),
			logger.StringV2("product_id", productID),
			logger.StringV2("account_name", accountName),
			logger.StringV2("token", token),
			logger.DurationV2("duration", time.Since(tokenApplyStartTime)))
		break
	}

	// 开始查询循环：基于token创建时间判断5分钟过期，不限制查询次数
	// queryInterval 已在上方定义
	tokenStartTime := time.Now() // 记录token开始使用的时间
	logger.InfoV2("token开始生命周期",
		logger.StringV2("token", token),
		logger.StringV2("start_time", tokenStartTime.Format("2006-01-02 15:04:05")),
		logger.StringV2("expected_expire_time", tokenStartTime.Add(5*time.Minute).Format("2006-01-02 15:04:05")),
		logger.StringV2("account_name", accountName))

	// 配置检查计时器
	configCheckInterval := 30 * time.Second // 每30秒检查一次配置
	lastConfigCheck := time.Now()

	for {
		select {
		case <-taskCtx.Done():
			logger.InfoV2("任务被主动停止",
				logger.Int64V2("task_id", taskID),
				logger.Int64V2("channel_id", int64(channelID)),
				logger.StringV2("product_id", productID))
			return
		case <-s.ctx.Done():
			return
		default:
		}

		// 定期检查配置是否有变更
		if time.Since(lastConfigCheck) >= configCheckInterval {
			if s.checkTaskConfigChanged(cfg) {
				logger.InfoV2("检测到任务配置变更，重启任务",
					logger.Int64V2("task_id", taskID),
					logger.Int64V2("channel_id", int64(channelID)),
					logger.StringV2("product_id", productID))
				return
			}
			lastConfigCheck = time.Now()
		}

		// 检查token是否已过期（5分钟）
		if time.Since(tokenStartTime) >= 5*time.Minute {
			tokenLifetime := time.Since(tokenStartTime)
			logger.InfoV2("token已过期，重新申请token",
				logger.StringV2("token", token),
				logger.Int64V2("channel_id", int64(channelID)),
				logger.StringV2("product_id", productID),
				logger.DurationV2("token_lifetime", tokenLifetime),
				logger.StringV2("expire_time", time.Now().Format("2006-01-02 15:04:05")),
				logger.StringV2("account_name", accountName))

			// 重新申请token而不是退出任务
			reapplyStartTime := time.Now()
			newToken, err := s.platformSvc.GetTokenWithContext(taskCtx, cfg.ID, channelID, productID, "", cfg.FaceValues, cfg.MinSettleAmounts, appkey, accountName, platform.ApiURL)
			if err != nil {
				logger.ErrorLogV2("重新申请token失败",
					logger.Int64V2("channel_id", int64(channelID)),
					logger.StringV2("product_id", productID),
					logger.StringV2("account_name", accountName),
					logger.DurationV2("duration", time.Since(reapplyStartTime)),
					logger.ErrorV2(err))
				logger.InfoV2("处理订单后重新申请token失败，稍后重试",
					logger.Int64V2("retry_seconds", int64(queryInterval)))
				select {
				case <-taskCtx.Done():
					logger.InfoV2("任务在处理订单后重新申请token等待中被停止",
						logger.Int64V2("task_id", taskID))
					return
				case <-time.After(time.Duration(queryInterval) * time.Second):
					continue
				}
			}

			token = newToken
			tokenStartTime = time.Now() // 重置token开始时间
			logger.InfoV2("重新申请token成功",
				logger.Int64V2("channel_id", int64(channelID)),
				logger.StringV2("product_id", productID),
				logger.StringV2("account_name", accountName),
				logger.StringV2("new_token", token),
				logger.DurationV2("duration", time.Since(reapplyStartTime)))
			continue
		}

		// 检查订单数量阈值，决定是否暂停拉单
		if err := s.checkOrderThresholds(taskCtx); err != nil {
			logger.ErrorLogV2("检查订单数量阈值失败",
				logger.Int64V2("task_id", taskID),
				logger.ErrorV2(err))
		}

		// 如果拉单被暂停，跳过本次查询
		if !s.isPullingAllowed() {
			logger.DebugV2("拉单已暂停，跳过查询",
				logger.Int64V2("task_id", taskID))
			continue
		}

		// 查询订单
		// apiurl := "http://60.205.159.182:5000/"
		order, err := s.platformSvc.QueryTask(taskCtx, token, platform.ApiURL, appkey, accountName)
		if err != nil {
			tokenLifetime := time.Since(tokenStartTime)
			logger.ErrorLogV2("查询任务匹配状态失败",
				logger.StringV2("token", token),
				logger.DurationV2("token_lifetime", tokenLifetime),
				logger.ErrorV2(err))

			// 检查任务是否被取消
			select {
			case <-taskCtx.Done():
				logger.InfoV2("任务在错误处理中被停止",
					logger.Int64V2("task_id", taskID))
				return
			default:
			}

			if strings.Contains(err.Error(), "匹配失败") {
				// 匹配失败，让当前token失效并重新申请token
				tokenLifetime := time.Since(tokenStartTime)
				logger.InfoV2("主动失效token",
					logger.StringV2("token", token),
					logger.StringV2("reason", "匹配失败"),
					logger.DurationV2("token_lifetime", tokenLifetime),
					logger.StringV2("invalid_time", time.Now().Format("2006-01-02 15:04:05")))
				_ = s.platformSvc.InvalidateToken(cfg.ID)

				logger.InfoV2("匹配订单失败重新申请token",
					logger.Int64V2("channel_id", int64(channelID)),
					logger.StringV2("product_id", productID),
					logger.StringV2("account_name", accountName))
				reapplyStartTime := time.Now()

				// 检查任务是否被取消
				select {
				case <-taskCtx.Done():
					logger.InfoV2("任务在重新申请token前被停止",
						logger.Int64V2("task_id", taskID))
					return
				default:
				}

				newToken, err := s.platformSvc.GetTokenWithContext(taskCtx, cfg.ID, channelID, productID, "", cfg.FaceValues, cfg.MinSettleAmounts, appkey, accountName, platform.ApiURL)
				if err != nil {
					logger.ErrorLogV2("匹配订单失败重新申请token失败",
						logger.Int64V2("channel_id", int64(channelID)),
						logger.StringV2("product_id", productID),
						logger.StringV2("account_name", accountName),
						logger.DurationV2("duration", time.Since(reapplyStartTime)),
						logger.ErrorV2(err))
					// 重新申请token失败时等待后重试，而不是直接退出
					logger.InfoV2("重新申请token失败，稍后重试",
						logger.Int64V2("retry_seconds", int64(queryInterval)))
					select {
					case <-taskCtx.Done():
						logger.InfoV2("任务在匹配失败重新申请token等待中被停止",
							logger.Int64V2("task_id", taskID))
						return
					case <-time.After(time.Duration(queryInterval) * time.Second):
						continue
					}
				}

				token = newToken
				tokenStartTime = time.Now() // 重置token开始时间
				logger.InfoV2("匹配订单失败重新申请token成功",
					logger.Int64V2("channel_id", int64(channelID)),
					logger.StringV2("product_id", productID),
					logger.StringV2("account_name", accountName),
					logger.StringV2("new_token", token),
					logger.DurationV2("duration", time.Since(reapplyStartTime)))
				logger.InfoV2("新token开始生命周期",
					logger.StringV2("token", token),
					logger.StringV2("start_time", tokenStartTime.Format("2006-01-02 15:04:05")),
					logger.StringV2("expected_expire_time", tokenStartTime.Add(5*time.Minute).Format("2006-01-02 15:04:05")))
				select {
				case <-taskCtx.Done():
					logger.InfoV2("任务在匹配失败处理后等待中被停止",
						logger.Int64V2("task_id", taskID))
					return
				case <-time.After(1 * time.Second):
					continue
				}
			}
			// 其他错误（非匹配失败），记录错误并等待后重试，而不是直接退出任务
			logger.WarnV2("查询订单遇到非匹配失败错误，稍后重试",
				logger.Int64V2("retry_seconds", int64(queryInterval)),
				logger.ErrorV2(err))
			select {
			case <-taskCtx.Done():
				logger.InfoV2("任务在非匹配失败错误等待中被停止",
					logger.Int64V2("task_id", taskID))
				return
			case <-time.After(time.Duration(queryInterval) * time.Second):
				continue
			}
		}

		if order != nil {
			// 匹配到订单，处理订单并重新申请token
			logger.InfoV2("匹配到订单",
				logger.StringV2("token", token),
				logger.StringV2("order_number", order.OrderNumber),
				logger.StringV2("account_num", order.AccountNum),
				logger.Float64V2("settlement_amount", order.SettlementAmount))

			// 让当前token失效
			tokenLifetime := time.Since(tokenStartTime)
			logger.InfoV2("主动失效token",
				logger.StringV2("token", token),
				logger.StringV2("reason", "匹配到订单"),
				logger.DurationV2("token_lifetime", tokenLifetime),
				logger.StringV2("invalid_time", time.Now().Format("2006-01-02 15:04:05")))
			_ = s.platformSvc.InvalidateToken(cfg.ID)

			// 处理订单
			s.handleMatchedOrder(order, cfg, channelID, productID, platformAccount, platform)

			// 重新申请token继续查询
			logger.InfoV2("开始重新申请token",
				logger.Int64V2("channel_id", int64(channelID)),
				logger.StringV2("product_id", productID),
				logger.StringV2("account_name", accountName))
			reapplyStartTime := time.Now()

			// 检查任务是否被取消
			select {
			case <-taskCtx.Done():
				logger.InfoV2("任务在处理订单后重新申请token前被停止",
					logger.Int64V2("task_id", taskID))
				return
			default:
			}

			newToken, err := s.platformSvc.GetTokenWithContext(taskCtx, cfg.ID, channelID, productID, "", cfg.FaceValues, cfg.MinSettleAmounts, appkey, accountName, platform.ApiURL)
			if err != nil {
				logger.ErrorLogV2("重新申请token失败",
					logger.Int64V2("channel_id", int64(channelID)),
					logger.StringV2("product_id", productID),
					logger.StringV2("account_name", accountName),
					logger.DurationV2("duration", time.Since(reapplyStartTime)),
					logger.ErrorV2(err))
				logger.InfoV2("处理订单后重新申请token失败，稍后重试",
					logger.Int64V2("retry_seconds", int64(queryInterval)))
				select {
				case <-taskCtx.Done():
					logger.InfoV2("任务在处理订单后重新申请token等待中被停止",
						logger.Int64V2("task_id", taskID))
					return
				case <-time.After(time.Duration(queryInterval) * time.Second):
					continue
				}
			}

			token = newToken
			tokenStartTime = time.Now() // 重置token开始时间
			logger.InfoV2("重新申请token成功",
				logger.Int64V2("channel_id", int64(channelID)),
				logger.StringV2("product_id", productID),
				logger.StringV2("account_name", accountName),
				logger.StringV2("new_token", token),
				logger.DurationV2("duration", time.Since(reapplyStartTime)))
			logger.InfoV2("新token开始生命周期",
				logger.StringV2("token", token),
				logger.StringV2("start_time", tokenStartTime.Format("2006-01-02 15:04:05")),
				logger.StringV2("expected_expire_time", tokenStartTime.Add(5*time.Minute).Format("2006-01-02 15:04:05")))
		} else {
			// 未匹配到订单，等待后继续查询
			tokenLifetime := time.Since(tokenStartTime)
			logger.DebugV2("未匹配到订单，继续查询",
				logger.StringV2("token", token),
				logger.DurationV2("token_lifetime", tokenLifetime))
		}

		// 等待查询间隔（可被任务取消）
		select {
		case <-taskCtx.Done():
			logger.InfoV2("任务在查询间隔等待中被停止",
				logger.Int64V2("task_id", int64(taskID)))
			return
		case <-time.After(time.Duration(queryInterval) * time.Second):
			// 继续下一轮
		}
	}
}
