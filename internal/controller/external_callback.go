package controller

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"recharge-go/internal/model"
	"recharge-go/internal/repository"
	"recharge-go/internal/service"
	"recharge-go/pkg/logger"
	"recharge-go/pkg/signature"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ExternalCallbackController 外部回调控制器
type ExternalCallbackController struct {
	orderService  service.OrderService
	apiKeyRepo    repository.ExternalAPIKeyRepository
	logRepo       repository.ExternalOrderLogRepository
	signValidator *signature.ExternalAPISignatureValidator
}

// NewExternalCallbackController 创建外部回调控制器
func NewExternalCallbackController(
	orderService service.OrderService,
	apiKeyRepo repository.ExternalAPIKeyRepository,
	logRepo repository.ExternalOrderLogRepository,
) *ExternalCallbackController {
	return &ExternalCallbackController{
		orderService:  orderService,
		apiKeyRepo:    apiKeyRepo,
		logRepo:       logRepo,
		signValidator: signature.NewExternalAPISignatureValidator(),
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
	apiKeyInfo, err := c.apiKeyRepo.GetByAppID(req.AppID)
	if err != nil {
		logData.ErrorMsg = fmt.Sprintf("Invalid app_id: %v", err)
		c.respondCallbackError(ctx, http.StatusUnauthorized, "Invalid app_id", &logData, startTime)
		return
	}

	// 检查API Key状态
	if !apiKeyInfo.IsActive() {
		logData.ErrorMsg = "API Key is inactive or expired"
		c.respondCallbackError(ctx, http.StatusUnauthorized, "API Key is inactive or expired", &logData, startTime)
		return
	}

	// 验证签名
	params := map[string]interface{}{
		"app_id":        req.AppID,
		"out_trade_num": req.OutTradeNum,
		"status":        strconv.Itoa(req.Status),
		"timestamp":     strconv.FormatInt(req.Timestamp, 10),
		"nonce":         req.Nonce,
	}

	// 添加调试日志
	logger.Info("接收端签名验证参数",
		"app_id", req.AppID,
		"out_trade_num", req.OutTradeNum,
		"status", req.Status,
		"status_str", strconv.Itoa(req.Status),
		"timestamp", req.Timestamp,
		"timestamp_str", strconv.FormatInt(req.Timestamp, 10),
		"nonce", req.Nonce,
		"received_sign", req.Sign,
		"app_secret_length", len(apiKeyInfo.AppSecret),
		"params_count", len(params),
	)

	if err := c.signValidator.ValidateExternalAPISignature(params, req.Sign, apiKeyInfo.AppSecret); err != nil {
		logData.ErrorMsg = fmt.Sprintf("Signature validation failed: %v", err)
		logger.Error("签名验证失败详细信息",
			"error", err,
			"received_sign", req.Sign,
			"app_secret_length", len(apiKeyInfo.AppSecret),
			"params", params,
		)
		c.respondCallbackError(ctx, http.StatusUnauthorized, "Signature validation failed", &logData, startTime)
		return
	}

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

	// 更新订单状态
	if err := c.orderService.UpdateOrderStatus(ctx, order.ID, model.OrderStatus(req.Status)); err != nil {
		logData.ErrorMsg = fmt.Sprintf("Update order status failed: %v", err)
		c.respondCallbackError(ctx, http.StatusInternalServerError, "Update order status failed", &logData, startTime)
		return
	}

	// 成功响应
	logData.Status = 1
	c.respondCallbackSuccess(ctx, "Success", &logData)
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
