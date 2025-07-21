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
	"recharge-go/pkg/logger"
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

	// 初始化日志
	logData = model.ExternalOrderLog{
		Platform:  "external_api",
		OrderID:   req.OutTradeNum,
		BizType:   "callback",
		Status:    0, // 默认失败
		Timestamp: time.Now().Unix(),
	}

	// 记录请求数据
	requestData, _ := json.Marshal(req)
	logData.RawData = string(requestData)

	// 验证API Key
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
	order, err := c.orderService.GetOrderByOutTradeNum(ctx, req.OutTradeNum)

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
		// 状态未变更，直接返回成功
		logData.Status = 1
		c.respondCallbackSuccess(ctx, "Status unchanged", &logData)
		return
	}

	// 处理订单状态更新
	if err := c.handleOrderStatusUpdate(ctx, order, model.OrderStatus(req.Status)); err != nil {
		logData.ErrorMsg = fmt.Sprintf("Update order status failed: %v", err)
		c.respondCallbackError(ctx, http.StatusInternalServerError, "Update order status failed", &logData, startTime)
		return
	}

	// 成功响应
	logData.Status = 1
	c.respondCallbackSuccess(ctx, "Success", &logData)
}

// handleOrderStatusUpdate 处理订单状态更新，失败时检查是否有其他可用通道
func (c *ExternalCallbackController) handleOrderStatusUpdate(ctx context.Context, order *model.Order, newStatus model.OrderStatus) error {
	// 如果是失败状态，检查是否有其他可用通道进行重试
	if newStatus == model.OrderStatusFailed {
		return c.handleFailedOrderWithRetry(ctx, order)
	}

	// 非失败状态，使用原有逻辑
	if c.unifiedOrderService != nil {
		return c.unifiedOrderService.ProcessOrderStatusChange(ctx, order.ID, newStatus, "external")
	} else {
		logger.Warn("统一订单服务未初始化，使用原有的简单状态更新", "order_id", order.ID)
		return c.orderService.UpdateOrderStatus(ctx, order.ID, newStatus)
	}
}

// handleFailedOrderWithRetry 处理失败订单，检查是否有其他可用通道进行重试
func (c *ExternalCallbackController) handleFailedOrderWithRetry(ctx context.Context, order *model.Order) error {
	logger.Info("处理失败订单，检查是否有其他可用通道",
		"order_id", order.ID,
		"order_number", order.OrderNumber)

	// 获取商品的所有API关系
	relations, err := c.productRepo.GetAPIRelationsByProductID(ctx, order.ProductID)
	if err != nil {
		logger.Error("获取API关系失败", "order_id", order.ID, "error", err)
		return err
	}

	// 解析已使用的API列表
	var usedAPIs []map[string]interface{}
	if order.UsedAPIs != "" {
		if err := json.Unmarshal([]byte(order.UsedAPIs), &usedAPIs); err != nil {
			logger.Error("解析已使用API列表失败", "order_id", order.ID, "error", err)
			usedAPIs = []map[string]interface{}{}
		}
	}

	// 检查是否还有未使用的API
	hasAvailableAPI := false
	for _, relation := range relations {
		alreadyUsed := false
		for _, usedAPI := range usedAPIs {
			if apiID, ok := usedAPI["api_id"].(float64); ok && int64(apiID) == relation.APIID {
				alreadyUsed = true
				break
			}
		}
		if !alreadyUsed {
			hasAvailableAPI = true
			break
		}
	}

	if hasAvailableAPI {
		// 有可用通道，推送重试任务到消息队列
		logger.Info("发现可用通道，推送重试任务到队列", "order_id", order.ID)
		if err := c.pushRetryTaskToQueue(ctx, order.ID, 2, "外部回调失败，切换通道重试"); err != nil {
			logger.Error("推送重试任务到队列失败", "order_id", order.ID, "error", err)
			// 推送失败，仍然更新订单状态为失败并发送通知
			return c.handleAllRetriesCompleted(ctx, order)
		}
		// 推送成功，不更新订单状态为失败，也不发送通知
		return nil
	} else {
		// 没有可用通道，更新订单状态为失败并发送通知
		logger.Info("没有可用通道，订单最终失败", "order_id", order.ID)
		return c.handleAllRetriesCompleted(ctx, order)
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

	logger.Info("重试任务推送到队列成功", "order_id", orderID, "retry_type", retryType, "queue_name", queueName)
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
		Status:        0, // 0: 待处理
		NextRetryTime: time.Now(), // 立即重试
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// 直接调用重试仓库创建记录
	if err := c.retryService.CreateRetryRecord(ctx, retryRecord); err != nil {
		return fmt.Errorf("创建重试记录失败: %v", err)
	}

	logger.Info("创建重试记录成功", "order_id", order.ID, "next_api_id", nextAPIID)
	return nil
}

// handleAllRetriesCompleted 处理所有重试完成后的通知发送
func (c *ExternalCallbackController) handleAllRetriesCompleted(ctx context.Context, order *model.Order) error {
	logger.Info("所有重试已完成，发送失败通知", "order_id", order.ID)
	return c.updateOrderStatusWithNotification(ctx, order, model.OrderStatusFailed)
}

// updateOrderStatusWithNotification 更新订单状态并发送通知
func (c *ExternalCallbackController) updateOrderStatusWithNotification(ctx context.Context, order *model.Order, newStatus model.OrderStatus) error {
	if c.unifiedOrderService != nil {
		return c.unifiedOrderService.ProcessOrderStatusChange(ctx, order.ID, newStatus, "external")
	} else {
		logger.Warn("统一订单服务未初始化，使用原有的简单状态更新", "order_id", order.ID)
		return c.orderService.UpdateOrderStatus(ctx, order.ID, newStatus)
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
			fmt.Printf("Failed to create callback error log: %v\n", err)
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
			fmt.Printf("Failed to create callback success log: %v\n", err)
		}
	}

	ctx.JSON(http.StatusOK, response)
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
