package controller

import (
	"net/http"
	"recharge-go/internal/model"
	"recharge-go/internal/service"
	"recharge-go/pkg/log"
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
		Platform:  "internal_api",
		RawData:   "", // 暂时为空
		CreatedAt: startTime,
	}

	var req ExternalRefundRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		logData.ErrorMsg = "Invalid request format: " + err.Error()
		c.respondError(ctx, http.StatusBadRequest, "Invalid request format", logData, startTime)
		return
	}

	// 使用中间件注入的 API Key 信息（与下单/查询一致）
	apiKeyInfo, exists := ctx.Get("api_key_info")
	if !exists {
		c.respondError(ctx, http.StatusUnauthorized, "API Key information not found", logData, startTime)
		return
	}
	apiKey := apiKeyInfo.(*model.ExternalAPIKey)
	if req.AppID != apiKey.AppID {
		logData.ErrorMsg = "App ID mismatch"
		c.respondError(ctx, http.StatusBadRequest, "App ID mismatch", logData, startTime)
		return
	}

	// 记录API Key信息
	logData.AppKey = req.AppID
	logData.OrderID = req.OutTradeNum

	log.Info(ctx, "external_refund_request",
		log.String("app_id", req.AppID),
		log.String("out_trade_num", req.OutTradeNum),
		log.String("reason", req.Reason))

	// 根据外部交易号获取订单
	order, err := c.orderService.GetOrderByOutTradeNum(ctx, req.OutTradeNum)
	if err != nil {
		c.respondError(ctx, http.StatusNotFound, "订单不存在", logData, startTime)
		return
	}

	// 校验订单归属：仅允许该订单所属用户（下单时使用的 API Key 对应用户）发起退款
	if order.CustomerID != apiKey.UserID {
		logData.ErrorMsg = "order does not belong to this app"
		c.respondError(ctx, http.StatusForbidden, "无权限操作该订单", logData, startTime)
		return
	}

	// 申请退款（待充值订单变为待审核，不直接退款；管理员审核通过后才执行退款）
	err = c.orderService.ProcessExternalRefund(ctx, req.OutTradeNum, req.Reason)
	if err != nil {
		c.respondError(ctx, http.StatusInternalServerError, err.Error(), logData, startTime)
		return
	}

	// 重新获取订单以得到更新后的状态
	order, _ = c.orderService.GetOrderByOutTradeNum(ctx, req.OutTradeNum)
	if order != nil {
		logData.OrderID = order.OrderNumber
		logData.Mobile = order.Mobile
		logData.Amount = order.Price
	}
	logData.Status = 1
	logData.UpdatedAt = time.Now()

	statusStr := "pending_review"
	if order != nil && order.Status == model.OrderStatusRefunded {
		statusStr = "refunded"
	}

	var data *ExternalRefundData
	if order != nil {
		data = &ExternalRefundData{
			OrderNumber: order.OrderNumber,
			OutTradeNum: order.OutTradeNum,
			Amount:      order.Price,
			Status:      statusStr,
		}
	} else {
		data = &ExternalRefundData{OutTradeNum: req.OutTradeNum, Status: statusStr}
	}
	response := ExternalRefundResponse{
		Code:    200,
		Message: "退款申请已提交，待管理员审核",
		Data:    data,
	}

	log.Info(ctx, "external_refund_apply_success",
		log.String("out_trade_num", req.OutTradeNum),
		log.String("status", statusStr))

	ctx.JSON(http.StatusOK, response)
}

func (c *ExternalRefundController) respondError(ctx *gin.Context, statusCode int, message string, logData *model.ExternalOrderLog, startTime time.Time) {
	logData.Status = 0
	logData.UpdatedAt = time.Now()

	response := ExternalRefundResponse{
		Code:    statusCode,
		Message: message,
	}

	log.Error(ctx, "external_refund_failed",
		log.Int("status_code", statusCode),
		log.String("message", message),
		log.String("error", logData.ErrorMsg))

	ctx.JSON(statusCode, response)
}

