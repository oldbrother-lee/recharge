package service

import (
	"context"
	"fmt"
	"recharge-go/configs"
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
	config           *configs.TaskConfig
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
	isPullingSuspended bool        // 是否暂停拉单
	suspendMutex       sync.RWMutex // 保护暂停状态的读写锁
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
				logger.Error(fmt.Sprintf("配置监听器启动失败: %v", err))
			}
		}()
		logger.Info("任务配置监听器已启动")
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

	s.ctx, s.cancel = context.WithCancel(context.Background())

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

		logger.Info("启动配置检查定时器，检查间隔: 30秒")

		for {
			select {
			case <-s.ctx.Done():
				return
			case <-configTicker.C:
				logger.Debug("开始定时检查配置变更")
				err := s.ReloadTaskConfig()
				if err != nil {
					logger.Error("定时配置检查失败", err)
				}
			}
		}
	}()
}

// StartOrderDetailsTask 启动订单详情查询任务
func (s *TaskService) StartOrderDetailsTask() {
	logger.Info(fmt.Sprintf("启动订单详情查询任务，执行间隔: %v", s.config.OrderDetailsInterval))
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
		logger.Info(fmt.Sprintf("主动停止任务: TaskID=%d", taskID))
		taskCtx.Cancel() // 触发context取消，会导致processTaskConfig中的defer清理逻辑执行
	} else {
		logger.Debug(fmt.Sprintf("任务不存在或已停止: TaskID=%d", taskID))
	}
}

// ReloadTaskConfig 重新加载任务配置（用于API调用时主动触发热更新）
func (s *TaskService) ReloadTaskConfig() error {
	logger.Debug("开始重新加载任务配置")

	// 获取最新的任务配置
	configs, err := s.taskConfigRepo.GetEnabledConfigs()
	if err != nil {
		logger.Error("重新加载任务配置失败", err)
		return err
	}

	// 获取当前运行的任务数量
	s.taskMutex.RLock()
	runningTaskCount := len(s.taskContexts)
	s.taskMutex.RUnlock()

	logger.Debug(fmt.Sprintf("配置检查: 数据库中启用配置=%d个, 当前运行任务=%d个", len(configs), runningTaskCount))

	// 检查并停止过时的任务
	s.checkAndStopObsoleteTasks(configs)

	// 启动新启用的任务
	s.startNewEnabledTasks(configs)

	// 获取更新后的运行任务数量
	s.taskMutex.RLock()
	newRunningTaskCount := len(s.taskContexts)
	s.taskMutex.RUnlock()

	if newRunningTaskCount != runningTaskCount {
		logger.Info(fmt.Sprintf("任务配置已更新: 启用配置=%d个, 运行任务数量: %d -> %d", len(configs), runningTaskCount, newRunningTaskCount))
	} else {
		logger.Debug(fmt.Sprintf("任务配置无变化: 启用配置=%d个, 运行任务=%d个", len(configs), newRunningTaskCount))
	}

	return nil
}

// processOrderDetails 处理订单详情查询
func (s *TaskService) processOrderDetails() {
	logger.Info("开始执行订单详情查询任务")
	configs, err := s.taskConfigRepo.GetEnabledConfigs()
	if err != nil {
		logger.Error(fmt.Sprintf("获取任务配置失败: %v", err))
		return
	}

	for i, cfg := range configs {
		// 如果不是第一个配置，添加2秒间隔
		if i > 0 {
			time.Sleep(2 * time.Second)
		}
		s.processOrderDetailsForConfig(&cfg)
	}
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
				logger.Debug(fmt.Sprintf("清理无效的任务上下文: TaskID=%d", config.ID))
			default:
				// 上下文仍然有效，任务确实在运行
				logger.Debug(fmt.Sprintf("任务已在运行，跳过: TaskID=%d, ChannelID=%d, ProductID=%s", config.ID, config.ChannelID, config.ProductID))
			}
		}
		s.taskMutex.Unlock()

		if isRunning {
			continue
		}

		logger.Info(fmt.Sprintf("启动新任务: TaskID=%d, ChannelID=%d, ProductID=%s", config.ID, config.ChannelID, config.ProductID))

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
		logger.Info(fmt.Sprintf("正在停止过时任务: TaskID=%d", taskID))
		s.StopTaskByID(taskID)
		// 注意：不需要手动删除任务上下文，StopTaskByID会触发defer清理逻辑
	}
}

// startNewEnabledTasks 启动新启用的任务
func (s *TaskService) startNewEnabledTasks(configs []model.TaskConfig) {
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
				logger.Debug(fmt.Sprintf("清理无效的任务上下文: TaskID=%d", config.ID))
			default:
				// 上下文仍然有效，任务确实在运行
				logger.Debug(fmt.Sprintf("任务已在运行，跳过: TaskID=%d, ChannelID=%d, ProductID=%s", config.ID, config.ChannelID, config.ProductID))
			}
		}
		s.taskMutex.Unlock()

		if isRunning {
			continue
		}

		newTaskCount++
		logger.Info(fmt.Sprintf("启动新任务: TaskID=%d, ChannelID=%d, ProductID=%s", config.ID, config.ChannelID, config.ProductID))

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
		logger.Info(fmt.Sprintf("新任务启动完成，共启动 %d 个新任务", newTaskCount))
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

	logger.Info(fmt.Sprintf("开始处理任务配置: TaskID=%d, PlatformAccountID=%d, ChannelID=%d, ProductID=%s",
		cfg.ID, cfg.PlatformAccountID, cfg.ChannelID, cfg.ProductID))

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
		logger.Warn(fmt.Sprintf("检测到重复任务，取消旧任务: TaskID=%d", taskID))
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

	logger.Info(fmt.Sprintf("任务上下文已注册: TaskID=%d, ChannelID=%d, ProductID=%s", taskID, channelID, productID))

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
			logger.Info(fmt.Sprintf("任务上下文已清理: TaskID=%d, ChannelID=%d, ProductID=%s", taskID, channelID, productID))
		} else {
			logger.Debug(fmt.Sprintf("任务上下文已被其他实例清理或替换: TaskID=%d", taskID))
		}
	}()

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

	token, err := s.platformSvc.GetTokenWithContext(taskCtx, cfg.ID, channelID, productID, "", cfg.FaceValues, cfg.MinSettleAmounts, appkey, accountName, platform.ApiURL)
	if err != nil {
		logger.Error(fmt.Sprintf("申请token失败: ChannelID=%d, ProductID=%s, AccountName=%s, 耗时=%v, error=%v",
			channelID, productID, accountName, time.Since(tokenApplyStartTime), err))
		return
	}

	logger.Info(fmt.Sprintf("申请token成功: ChannelID=%d, ProductID=%s, AccountName=%s, token=%s, 耗时=%v",
		channelID, productID, accountName, token, time.Since(tokenApplyStartTime)))

	// 开始查询循环：基于token创建时间判断5分钟过期，不限制查询次数
	queryInterval := s.config.Interval
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
			logger.Info(fmt.Sprintf("token已过期，重新申请token: token=%s, ChannelID=%d, ProductID=%s, 生命周期=%v, 过期时间=%s AccountName=%s",
				token, channelID, productID, tokenLifetime, time.Now().Format("2006-01-02 15:04:05"), accountName))

			// 重新申请token而不是退出任务
			reapplyStartTime := time.Now()
			newToken, err := s.platformSvc.GetTokenWithContext(taskCtx, cfg.ID, channelID, productID, "", cfg.FaceValues, cfg.MinSettleAmounts, appkey, accountName, platform.ApiURL)
			if err != nil {
				logger.Error(fmt.Sprintf("token过期后重新申请失败: ChannelID=%d, ProductID=%s, AccountName=%s, 耗时=%v, error=%v",
					channelID, productID, accountName, time.Since(reapplyStartTime), err))
				return
			}

			token = newToken
			tokenStartTime = time.Now() // 重置token开始时间
			logger.Info(fmt.Sprintf("token过期后重新申请成功: ChannelID=%d, ProductID=%s, AccountName=%s, newToken=%s, 耗时=%v",
				channelID, productID, accountName, newToken, time.Since(reapplyStartTime)))
			continue
		}

		// 检查订单数量阈值，决定是否暂停拉单
		if err := s.checkOrderThresholds(taskCtx); err != nil {
			logger.Error(fmt.Sprintf("检查订单数量阈值失败: TaskID=%d, error=%v", taskID, err))
		}

		// 如果拉单被暂停，跳过本次查询
		if !s.isPullingAllowed() {
			logger.Debug(fmt.Sprintf("拉单已暂停，跳过查询: TaskID=%d", taskID))
			continue
		}

		// 查询订单
		// apiurl := "http://60.205.159.182:5000/"
		order, err := s.platformSvc.QueryTask(token, platform.ApiURL, appkey, accountName)
		if err != nil {
			tokenLifetime := time.Since(tokenStartTime)
			logger.Error(fmt.Sprintf("查询任务匹配状态失败: token=%s, 生命周期=%v, error=%v", token, tokenLifetime, err))

			// 检查任务是否被取消
			select {
			case <-taskCtx.Done():
				logger.Info(fmt.Sprintf("任务在错误处理中被停止: TaskID=%d", taskID))
				return
			default:
			}

			if strings.Contains(err.Error(), "匹配失败") {
				// 匹配失败，让当前token失效并重新申请token
				tokenLifetime := time.Since(tokenStartTime)
				logger.Info(fmt.Sprintf("主动失效token: token=%s, 原因=匹配失败, 生命周期=%v, 失效时间=%s",
					token, tokenLifetime, time.Now().Format("2006-01-02 15:04:05")))
				_ = s.platformSvc.InvalidateToken(cfg.ID)

				logger.Info(fmt.Sprintf("匹配订单失败重新申请token: ChannelID=%d, ProductID=%s, AccountName=%s", channelID, productID, accountName))
				reapplyStartTime := time.Now()

				// 检查任务是否被取消
				select {
				case <-taskCtx.Done():
					logger.Info(fmt.Sprintf("任务在重新申请token前被停止: TaskID=%d", taskID))
					return
				default:
				}

				newToken, err := s.platformSvc.GetTokenWithContext(taskCtx, cfg.ID, channelID, productID, "", cfg.FaceValues, cfg.MinSettleAmounts, appkey, accountName, platform.ApiURL)
				if err != nil {
					logger.Error(fmt.Sprintf("匹配订单失败重新申请token失败: ChannelID=%d, ProductID=%s, AccountName=%s, 耗时=%v, error=%v",
						channelID, productID, accountName, time.Since(reapplyStartTime), err))
					// 重新申请token失败时等待后重试，而不是直接退出
					logger.Info(fmt.Sprintf("重新申请token失败，等待%v后重试", queryInterval))
					time.Sleep(time.Duration(queryInterval) * time.Second)
					continue
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
			// 其他错误（非匹配失败），记录错误并等待后重试，而不是直接退出任务
			logger.Warn(fmt.Sprintf("查询订单遇到非匹配失败错误，等待%v后重试: error=%v", queryInterval, err))
			time.Sleep(time.Duration(queryInterval) * time.Second)
			continue
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

			// 检查任务是否被取消
			select {
			case <-taskCtx.Done():
				logger.Info(fmt.Sprintf("任务在处理订单后重新申请token前被停止: TaskID=%d", taskID))
				return
			default:
			}

			newToken, err := s.platformSvc.GetTokenWithContext(taskCtx, cfg.ID, channelID, productID, "", cfg.FaceValues, cfg.MinSettleAmounts, appkey, accountName, platform.ApiURL)
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
		time.Sleep(time.Duration(queryInterval) * time.Second)
	}
}

// processOrderDetailsForConfig 为指定配置处理订单详情查询
func (s *TaskService) processOrderDetailsForConfig(cfg *model.TaskConfig) {
	logger.Info(fmt.Sprintf("开始为配置处理订单详情查询: TaskID=%d, ChannelID=%d, ProductID=%s", cfg.ID, cfg.ChannelID, cfg.ProductID))
	// 获取平台账号信息
	platformAccount, err := s.platformAccountRepo.GetByID(cfg.PlatformAccountID)
	if err != nil {
		logger.Error(fmt.Sprintf("获取平台账号失败: PlatformAccountID=%d, error=%v", cfg.PlatformAccountID, err))
		return
	}

	// 获取平台信息
	_, platform, _, err := s.platformSvc.GetAPIKeyAndSecret(cfg.PlatformAccountID)
	if err != nil {
		logger.Error(fmt.Sprintf("获取平台信息失败: %v", err))
		return
	}

	// 查询订单列表 - 获取第一页数据，查询状态为1的订单
	orderList, pageResult, err := s.platformSvc.GetOrderList("", 4, 0, 1, 100, platform.ApiURL, platformAccount) // 查询订单状态为1的订单
	if err != nil {
		logger.Error(fmt.Sprintf("查询订单列表失败: %v", err))
		return
	}

	// 处理查询到的订单
	for _, order := range orderList {
		s.processOrderIfNotExists(&order, cfg, platformAccount, platform)
	}

	// 记录查询结果
	logger.Info(fmt.Sprintf("查询到 %d 条状态为4的订单，总页数: %d", len(orderList), pageResult.Pages))
}

// processOrderIfNotExists 检查订单是否存在，如果不存在则创建
func (s *TaskService) processOrderIfNotExists(order *platform.PlatformOrder, cfg *model.TaskConfig, platformAccount *model.PlatformAccount, platformInfo *model.Platform) {
	// 检查任务订单表中是否已存在
	existingTaskOrder, err := s.taskOrderRepo.GetByOrderNumber(order.OrderNumber)
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
	logger.Info(fmt.Sprintf("开始匹配产品: OrderNumber=%s, FaceValue=%.2f, ProductName=%s", order.OrderNumber, order.FaceValue, order.ProductName))

	// 使用订单面值匹配商品表中name字段包含该数字的产品
	productObject, err := s.orderService.GetProductByNameValue(order.FaceValue, utils.ISPNameToCode(order.ProductName), 1)
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

	// 根据订单状态设置初始状态和备注
	var initialStatus model.OrderStatus
	var remark string
	if order.OrderStatus == 4 {
		// 状态为4的订单直接设置为失败状态
		initialStatus = model.OrderStatusFailed
		remark = "系统检测到第三方平台订单状态为失败，自动创建失败订单"
		logger.Info(fmt.Sprintf("检测到状态为4的订单，将创建为失败状态: OrderNumber=%s", order.OrderNumber))
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
		logger.Error(fmt.Sprintf("保存订单失败: OrderNumber=%s, error=%v", order.OrderNumber, err))
		return
	}

	// 如果是状态为4的订单，创建后需要处理失败逻辑（包括发送通知）
	if order.OrderStatus == 4 {
		if err := s.orderService.ProcessOrderFail(s.ctx, orderRecord.ID, "第三方平台订单状态为失败，自动处理"); err != nil {
			logger.Error(fmt.Sprintf("处理失败订单失败: OrderNumber=%s, OrderID=%d, error=%v", order.OrderNumber, orderRecord.ID, err))
		} else {
			logger.Info(fmt.Sprintf("失败订单处理完成，已发送通知: OrderNumber=%s, OrderID=%d", order.OrderNumber, orderRecord.ID))
		}
	}

	logger.Info(fmt.Sprintf("通过查单创建新订单成功: OrderNumber=%s, Status=%s", order.OrderNumber, initialStatus.String()))
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
	logger.Info(fmt.Sprintf("开始匹配产品: OrderNumber=%s, FaceValue=%.2f, ProductName=%s", order.OrderNumber, order.FaceValue, order.ProductName))

	// 使用订单面值匹配商品表中name字段包含该数字的产品
	productObject, err := s.orderService.GetProductByNameValue(order.FaceValue, utils.ISPNameToCode(order.ProductName), 1)
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

// checkOrderThresholds 检查订单数量阈值并决定是否暂停或恢复拉单
func (s *TaskService) checkOrderThresholds(ctx context.Context) error {
	s.suspendMutex.RLock()
	currentSuspended := s.isPullingSuspended
	s.suspendMutex.RUnlock()

	if currentSuspended {
		// 当前已暂停，检查是否可以恢复
		processingCount, err := s.orderRepo.CountProcessingOrders(ctx)
		if err != nil {
			logger.Error("统计处理中订单数量失败: %v", err)
			return err
		}

		if processingCount < int64(s.orderThresholds.ResumeThreshold) {
			s.resumePulling()
			logger.Info(fmt.Sprintf("处理中订单数量(%d)低于恢复阈值(%d)，恢复拉单", processingCount, s.orderThresholds.ResumeThreshold))
		}
	} else {
		// 当前未暂停，检查是否需要暂停
		rechargeStatuses := []model.OrderStatus{
			model.OrderStatusPendingRecharge, // 待充值 (2)
			model.OrderStatusRecharging,      // 充值中 (3)
		}
		rechargeCount, err := s.orderRepo.CountByStatuses(ctx, rechargeStatuses)
		if err != nil {
			logger.Error("统计充值中和待充值订单数量失败: %v", err)
			return err
		}

		if rechargeCount >= int64(s.orderThresholds.SuspendThreshold) {
			s.suspendPulling()
			logger.Warn(fmt.Sprintf("充值中和待充值订单数量(%d)达到暂停阈值(%d)，暂停拉单", rechargeCount, s.orderThresholds.SuspendThreshold))
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
