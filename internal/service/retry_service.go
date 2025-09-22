package service

import (
	"context"
	"encoding/json"
	"fmt"
	"recharge-go/internal/model"
	"recharge-go/internal/repository"
	"recharge-go/internal/signature"
	"recharge-go/pkg/logger"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"
)

// RetryService 重试服务
type RetryService struct {
	retryRepo              repository.RetryRepository
	orderRepo              repository.OrderRepository
	platformRepo           repository.PlatformRepository
	productRepo            repository.ProductRepository
	productAPIRelationRepo repository.ProductAPIRelationRepository
	submitters             map[string]OrderSubmitter
	rechargeService        RechargeService
	orderService           OrderService
}

// NewRetryService 创建重试服务实例
func NewRetryService(
	retryRepo repository.RetryRepository,
	orderRepo repository.OrderRepository,
	platformRepo repository.PlatformRepository,
	productRepo repository.ProductRepository,
	productAPIRelationRepo repository.ProductAPIRelationRepository,
	rechargeService RechargeService,
	orderService OrderService,
) *RetryService {
	// 创建签名处理器
	kekebangConfig := &signature.Config{
		AppID:     "your_app_id",
		AppSecret: "your_app_secret",
	}
	kekebangHandler := signature.NewKekebangHandler(kekebangConfig)

	// 创建订单提交器
	submitters := map[string]OrderSubmitter{
		"kekebang": NewKekebangSubmitter(kekebangHandler),
		// 添加其他平台的提交器...
	}

	return &RetryService{
		retryRepo:              retryRepo,
		orderRepo:              orderRepo,
		platformRepo:           platformRepo,
		productRepo:            productRepo,
		productAPIRelationRepo: productAPIRelationRepo,
		submitters:             submitters,
		rechargeService:        rechargeService,
		orderService:           orderService,
	}
}

// HandleRetry 处理重试
func (s *RetryService) HandleRetry(ctx context.Context, order *model.Order, retryType int) error {
	// 将订单号注入上下文，便于全链路日志携带
	ctx = logger.InjectOrderNumber(ctx, order.OrderNumber)
	// 入口诊断日志（串联订单号）
	logger.WithContext(ctx).Info(fmt.Sprintf("【重试入口】order_id: %d, order_number: %s, product_id: %d, retry_type: %d", order.ID, order.OrderNumber, order.ProductID, retryType))

	// 优先判断是否应走同通道重试（即便上游传入的是跨通道重试类型2）
	// 条件：商品仅绑定1个启用通道，且该通道开启了同通道重试
	{
		relations, _, err := s.productAPIRelationRepo.List(ctx, order.ProductID, 0, 1, 1, 100)
		if err == nil && len(relations) > 0 {
			enabled := 0
			for _, r := range relations {
				if r.Status == 1 {
					enabled++
				}
			}
			// 关系统计日志
			logger.Info("【同通道判断·关系统计】order_id: %d, order_number: %s, product_id: %d, enabled_count: %d, relations_total: %d", order.ID, order.OrderNumber, order.ProductID, enabled, len(relations))

			curAPI := order.APICurID
			if curAPI == 0 {
				curAPI = order.APIID
			}
			logger.Info("【同通道判断·当前通道】order_id: %d, order_number: %s, cur_api_id: %d", order.ID, order.OrderNumber, curAPI)

			var curRel *model.ProductAPIRelation
			for _, r := range relations {
				if r.APIID == curAPI {
					curRel = r
					break
				}
			}
			if enabled == 1 && curRel != nil && curRel.SameChannelRetryEnabled {
				// 当前关系命中同通道条件
				logger.Info("【同通道优先命中】order_id: %d, order_number: %s, relation_id: %d, api_id: %d, relation_status: %d, same_channel_enabled: %t, same_channel_times: %d", order.ID, order.OrderNumber, curRel.ID, curRel.APIID, curRel.Status, curRel.SameChannelRetryEnabled, curRel.SameChannelRetryTimes)
				// 记录API状态（仅日志）
				if api, err2 := s.platformRepo.GetAPIByID(ctx, curRel.APIID); err2 == nil && api != nil {
					logger.Info("【同通道判断·API状态】order_id: %d, order_number: %s, api_id: %d, api_status: %d", order.ID, order.OrderNumber, api.ID, api.Status)
				}
				return s.handleSameChannelRetry(ctx, order)
			}
		}
	}

	// 根据重试类型执行不同的逻辑
	if retryType == model.RetryTypeSameChannel {
		// 同通道重试：只使用当前失败的通道
		return s.handleSameChannelRetry(ctx, order)
	}

	// 跨通道重试：获取所有可用的API关系列表
	relations, err := s.GetAvailableAPIRelations(ctx, order.ID, order.ProductID)
	if err != nil {
		return fmt.Errorf("获取可用API失败: %v", err)
	}

	if len(relations) == 0 {
		return fmt.Errorf("没有可用的API进行重试")
	}

	// 2. 创建重试记录
	for _, relation := range relations {
		// 获取已使用的API列表
		records, err := s.retryRepo.GetByOrderID(ctx, order.ID)
		if err != nil {
			return fmt.Errorf("获取已使用API失败: %v", err)
		}

		// 收集已使用的API ID
		usedAPIs := make([]int64, 0)
		for _, record := range records {
			// 尝试解析为对象数组格式
			var usedAPIList []struct {
				APIID   int64 `json:"api_id"`
				ParamID int64 `json:"param_id,omitempty"`
			}
			if err := json.Unmarshal([]byte(record.UsedAPIs), &usedAPIList); err != nil {
				// 如果解析失败，尝试解析为简单数字数组格式
				var simpleAPIList []int64
				if err2 := json.Unmarshal([]byte(record.UsedAPIs), &simpleAPIList); err2 != nil {
					return fmt.Errorf("解析已使用API失败: %v", err)
				}
				usedAPIs = append(usedAPIs, simpleAPIList...)
			} else {
				for _, u := range usedAPIList {
					usedAPIs = append(usedAPIs, u.APIID)
				}
			}
		}

		// 添加当前API到已使用列表，使用对象格式
		usedAPIs = append(usedAPIs, relation.APIID)
		usedAPIObjects := make([]struct {
			APIID   int64 `json:"api_id"`
			ParamID int64 `json:"param_id,omitempty"`
		}, len(usedAPIs))
		for i, apiID := range usedAPIs {
			usedAPIObjects[i] = struct {
				APIID   int64 `json:"api_id"`
				ParamID int64 `json:"param_id,omitempty"`
			}{APIID: apiID}
		}
		usedAPIsJSON, err := json.Marshal(usedAPIObjects)
		if err != nil {
			return fmt.Errorf("序列化已使用API失败: %v", err)
		}

		// 设置重试时间：如果是第一次重试（RetryCount为0），立即执行；否则设置3秒后重试
		retryCount := len(records)
		nextRetryTime := time.Now()
		if retryCount > 0 {
			// 实现秒级充值：缩短重试间隔到3秒
			nextRetryTime = time.Now().Add(3 * time.Second)
		}

		retryRecord := &model.OrderRetryRecord{
			OrderID:       order.ID,
			APIID:         relation.APIID,
			ParamID:       relation.ParamID,
			RetryType:     retryType,
			Status:        0, // 待重试
			NextRetryTime: nextRetryTime,
			RetryParams:   "{}", // 设置为空JSON对象
			UsedAPIs:      string(usedAPIsJSON),
			RetryCount:    retryCount, // 设置重试次数为已存在的记录数
		}

		if err := s.retryRepo.Create(ctx, retryRecord); err != nil {
			logger.Log.Error("创建重试记录失败",
				zap.Int64("order_id", order.ID),
				zap.Int64("api_id", relation.APIID),
				zap.Error(err),
			)
			continue
		}

		// 如果是第一次重试（RetryCount为0），立即执行
		if retryRecord.RetryCount == 0 {
			logger.Info("【首次重试】立即执行重试 record_id: %d, order_id: %d", retryRecord.ID, order.ID)
			if err := s.executeRetry(ctx, retryRecord); err != nil {
				// 更新重试记录状态为失败
				retryRecord.Status = 3 // 重试失败
				retryRecord.LastError = err.Error()
				if err := s.retryRepo.Update(ctx, retryRecord); err != nil {
					logger.Error("【更新重试记录状态失败】record_id: %d, order_id: %d, error: %v",
						retryRecord.ID, retryRecord.OrderID, err)
				}
				logger.Error("【首次重试失败】record_id: %d, order_id: %d, error: %v, 继续尝试其他通道",
					retryRecord.ID, order.ID, err)
				// 不要立即返回错误，继续尝试其他通道
			} else {
				// 更新重试记录状态为成功
				retryRecord.Status = 2 // 重试成功
				if err := s.retryRepo.Update(ctx, retryRecord); err != nil {
					logger.Error("【更新重试记录状态失败】record_id: %d, order_id: %d, error: %v",
						retryRecord.ID, retryRecord.OrderID, err)
				}
				logger.Info("【首次重试成功】record_id: %d, order_id: %d", retryRecord.ID, order.ID)
				// 如果重试成功，可以直接返回
				return nil
			}
		}
	}

	// 检查当前订单是否所有重试通道都已完成
	orderRetries, err := s.retryRepo.GetByOrderID(ctx, order.ID)
	if err != nil {
		logger.Error("【获取订单重试记录失败】order_id: %d, error: %v", order.ID, err)
		return nil // 不影响主流程
	}

	// 检查是否所有重试都已完成（成功或失败）
	allCompleted := true
	hasSuccess := false
	for _, r := range orderRetries {
		if r.Status == 0 || r.Status == 1 { // 0: 待处理, 1: 处理中
			allCompleted = false
			break
		} else if r.Status == 2 { // 2: 重试成功
			hasSuccess = true
		}
	}

	// 如果所有重试都已完成且没有成功的，将订单标记为失败
	if allCompleted && !hasSuccess {
		logger.Info("【所有重试通道均已失败，更新订单状态为失败】order_id: %d", order.ID)
		if err := s.orderService.ProcessOrderFail(ctx, order.ID, "所有通道重试失败，自动失败"); err != nil {
			logger.Error("【订单失败处理失败】order_id: %d, error: %v", order.ID, err)
			// 如果是获取锁失败，创建一个延迟重试任务
			if strings.Contains(err.Error(), "获取分布式锁超时") || strings.Contains(err.Error(), "获取退款锁失败") {
				logger.Info("【因锁获取失败，创建延迟重试任务】order_id: %d", order.ID)
				retryRecord := &model.OrderRetryRecord{
					OrderID:       order.ID,
					RetryType:     model.RetryTypeOrderFail,
					Status:        0, // 待处理
					RetryCount:    0,
					NextRetryTime: time.Now().Add(30 * time.Second), // 30秒后重试
				}
				if createErr := s.retryRepo.Create(ctx, retryRecord); createErr != nil {
					logger.Error("【创建延迟重试任务失败】order_id: %d, error: %v", order.ID, createErr)
				}
			}
		} else {
			logger.Info("【订单状态已更新为失败并已发送通知】order_id: %d", order.ID)
		}
	}

	return nil
}

// ProcessRetries 处理待重试的记录
func (s *RetryService) ProcessRetries(ctx context.Context) error {
	logger.Info("【开始处理待重试记录】")

	// 1. 获取待重试的记录
	records, err := s.retryRepo.GetPendingRetries(ctx)
	if err != nil {
		logger.Error("【获取待重试记录失败】error: %v", err)
		return fmt.Errorf("获取待重试记录失败: %v", err)
	}

	if len(records) == 0 {
		logger.Info("【没有待重试的记录】")
		return nil
	}

	logger.Info("【获取到待重试记录】数量: %d", len(records))

	// 2. 处理每条重试记录
	for _, record := range records {
		// 检查重试时间是否到达
		if time.Now().Before(record.NextRetryTime) {
			logger.Info(fmt.Sprintf("【重试时间未到】record_id: %d, order_id: %d, next_retry_time: %v, current_time: %v",
				record.ID, record.OrderID, record.NextRetryTime, time.Now()))
			continue
		}

		// 更新重试记录状态为处理中
		record.Status = 1 // 处理中
		if err := s.retryRepo.Update(ctx, record); err != nil {
			logger.Error(fmt.Sprintf("【更新重试记录状态失败】record_id: %d, order_id: %d, error: %v",
				record.ID, record.OrderID, err))
			continue
		}

		logger.Info(fmt.Sprintf("【开始执行重试】record_id: %d, order_id: %d, retry_count: %d",
			record.ID, record.OrderID, record.RetryCount))

		// 执行重试
		if err := s.executeRetry(ctx, record); err != nil {
			logger.Error(fmt.Sprintf("【重试执行失败】record_id: %d, order_id: %d, error: %v",
				record.ID, record.OrderID, err))

			// 更新重试记录状态为失败
			record.Status = 3 // 重试失败
			record.LastError = err.Error()
			if err := s.retryRepo.Update(ctx, record); err != nil {
				logger.Error(fmt.Sprintf("【更新重试记录状态失败】record_id: %d, order_id: %d, error: %v",
					record.ID, record.OrderID, err))
			}

			// 检查当前订单是否所有重试都失败
			orderRetries, err := s.retryRepo.GetByOrderID(ctx, record.OrderID)
			if err == nil {
				allCompleted := true
				for _, r := range orderRetries {
					if r.Status == 0 || r.Status == 1 { // 0: 待处理, 1: 处理中
						allCompleted = false
						break
					}
				}
				if allCompleted {
					logger.Info(fmt.Sprintf("【所有重试均已完成，更新订单状态为失败并发送通知】order_id: %d", record.OrderID))
					if err := s.orderService.ProcessOrderFail(ctx, record.OrderID, "所有通道重试失败，自动失败"); err != nil {
						logger.Error("【订单失败处理失败】order_id: %d, error: %v", record.OrderID, err)
					} else {
						logger.Info("【订单状态已更新为失败并已发送通知】order_id: %d", record.OrderID)
					}
				}
			}

			continue
		}

		// 更新重试记录状态为成功
		record.Status = 2 // 重试成功
		if err := s.retryRepo.Update(ctx, record); err != nil {
			logger.Error(fmt.Sprintf("【更新重试记录状态失败】record_id: %d, order_id: %d, error: %v",
				record.ID, record.OrderID, err))
			continue
		}

		logger.Info(fmt.Sprintf("【重试执行成功】record_id: %d, order_id: %d", record.ID, record.OrderID))
	}

	logger.Info("【所有重试记录处理完成】")

	// 检查是否有订单需要更新为失败状态
	records, err = s.retryRepo.GetByOrderID(ctx, 0) // 获取所有重试记录
	if err != nil {
		logger.Error("【获取重试记录失败】error: %v", err)
		return fmt.Errorf("get retry records failed: %v", err)
	}

	// 按订单ID分组统计
	orderRetries := make(map[int64][]*model.OrderRetryRecord)
	for _, record := range records {
		orderRetries[record.OrderID] = append(orderRetries[record.OrderID], record)
	}

	// 检查每个订单的重试情况
	for orderID, retries := range orderRetries {
		// 获取订单信息
		order, err := s.orderRepo.GetByID(ctx, orderID)
		if err != nil {
			logger.Error("【获取订单信息失败】order_id: %d, error: %v", orderID, err)
			continue
		}

		// 如果订单已经是成功或失败状态，跳过
		if order.Status == model.OrderStatusSuccess || order.Status == model.OrderStatusFailed {
			continue
		}

		// 检查是否所有重试都失败了
		allFailed := true
		for _, retry := range retries {
			if retry.Status != 3 { // 3 表示重试失败
				allFailed = false
				break
			}
		}

		// 如果所有重试都失败了，更新订单状态为失败
		if allFailed {
			logger.Info("【所有平台重试均失败，更新订单状态为失败】order_id: %d", orderID)
			if err := s.orderService.ProcessOrderFail(ctx, orderID, "所有平台重试失败，自动失败"); err != nil {
				logger.Error("【订单失败处理失败】order_id: %d, error: %v", orderID, err)
			} else {
				logger.Info("【订单状态已更新为失败】order_id: %d", orderID)
			}
		}
	}

	return nil
}

// executeRetry 执行重试
func (s *RetryService) executeRetry(ctx context.Context, record *model.OrderRetryRecord) error {
	logger.Info(fmt.Sprintf("【开始执行重试】record_id: %d, order_id: %d, retry_type: %s", record.ID, record.OrderID, record.RetryType))

	// 检查重试类型，如果是订单失败重试，直接调用ProcessOrderFail
	if record.RetryType == model.RetryTypeOrderFail {
		logger.Info(fmt.Sprintf("【执行订单失败重试】record_id: %d, order_id: %d", record.ID, record.OrderID))
		return s.orderService.ProcessOrderFail(ctx, record.OrderID, "重试处理订单失败")
	}

	// 1. 获取订单信息
	order, err := s.orderRepo.GetByID(ctx, record.OrderID)
	if err != nil {
		logger.Error(fmt.Sprintf("【获取订单信息失败】record_id: %d, order_id: %d, error: %v",
			record.ID, record.OrderID, err))
		return fmt.Errorf("获取订单信息失败: %v", err)
	}
	// 注入订单号到上下文，便于全链路日志携带
	ctx = logger.InjectOrderNumber(ctx, order.OrderNumber)
	logger.WithContext(ctx).Info(fmt.Sprintf("【获取订单信息成功】record_id: %d, order_id: %d, status: %d, order_number: %s",
		record.ID, record.OrderID, order.Status, order.OrderNumber))
	fmt.Println(order, "order+@@@@@@!!!!!!!!!!!!!!!!!!1+++++++")
	// 2. 获取可用的API关系列表
	relations, err := s.GetAvailableAPIRelations(ctx, record.OrderID, order.ProductID)
	if err != nil {
		logger.Error(fmt.Sprintf("【获取可用API关系失败】record_id: %d, order_id: %d, error: %v",
			record.ID, record.OrderID, err))
		return fmt.Errorf("获取可用API关系失败: %v", err)
	}

	if len(relations) == 0 {
		logger.Error(fmt.Sprintf("【没有可用的API关系】record_id: %d, order_id: %d", record.ID, record.OrderID))
		return fmt.Errorf("没有可用的API关系")
	}

	logger.WithContext(ctx).Info(fmt.Sprintf("【获取到可用API关系】record_id: %d, order_id: %d, count: %d",
		record.ID, record.OrderID, len(relations)))

	// 3. 选择第一个可用的API关系
	relation := relations[0]
	logger.WithContext(ctx).Info(fmt.Sprintf("【选择API关系】record_id: %d, order_id: %d, api_id: %d, param_id: %d",
		record.ID, record.OrderID, relation.APIID, relation.ParamID))

	// 4. 获取API信息
	api, err := s.platformRepo.GetAPIByID(ctx, relation.APIID)
	if err != nil {
		logger.Error(fmt.Sprintf("【获取API信息失败】record_id: %d, order_id: %d, error: %v",
			record.ID, record.OrderID, err))
		return fmt.Errorf("获取API信息失败: %v", err)
	}
	logger.WithContext(ctx).Info(fmt.Sprintf("【获取API信息成功】record_id: %d, order_id: %d, api_id: %d, api_name: %s",
		record.ID, record.OrderID, api.ID, api.Name))

	// 5. 获取API参数
	param, err := s.platformRepo.GetAPIParamByID(ctx, relation.ParamID)
	if err != nil {
		logger.Error("\t【获取API参数失败】record_id: %d, order_id: %d, error: %v",
			record.ID, record.OrderID, err)
		return fmt.Errorf("获取API参数失败: %v", err)
	}
	logger.WithContext(ctx).Info(fmt.Sprintf("【获取API参数成功】record_id: %d, order_id: %d, param_id: %d",
		record.ID, record.OrderID, param.ID))

	// 6. 更新重试记录中的API信息
	record.APIID = relation.APIID
	record.ParamID = relation.ParamID
	if err := s.retryRepo.Update(ctx, record); err != nil {
		logger.Error("\t【更新重试记录API信息失败】record_id: %d, order_id: %d, error: %v",
			record.ID, record.OrderID, err)
	}
	logger.WithContext(ctx).Info(fmt.Sprintf("【更新重试记录API信息成功】record_id: %d, order_id: %d, api_id: %d, param_id: %d",
		record.ID, record.OrderID, record.APIID, record.ParamID))

	// 【同通道重试】如果所选关系与当前通道一致且开启开关，则为本次尝试生成新的 ActiveOutTradeNum
	currentAPIID := order.APICurID
	if currentAPIID == 0 {
		currentAPIID = order.APIID
	}

	// 6. 【同通道重试逻辑】检查是否为同通道重试
	retryAttempt := record.AttemptNo + 1 // 提前定义，供后续事务中使用
	if relation.APIID == currentAPIID && relation.SameChannelRetryEnabled {
		// 检查是否超过最大同通道重试次数
		if retryAttempt > relation.SameChannelRetryTimes {
			logger.Info("【同通道重试】已达到最大重试次数，调用失败处理逻辑",
				"order_id", order.ID,
				"record_id", record.ID,
				"attempt_no", retryAttempt,
				"max_retry_times", relation.SameChannelRetryTimes,
			)
			
			// 调用订单失败处理逻辑（包括退款、发送通知等）
			if err := s.orderService.ProcessOrderFail(ctx, record.OrderID, "同通道重试达到最大次数"); err != nil {
				logger.Error("【订单失败处理失败】record_id: %d, order_id: %d, error: %v",
					record.ID, record.OrderID, err)
				return fmt.Errorf("订单失败处理失败: %v", err)
			}
			
			// 更新重试记录状态为失败
			record.Status = 3 // 重试失败
			record.LastError = fmt.Sprintf("同通道重试已达到最大次数(%d)，订单已失败", relation.SameChannelRetryTimes)
			if err := s.retryRepo.Update(ctx, record); err != nil {
				logger.Error("【更新重试记录状态失败】record_id: %d, order_id: %d, error: %v", record.ID, record.OrderID, err)
			}
			return fmt.Errorf("同通道重试已达到最大次数(%d)，订单已失败", relation.SameChannelRetryTimes)
		} else {
			// 生成基于重试次数的后缀，格式更简洁：原订单号-r重试次数
			suffix := fmt.Sprintf("-r%d", retryAttempt)
			base := order.OrderNumber
			maxBaseLen := 64 - len(suffix)
			if maxBaseLen < 1 {
				// 保护性处理，至少保留1个字符
				maxBaseLen = 1
			}
			if len(base) > maxBaseLen {
				// 截断基础部分，避免超出长度限制
				orig := base
				base = base[:maxBaseLen]
				logger.Info("【同通道重试】原始订单号过长，已截断以适配长度限制",
					"order_id", order.ID,
					"record_id", record.ID,
					"order_number", orig,
					"max_base_len", maxBaseLen,
				)
			}
			newActiveOutTradeNum := base + suffix

			// 【重要】不修改订单表的OutTradeNum，而是将新订单号存储在重试记录的ActiveOutTradeNum中
			record.ActiveOutTradeNum = newActiveOutTradeNum
			record.RetryType = 3 // RetryTypeSameChannel
			record.AttemptNo = retryAttempt
			now := time.Now()
			record.LastAttemptAt = &now
			
			if err := s.retryRepo.Update(ctx, record); err != nil {
				logger.Error("【更新重试记录ActiveOutTradeNum失败】record_id: %d, order_id: %d, error: %v", record.ID, record.OrderID, err)
				return fmt.Errorf("更新重试记录ActiveOutTradeNum失败: %v", err)
			}
			
			logger.WithContext(ctx).Info("【同通道重试】已生成新的ActiveOutTradeNum",
				logger.Int64("record_id", record.ID),
				logger.Int64("order_id", order.ID),
				logger.String("active_out_trade_num", newActiveOutTradeNum),
				logger.String("order_number", order.OrderNumber),
				logger.String("out_trade_num", order.OutTradeNum), // 保持不变
				logger.Int("attempt_no", retryAttempt),
				logger.String("retry_suffix", suffix),
			)
		}
	}

	// 7. 开启事务
	tx := s.orderRepo.(*repository.OrderRepositoryImpl).DB().Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			logger.Error("【事务回滚】record_id: %d, order_id: %d, panic: %v",
				record.ID, record.OrderID, r)
		}
	}()

	// 8. 调用 RechargeService 的 SubmitOrder 方法
	// 【同通道重试】如果有ActiveOutTradeNum，临时替换order.OrderNumber用于提交
	originalOrderNumber := order.OrderNumber
	if record.ActiveOutTradeNum != "" {
		order.OrderNumber = record.ActiveOutTradeNum
		logger.WithContext(ctx).Info("【同通道重试】临时使用ActiveOutTradeNum作为OrderNumber提交订单",
			logger.Int64("record_id", record.ID),
			logger.Int64("order_id", record.OrderID),
			logger.String("original_order_number", originalOrderNumber),
			logger.String("active_out_trade_num", record.ActiveOutTradeNum),
		)
	}
	
	logger.WithContext(ctx).Info(fmt.Sprintf("【开始提交订单】record_id: %d, order_id: %d, order_number: %s",
		record.ID, record.OrderID, order.OrderNumber))
	submitErr := s.rechargeService.SubmitOrder(ctx, order, api, param)
	
	// 【重要】提交完成后立即恢复原始OrderNumber
	order.OrderNumber = originalOrderNumber
	
	if submitErr != nil {
		tx.Rollback()
		logger.Error("【提交订单失败】record_id: %d, order_id: %d, error: %v",
			record.ID, record.OrderID, submitErr)
		return fmt.Errorf("提交订单失败: %v", submitErr)
	}
	fmt.Println("-----------")
	logger.WithContext(ctx).Info(fmt.Sprintf("【提交订单成功】record_id: %d, order_id: %d, order_number: %s",
		record.ID, record.OrderID, order.OrderNumber))

	// 9. 更新订单状态
	logger.WithContext(ctx).Info(fmt.Sprintf("【开始更新订单状态】record_id: %d, order_id: %d, order_number: %s, old_status: %d, new_status: %d",
		record.ID, record.OrderID, order.OrderNumber, order.Status, model.OrderStatusRecharging))
	result := tx.Model(&model.Order{}).Where("id = ?", record.OrderID).Update("status", model.OrderStatusRecharging)
	if result.Error != nil {
		tx.Rollback()
		logger.Error("【更新订单状态失败】record_id: %d, order_id: %d, error: %v",
			record.ID, record.OrderID, result.Error)
		return fmt.Errorf("更新订单状态失败: %v", result.Error)
	}
	if result.RowsAffected == 0 {
		tx.Rollback()
		logger.Error("【更新订单状态失败】record_id: %d, order_id: %d, 没有记录被更新",
			record.ID, record.OrderID)
		return fmt.Errorf("没有记录被更新")
	}

	// 10. 更新重试记录的AttemptNo字段
	logger.WithContext(ctx).Info(fmt.Sprintf("【更新重试记录AttemptNo】record_id: %d, order_id: %d, attempt_no: %d -> %d",
		record.ID, record.OrderID, record.AttemptNo, retryAttempt))
	retryUpdateResult := tx.Model(&model.OrderRetryRecord{}).Where("id = ?", record.ID).Update("attempt_no", retryAttempt)
	if retryUpdateResult.Error != nil {
		tx.Rollback()
		logger.Error("【更新重试记录AttemptNo失败】record_id: %d, order_id: %d, error: %v",
			record.ID, record.OrderID, retryUpdateResult.Error)
		return fmt.Errorf("更新重试记录AttemptNo失败: %v", retryUpdateResult.Error)
	}

	// 11. 提交事务
	if err := tx.Commit().Error; err != nil {
		logger.Error("【提交事务失败】record_id: %d, order_id: %d, error: %v",
			record.ID, record.OrderID, err)
		return fmt.Errorf("提交事务失败: %v", err)
	}

	logger.Info("【订单状态更新成功】record_id: %d, order_id: %d, order_number: %s",
		record.ID, record.OrderID, order.OrderNumber)

	// 11. 重试成功，取消同一订单的其他待重试记录
	if err := s.cancelPendingRetries(ctx, record.OrderID, record.ID); err != nil {
		logger.Error("【取消其他待重试记录失败】record_id: %d, order_id: %d, error: %v",
			record.ID, record.OrderID, err)
		// 不返回错误，因为主要任务已完成
	}

	return nil
}

// getAvailableAPIRelations 获取可用的API关系列表
func (s *RetryService) GetAvailableAPIRelations(ctx context.Context, orderID int64, productID int64) ([]*model.ProductAPIRelation, error) {
	logger.Info("开始获取可用的API关系列表",
		"order_id", orderID,
		"product_id", productID,
	)

	// 1. 获取已使用的API列表
	records, err := s.retryRepo.GetByOrderID(ctx, orderID)
	if err != nil {
		logger.Error("获取已使用API失败",
			"error", err,
			"order_id", orderID,
		)
		return nil, fmt.Errorf("获取已使用API失败: %v", err)
	}

	logger.Info("DEBUG: 获取到的重试记录",
		"order_id", orderID,
		"records_count", len(records),
	)

	// 收集已使用的API ID（统计所有已尝试过的API，无论成功还是失败）
	usedAPIs := make([]int64, 0)
	for _, record := range records {
		// 统计所有已尝试过的API，避免重复使用失败的API
		// 状态说明：0: 待处理, 1: 处理中, 2: 重试成功, 3: 重试失败/已取消
		
		logger.Info("DEBUG: 处理重试记录",
			"record_id", record.ID,
			"order_id", record.OrderID,
			"api_id", record.APIID,
			"status", record.Status,
			"used_apis_raw", record.UsedAPIs,
		)
		
		// 解析 UsedAPIs 字段，兼容两种格式
		var usedAPIList []struct {
			APIID   int64 `json:"api_id"`
			ParamID int64 `json:"param_id,omitempty"`
		}
		if err := json.Unmarshal([]byte(record.UsedAPIs), &usedAPIList); err != nil {
			// 如果解析对象数组失败，尝试解析为简单数字数组
			var simpleAPIList []int64
			if err2 := json.Unmarshal([]byte(record.UsedAPIs), &simpleAPIList); err2 != nil {
				logger.Error("解析已使用API失败",
					"error", err,
					"record_id", record.ID,
				)
				return nil, fmt.Errorf("解析已使用API失败: %v", err)
			}
			logger.Info("DEBUG: 解析为简单数组", "simple_apis", simpleAPIList)
			usedAPIs = append(usedAPIs, simpleAPIList...)
		} else {
			logger.Info("DEBUG: 解析为对象数组", "object_apis", usedAPIList)
			for _, u := range usedAPIList {
				usedAPIs = append(usedAPIs, u.APIID)
			}
		}
	}

	logger.Info("已使用的API列表",
		"order_id", orderID,
		"used_apis", usedAPIs,
	)

	// 2. 获取可用的API关系列表
	relations, _, err := s.productAPIRelationRepo.List(ctx, productID, 0, 1, 1, 100)
	if err != nil {
		logger.Error("获取API关系列表失败",
			"error", err,
			"product_id", productID,
		)
		return nil, fmt.Errorf("获取API关系列表失败: %v", err)
	}

	// 【同通道重试优先】只有在绑定单个通道且同通道重试开关为真时才走同通道重试
	if order, err2 := s.orderRepo.GetByID(ctx, orderID); err2 == nil && order != nil {
		// 统计启用状态的通道数量
		enabledChannelCount := 0
		for _, rel := range relations {
			if rel.Status == 1 {
				enabledChannelCount++
			}
		}
		
		// 只有绑定单个通道时才考虑同通道重试
		if enabledChannelCount == 1 {
			curAPI := order.APICurID
			if curAPI == 0 {
				curAPI = order.APIID
			}
			var curRel *model.ProductAPIRelation
			for _, rel := range relations {
				if rel.APIID == curAPI {
					curRel = rel
					break
				}
			}
			if curRel != nil && curRel.Status == 1 && curRel.SameChannelRetryEnabled {
				if api, err3 := s.platformRepo.GetAPIByID(ctx, curRel.APIID); err3 == nil && api.Status == 1 {
					logger.Info("单通道绑定且同通道重试开启，优先返回当前通道",
						"order_id", orderID,
						"api_id", curRel.APIID,
						"param_id", curRel.ParamID,
						"enabled_channel_count", enabledChannelCount,
					)
					return []*model.ProductAPIRelation{curRel}, nil
				}
			}
		} else {
			logger.Info("多通道绑定，跳过同通道重试，走多通道重试逻辑",
				"order_id", orderID,
				"enabled_channel_count", enabledChannelCount,
			)
		}
	}

	// 3. 过滤和排序可用的API
	availableRelations := make([]*model.ProductAPIRelation, 0)
	for _, relation := range relations {
		// 3.1 检查ProductAPIRelation状态
		if relation.Status != 1 { // 1 表示启用
			logger.Info("ProductAPIRelation未启用，跳过",
				"relation_id", relation.ID,
				"api_id", relation.APIID,
				"status", relation.Status,
			)
			continue
		}

		// 3.2 检查API是否已使用
		isUsed := false
		for _, usedAPI := range usedAPIs {
			if relation.APIID == usedAPI {
				isUsed = true
				break
			}
		}
		if isUsed {
			continue
		}

		// 3.3 获取API信息
		api, err := s.platformRepo.GetAPIByID(ctx, relation.APIID)
		if err != nil {
			logger.Error("获取API信息失败",
				"error", err,
				"api_id", relation.APIID,
			)
			continue
		}

		// 3.4 检查API状态
		if api.Status != 1 { // 1 表示启用
			logger.Info("API未启用，跳过",
				"api_id", relation.APIID,
				"status", api.Status,
			)
			continue
		}

		// 3.5 添加到可用列表
		availableRelations = append(availableRelations, relation)
	}

	// 4. 按排序字段排序
	sort.Slice(availableRelations, func(i, j int) bool {
		return availableRelations[i].Sort < availableRelations[j].Sort
	})

	logger.Info("获取到可用的API关系列表",
		"order_id", orderID,
		"product_id", productID,
		"total", len(relations),
		"available", len(availableRelations),
	)

	return availableRelations, nil
}

// cancelPendingRetries 取消同一订单的其他待重试记录
func (s *RetryService) cancelPendingRetries(ctx context.Context, orderID int64, excludeRecordID int64) error {
	logger.Info("【开始取消其他待重试记录】order_id: %d, exclude_record_id: %d", orderID, excludeRecordID)

	// 获取同一订单的所有待重试记录
	records, err := s.retryRepo.GetByOrderID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("获取订单重试记录失败: %v", err)
	}

	// 取消除当前记录外的其他待重试记录
	for _, record := range records {
		if record.ID != excludeRecordID && record.Status == 0 { // 0: 待处理
			record.Status = 3 // 3: 已取消
			if err := s.retryRepo.Update(ctx, record); err != nil {
				logger.Error("【取消重试记录失败】record_id: %d, order_id: %d, error: %v",
					record.ID, orderID, err)
				continue
			}
			logger.Info("【取消重试记录成功】record_id: %d, order_id: %d", record.ID, orderID)
		}
	}

	return nil
}

// CreateRetryRecord 创建重试记录
func (s *RetryService) CreateRetryRecord(ctx context.Context, record *model.OrderRetryRecord) error {
	return s.retryRepo.Create(ctx, record)
}

// checkAllRetriesCompleted 检查订单的所有重试是否已完成
func (s *RetryService) checkAllRetriesCompleted(ctx context.Context, orderID int64) (bool, error) {
	// 获取订单的所有重试记录
	records, err := s.retryRepo.GetByOrderID(ctx, orderID)
	if err != nil {
		return false, fmt.Errorf("获取订单重试记录失败: %v", err)
	}

	// 检查是否还有待处理或处理中的记录
	for _, record := range records {
		if record.Status == 0 || record.Status == 1 { // 0: 待处理, 1: 处理中
			return false, nil
		}
	}

	return true, nil
}

// TriggerRetry 手动触发重试
func (s *RetryService) TriggerRetry(ctx context.Context, recordID int64) error {
	// 1. 获取重试记录
	record, err := s.retryRepo.GetByID(ctx, recordID)
	if err != nil {
		return fmt.Errorf("获取重试记录失败: %v", err)
	}

	// 2. 更新重试时间为当前时间
	record.NextRetryTime = time.Now()
	if err := s.retryRepo.Update(ctx, record); err != nil {
		return fmt.Errorf("更新重试时间失败: %v", err)
	}

	// 3. 执行重试
	return s.executeRetry(ctx, record)
}

// handleSameChannelRetry 处理同通道重试
func (s *RetryService) handleSameChannelRetry(ctx context.Context, order *model.Order) error {
	// 注入订单号到上下文
	ctx = logger.InjectOrderNumber(ctx, order.OrderNumber)
	logger.WithContext(ctx).Info(fmt.Sprintf("【开始处理同通道重试】order_id: %d, order_number: %s, product_id: %d", order.ID, order.OrderNumber, order.ProductID))

	// 1. 获取订单当前使用的API信息（优先使用APICurID，其次APIID）
	currentAPIID := order.APICurID
	if currentAPIID == 0 {
		currentAPIID = order.APIID
	}
	logger.Info("【同通道重试·当前通道】order_id: %d, order_number: %s, current_api_id: %d", order.ID, order.OrderNumber, currentAPIID)
	if currentAPIID == 0 {
		logger.Error("【同通道重试中止】order_id: %d, order_number: %s, 原因: 订单未绑定API", order.ID, order.OrderNumber)
		return fmt.Errorf("订单未绑定API，无法进行同通道重试")
	}

	// 2. 获取API关系配置
	relations, _, err := s.productAPIRelationRepo.List(ctx, order.ProductID, currentAPIID, 1, 1, 1)
	if err != nil {
		return fmt.Errorf("获取API关系配置失败: %v", err)
	}
	if len(relations) == 0 {
		return fmt.Errorf("未找到对应的API关系配置")
	}
	relation := relations[0]

	// 3. 检查是否启用同通道重试
	if !relation.SameChannelRetryEnabled {
		return fmt.Errorf("当前通道未启用同通道重试")
	}

	// 4. 获取已有的重试记录
	records, err := s.retryRepo.GetByOrderID(ctx, order.ID)
	if err != nil {
		return fmt.Errorf("获取重试记录失败: %v", err)
	}

	// 4.1 如果已经存在同通道且处于待处理/处理中状态的重试记录，则跳过，避免重复触发
	for _, r := range records {
		if r.APIID == currentAPIID && r.RetryType == model.RetryTypeSameChannel && (r.Status == 0 || r.Status == 1) {
			logger.Info("【同通道重试已在进行中，跳过重复触发】order_id: %d, record_id: %d", order.ID, r.ID)
			return nil
		}
	}

	// 5. 计算已执行的尝试次数：优先取该通道的最大 AttemptNo；若历史记录未写 AttemptNo，则退化为统计该通道的记录数
	maxAttemptNo := 0
	countForChannel := 0
	for _, r := range records {
		if r.APIID == currentAPIID {
			countForChannel++
			if r.AttemptNo > maxAttemptNo {
				maxAttemptNo = r.AttemptNo
			}
		}
	}
	executedAttempts := maxAttemptNo
	if executedAttempts == 0 {
		executedAttempts = countForChannel
	}

	// 6. 检查是否超过最大重试次数
	if executedAttempts >= relation.SameChannelRetryTimes {
		logger.Info("【同通道重试次数已达上限】order_id: %d, current_count: %d, max_count: %d", 
			order.ID, executedAttempts, relation.SameChannelRetryTimes)
		// 达到上限则将订单处理为失败
		if err := s.orderService.ProcessOrderFail(ctx, order.ID, "同通道重试次数已达上限"); err != nil {
			return fmt.Errorf("处理订单失败: %v", err)
		}
		return nil
	}

	// 7. 生成新的 OutTradeNum（平台订单号），下一次尝试序号 = executedAttempts + 1
	nextAttemptNo := executedAttempts + 1
	newOutTradeNum := fmt.Sprintf("%s_%d", order.OrderNumber, nextAttemptNo)

	// 8. 获取API信息以获取通道编码
	api, err := s.platformRepo.GetAPIByID(ctx, currentAPIID)
	if err != nil {
		return fmt.Errorf("获取API信息失败: %v", err)
	}
	channelCode := api.Code
	if channelCode == "" {
		return fmt.Errorf("通道编码为空")
	}

	// 9. 创建同通道重试记录前，构建 JSON 字段，避免空字符串写入 JSON 列
	usedAPIs := []map[string]interface{}{
		{
			"api_id":  currentAPIID,
			"param_id": relation.ParamID,
		},
	}
	usedAPIsJSON, _ := json.Marshal(usedAPIs)

	// 9. 创建同通道重试记录（AttemptNo 记录已执行次数，真正执行的尝试号由 executeRetry 统一 +1），并直接置为处理中，避免扫描任务并发触发
	retryRecord := &model.OrderRetryRecord{
		OrderID:           order.ID,
		APIID:             currentAPIID,
		ParamID:           relation.ParamID,
		RetryType:         model.RetryTypeSameChannel,
		RetryCount:        executedAttempts,
		Status:            1, // 直接置为处理中，避免被 ProcessRetries 并发捞起
		NextRetryTime:     time.Now(),
		ChannelCode:       channelCode,
		AttemptNo:         executedAttempts,
		ActiveOutTradeNum: newOutTradeNum,
		LastAttemptAt:     &[]time.Time{time.Now()}[0],
		RetryParams:       "{}",
		UsedAPIs:          string(usedAPIsJSON),
	}

	if err := s.retryRepo.Create(ctx, retryRecord); err != nil {
		return fmt.Errorf("创建同通道重试记录失败: %v", err)
	}

	// 10. 立即执行重试
	logger.Info("【立即执行同通道重试】record_id: %d, order_id: %d, attempt_no: %d", 
		retryRecord.ID, order.ID, nextAttemptNo)
	
	if err := s.executeRetry(ctx, retryRecord); err != nil {
		// 更新重试记录状态为失败
		retryRecord.Status = 3 // 重试失败
		retryRecord.LastError = err.Error()
		if updateErr := s.retryRepo.Update(ctx, retryRecord); updateErr != nil {
			logger.Error("【更新重试记录状态失败】record_id: %d, error: %v", retryRecord.ID, updateErr)
		}
		return fmt.Errorf("同通道重试失败: %v", err)
	}

	// 更新重试记录状态为成功
	retryRecord.Status = 2 // 重试成功
	if err := s.retryRepo.Update(ctx, retryRecord); err != nil {
		logger.Error("【更新重试记录状态失败】record_id: %d, error: %v", retryRecord.ID, err)
	}

	logger.Info("【同通道重试成功】record_id: %d, order_id: %d", retryRecord.ID, order.ID)
	return nil
}

// GetOrderByID 根据订单ID获取订单信息
func (s *RetryService) GetOrderByID(ctx context.Context, orderID int64) (*model.Order, error) {
	return s.orderRepo.GetByID(ctx, orderID)
}
