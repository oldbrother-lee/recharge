package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"recharge-go/internal/model"
	"recharge-go/internal/repository"
	"recharge-go/internal/service"
	"recharge-go/internal/signature"
	"recharge-go/pkg/log"
	logger "recharge-go/pkg/log"
	"recharge-go/pkg/queue"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ExternalCallbackController 外部回调控制器
type ExternalCallbackController struct {
	orderService        service.OrderService
	unifiedOrderService *service.UnifiedOrderService
	apiKeyRepo          repository.ExternalAPIKeyRepository
	logRepo             repository.ExternalOrderLogRepository
	signValidator       signature.SignatureHandler
	retryService        *service.RetryService
	productRepo         repository.ProductRepository
	queue               queue.Queue
}

// NewExternalCallbackController 创建外部回调控制器
func NewExternalCallbackController(
	orderService service.OrderService,
	unifiedOrderService *service.UnifiedOrderService,
	apiKeyRepo repository.ExternalAPIKeyRepository,
	logRepo repository.ExternalOrderLogRepository,
	signValidator signature.SignatureHandler,
	retryService *service.RetryService,
	productRepo repository.ProductRepository,
	queue queue.Queue,
) *ExternalCallbackController {
	return &ExternalCallbackController{
		orderService:        orderService,
		unifiedOrderService: unifiedOrderService,
		apiKeyRepo:          apiKeyRepo,
		logRepo:             logRepo,
		signValidator:       signValidator,
		retryService:        retryService,
		productRepo:         productRepo,
		queue:               queue,
	}
}

// CallbackRequest 回调请求结构
type CallbackRequest struct {
	AppID       string `json:"app_id" binding:"required"`
	OutTradeNum string `json:"out_trade_num" binding:"required"`
	Status      int    `json:"status" binding:"required"`
	Timestamp   int64  `json:"timestamp" binding:"required"`
	Nonce       string `json:"nonce" binding:"required"`
	Sign        string `json:"sign" binding:"required"`
}

// CallbackResponse 回调响应结构
type CallbackResponse struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}

// HandleCallback 处理外部系统回调
func (c *ExternalCallbackController) HandleCallback(ctx *gin.Context) {
	startTime := time.Now()
	var req CallbackRequest
	var logData model.ExternalOrderLog

	// 获取客户端IP
	_ = getClientIP(ctx)

	// 解析请求参数
	if err := ctx.ShouldBindJSON(&req); err != nil {
		c.respondCallbackError(ctx, http.StatusBadRequest, "Invalid request parameters", &logData, startTime)
		return
	}

	// 基于请求上下文注入订单号（先用 out_trade_num 兜底，稍后查到内部订单再覆盖）
	baseCtx := ctx.Request.Context()
	ctxWithOrder := log.InjectOrderNumber(baseCtx, req.OutTradeNum)
	log.WithContextCategory(ctxWithOrder, "callback").Info("收到外部回调请求",
		log.StringV2("app_id", req.AppID),
		log.StringV2("out_trade_num", req.OutTradeNum),
		log.IntV2("status", req.Status),
		log.Int64V2("timestamp", req.Timestamp),
	)

	// 初始化日志
	logData = model.ExternalOrderLog{
		Platform:  "internal_api",
		OrderID:   req.OutTradeNum,
		BizType:   "callback",
		Status:    0, // 默认失败
		Timestamp: time.Now().Unix(),
	}

	// 记录请求数据
	requestData, _ := json.Marshal(req)
	logData.RawData = string(requestData)

	// 验证API Key（暂时关闭）
	// apiKeyInfo, err := c.apiKeyRepo.GetByAppID(req.AppID)
	// if err != nil {
	// 	logData.ErrorMsg = fmt.Sprintf("Invalid app_id: %v", err)
	// 	c.respondCallbackError(ctx, http.StatusUnauthorized, "Invalid app_id", &logData, startTime)
	// 	return
	// }

	// 检查API Key状态
	// if !apiKeyInfo.IsActive() {
	// 	logData.ErrorMsg = "API Key is inactive or expired"
	// 	c.respondCallbackError(ctx, http.StatusUnauthorized, "API Key is inactive or expired", &logData, startTime)
	// 	return
	// }

	// 验证签名
	// params := map[string]interface{}{
	// 	"app_id":        req.AppID,
	// 	"out_trade_num": req.OutTradeNum,
	// 	"status":        strconv.Itoa(req.Status),
	// 	"timestamp":     strconv.FormatInt(req.Timestamp, 10),
	// 	"nonce":         req.Nonce,
	// }

	// 添加调试日志
	// logger.Info("接收端签名验证参数",
	// 	"app_id", req.AppID,
	// 	"out_trade_num", req.OutTradeNum,
	// 	"status", req.Status,
	// 	"status_str", strconv.Itoa(req.Status),
	// 	"timestamp", req.Timestamp,
	// 	"timestamp_str", strconv.FormatInt(req.Timestamp, 10),
	// 	"nonce", req.Nonce,
	// 	"received_sign", req.Sign,
	// 	"app_secret_length", len(apiKeyInfo.AppSecret),
	// 	"params_count", len(params),
	// )

	// if err := c.signValidator.ValidateExternalAPISignature(params, req.Sign, apiKeyInfo.AppSecret); err != nil {
	// 	logData.ErrorMsg = fmt.Sprintf("Signature validation failed: %v", err)
	// 	logger.Error("签名验证失败详细信息",
	// 		"error", err,
	// 		"received_sign", req.Sign,
	// 		"app_secret_length", len(apiKeyInfo.AppSecret),
	// 		"params", params,
	// 	)
	// 	c.respondCallbackError(ctx, http.StatusUnauthorized, "Signature validation failed", &logData, startTime)
	// 	return
	// }

	// 查询订单
	order, err := c.orderService.GetOrderByOutTradeNum(ctxWithOrder, req.OutTradeNum)
	if err != nil {
		logData.ErrorMsg = fmt.Sprintf("Order not found: %v", err)
		c.respondCallbackError(ctx, http.StatusNotFound, "Order not found", &logData, startTime)
		return
	}

	// 更新日志信息
	logData.OrderID = strconv.FormatInt(order.ID, 10)
	logData.GoodsID = order.ProductID
	logData.Amount = order.TotalPrice

	// 检查订单状态是否需要更新
	if int(order.Status) == req.Status {
		// 状态未变更，但如果是成功状态，仍需要确保通知已发送
		if req.Status == int(model.OrderStatusSuccess) {
			logger.WithContextCategory(ctxWithOrder, "callback").Info("订单状态未变更但为成功状态，直接创建通知记录",
				logger.Int64V2("order_id", order.ID),
				logger.IntV2("status", req.Status))
			// 对于成功状态，即使状态未变更也要直接创建通知记录
			if err := c.createNotificationForUnchangedSuccessOrder(ctxWithOrder, order); err != nil {
				logData.ErrorMsg = fmt.Sprintf("Create notification failed: %v", err)
				c.respondCallbackError(ctx, http.StatusInternalServerError, "Create notification failed", &logData, startTime)
				return
			}
		} else {
			logger.WithContextCategory(ctxWithOrder, "callback").Info("订单状态未变更，直接返回成功",
				logger.Int64V2("order_id", order.ID),
				logger.IntV2("status", req.Status))
		}
		logData.Status = 1
		c.respondCallbackSuccess(ctx, "Status unchanged", &logData)
		return
	}

	// 处理订单状态更新
	if err := c.handleOrderStatusUpdate(ctxWithOrder, order, model.OrderStatus(req.Status), req.OutTradeNum); err != nil {
		logData.ErrorMsg = fmt.Sprintf("Update order status failed: %v", err)
		c.respondCallbackError(ctx, http.StatusInternalServerError, "Update order status failed", &logData, startTime)
		return
	}

	// 成功响应
	logData.Status = 1
	c.respondCallbackSuccess(ctx, "Success", &logData)
}

// handleOrderStatusUpdate 处理订单状态更新，失败时检查是否有其他可用通道进行重试
func (c *ExternalCallbackController) handleOrderStatusUpdate(ctx context.Context, order *model.Order, newStatus model.OrderStatus, outTradeNum string) error {
	logger.WithContextCategory(ctx, "callback").Info("处理订单状态更新",
		logger.Int64V2("order_id", order.ID),
		logger.StringV2("order_number", order.OrderNumber),
		logger.Int64V2("product_id", order.ProductID),
		logger.IntV2("old_status", int(order.Status)),
		logger.IntV2("new_status", int(newStatus)))

	// 如果是失败状态，检查是否有其他可用通道进行重试
	if newStatus == model.OrderStatusFailed {
		return c.handleFailedOrderWithRetry(ctx, order, outTradeNum)
	}

	// 非失败状态（成功、处理中等），直接更新状态并发送通知
	if c.unifiedOrderService != nil {
		return c.unifiedOrderService.ProcessOrderStatusChange(ctx, order.ID, newStatus, "external")
	} else {
		// 统一订单服务未初始化时，使用原有逻辑但确保通知发送
		logger.WithContextCategory(ctx, "callback").Warn("统一订单服务未初始化，使用原有的简单状态更新", logger.Int64V2("order_id", order.ID))
		err := c.orderService.UpdateOrderStatus(ctx, order.ID, newStatus)
		if err != nil {
			logger.WithContextCategory(ctx, "callback").Error("订单状态更新失败", logger.Int64V2("order_id", order.ID), logger.ErrorV2(err))
			return err
		}

		// 确保通知发送成功，如果 orderService.UpdateOrderStatus 的通知发送失败，
		// 这里可以添加额外的通知发送逻辑作为备用方案
		logger.WithContextCategory(ctx, "callback").Info("订单状态更新完成，通知已推送到队列", logger.Int64V2("order_id", order.ID), logger.IntV2("new_status", int(newStatus)))
		return nil
	}
}

// handleFailedOrderWithRetry 处理失败订单，检查是否有其他可用通道进行重试
func (c *ExternalCallbackController) handleFailedOrderWithRetry(ctx context.Context, order *model.Order, outTradeNum string) error {
	logger.WithContextCategory(ctx, "callback").Info("处理失败订单，检查是否有其他可用通道",
		logger.Int64V2("order_id", order.ID),
		logger.StringV2("order_number", order.OrderNumber),
		logger.Int64V2("product_id", order.ProductID),
		logger.StringV2("used_apis", order.UsedAPIs))

	// 获取商品的所有API关系
	relations, err := c.productRepo.GetAPIRelationsByProductID(ctx, order.ProductID)
	if err != nil {
		logger.WithContextCategory(ctx, "callback").Error("获取API关系失败", logger.Int64V2("order_id", order.ID), logger.ErrorV2(err))
		return err
	}

	logger.WithContextCategory(ctx, "callback").Info("获取到的API关系列表",
		logger.Int64V2("order_id", order.ID),
		logger.IntV2("relations_count", len(relations)))

	// 解析已使用的API列表
	var usedAPIs []map[string]interface{}
	if order.UsedAPIs != "" {
		if err := json.Unmarshal([]byte(order.UsedAPIs), &usedAPIs); err != nil {
			logger.WithContextCategory(ctx, "callback").Error("解析已使用API列表失败", logger.Int64V2("order_id", order.ID), logger.ErrorV2(err))
			usedAPIs = []map[string]interface{}{}
		}
	}

	// 检查是否还有未使用的API
	hasAvailableAPI := false
	logger.WithContextCategory(ctx, "callback").Info("开始检查可用API",
		logger.Int64V2("order_id", order.ID),
		logger.IntV2("total_relations", len(relations)),
		logger.IntV2("used_apis_count", len(usedAPIs)))

	// 记录当前通道关系和同通道配置
	type curRelInfo struct {
		APIID                   int64
		SameChannelRetryEnabled bool
		SameChannelRetryTimes   int
	}
	var curRel *curRelInfo
	currentAPIID := order.APICurID
	if currentAPIID == 0 {
		currentAPIID = order.APIID
	}

	for _, relation := range relations {
		alreadyUsed := false

		// 兜底保护：排除当前订单正在使用的通道（无论 UsedAPIs 是否已写入）
		if relation.APIID == order.APICurID || relation.APIID == order.APIID {
			alreadyUsed = true
			logger.WithContextCategory(ctx, "callback").Info("当前订单正在使用的API，默认排除",
				logger.Int64V2("order_id", order.ID),
				logger.Int64V2("api_id", relation.APIID),
				logger.Int64V2("api_cur_id", order.APICurID),
				logger.Int64V2("api_id_field", order.APIID))
		}

		// 记录当前通道的关系配置
		if relation.APIID == currentAPIID {
			curRel = &curRelInfo{
				APIID:                   relation.APIID,
				SameChannelRetryEnabled: relation.SameChannelRetryEnabled,
				SameChannelRetryTimes:   relation.SameChannelRetryTimes,
			}
		}

		if !alreadyUsed {
			for _, usedAPI := range usedAPIs {
				if apiID, ok := usedAPI["api_id"].(float64); ok && int64(apiID) == relation.APIID {
					alreadyUsed = true
					logger.WithContextCategory(ctx, "callback").Info("API已被使用",
						logger.Int64V2("order_id", order.ID),
						logger.Int64V2("api_id", relation.APIID))
					break
				}
			}
		}

		if !alreadyUsed {
			logger.WithContextCategory(ctx, "callback").Info("发现可用API",
				logger.Int64V2("order_id", order.ID),
				logger.Int64V2("api_id", relation.APIID))
			hasAvailableAPI = true
			break
		}
	}

	logger.WithContextCategory(ctx, "callback").Info("API可用性检查结果",
		logger.Int64V2("order_id", order.ID),
		logger.BoolV2("has_available_api", hasAvailableAPI))

	if hasAvailableAPI {
		// 有可用通道，推送重试任务到消息队列，不更新订单状态，不发送通知
		logger.WithContextCategory(ctx, "callback").Info("发现可用通道，推送重试任务到队列，暂不发送失败通知", logger.Int64V2("order_id", order.ID))
		if err := c.pushRetryTaskToQueue(ctx, order.ID, 2, "外部回调失败，切换通道重试"); err != nil {
			logger.WithContextCategory(ctx, "callback").Error("推送重试任务到队列失败", logger.Int64V2("order_id", order.ID), logger.ErrorV2(err))
			// 推送失败，仍然更新订单状态为失败并发送通知
			logger.WithContextCategory(ctx, "callback").Info("推送重试任务失败，执行最终失败处理", logger.Int64V2("order_id", order.ID))
			return c.handleAllRetriesCompleted(ctx, order)
		}
		// 推送成功，不更新订单状态为失败，也不发送通知，等待重试结果
		logger.WithContextCategory(ctx, "callback").Info("重试任务推送成功，等待重试处理，暂不发送失败通知", logger.Int64V2("order_id", order.ID))
		return nil
	} else {
		// 没有可用通道：根据 outTradeNum 的 -rN 后缀与关系配置判定是否继续同通道重试
		attemptNo := 0
		if i := strings.LastIndex(outTradeNum, "-r"); i >= 0 && i+2 < len(outTradeNum) {
			if n, err := strconv.Atoi(outTradeNum[i+2:]); err == nil {
				attemptNo = n
			}
		}
		// 从 curRel 中读取开关与上限
		sameEnabled := false
		maxTimes := 0
		if curRel != nil {
			sameEnabled = curRel.SameChannelRetryEnabled
			maxTimes = curRel.SameChannelRetryTimes
		}

		logger.WithContextCategory(ctx, "callback").Info("同通道重试判定",
			logger.Int64V2("order_id", order.ID),
			logger.StringV2("out_trade_num", outTradeNum),
			logger.IntV2("attempt_no", attemptNo),
			logger.BoolV2("same_channel_enabled", sameEnabled),
			logger.IntV2("same_channel_max_times", maxTimes))

		// 若未开启同通道或已达到上限，则直接做最终失败处理
		if !sameEnabled {
			logger.WithContextCategory(ctx, "callback").Info("当前通道未开启同通道重试，执行最终失败处理", logger.Int64V2("order_id", order.ID))
			return c.handleAllRetriesCompleted(ctx, order)
		}
		if attemptNo >= maxTimes && maxTimes > 0 {
			logger.WithContextCategory(ctx, "callback").Info("同通道重试次数已达上限，执行最终失败处理",
				logger.Int64V2("order_id", order.ID),
				logger.IntV2("attempt_no", attemptNo),
				logger.IntV2("max_times", maxTimes))
			return c.handleAllRetriesCompleted(ctx, order)
		}

		// 仍可继续同通道重试
		logger.WithContextCategory(ctx, "callback").Info("没有可用通道，改为同通道重试，推送重试任务到队列", logger.Int64V2("order_id", order.ID))
		if err := c.pushRetryTaskToQueue(ctx, order.ID, model.RetryTypeSameChannel, "外部回调失败，改为同通道重试"); err != nil {
			logger.WithContextCategory(ctx, "callback").Error("推送同通道重试任务到队列失败，执行最终失败处理", logger.Int64V2("order_id", order.ID), logger.ErrorV2(err))
			return c.handleAllRetriesCompleted(ctx, order)
		}
		logger.WithContextCategory(ctx, "callback").Info("同通道重试任务推送成功，等待重试处理，暂不发送失败通知", logger.Int64V2("order_id", order.ID))
		return nil
	}
}

// pushRetryTaskToQueue 推送重试任务到队列
func (c *ExternalCallbackController) pushRetryTaskToQueue(ctx context.Context, orderID int64, retryType int, reason string) error {
	task := model.NewRetryTaskMessage(orderID, retryType, reason)

	// 使用默认队列名称，也可以从配置中读取
	queueName := "retry_queue"
	if err := c.queue.Push(ctx, queueName, task); err != nil {
		return fmt.Errorf("推送重试任务到队列失败: %v", err)
	}

	logger.WithContextCategory(ctx, "callback").Info("重试任务推送到队列成功", logger.Int64V2("order_id", orderID), logger.IntV2("retry_type", retryType), logger.StringV2("queue_name", queueName))
	return nil
}

// createRetryRecord 创建重试记录
func (c *ExternalCallbackController) createRetryRecord(ctx context.Context, order *model.Order, nextAPIID int64, usedAPIs []int64) error {
	// 更新已使用的API列表
	updatedUsedAPIs := append(usedAPIs, nextAPIID)
	usedAPIListBytes, err := json.Marshal(updatedUsedAPIs)
	if err != nil {
		return fmt.Errorf("序列化已使用API列表失败: %v", err)
	}

	// 创建重试记录
	retryRecord := &model.OrderRetryRecord{
		OrderID:       order.ID,
		APIID:         nextAPIID,
		ParamID:       0, // 暂时设为0，后续在重试时会更新
		RetryType:     2, // 2: 外部回调重试
		RetryCount:    1,
		LastError:     "外部回调失败，切换通道重试",
		RetryParams:   "", // 可以根据需要设置重试参数
		UsedAPIs:      string(usedAPIListBytes),
		Status:        0,          // 0: 待处理
		NextRetryTime: time.Now(), // 立即重试
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// 直接调用重试仓库创建记录
	if err := c.retryService.CreateRetryRecord(ctx, retryRecord); err != nil {
		return fmt.Errorf("创建重试记录失败: %v", err)
	}

	logger.WithContextCategory(ctx, "callback").Info("创建重试记录成功", logger.Int64V2("order_id", order.ID), logger.Int64V2("next_api_id", nextAPIID))
	return nil
}

// handleAllRetriesCompleted 处理所有重试完成后的通知发送
func (c *ExternalCallbackController) handleAllRetriesCompleted(ctx context.Context, order *model.Order) error {
	logger.WithContextCategory(ctx, "callback").Info("所有重试已完成，发送失败通知", logger.Int64V2("order_id", order.ID))
	return c.updateOrderStatusWithNotification(ctx, order, model.OrderStatusFailed)
}

// updateOrderStatusWithNotification 更新订单状态并发送通知
func (c *ExternalCallbackController) updateOrderStatusWithNotification(ctx context.Context, order *model.Order, newStatus model.OrderStatus) error {
	// 优先使用统一订单服务
	if c.unifiedOrderService != nil {
		return c.unifiedOrderService.ProcessOrderStatusChange(ctx, order.ID, newStatus, "external")
	} else {
		// 统一订单服务未初始化时，使用原有逻辑但确保通知发送
		logger.WithContextCategory(ctx, "callback").Warn("统一订单服务未初始化，使用原有的简单状态更新", logger.Int64V2("order_id", order.ID))
		err := c.orderService.UpdateOrderStatus(ctx, order.ID, newStatus)
		if err != nil {
			logger.WithContextCategory(ctx, "callback").Error("订单状态更新失败", logger.Int64V2("order_id", order.ID), logger.ErrorV2(err))
			return err
		}

		// 确保通知发送成功，如果 orderService.UpdateOrderStatus 的通知发送失败，
		// 这里可以添加额外的通知发送逻辑作为备用方案
		logger.WithContextCategory(ctx, "callback").Info("订单状态更新完成，通知已推送到队列", logger.Int64V2("order_id", order.ID), logger.AnyV2("new_status", newStatus))
		return nil
	}
}

// respondCallbackError 回调错误响应
func (c *ExternalCallbackController) respondCallbackError(ctx *gin.Context, statusCode int, message string, logData *model.ExternalOrderLog, startTime time.Time) {
	logData.Status = 0
	if logData.ErrorMsg == "" {
		logData.ErrorMsg = message
	}
	logData.ProcessTime = int(time.Since(startTime).Milliseconds())

	response := &CallbackResponse{
		Code:      statusCode,
		Message:   message,
		Timestamp: time.Now().Unix(),
	}

	// 记录日志到数据库
	if logData.OrderID != "" {
		if err := c.logRepo.Create(ctx, logData); err != nil {
			// 日志记录失败不影响主流程，只记录错误
			logger.WithContextCategory(ctx.Request.Context(), "external_callback").Error("创建回调错误日志失败", logger.ErrorV2(err), logger.StringV2("order_id", logData.OrderID))
		}
	}

	ctx.JSON(statusCode, response)
}

// respondCallbackSuccess 回调成功响应
func (c *ExternalCallbackController) respondCallbackSuccess(ctx *gin.Context, message string, logData *model.ExternalOrderLog) {
	logData.Status = 1 // 成功状态

	response := &CallbackResponse{
		Code:      200,
		Message:   message,
		Timestamp: time.Now().Unix(),
	}

	// 记录日志到数据库
	if logData.OrderID != "" {
		if err := c.logRepo.Create(ctx, logData); err != nil {
			// 日志记录失败不影响主流程，只记录错误
			logger.WithContextCategory(ctx.Request.Context(), "external_callback").Error("创建回调成功日志失败", logger.ErrorV2(err), logger.StringV2("order_id", logData.OrderID))
		}
	}

	ctx.JSON(http.StatusOK, response)
}

// createNotificationForUnchangedSuccessOrder 为状态未变更的成功订单创建通知记录
func (c *ExternalCallbackController) createNotificationForUnchangedSuccessOrder(ctx context.Context, order *model.Order) error {
	logger.WithContextCategory(ctx, "callback").Info("开始为状态未变更的成功订单创建通知记录", logger.Int64V2("order_id", order.ID))

	// 优先使用统一订单服务
	if c.unifiedOrderService != nil {
		return c.unifiedOrderService.ProcessOrderStatusChange(ctx, order.ID, model.OrderStatusSuccess, "external")
	} else {
		// 统一订单服务未初始化时，直接创建通知记录
		logger.WithContextCategory(ctx, "callback").Warn("统一订单服务未初始化，直接创建通知记录", logger.Int64V2("order_id", order.ID))
		return c.orderService.SendNotification(ctx, order.ID)
	}
}

// getClientIP 获取客户端真实IP（复用中间件中的函数）
func getClientIP(c *gin.Context) string {
	// 尝试从各种头部获取真实IP
	if ip := c.GetHeader("X-Forwarded-For"); ip != "" {
		// X-Forwarded-For可能包含多个IP，取第一个
		if ips := strings.Split(ip, ","); len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	if ip := c.GetHeader("X-Real-IP"); ip != "" {
		return ip
	}

	if ip := c.GetHeader("X-Original-Forwarded-For"); ip != "" {
		return ip
	}

	// 从RemoteAddr获取
	if ip, _, err := net.SplitHostPort(c.Request.RemoteAddr); err == nil {
		return ip
	}

	return c.Request.RemoteAddr
}
