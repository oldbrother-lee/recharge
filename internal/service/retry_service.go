package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"recharge-go/internal/model"
	"recharge-go/internal/repository"
    "recharge-go/internal/signature"
    logger "recharge-go/pkg/log"
	"sort"
	"strings"
	"time"
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
    // v2 类别日志：retry
    lg := logger.WithContextCategory(ctx, "retry")
	// 入口诊断日志（串联订单号）
	lg.Info("重试入口",
		logger.Int64V2("order_id", order.ID),
		logger.StringV2("order_number", order.OrderNumber),
		logger.Int64V2("product_id", order.ProductID),
		logger.IntV2("retry_type", retryType),
	)

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
			lg.Info("同通道判断·关系统计",
				logger.Int64V2("order_id", order.ID),
				logger.StringV2("order_number", order.OrderNumber),
				logger.Int64V2("product_id", order.ProductID),
				logger.IntV2("enabled_count", enabled),
				logger.IntV2("relations_total", len(relations)),
			)

			curAPI := order.APICurID
			if curAPI == 0 {
				curAPI = order.APIID
			}
			lg.Info("同通道判断·当前通道",
				logger.Int64V2("order_id", order.ID),
				logger.StringV2("order_number", order.OrderNumber),
				logger.Int64V2("cur_api_id", curAPI),
			)

			var curRel *model.ProductAPIRelation
			for _, r := range relations {
				if r.APIID == curAPI {
					curRel = r
					break
				}
			}
			if enabled == 1 && curRel != nil && curRel.SameChannelRetryEnabled {
				// 当前关系命中同通道条件
				lg.Info("同通道优先命中",
					logger.Int64V2("order_id", order.ID),
					logger.StringV2("order_number", order.OrderNumber),
					logger.Int64V2("relation_id", curRel.ID),
					logger.Int64V2("api_id", curRel.APIID),
					logger.IntV2("relation_status", curRel.Status),
					logger.BoolV2("same_channel_enabled", curRel.SameChannelRetryEnabled),
					logger.IntV2("same_channel_times", curRel.SameChannelRetryTimes),
				)
				// 记录API状态（仅日志）
				if api, err2 := s.platformRepo.GetAPIByID(ctx, curRel.APIID); err2 == nil && api != nil {
					lg.Info("同通道判断·API状态",
						logger.Int64V2("order_id", order.ID),
						logger.StringV2("order_number", order.OrderNumber),
						logger.Int64V2("api_id", api.ID),
						logger.IntV2("api_status", api.Status),
					)
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

	// 2. 选择下一个未尝试的关系并创建单条重试记录
	records, err := s.retryRepo.GetByOrderID(ctx, order.ID)
	if err != nil {
		return fmt.Errorf("获取已使用API失败: %v", err)
	}

	usedPairs := make(map[string]struct{})
	for _, record := range records {
		var usedAPIList []struct {
			APIID   int64 `json:"api_id"`
			ParamID int64 `json:"param_id"`
			Result  string `json:"result,omitempty"`
			Error   string `json:"error,omitempty"`
			Time    string `json:"time,omitempty"`
		}
		if err := json.Unmarshal([]byte(record.UsedAPIs), &usedAPIList); err == nil {
			for _, u := range usedAPIList {
				usedPairs[fmt.Sprintf("%d:%d", u.APIID, u.ParamID)] = struct{}{}
			}
		} else {
			var simpleAPIList []int64
			if err2 := json.Unmarshal([]byte(record.UsedAPIs), &simpleAPIList); err2 == nil {
				for _, apiID := range simpleAPIList {
					usedPairs[fmt.Sprintf("%d:%d", apiID, 0)] = struct{}{}
				}
			}
		}
	}

	var nextRel *model.ProductAPIRelation
	for _, rel := range relations {
		key := fmt.Sprintf("%d:%d", rel.APIID, rel.ParamID)
		if _, ok := usedPairs[key]; !ok {
			nextRel = rel
			break
		}
	}
	if nextRel == nil {
		return fmt.Errorf("没有可用的API进行重试")
	}

	// 历史已用集合标准化持久化（不包含本次候选）
	uniquePairs := make([]struct {
		APIID   int64 `json:"api_id"`
		ParamID int64 `json:"param_id,omitempty"`
	}, 0, len(usedPairs))
	for k := range usedPairs {
		var a, p int64
		fmt.Sscanf(k, "%d:%d", &a, &p)
		uniquePairs = append(uniquePairs, struct {
			APIID   int64 `json:"api_id"`
			ParamID int64 `json:"param_id,omitempty"`
		}{APIID: a, ParamID: p})
	}
	usedAPIsJSON, err := json.Marshal(uniquePairs)
	if err != nil {
		return fmt.Errorf("序列化已使用API失败: %v", err)
	}

    // 计算同组合的最大 AttemptNo，并检查是否存在待处理/处理中记录
    maxAttempt := 0
    var pendingRecord *model.OrderRetryRecord
    for _, r := range records {
        if r.APIID == nextRel.APIID && r.ParamID == nextRel.ParamID {
            if int(r.AttemptNo) > maxAttempt {
                maxAttempt = int(r.AttemptNo)
            }
            if r.Status == 0 || r.Status == 1 { // 待处理或处理中
                pendingRecord = r
            }
        }
    }

    // 若存在同组合的待处理记录，复用并更新下一重试时间，避免新建重复记录
    if pendingRecord != nil {
        lg.Info("复用同组合待处理重试记录",
            logger.Int64V2("record_id", pendingRecord.ID),
            logger.Int64V2("order_id", order.ID),
            logger.Int64V2("api_id", nextRel.APIID),
            logger.Int64V2("param_id", nextRel.ParamID),
            logger.IntV2("attempt_no", int(pendingRecord.AttemptNo)),
        )
        // 设定秒级重试
        pendingRecord.NextRetryTime = time.Now().Add(3 * time.Second)
        _ = s.retryRepo.Update(ctx, pendingRecord)
        return nil
    }

    retryCount := len(records)
    // 首次该组合重试：立即；否则秒级重试
    nextRetryTime := time.Now()
    if maxAttempt > 0 || retryCount > 0 {
        nextRetryTime = time.Now().Add(3 * time.Second)
    }

    retryRecord := &model.OrderRetryRecord{
        OrderID:       order.ID,
        APIID:         nextRel.APIID,
        ParamID:       nextRel.ParamID,
        RetryType:     retryType,
        Status:        0,
        NextRetryTime: nextRetryTime,
        RetryParams:   "{}",
        UsedAPIs:      string(usedAPIsJSON),
        RetryCount:    retryCount,
        AttemptNo:     int(maxAttempt), // 继承上次尝试号，执行时会递增
    }

    if err := s.retryRepo.Create(ctx, retryRecord); err != nil {
        lg.Error("创建重试记录失败",
            logger.Int64V2("order_id", order.ID),
            logger.Int64V2("api_id", nextRel.APIID),
            logger.ErrorV2(err),
        )
        return nil
    }

    if retryRecord.RetryCount == 0 && maxAttempt == 0 {
        lg.Info("首次重试·立即执行重试",
            logger.Int64V2("record_id", retryRecord.ID),
            logger.Int64V2("order_id", order.ID),
        )
		if err := s.executeRetry(ctx, retryRecord); err != nil {
			retryRecord.Status = 3
			retryRecord.LastError = err.Error()
			if err := s.retryRepo.Update(ctx, retryRecord); err != nil {
				lg.Error("更新重试记录状态失败",
					logger.Int64V2("record_id", retryRecord.ID),
					logger.Int64V2("order_id", retryRecord.OrderID),
					logger.ErrorV2(err),
				)
			}
			lg.Error("首次重试失败",
				logger.Int64V2("record_id", retryRecord.ID),
				logger.Int64V2("order_id", order.ID),
				logger.ErrorV2(err),
			)
		} else {
			retryRecord.Status = 2
			if err := s.retryRepo.Update(ctx, retryRecord); err != nil {
				lg.Error("更新重试记录状态失败",
					logger.Int64V2("record_id", retryRecord.ID),
					logger.Int64V2("order_id", retryRecord.OrderID),
					logger.ErrorV2(err),
				)
			}
			lg.Info("首次重试成功",
				logger.Int64V2("record_id", retryRecord.ID),
				logger.Int64V2("order_id", order.ID),
			)
			return nil
		}
	}

	// 检查当前订单是否所有重试通道都已完成
	orderRetries, err := s.retryRepo.GetByOrderID(ctx, order.ID)
	if err != nil {
		lg.Error("获取订单重试记录失败",
			logger.Int64V2("order_id", order.ID),
			logger.ErrorV2(err),
		)
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
		lg.Info("所有重试通道均已失败，更新订单状态为失败",
			logger.Int64V2("order_id", order.ID),
		)
		if err := s.orderService.ProcessOrderFail(ctx, order.ID, "所有通道重试失败，自动失败"); err != nil {
			lg.Error("订单失败处理失败",
				logger.Int64V2("order_id", order.ID),
				logger.ErrorV2(err),
			)
			// 如果是获取锁失败，创建一个延迟重试任务
			if strings.Contains(err.Error(), "获取分布式锁超时") || strings.Contains(err.Error(), "获取退款锁失败") {
				lg.Info("因锁获取失败，创建延迟重试任务",
					logger.Int64V2("order_id", order.ID),
				)
				retryRecord := &model.OrderRetryRecord{
					OrderID:       order.ID,
					RetryType:     model.RetryTypeOrderFail,
					Status:        0, // 待处理
					RetryCount:    0,
					NextRetryTime: time.Now().Add(30 * time.Second), // 30秒后重试
				}
				if createErr := s.retryRepo.Create(ctx, retryRecord); createErr != nil {
					lg.Error("创建延迟重试任务失败",
						logger.Int64V2("order_id", order.ID),
						logger.ErrorV2(createErr),
					)
				}
			}
		} else {
			lg.Info("订单状态已更新为失败并已发送通知",
				logger.Int64V2("order_id", order.ID),
			)
		}
	}

	return nil
}

// UsedAPI 记录已使用的API信息
type UsedAPI struct {
	APIID   int64  `json:"api_id"`
	ParamID int64  `json:"param_id"`
	Result  string `json:"result,omitempty"` // API响应结果
	Error   string `json:"error,omitempty"`   // 错误信息
	Time    string `json:"time,omitempty"`    // 执行时间
}

// generateRandomOrderSuffix 生成随机订单号后缀
func generateRandomOrderSuffix() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const length = 16
	
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// ProcessRetries 处理待重试的记录
func (s *RetryService) ProcessRetries(ctx context.Context) error {
	// v2 类别日志：retry
	lg := logger.WithContextCategory(ctx, "retry")
	lg.Info("开始处理待重试记录")

	// 1. 获取待重试的记录
	records, err := s.retryRepo.GetPendingRetries(ctx)
	if err != nil {
		lg.Error("获取待重试记录失败",
			logger.ErrorV2(err),
		)
		return fmt.Errorf("获取待重试记录失败: %v", err)
	}

	if len(records) == 0 {
		lg.Info("没有待重试的记录")
		return nil
	}

	lg.Info("获取到待重试记录",
		logger.IntV2("count", len(records)),
	)

	// 2. 处理每条重试记录
	for _, record := range records {
		// 检查重试时间是否到达
		if time.Now().Before(record.NextRetryTime) {
			lg.Info("重试时间未到",
				logger.Int64V2("record_id", record.ID),
				logger.Int64V2("order_id", record.OrderID),
				logger.StringV2("next_retry_time", record.NextRetryTime.Format(time.RFC3339)),
				logger.StringV2("current_time", time.Now().Format(time.RFC3339)),
			)
			continue
		}

		// 更新重试记录状态为处理中
		record.Status = 1 // 处理中
		if err := s.retryRepo.Update(ctx, record); err != nil {
			lg.Error("更新重试记录状态失败",
				logger.Int64V2("record_id", record.ID),
				logger.Int64V2("order_id", record.OrderID),
				logger.ErrorV2(err),
			)
			continue
		}

		lg.Info("开始执行重试",
			logger.Int64V2("record_id", record.ID),
			logger.Int64V2("order_id", record.OrderID),
			logger.IntV2("retry_count", record.RetryCount),
		)

		// 执行重试
		if err := s.executeRetry(ctx, record); err != nil {
			lg.Error("重试执行失败",
				logger.Int64V2("record_id", record.ID),
				logger.Int64V2("order_id", record.OrderID),
				logger.ErrorV2(err),
			)

			// 更新重试记录状态为失败
			record.Status = 3 // 重试失败
			record.LastError = err.Error()
			if err := s.retryRepo.Update(ctx, record); err != nil {
				lg.Error("更新重试记录状态失败",
					logger.Int64V2("record_id", record.ID),
					logger.Int64V2("order_id", record.OrderID),
					logger.ErrorV2(err),
				)
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
					lg.Info("所有重试均已完成，更新订单状态为失败并发送通知",
						logger.Int64V2("order_id", record.OrderID),
					)
					if err := s.orderService.ProcessOrderFail(ctx, record.OrderID, "所有通道重试失败，自动失败"); err != nil {
						lg.Error("订单失败处理失败",
							logger.Int64V2("order_id", record.OrderID),
							logger.ErrorV2(err),
						)
					} else {
						lg.Info("订单状态已更新为失败并已发送通知",
							logger.Int64V2("order_id", record.OrderID),
						)
					}
				}
			}

			continue
		}

		// 更新重试记录状态为成功
		record.Status = 2 // 重试成功
		if err := s.retryRepo.Update(ctx, record); err != nil {
			lg.Error("更新重试记录状态失败",
				logger.Int64V2("record_id", record.ID),
				logger.Int64V2("order_id", record.OrderID),
				logger.ErrorV2(err),
			)
			continue
		}

		lg.Info("重试执行成功",
			logger.Int64V2("record_id", record.ID),
			logger.Int64V2("order_id", record.OrderID),
		)
	}

	lg.Info("所有重试记录处理完成")

	// 检查是否有订单需要更新为失败状态
	records, err = s.retryRepo.GetByOrderID(ctx, 0) // 获取所有重试记录
	if err != nil {
		lg.Error("获取重试记录失败",
			logger.ErrorV2(err),
		)
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
			lg.Error("获取订单信息失败",
				logger.Int64V2("order_id", orderID),
				logger.ErrorV2(err),
			)
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
			lg.Info("所有平台重试均失败，更新订单状态为失败",
				logger.Int64V2("order_id", orderID),
			)
			if err := s.orderService.ProcessOrderFail(ctx, orderID, "所有平台重试失败，自动失败"); err != nil {
				lg.Error("订单失败处理失败",
					logger.Int64V2("order_id", orderID),
					logger.ErrorV2(err),
				)
			} else {
				lg.Info("订单状态已更新为失败",
					logger.Int64V2("order_id", orderID),
				)
			}
		}
	}

	return nil
}

// executeRetry 执行重试
func (s *RetryService) executeRetry(ctx context.Context, record *model.OrderRetryRecord) error {
	// v2 类别日志：retry
	lg := logger.WithContextCategory(ctx, "retry")
	lg.Info("开始执行重试",
		logger.Int64V2("record_id", record.ID),
		logger.Int64V2("order_id", record.OrderID),
		logger.IntV2("retry_type", record.RetryType),
	)

	// 检查重试类型，如果是订单失败重试，直接调用ProcessOrderFail
	if record.RetryType == model.RetryTypeOrderFail {
		lg.Info("执行订单失败重试",
			logger.Int64V2("record_id", record.ID),
			logger.Int64V2("order_id", record.OrderID),
		)
		return s.orderService.ProcessOrderFail(ctx, record.OrderID, "重试处理订单失败")
	}

	// 1. 获取订单信息
	order, err := s.orderRepo.GetByID(ctx, record.OrderID)
	if err != nil {
		lg.Error("获取订单信息失败",
			logger.Int64V2("record_id", record.ID),
			logger.Int64V2("order_id", record.OrderID),
			logger.ErrorV2(err),
		)
		return fmt.Errorf("获取订单信息失败: %v", err)
	}
	// 注入订单号到上下文，便于全链路日志携带
	ctx = logger.InjectOrderNumber(ctx, order.OrderNumber)
	lg = logger.WithContextCategory(ctx, "retry")
	lg.Info("获取订单信息成功",
		logger.Int64V2("record_id", record.ID),
		logger.Int64V2("order_id", record.OrderID),
		logger.IntV2("status", int(order.Status)),
		logger.StringV2("order_number", order.OrderNumber),
	)
	// 2. 获取可用的API关系列表
	relations, err := s.GetAvailableAPIRelations(ctx, record.OrderID, order.ProductID)
	if err != nil {
		lg.Error("获取可用API关系失败",
			logger.Int64V2("record_id", record.ID),
			logger.Int64V2("order_id", record.OrderID),
			logger.ErrorV2(err),
		)
		return fmt.Errorf("获取可用API关系失败: %v", err)
	}

	if len(relations) == 0 {
		lg.Error("没有可用的API关系",
			logger.Int64V2("record_id", record.ID),
			logger.Int64V2("order_id", record.OrderID),
		)
		return fmt.Errorf("没有可用的API关系")
	}

	lg.Info("获取到可用API关系",
		logger.Int64V2("record_id", record.ID),
		logger.Int64V2("order_id", record.OrderID),
		logger.IntV2("count", len(relations)),
	)

	var usedAPIList []UsedAPI
	if record.UsedAPIs != "" {
		_ = json.Unmarshal([]byte(record.UsedAPIs), &usedAPIList)
	}
	used := make(map[string]struct{})
	for _, u := range usedAPIList {
		used[fmt.Sprintf("%d:%d", u.APIID, u.ParamID)] = struct{}{}
	}
	var relation *model.ProductAPIRelation
	for _, r := range relations {
		key := fmt.Sprintf("%d:%d", r.APIID, r.ParamID)
		if _, ok := used[key]; ok {
			continue
		}
		relation = r
		break
	}
	if relation == nil {
		logger.WithContextCategory(ctx, "retry").Info("没有未尝试的API关系",
			logger.Int64V2("record_id", record.ID),
			logger.Int64V2("order_id", record.OrderID),
		)
		return fmt.Errorf("没有未尝试的API关系")
	}
	lg.Info("选择API关系",
		logger.Int64V2("record_id", record.ID),
		logger.Int64V2("order_id", record.OrderID),
		logger.Int64V2("api_id", relation.APIID),
		logger.Int64V2("param_id", relation.ParamID),
	)

	// 4. 获取API信息
	api, err := s.platformRepo.GetAPIByID(ctx, relation.APIID)
	if err != nil {
		lg.Error("获取API信息失败",
			logger.Int64V2("record_id", record.ID),
			logger.Int64V2("order_id", record.OrderID),
			logger.ErrorV2(err),
		)
		return fmt.Errorf("获取API信息失败: %v", err)
	}
	lg.Info("获取API信息成功",
		logger.Int64V2("record_id", record.ID),
		logger.Int64V2("order_id", record.OrderID),
		logger.Int64V2("api_id", api.ID),
		logger.StringV2("api_name", api.Name),
	)

	// 5. 获取API参数
	param, err := s.platformRepo.GetAPIParamByID(ctx, relation.ParamID)
	if err != nil {
		lg.Error("获取API参数失败",
			logger.Int64V2("record_id", record.ID),
			logger.Int64V2("order_id", record.OrderID),
			logger.ErrorV2(err),
		)
		return fmt.Errorf("获取API参数失败: %v", err)
	}
	lg.Info("获取API参数成功",
		logger.Int64V2("record_id", record.ID),
		logger.Int64V2("order_id", record.OrderID),
		logger.Int64V2("param_id", param.ID),
	)

	record.APIID = relation.APIID
	record.ParamID = relation.ParamID
	var exists bool
	for _, u := range usedAPIList {
		if u.APIID == relation.APIID && u.ParamID == relation.ParamID { exists = true; break }
	}
	if !exists {
		usedAPIList = append(usedAPIList, UsedAPI{APIID: relation.APIID, ParamID: relation.ParamID})
		if b, err := json.Marshal(usedAPIList); err == nil {
			record.UsedAPIs = string(b)
		}
	}
	_ = s.retryRepo.Update(ctx, record)
	lg.Info("更新重试记录API信息成功",
		logger.Int64V2("record_id", record.ID),
		logger.Int64V2("order_id", record.OrderID),
		logger.Int64V2("api_id", record.APIID),
		logger.Int64V2("param_id", record.ParamID),
	)

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
			lg.Info("同通道重试已达到最大重试次数，调用失败处理逻辑",
				logger.Int64V2("order_id", order.ID),
				logger.Int64V2("record_id", record.ID),
				logger.IntV2("attempt_no", retryAttempt),
				logger.IntV2("max_retry_times", relation.SameChannelRetryTimes),
			)

			// 调用订单失败处理逻辑（包括退款、发送通知等）
			if err := s.orderService.ProcessOrderFail(ctx, record.OrderID, "同通道重试达到最大次数"); err != nil {
				lg.Error("订单失败处理失败",
					logger.Int64V2("record_id", record.ID),
					logger.Int64V2("order_id", record.OrderID),
					logger.ErrorV2(err),
				)
				return fmt.Errorf("订单失败处理失败: %v", err)
			}

			// 更新重试记录状态为失败
			record.Status = 3 // 重试失败
			record.LastError = fmt.Sprintf("同通道重试已达到最大次数(%d)，订单已失败", relation.SameChannelRetryTimes)
			if err := s.retryRepo.Update(ctx, record); err != nil {
				lg.Error("更新重试记录状态失败",
					logger.Int64V2("record_id", record.ID),
					logger.Int64V2("order_id", record.OrderID),
					logger.ErrorV2(err),
				)
			}
			return fmt.Errorf("同通道重试已达到最大次数(%d)，订单已失败", relation.SameChannelRetryTimes)
		} else {
			// 生成随机C开头的重试订单号
			newActiveOutTradeNum := fmt.Sprintf("C%s", generateRandomOrderSuffix())

			// 【重要】不修改订单表的OutTradeNum，而是将新订单号存储在重试记录的ActiveOutTradeNum中
			record.ActiveOutTradeNum = newActiveOutTradeNum
			record.RetryType = 3 // RetryTypeSameChannel
			record.AttemptNo = retryAttempt
			now := time.Now()
			record.LastAttemptAt = &now

			if err := s.retryRepo.Update(ctx, record); err != nil {
				logger.WithContext(ctx).Error("更新重试记录 ActiveOutTradeNum 失败",
					logger.Int64V2("record_id", record.ID),
					logger.Int64V2("order_id", record.OrderID),
					logger.ErrorV2(err),
				)
				return fmt.Errorf("更新重试记录ActiveOutTradeNum失败: %v", err)
			}

			logger.WithContext(ctx).Info("【同通道重试】已生成新的ActiveOutTradeNum",
				logger.Int64V2("record_id", record.ID),
				logger.Int64V2("order_id", order.ID),
				logger.StringV2("active_out_trade_num", newActiveOutTradeNum),
				logger.StringV2("order_number", order.OrderNumber),
				logger.StringV2("out_trade_num", order.OutTradeNum), // 保持不变
				logger.IntV2("attempt_no", retryAttempt),
				logger.StringV2("retry_suffix", "C[随机]"),
			)
		}
	}

	// 7. 开启事务前生成新的 ActiveOutTradeNum（使用随机C开头订单号）
	newActiveOutTradeNum := fmt.Sprintf("C%s", generateRandomOrderSuffix())
	record.ActiveOutTradeNum = newActiveOutTradeNum
	record.AttemptNo = retryAttempt
	now := time.Now()
	record.LastAttemptAt = &now
	if err := s.retryRepo.Update(ctx, record); err != nil {
		logger.WithContext(ctx).Error("更新重试记录 ActiveOutTradeNum 失败",
			logger.Int64V2("record_id", record.ID),
			logger.Int64V2("order_id", record.OrderID),
			logger.ErrorV2(err),
		)
		return fmt.Errorf("更新重试记录ActiveOutTradeNum失败: %v", err)
	}
	logger.WithContext(ctx).Info("跨通道重试生成ActiveOutTradeNum",
		logger.Int64V2("record_id", record.ID),
		logger.Int64V2("order_id", order.ID),
		logger.StringV2("active_out_trade_num", newActiveOutTradeNum),
		logger.StringV2("order_number", order.OrderNumber),
		logger.IntV2("attempt_no", retryAttempt),
	)

	tx := s.orderRepo.(*repository.OrderRepositoryImpl).DB().Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			logger.WithContext(ctx).Error("事务回滚",
				logger.Int64V2("record_id", record.ID),
				logger.Int64V2("order_id", record.OrderID),
				logger.AnyV2("panic", r),
			)
		}
	}()

	// 8. 调用 RechargeService 的 SubmitOrder 方法
	// 【同通道重试】如果有ActiveOutTradeNum，临时替换order.OrderNumber用于提交
	originalOrderNumber := order.OrderNumber
	if record.ActiveOutTradeNum != "" {
		order.OrderNumber = record.ActiveOutTradeNum
		logger.WithContext(ctx).Info("【同通道重试】临时使用ActiveOutTradeNum作为OrderNumber提交订单",
			logger.Int64V2("record_id", record.ID),
			logger.Int64V2("order_id", record.OrderID),
			logger.StringV2("original_order_number", originalOrderNumber),
			logger.StringV2("active_out_trade_num", record.ActiveOutTradeNum),
		)
	}

	logger.WithContext(ctx).Info("开始提交订单",
		logger.Int64V2("record_id", record.ID),
		logger.Int64V2("order_id", record.OrderID),
		logger.StringV2("order_number", order.OrderNumber),
	)
	submitErr := s.rechargeService.SubmitOrder(ctx, order, api, param)

	// 【重要】提交完成后立即恢复原始OrderNumber
	order.OrderNumber = originalOrderNumber

	// 记录API响应信息到used_apis
	apiResponse := ""
	if submitErr != nil {
		apiResponse = fmt.Sprintf("错误: %v", submitErr)
	} else {
		apiResponse = "提交成功"
	}
	
	// 更新used_apis记录当前API的响应信息
	var responseAPIList []UsedAPI
	if record.UsedAPIs != "" {
		_ = json.Unmarshal([]byte(record.UsedAPIs), &responseAPIList)
	}
	
	// 查找或创建当前API的记录
	found := false
	for i, responseAPI := range responseAPIList {
		if responseAPI.APIID == api.ID && responseAPI.ParamID == param.ID {
			responseAPIList[i].Result = apiResponse
			responseAPIList[i].Time = time.Now().Format("2006-01-02 15:04:05")
			if submitErr != nil {
				responseAPIList[i].Error = submitErr.Error()
			}
			found = true
			break
		}
	}
	
	if !found {
		responseAPI := UsedAPI{
			APIID:   api.ID,
			ParamID: param.ID,
			Result:  apiResponse,
			Time:    time.Now().Format("2006-01-02 15:04:05"),
		}
		if submitErr != nil {
			responseAPI.Error = submitErr.Error()
		}
		responseAPIList = append(responseAPIList, responseAPI)
	}
	
	// 更新重试记录的used_apis字段
	if responseAPIsJSON, err := json.Marshal(responseAPIList); err == nil {
		record.UsedAPIs = string(responseAPIsJSON)
	}

	if submitErr != nil {
		tx.Rollback()
		logger.WithContext(ctx).Error("提交订单失败",
			logger.Int64V2("record_id", record.ID),
			logger.Int64V2("order_id", record.OrderID),
			logger.ErrorV2(submitErr),
		)
		return fmt.Errorf("提交订单失败: %v", submitErr)
	}

	logger.WithContext(ctx).Info("提交订单成功",
		logger.Int64V2("record_id", record.ID),
		logger.Int64V2("order_id", record.OrderID),
		logger.StringV2("order_number", order.OrderNumber),
	)

	// 9. 更新订单状态
	logger.WithContext(ctx).Info("开始更新订单状态",
		logger.Int64V2("record_id", record.ID),
		logger.Int64V2("order_id", record.OrderID),
		logger.StringV2("order_number", order.OrderNumber),
		logger.IntV2("old_status", int(order.Status)),
		logger.IntV2("new_status", int(model.OrderStatusRecharging)),
	)
	result := tx.Model(&model.Order{}).Where("id = ?", record.OrderID).Update("status", model.OrderStatusRecharging)
	if result.Error != nil {
		tx.Rollback()
		logger.WithContext(ctx).Error("更新订单状态失败",
			logger.Int64V2("record_id", record.ID),
			logger.Int64V2("order_id", record.OrderID),
			logger.ErrorV2(result.Error),
		)
		return fmt.Errorf("更新订单状态失败: %v", result.Error)
	}
	if result.RowsAffected == 0 {
		tx.Rollback()
		logger.WithContext(ctx).Error("更新订单状态失败（没有记录被更新）",
			logger.Int64V2("record_id", record.ID),
			logger.Int64V2("order_id", record.OrderID),
		)
		return fmt.Errorf("没有记录被更新")
	}

	// 10. 更新重试记录的AttemptNo字段
	logger.WithContext(ctx).Info("更新重试记录 AttemptNo",
		logger.Int64V2("record_id", record.ID),
		logger.Int64V2("order_id", record.OrderID),
		logger.IntV2("attempt_no_old", record.AttemptNo),
		logger.IntV2("attempt_no_new", retryAttempt),
	)
	retryUpdateResult := tx.Model(&model.OrderRetryRecord{}).Where("id = ?", record.ID).Update("attempt_no", retryAttempt)
	if retryUpdateResult.Error != nil {
		tx.Rollback()
		logger.WithContext(ctx).Error("更新重试记录 AttemptNo 失败",
			logger.Int64V2("record_id", record.ID),
			logger.Int64V2("order_id", record.OrderID),
			logger.ErrorV2(retryUpdateResult.Error),
		)
		return fmt.Errorf("更新重试记录AttemptNo失败: %v", retryUpdateResult.Error)
	}

	// 11. 提交事务
	if err := tx.Commit().Error; err != nil {
		logger.WithContext(ctx).Error("提交事务失败",
			logger.Int64V2("record_id", record.ID),
			logger.Int64V2("order_id", record.OrderID),
			logger.ErrorV2(err),
		)
		return fmt.Errorf("提交事务失败: %v", err)
	}

	logger.WithContext(ctx).Info("订单状态更新成功",
		logger.Int64V2("record_id", record.ID),
		logger.Int64V2("order_id", record.OrderID),
		logger.StringV2("order_number", order.OrderNumber),
	)

	// 11. 重试成功，取消同一订单的其他待重试记录
	if err := s.cancelPendingRetries(ctx, record.OrderID, record.ID); err != nil {
		logger.WithContext(ctx).Error("取消其他待重试记录失败",
			logger.Int64V2("record_id", record.ID),
			logger.Int64V2("order_id", record.OrderID),
			logger.ErrorV2(err),
		)
		// 不返回错误，因为主要任务已完成
	}

	return nil
}

// getAvailableAPIRelations 获取可用的API关系列表
func (s *RetryService) GetAvailableAPIRelations(ctx context.Context, orderID int64, productID int64) ([]*model.ProductAPIRelation, error) {
	// v2 类别日志：retry
	lg := logger.WithContextCategory(ctx, "retry")
	lg.Info("开始获取可用的API关系列表",
		logger.Int64V2("order_id", orderID),
		logger.Int64V2("product_id", productID),
	)

	// 1. 获取已使用的API列表
	records, err := s.retryRepo.GetByOrderID(ctx, orderID)
	if err != nil {
		lg.Error("获取已使用API失败",
			logger.ErrorV2(err),
			logger.Int64V2("order_id", orderID),
		)
		return nil, fmt.Errorf("获取已使用API失败: %v", err)
	}

	lg.Info("DEBUG: 获取到的重试记录",
		logger.Int64V2("order_id", orderID),
		logger.IntV2("records_count", len(records)),
	)

	usedPairs := make([]struct {
		APIID   int64 `json:"api_id"`
		ParamID int64 `json:"param_id,omitempty"`
	}, 0)
	for _, record := range records {
		// 统计所有已尝试过的API，避免重复使用失败的API
		// 状态说明：0: 待处理, 1: 处理中, 2: 重试成功, 3: 重试失败/已取消

		lg.Info("DEBUG: 处理重试记录",
			logger.Int64V2("record_id", record.ID),
			logger.Int64V2("order_id", record.OrderID),
			logger.Int64V2("api_id", record.APIID),
			logger.IntV2("status", int(record.Status)),
			logger.StringV2("used_apis_raw", record.UsedAPIs),
		)

		var usedAPIList []struct {
			APIID   int64 `json:"api_id"`
			ParamID int64 `json:"param_id,omitempty"`
		}
		if err := json.Unmarshal([]byte(record.UsedAPIs), &usedAPIList); err != nil {
			var simpleAPIList []int64
			if err2 := json.Unmarshal([]byte(record.UsedAPIs), &simpleAPIList); err2 != nil {
				lg.Error("解析已使用API失败",
					logger.ErrorV2(err),
					logger.Int64V2("record_id", record.ID),
				)
				return nil, fmt.Errorf("解析已使用API失败: %v", err)
			}
			lg.Info("DEBUG: 解析为简单数组", logger.AnyV2("simple_apis", simpleAPIList))
			for _, apiID := range simpleAPIList {
				usedPairs = append(usedPairs, struct {
					APIID   int64 `json:"api_id"`
					ParamID int64 `json:"param_id,omitempty"`
				}{APIID: apiID, ParamID: 0})
			}
		} else {
			lg.Info("DEBUG: 解析为对象数组", logger.AnyV2("object_apis", usedAPIList))
			for _, u := range usedAPIList {
				usedPairs = append(usedPairs, struct {
					APIID   int64 `json:"api_id"`
					ParamID int64 `json:"param_id,omitempty"`
				}{APIID: u.APIID, ParamID: u.ParamID})
			}
		}

		// 额外兼容：如果记录中有明确的 APIID/ParamID，直接计入已用集合（避免 UsedAPIs 为空导致重复选择）
		if record.APIID != 0 {
			usedPairs = append(usedPairs, struct {
				APIID   int64 `json:"api_id"`
				ParamID int64 `json:"param_id,omitempty"`
			}{APIID: record.APIID, ParamID: record.ParamID})
		}
	}

	lg.Info("已使用的API列表",
		logger.Int64V2("order_id", orderID),
		logger.AnyV2("used_api_pairs", usedPairs),
	)

	// 2. 获取可用的API关系列表
	relations, _, err := s.productAPIRelationRepo.List(ctx, productID, 0, 1, 1, 100)
	if err != nil {
		lg.Error("获取API关系列表失败",
			logger.ErrorV2(err),
			logger.Int64V2("product_id", productID),
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
					lg.Info("单通道绑定且同通道重试开启，优先返回当前通道",
						logger.Int64V2("order_id", orderID),
						logger.Int64V2("api_id", curRel.APIID),
						logger.Int64V2("param_id", curRel.ParamID),
						logger.IntV2("enabled_channel_count", enabledChannelCount),
					)
					return []*model.ProductAPIRelation{curRel}, nil
				}
			}
		} else {
			lg.Info("多通道绑定，跳过同通道重试，走多通道重试逻辑",
				logger.Int64V2("order_id", orderID),
				logger.IntV2("enabled_channel_count", enabledChannelCount),
			)
		}
	}

	// 3. 过滤和排序可用的API
	availableRelations := make([]*model.ProductAPIRelation, 0)
	for _, relation := range relations {
		// 3.1 检查ProductAPIRelation状态
		if relation.Status != 1 { // 1 表示启用
			lg.Info("ProductAPIRelation未启用，跳过",
				logger.Int64V2("relation_id", relation.ID),
				logger.Int64V2("api_id", relation.APIID),
				logger.IntV2("status", relation.Status),
			)
			continue
		}

		isUsed := false
		for _, up := range usedPairs {
			if relation.APIID == up.APIID && relation.ParamID == up.ParamID {
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
			lg.Error("获取API信息失败",
				logger.ErrorV2(err),
				logger.Int64V2("api_id", relation.APIID),
			)
			continue
		}

		// 3.4 检查API状态
		if api.Status != 1 { // 1 表示启用
			lg.Info("API未启用，跳过",
				logger.Int64V2("api_id", relation.APIID),
				logger.IntV2("status", api.Status),
			)
			continue
		}

		// 3.5 添加到可用列表
		availableRelations = append(availableRelations, relation)
	}

	// 4. 稳定排序：sort 升序 → api_id 升序 → param_id 升序
	sort.SliceStable(availableRelations, func(i, j int) bool {
		if availableRelations[i].Sort != availableRelations[j].Sort {
			return availableRelations[i].Sort < availableRelations[j].Sort
		}
		if availableRelations[i].APIID != availableRelations[j].APIID {
			return availableRelations[i].APIID < availableRelations[j].APIID
		}
		return availableRelations[i].ParamID < availableRelations[j].ParamID
	})

	lg.Info("获取到可用的API关系列表",
		logger.Int64V2("order_id", orderID),
		logger.Int64V2("product_id", productID),
		logger.IntV2("total", len(relations)),
		logger.IntV2("available", len(availableRelations)),
	)

	return availableRelations, nil
}

// cancelPendingRetries 取消同一订单的其他待重试记录
func (s *RetryService) cancelPendingRetries(ctx context.Context, orderID int64, excludeRecordID int64) error {
	// v2 类别日志：retry
	lg := logger.WithContextCategory(ctx, "retry")
	lg.Info("开始取消其他待重试记录",
		logger.Int64V2("order_id", orderID),
		logger.Int64V2("exclude_record_id", excludeRecordID),
	)

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
				lg.Error("取消重试记录失败",
					logger.Int64V2("record_id", record.ID),
					logger.Int64V2("order_id", orderID),
					logger.ErrorV2(err),
				)
				continue
			}
			lg.Info("取消重试记录成功",
				logger.Int64V2("record_id", record.ID),
				logger.Int64V2("order_id", orderID),
			)
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
	// v2 类别日志：retry
	lg := logger.WithContextCategory(ctx, "retry")
	lg.Info("开始处理同通道重试",
		logger.Int64V2("order_id", order.ID),
		logger.StringV2("order_number", order.OrderNumber),
		logger.Int64V2("product_id", order.ProductID),
	)

	// 1. 获取订单当前使用的API信息（优先使用APICurID，其次APIID）
	currentAPIID := order.APICurID
	if currentAPIID == 0 {
		currentAPIID = order.APIID
	}
	lg.Info("同通道重试·当前通道",
		logger.Int64V2("order_id", order.ID),
		logger.StringV2("order_number", order.OrderNumber),
		logger.Int64V2("current_api_id", currentAPIID),
	)
	if currentAPIID == 0 {
		lg.Error("同通道重试中止",
			logger.Int64V2("order_id", order.ID),
			logger.StringV2("order_number", order.OrderNumber),
			logger.StringV2("reason", "订单未绑定API"),
		)
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
			lg.Info("同通道重试已在进行中，跳过重复触发",
				logger.Int64V2("order_id", order.ID),
				logger.Int64V2("record_id", r.ID),
			)
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
		lg.Info("同通道重试次数已达上限",
			logger.Int64V2("order_id", order.ID),
			logger.IntV2("current_count", executedAttempts),
			logger.IntV2("max_count", relation.SameChannelRetryTimes),
		)
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
			"api_id":   currentAPIID,
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
		AttemptNo:         nextAttemptNo,
		ActiveOutTradeNum: newOutTradeNum,
		LastAttemptAt:     &[]time.Time{time.Now()}[0],
		RetryParams:       "{}",
		UsedAPIs:          string(usedAPIsJSON),
	}

	if err := s.retryRepo.Create(ctx, retryRecord); err != nil {
		return fmt.Errorf("创建同通道重试记录失败: %v", err)
	}

	// 10. 立即执行重试
	lg.Info("立即执行同通道重试",
		logger.Int64V2("record_id", retryRecord.ID),
		logger.Int64V2("order_id", order.ID),
		logger.IntV2("attempt_no", nextAttemptNo),
	)

	if err := s.executeRetry(ctx, retryRecord); err != nil {
		// 更新重试记录状态为失败
		retryRecord.Status = 3 // 重试失败
		retryRecord.LastError = err.Error()
		if updateErr := s.retryRepo.Update(ctx, retryRecord); updateErr != nil {
			lg.Error("更新重试记录状态失败",
				logger.Int64V2("record_id", retryRecord.ID),
				logger.ErrorV2(updateErr),
			)
		}
		return fmt.Errorf("同通道重试失败: %v", err)
	}

	// 更新重试记录状态为成功
	retryRecord.Status = 2 // 重试成功
	if err := s.retryRepo.Update(ctx, retryRecord); err != nil {
		lg.Error("更新重试记录状态失败",
			logger.Int64V2("record_id", retryRecord.ID),
			logger.ErrorV2(err),
		)
	}

	lg.Info("同通道重试成功",
		logger.Int64V2("record_id", retryRecord.ID),
		logger.Int64V2("order_id", order.ID),
	)
	return nil
}

// GetOrderByID 根据订单ID获取订单信息
func (s *RetryService) GetOrderByID(ctx context.Context, orderID int64) (*model.Order, error) {
	return s.orderRepo.GetByID(ctx, orderID)
}
