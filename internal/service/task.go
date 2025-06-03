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

	maxConcurrent := s.config.MaxConcurrent
	if maxConcurrent <= 0 {
		maxConcurrent = 5 // 默认最大并发数
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
				logger.Error(fmt.Sprintf("获取平台账号信息失败: OrderNumber=%s, error=%+v", err))
				return
			}

			fmt.Printf("userid %d platformAccount++++++++!!!!!!!!%+v", *platformAccount.BindUserID, platformAccount)
			logger.Info(fmt.Sprintf("处理任务配置: ChannelID=%d, ProductID=%s accountName=%s provinces=%s faceValues=%s minSettleAmounts=%s", channelID, productID, accountName, provinces, faceValues, minSettleAmounts))
			token, err := s.platformSvc.GetToken(channelID, productID, "", cfg.FaceValues, cfg.MinSettleAmounts, appkey, accountName, platform.ApiURL, cfg.ID)
			if err != nil {
				logger.Error(fmt.Sprintf("获取 token 失败: ChannelID=%d, ProductID=%s, error=%v", channelID, productID, err))
				return
			}
			logger.Info(fmt.Sprintf("获取 token 成功: ChannelID=%d, ProductID=%s, token=%s", channelID, productID, token))

			order, err := s.platformSvc.QueryTask(token, platform.ApiURL, appkey, accountName)
			if err != nil {
				logger.Error(fmt.Sprintf("查询任务匹配状态失败: token=%s, error=%v", token, err))
				return
			}
			if order == nil {
				logger.Info(fmt.Sprintf("未匹配到订单: token=%s", token))
				return
			}

			logger.Info(fmt.Sprintf("匹配到订单: OrderNumber=%s, AccountNum=%s, SettlementAmount=%.2f",
				order.OrderNumber, order.AccountNum, order.SettlementAmount))

			_ = s.platformSvc.InvalidateToken(cfg.ID)

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

			// 7. 保存订单到 order 订单表
			// productIDInt, _ := strconv.ParseInt(productID, 10, 64)
			// 获取产品id 通过面值price 运营商isp 状态status 获取产品id
			productIDInt, err := s.orderService.GetProductID(order.FaceValue, utils.ISPNameToCode(order.ProductName), 1)
			if err != nil {
				logger.Error(fmt.Sprintf("获取产品id失败: OrderNumber=%s, error=%v", order.OrderNumber, err))
				return
			}

			orderRecord := &model.Order{
				Mobile:            order.AccountNum,
				ProductID:         productIDInt,
				Denom:             order.FaceValue,
				OfficialPayment:   order.SettlementAmount,
				UserQuotePayment:  order.SettlementAmount,
				UserPayment:       order.SettlementAmount,
				Price:             order.SettlementAmount,
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
				CustomerID:        *platformAccount.BindUserID,
				PlatformName:      platform.Name,
				PlatformCode:      platform.Code,
			}

			if err := s.orderService.CreateOrder(s.ctx, orderRecord); err != nil {
				logger.Error(fmt.Sprintf("保存订单失败: OrderNumber=%s, error=%v", order.OrderNumber, err))
				return
			}

			logger.Info(fmt.Sprintf("保存任务订单成功: OrderNumber=%s", order.OrderNumber))
		}(&config)
	}
	wg.Wait()
	// logger.Info("定时任务执行完成")
}
