package controller

import (
	"net/http"
	"recharge-go/internal/model"
	"recharge-go/internal/service"
	"recharge-go/pkg/logger"
	"time"

	"github.com/gin-gonic/gin"
)

type ExternalRefundController struct {
	orderService service.OrderService
}

func NewExternalRefundController(orderService service.OrderService) *ExternalRefundController {
	return &ExternalRefundController{orderService: orderService}
}

// ExternalRefundRequest 外部退款请求
type ExternalRefundRequest struct {
	AppID       string `json:"app_id" binding:"required"`        // 应用ID
	OutTradeNum string `json:"out_trade_num" binding:"required"` // 外部交易号
	Reason      string `json:"reason"`                           // 退款原因
	Timestamp   int64  `json:"timestamp" binding:"required"`     // 时间戳
	Nonce       string `json:"nonce" binding:"required"`         // 随机字符串
	Sign        string `json:"sign" binding:"required"`          // 签名
}

// ExternalRefundResponse 外部退款响应
type ExternalRefundResponse struct {
	Code    int                 `json:"code"`
	Message string              `json:"message"`
	Data    *ExternalRefundData `json:"data,omitempty"`
}

type ExternalRefundData struct {
	OrderNumber string  `json:"order_number"`  // 系统订单号
	OutTradeNum string  `json:"out_trade_num"` // 外部交易号
	Amount      float64 `json:"amount"`        // 退款金额
	Status      string  `json:"status"`        // 退款状态
}

// ProcessRefund 处理外部订单退款
func (c *ExternalRefundController) ProcessRefund(ctx *gin.Context) {
	startTime := time.Now()

	// 创建日志记录
	logData := &model.ExternalOrderLog{
		Platform:  "external_api",
		RawData:   "", // 暂时为空
		CreatedAt: startTime,
	}

	var req ExternalRefundRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		logData.ErrorMsg = "Invalid request format: " + err.Error()
		c.respondError(ctx, http.StatusBadRequest, "Invalid request format", logData, startTime)
		return
	}

	// 验证API Key
	_, err := c.validateAPIKey(ctx, req.AppID)
	if err != nil {
		c.respondError(ctx, http.StatusUnauthorized, "无效的API Key", logData, startTime)
		return
	}

	// 记录API Key信息
	logData.AppKey = req.AppID
	logData.OrderID = req.OutTradeNum

	logger.Info("收到外部退款请求",
		"app_id", req.AppID,
		"out_trade_num", req.OutTradeNum,
		"reason", req.Reason)

	// 根据外部交易号获取订单
	order, err := c.orderService.GetOrderByOutTradeNum(ctx, req.OutTradeNum)
	if err != nil {
		c.respondError(ctx, http.StatusNotFound, "订单不存在", logData, startTime)
		return
	}

	// 处理退款
	err = c.orderService.ProcessExternalRefund(ctx, req.OutTradeNum, req.Reason)
	if err != nil {
		c.respondError(ctx, http.StatusInternalServerError, "退款处理失败: "+err.Error(), logData, startTime)
		return
	}

	// 更新日志记录
	logData.OrderID = order.OrderNumber
	logData.Mobile = order.Mobile
	logData.Amount = order.Price
	logData.Status = 1

	// 记录处理时间
	logData.UpdatedAt = time.Now()

	// 构造响应数据
	response := ExternalRefundResponse{
		Code:    200,
		Message: "success",
		Data: &ExternalRefundData{
			OrderNumber: order.OrderNumber,
			OutTradeNum: order.OutTradeNum,
			Amount:      order.Price,
			Status:      "refunded",
		},
	}

	logger.Info("外部退款处理成功",
		"order_id", order.ID,
		"order_number", order.OrderNumber,
		"out_trade_num", order.OutTradeNum,
		"amount", order.Price)

	ctx.JSON(http.StatusOK, response)
}

func (c *ExternalRefundController) respondError(ctx *gin.Context, statusCode int, message string, logData *model.ExternalOrderLog, startTime time.Time) {
	logData.Status = 0
	logData.UpdatedAt = time.Now()

	response := ExternalRefundResponse{
		Code:    statusCode,
		Message: message,
	}

	logger.Error("外部退款请求失败",
		"status_code", statusCode,
		"message", message,
		"error", logData.ErrorMsg)

	ctx.JSON(statusCode, response)
}

// validateAPIKey 验证API Key
func (c *ExternalRefundController) validateAPIKey(ctx *gin.Context, appID string) (*model.ExternalAPIKey, error) {
	// 这里应该实现API Key验证逻辑
	// 暂时返回一个模拟的API Key对象
	return &model.ExternalAPIKey{
		AppID:  appID,
		UserID: 1, // 模拟用户ID
	}, nil
}
