package service

import (
	"context"
	"fmt"
	"recharge-go/internal/model"
	"recharge-go/internal/repository"
	"recharge-go/internal/service/platform"
	"recharge-go/internal/utils"
	"recharge-go/pkg/logger"
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
	config              *TaskConfig
	ctx                 context.Context
	cancel              context.CancelFunc
	wg                  sync.WaitGroup
	mu                  sync.Mutex
	isRunning           bool
	platformAccountRepo *repository.PlatformAccountRepository
	// 任务上下文管理
	taskContexts        map[int64]context.CancelFunc // 任务ID -> 取消函数
	taskMutex           sync.RWMutex                 // 保护taskContexts的读写锁
}

type TaskConfig struct {
	Interval      time.Duration // 任务执行间隔
	MaxRetries    int           // 最大重试次数
	RetryDelay    time.Duration // 重试延迟
	MaxConcurrent int           // 最大并发数
	APIKey        string        // API密钥
	UserID        string        // 用户ID
	BaseURL       string        // API基础URL
}

func NewTaskService(
	taskConfigRepo *repository.TaskConfigRepository,
	taskOrderRepo *repository.TaskOrderRepository,
	orderRepo repository.OrderRepository,
	daichongOrderRepo *repository.DaichongOrderRepository,
	platformSvc *platform.Service,
	orderService OrderService,
	config *TaskConfig,
	platformAccountRepo *repository.PlatformAccountRepository,
) *TaskService {
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
		taskContexts:        make(map[int64]context.CancelFunc),
	}
}

// StartTask 启动自动取单任务
func (s *TaskService) StartTask() {
	s.mu.Lock()
	if s.isRunning {
		s.mu.Unlock()
		return
	}
	s.isRunning = true
	s.mu.Unlock()

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(s.config.Interval)
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

	s.cancel()
	s.wg.Wait()
}

// StopTaskByID 主动停止特定任务
func (s *TaskService) StopTaskByID(taskID int64) {
	s.taskMutex.RLock()
	if cancel, exists := s.taskContexts[taskID]; exists {
		cancel()
		logger.Info(fmt.Sprintf("主动停止任务: TaskID=%d", taskID))
	}
	s.taskMutex.RUnlock()
}

// ReloadTaskConfig 重新加载任务配置（用于API调用时主动触发热更新）
func (s *TaskService) ReloadTaskConfig() error {
	logger.Info("开始重新加载任务配置")

	// 获取最新的任务配置
	configs, err := s.taskConfigRepo.GetEnabledConfigs()
	if err != nil {
		logger.Error("重新加载任务配置失败", err)
		return err
	}

	// 检查并停止过时的任务
	s.checkAndStopObsoleteTasks(configs)

	logger.Info(fmt.Sprintf("任务配置重新加载完成，当前启用配置数量: %d", len(configs)))
	return nil
}

// processTask 处理取单任务
func (s *TaskService) processTask() {
	logger.Info("开始执行定时任务")

	// 获取所有启用的任务配置
	configs, err := s.taskConfigRepo.GetEnabledConfigs()
	if err != nil {
		logger.Error("获取任务配置失败", err)
		return
	}
	logger.Info(fmt.Sprintf("获取到 %d 个启用的任务配置", len(configs)))

	// 检查配置变更，停止已删除或禁用的任务
	s.checkAndStopObsoleteTasks(configs)

	maxConcurrent := s.config.MaxConcurrent
	logger.Info(fmt.Sprintf("最大并发数: %d", maxConcurrent))
	if maxConcurrent <= 0 {
		maxConcurrent = 20 // 默认最大并发数
	}
	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup

	for _, config := range configs {
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
		s.StopTaskByID(taskID)
		// 从任务上下文中移除
		s.taskMutex.Lock()
		delete(s.taskContexts, taskID)
		s.taskMutex.Unlock()
		logger.Info(fmt.Sprintf("已停止过时任务: TaskID=%d", taskID))
	}
}

// checkTaskConfigChanged 检查任务配置是否发生变更
func (s *TaskService) checkTaskConfigChanged(oldCfg *model.TaskConfig) bool {
	// 从数据库获取最新配置
	newCfg, err := s.taskConfigRepo.GetByID(oldCfg.ID)
	if err != nil {
		logger.Error(fmt.Sprintf("获取任务配置失败: TaskID=%d, error=%v", oldCfg.ID, err))
		return false
	}

	// 检查任务是否被禁用
	if newCfg.Status != 1 {
		logger.Info(fmt.Sprintf("任务配置已被禁用: TaskID=%d", oldCfg.ID))
		return true
	}

	// 检查关键配置是否发生变更
	if oldCfg.PlatformAccountID != newCfg.PlatformAccountID ||
		oldCfg.ChannelID != newCfg.ChannelID ||
		oldCfg.ProductID != newCfg.ProductID ||
		oldCfg.FaceValues != newCfg.FaceValues ||
		oldCfg.MinSettleAmounts != newCfg.MinSettleAmounts ||
		oldCfg.Provinces != newCfg.Provinces {
		logger.Info(fmt.Sprintf("任务配置发生变更: TaskID=%d", oldCfg.ID))
		return true
	}

	return false
}

// processTaskConfig 处理单个任务配置
func (s *TaskService) processTaskConfig(cfg *model.TaskConfig) {
	// 创建任务专用的上下文
	taskCtx, taskCancel := context.WithCancel(s.ctx)
	taskID := cfg.ID

	// 注册任务上下文
	s.taskMutex.Lock()
	s.taskContexts[taskID] = taskCancel
	s.taskMutex.Unlock()

	// 确保任务结束时清理上下文
	defer func() {
		s.taskMutex.Lock()
		delete(s.taskContexts, taskID)
		s.taskMutex.Unlock()
		taskCancel()
	}()

	channelID := int(cfg.ChannelID)
	productID := cfg.ProductID
	provinces := cfg.Provinces
	faceValues := cfg.FaceValues
	minSettleAmounts := cfg.MinSettleAmounts

	appkey, platform, accountName, err := s.platformSvc.GetAPIKeyAndSecret(cfg.PlatformAccountID)
	if err != nil {
		logger.Error(fmt.Sprintf("获取账号信息失败: %v", err))
		return
	}
	//获取平台账号信息
	platformAccount, err := s.platformAccountRepo.GetByID(cfg.PlatformAccountID)
	if err != nil {
		logger.Error(fmt.Sprintf("获取平台账号信息失败: error=%+v", err))
		return
	}

	if platformAccount.BindUserID != nil {
		fmt.Printf("userid %d platformAccount++++++++!!!!!!!!%+v", *platformAccount.BindUserID, platformAccount)
	} else {
		fmt.Printf("userid <nil> platformAccount++++++++!!!!!!!!%+v", platformAccount)
		logger.Warn(fmt.Sprintf("平台账号未绑定用户，跳过任务处理: PlatformAccountID=%d, ChannelID=%d, ProductID=%s", cfg.PlatformAccountID, channelID, productID))
		return
	}
	logger.Info(fmt.Sprintf("处理任务配置: ChannelID=%d, ProductID=%s accountName=%s provinces=%s faceValues=%s minSettleAmounts=%s", channelID, productID, accountName, provinces, faceValues, minSettleAmounts))

	// 获取或申请token
	logger.Info(fmt.Sprintf("开始申请token: ChannelID=%d, ProductID=%s, AccountName=%s provinces=%s faceValues=%s minSettleAmounts=%s", channelID, productID, accountName, provinces, faceValues, minSettleAmounts))
	tokenApplyStartTime := time.Now()

	token, err := s.platformSvc.GetToken(channelID, productID, "", cfg.FaceValues, cfg.MinSettleAmounts, appkey, accountName, platform.ApiURL, cfg.ID)
	if err != nil {
		logger.Error(fmt.Sprintf("申请token失败: ChannelID=%d, ProductID=%s, AccountName=%s, 耗时=%v, error=%v",
			channelID, productID, accountName, time.Since(tokenApplyStartTime), err))
		return
	}

	logger.Info(fmt.Sprintf("申请token成功: ChannelID=%d, ProductID=%s, AccountName=%s, token=%s, 耗时=%v",
		channelID, productID, accountName, token, time.Since(tokenApplyStartTime)))

	// 开始查询循环：基于token创建时间判断5分钟过期，不限制查询次数
	queryInterval := 2 * time.Second
	tokenStartTime := time.Now() // 记录token开始使用的时间
	logger.Info(fmt.Sprintf("token开始生命周期: token=%s, 开始时间=%s, 预计过期时间=%s AccountName=%s",
		token, tokenStartTime.Format("2006-01-02 15:04:05"), tokenStartTime.Add(5*time.Minute).Format("2006-01-02 15:04:05"), accountName))

	// 配置检查计时器
	configCheckInterval := 30 * time.Second // 每30秒检查一次配置
	lastConfigCheck := time.Now()

	for {
		select {
		case <-taskCtx.Done():
			logger.Info(fmt.Sprintf("任务被主动停止: TaskID=%d, ChannelID=%d, ProductID=%s", taskID, channelID, productID))
			return
		case <-s.ctx.Done():
			return
		default:
		}

		// 定期检查配置是否有变更
		if time.Since(lastConfigCheck) >= configCheckInterval {
			if s.checkTaskConfigChanged(cfg) {
				logger.Info(fmt.Sprintf("检测到任务配置变更，重启任务: TaskID=%d, ChannelID=%d, ProductID=%s", taskID, channelID, productID))
				return
			}
			lastConfigCheck = time.Now()
		}

		// 检查token是否已过期（5分钟）
		if time.Since(tokenStartTime) >= 5*time.Minute {
			tokenLifetime := time.Since(tokenStartTime)
			logger.Info(fmt.Sprintf("token已过期，结束生命周期: token=%s, ChannelID=%d, ProductID=%s, 生命周期=%v, 过期时间=%s AccountName=%s",
				token, channelID, productID, tokenLifetime, time.Now().Format("2006-01-02 15:04:05"), accountName))
			return
		}

		// 查询订单
		order, err := s.platformSvc.QueryTask(token, platform.ApiURL, appkey, accountName)
		if err != nil {
			tokenLifetime := time.Since(tokenStartTime)
			logger.Error(fmt.Sprintf("查询任务匹配状态失败: token=%s, 生命周期=%v, error=%v", token, tokenLifetime, err))
			if strings.Contains(err.Error(), "匹配失败") {
				// 匹配失败，让当前token失效并重新申请token
				tokenLifetime := time.Since(tokenStartTime)
				logger.Info(fmt.Sprintf("主动失效token: token=%s, 原因=匹配失败, 生命周期=%v, 失效时间=%s",
					token, tokenLifetime, time.Now().Format("2006-01-02 15:04:05")))
				_ = s.platformSvc.InvalidateToken(cfg.ID)

				logger.Info(fmt.Sprintf("匹配订单失败重新申请token: ChannelID=%d, ProductID=%s, AccountName=%s", channelID, productID, accountName))
				reapplyStartTime := time.Now()

				newToken, err := s.platformSvc.GetToken(channelID, productID, "", cfg.FaceValues, cfg.MinSettleAmounts, appkey, accountName, platform.ApiURL, cfg.ID)
				if err != nil {
					logger.Error(fmt.Sprintf("匹配订单失败重新申请token失败: ChannelID=%d, ProductID=%s, AccountName=%s, 耗时=%v, error=%v",
						channelID, productID, accountName, time.Since(reapplyStartTime), err))
					return
				}

				token = newToken
				tokenStartTime = time.Now() // 重置token开始时间
				logger.Info(fmt.Sprintf("匹配订单失败重新申请token成功: ChannelID=%d, ProductID=%s, AccountName=%s, 新token=%s, 耗时=%v",
					channelID, productID, accountName, token, time.Since(reapplyStartTime)))
				logger.Info(fmt.Sprintf("新token开始生命周期: token=%s, 开始时间=%s, 预计过期时间=%s",
					token, tokenStartTime.Format("2006-01-02 15:04:05"), tokenStartTime.Add(5*time.Minute).Format("2006-01-02 15:04:05")))
				time.Sleep(1 * time.Second)
				continue
			}
			return
		}

		if order != nil {
			// 匹配到订单，处理订单并重新申请token
			logger.Info(fmt.Sprintf("匹配到订单: token=%s OrderNumber=%s, AccountNum=%s,SettlementAmount=%.2f",
				token, order.OrderNumber, order.AccountNum, order.SettlementAmount))

			// 让当前token失效
			tokenLifetime := time.Since(tokenStartTime)
			logger.Info(fmt.Sprintf("主动失效token: token=%s, 原因=匹配到订单, 生命周期=%v, 失效时间=%s",
				token, tokenLifetime, time.Now().Format("2006-01-02 15:04:05")))
			_ = s.platformSvc.InvalidateToken(cfg.ID)

			// 处理订单
			s.handleMatchedOrder(order, cfg, channelID, productID, platformAccount, platform)

			// 重新申请token继续查询
			logger.Info(fmt.Sprintf("开始重新申请token: ChannelID=%d, ProductID=%s, AccountName=%s", channelID, productID, accountName))
			reapplyStartTime := time.Now()

			newToken, err := s.platformSvc.GetToken(channelID, productID, "", cfg.FaceValues, cfg.MinSettleAmounts, appkey, accountName, platform.ApiURL, cfg.ID)
			if err != nil {
				logger.Error(fmt.Sprintf("重新申请token失败: ChannelID=%d, ProductID=%s, AccountName=%s, 耗时=%v, error=%v",
					channelID, productID, accountName, time.Since(reapplyStartTime), err))
				return
			}

			token = newToken
			tokenStartTime = time.Now() // 重置token开始时间
			logger.Info(fmt.Sprintf("重新申请token成功: ChannelID=%d, ProductID=%s, AccountName=%s, 新token=%s, 耗时=%v",
				channelID, productID, accountName, token, time.Since(reapplyStartTime)))
			logger.Info(fmt.Sprintf("新token开始生命周期: token=%s, 开始时间=%s, 预计过期时间=%s",
				token, tokenStartTime.Format("2006-01-02 15:04:05"), tokenStartTime.Add(5*time.Minute).Format("2006-01-02 15:04:05")))
		} else {
			// 未匹配到订单，等待后继续查询
			tokenLifetime := time.Since(tokenStartTime)
			logger.Debug(fmt.Sprintf("未匹配到订单，继续查询: token=%s, 生命周期=%v", token, tokenLifetime))
		}

		// 等待查询间隔
		time.Sleep(queryInterval)
	}
}

// handleMatchedOrder 处理匹配到的订单
func (s *TaskService) handleMatchedOrder(order *platform.PlatformOrder, cfg *model.TaskConfig, channelID int, productID string, platformAccount *model.PlatformAccount, platformInfo *model.Platform) {
	taskOrder := &model.TaskOrder{
		OrderNumber:      order.OrderNumber,
		ChannelID:        channelID,
		ProductID:        productID,
		AccountNum:       order.AccountNum,
		AccountLocation:  order.AccountLocation,
		SettlementAmount: order.SettlementAmount,
		OrderStatus:      order.OrderStatus,
		FaceValue:        order.FaceValue,
		SettlementStatus: 1, // 待结算
		CreateTime:       order.CreateTime.UnixMilli(),
		ExpirationTime:   order.ExpirationTime.UnixMilli(),
	}

	if err := s.taskOrderRepo.Create(taskOrder); err != nil {
		logger.Error(fmt.Sprintf("保存任务订单失败: OrderNumber=%s, error=%v", order.OrderNumber, err))
		return
	}

	// 保存订单到 order 订单表
	productObject, err := s.orderService.GetProductID(order.FaceValue, utils.ISPNameToCode(order.ProductName), 1)
	if err != nil {
		logger.Error(fmt.Sprintf("获取产品id失败: OrderNumber=%s, error=%v", order.OrderNumber, err))
		return
	}

	var customerID int64
	if platformAccount.BindUserID != nil {
		customerID = *platformAccount.BindUserID
	} else {
		// 如果没有绑定用户，跳过订单创建
		logger.Warn(fmt.Sprintf("平台账号未绑定用户，跳过订单创建: PlatformAccountID=%d, OrderNumber=%s", cfg.PlatformAccountID, order.OrderNumber))
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
		logger.Error(fmt.Sprintf("保存订单失败: OrderNumber=%s, error=%v", order.OrderNumber, err))
		return
	}

	logger.Info(fmt.Sprintf("保存任务订单成功: OrderNumber=%s", order.OrderNumber))
}
