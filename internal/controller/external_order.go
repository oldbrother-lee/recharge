package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"recharge-go/internal/model"
	"recharge-go/internal/repository"
	"recharge-go/internal/service"
	"recharge-go/pkg/logger"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ExternalOrderController struct {
	orderService   service.OrderService
	productService *service.ProductService
	logRepo        repository.ExternalOrderLogRepository
}

func NewExternalOrderController(orderService service.OrderService, productService *service.ProductService, logRepo repository.ExternalOrderLogRepository) *ExternalOrderController {
	return &ExternalOrderController{
		orderService:   orderService,
		productService: productService,
		logRepo:        logRepo,
	}
}

// ExternalOrderCreateRequest 外部订单创建请求
type ExternalOrderCreateRequest struct {
	AppID       string  `json:"app_id" binding:"required"`        // 应用ID
	Mobile      string  `json:"mobile" binding:"required"`        // 手机号
	ProductID   int64   `json:"product_id" binding:"required"`    // 产品ID
	OutTradeNum string  `json:"out_trade_num" binding:"required"` // 外部交易号
	Amount      float64 `json:"amount" binding:"required"`        // 金额
	BizType     string  `json:"biz_type"`                         // 业务类型
	NotifyURL   string  `json:"notify_url"`                       // 回调通知URL
	Param1      string  `json:"param1"`                           // 扩展参数1
	Param2      string  `json:"param2"`                           // 扩展参数2
	Param3      string  `json:"param3"`                           // 扩展参数3
	CustomerID  int64   `json:"customer_id"`                      // 外部客户ID
	ISP         int     `json:"isp"`                              // 运营商
	Remark      string  `json:"remark"`                           // 备注
	Timestamp   int64   `json:"timestamp" binding:"required"`     // 时间戳
	Nonce       string  `json:"nonce" binding:"required"`         // 随机字符串

	Sign string `json:"sign" binding:"required"` // 签名
}

// ExternalOrderCreateResponse 外部订单创建响应
type ExternalOrderCreateResponse struct {
	Code      int                `json:"code"`
	Message   string             `json:"message"`
	Data      *ExternalOrderData `json:"data,omitempty"`
	Timestamp int64              `json:"timestamp"`
}

// ExternalOrderData 外部订单数据
type ExternalOrderData struct {
	OrderNumber string  `json:"order_number"`
	OutTradeNum string  `json:"out_trade_num"`
	Status      int     `json:"status"`
	StatusDesc  string  `json:"status_desc"`
	Amount      float64 `json:"amount"`
	CreateTime  int64   `json:"create_time"`
}

// ExternalOrderQueryRequest 外部订单查询请求
type ExternalOrderQueryRequest struct {
	AppID       string `json:"app_id" binding:"required"`
	OutTradeNum string `json:"out_trade_num"`
	OrderNumber string `json:"order_number"`
	Timestamp   int64  `json:"timestamp" binding:"required"`
	Nonce       string `json:"nonce" binding:"required"`

	Sign string `json:"sign" binding:"required"`
}

// CreateOrder 创建外部订单
func (c *ExternalOrderController) CreateOrder(ctx *gin.Context) {
	startTime := time.Now()
	var req ExternalOrderCreateRequest
	var logData model.ExternalOrderLog

	// 获取API Key信息
	apiKeyInfo, exists := ctx.Get("api_key_info")
	if !exists {
		c.respondError(ctx, http.StatusUnauthorized, "API Key information not found", &logData, startTime)
		return
	}
	apiKey := apiKeyInfo.(*model.ExternalAPIKey)

	// 初始化日志
	logData = model.ExternalOrderLog{
		Platform:  "external_api",
		BizType:   "create_order",
		Status:    0, // 默认失败
		Timestamp: time.Now().Unix(),
	}

	// 解析请求参数
	if err := ctx.ShouldBindJSON(&req); err != nil {
		logData.ErrorMsg = fmt.Sprintf("请求参数错误 %v", err)
		c.respondError(ctx, http.StatusBadRequest, "请求参数错误", &logData, startTime)
		logger.Error(fmt.Sprintf("请求参数错误: %v", err))
		return
	}

	// 记录请求数据
	requestData, _ := json.Marshal(req)
	logData.RawData = string(requestData)
	logData.OrderID = req.OutTradeNum

	// 验证应用ID
	if req.AppID != apiKey.AppID {
		logData.ErrorMsg = "App ID mismatch"
		c.respondError(ctx, http.StatusBadRequest, "App ID mismatch", &logData, startTime)
		return
	}

	// 检查外部交易号是否已存在
	existingOrder, err := c.orderService.GetOrderByOutTradeNum(ctx, req.OutTradeNum)
	if err != nil && err != gorm.ErrRecordNotFound {
		logData.ErrorMsg = fmt.Sprintf("Database error: %v", err)
		c.respondError(ctx, http.StatusInternalServerError, "Database error！！！", &logData, startTime)
		logger.Error(fmt.Sprintf("Database error: %v", err))
		return
	}
	if existingOrder != nil {
		// 订单已存在，返回现有订单信息
		logData.OrderID = strconv.FormatInt(existingOrder.ID, 10)
		logData.GoodsID = existingOrder.ProductID
		logData.Amount = existingOrder.TotalPrice
		logData.Status = 1

		response := &ExternalOrderCreateResponse{
			Code:      200,
			Message:   "Order already exists",
			Timestamp: time.Now().Unix(),
			Data: &ExternalOrderData{

				OrderNumber: existingOrder.OrderNumber,
				OutTradeNum: existingOrder.OutTradeNum,
				Status:      int(existingOrder.Status),
				StatusDesc:  c.getStatusDesc(int(existingOrder.Status)),
				Amount:      existingOrder.TotalPrice,
				CreateTime:  existingOrder.CreateTime.Unix(),
			},
		}

		// 记录成功响应
		logData.Status = 1
		logData.OrderID = strconv.FormatInt(existingOrder.ID, 10)
		logData.GoodsID = existingOrder.ProductID
		logData.Amount = existingOrder.TotalPrice

		// 记录日志到数据库，但只有当order_id不为空时才插入
		if logData.OrderID != "" {
			if err := c.logRepo.Create(ctx, &logData); err != nil {
				logger.Error("Failed to create external order log", "error", err)
			}
		}

		ctx.JSON(http.StatusOK, response)
		return
	}

	// 通过商品ID获取商品信息（基础验证，service层会再次验证确保数据一致性）
	product, err := c.productService.GetByID(ctx, req.ProductID)
	if err != nil {
		logData.ErrorMsg = fmt.Sprintf("Get product failed: %v", err)
		c.respondError(ctx, http.StatusBadRequest, "商品不存在", &logData, startTime)
		return
	}

	// 验证商品状态（基础验证）
	if product.Product.Status != 1 {
		logData.ErrorMsg = "Product is disabled"
		c.respondError(ctx, http.StatusBadRequest, "商品已下架", &logData, startTime)
		return
	}

	// 验证请求金额是否与商品价格匹配（可选的业务逻辑验证）
	productPrice := product.Product.Price
	// if req.Amount != productPrice {
	// 	logData.ErrorMsg = fmt.Sprintf("Amount mismatch: request=%v, product=%v", req.Amount, productPrice)
	// 	c.respondError(ctx, http.StatusBadRequest, "订单金额与商品价格不匹配", &logData, startTime)
	// 	return
	// }

	// 创建新订单
	order := &model.Order{
		Mobile:              req.Mobile,
		ProductID:           req.ProductID,
		OutTradeNum:         req.OutTradeNum,
		TotalPrice:          productPrice, // 使用商品价格作为总价
		Price:               productPrice, // 使用商品价格作为面值
		Denom:               productPrice,
		IsDel:               0,
		Param1:              req.Param1,
		Param2:              req.Param2,
		Param3:              req.Param3,
		CustomerID:          req.CustomerID,
		ISP:                 req.ISP,
		Remark:              req.Remark,
		Client:              2, // 外部API
		PlatformCallbackURL: req.NotifyURL,
	}

	// 使用事务性的外部订单创建方法（先扣款再创建订单）
	if err := c.orderService.CreateExternalOrder(ctx, order, apiKey.UserID); err != nil {
		logData.ErrorMsg = fmt.Sprintf("Create external order failed: %v", err)

		// 根据错误类型返回不同的HTTP状态码和消息
		errorMsg := err.Error()
		switch {
		case strings.Contains(errorMsg, "余额不足"):
			c.respondError(ctx, http.StatusPaymentRequired, "账号余额不足", &logData, startTime)
		case strings.Contains(errorMsg, "商品不存在"):
			c.respondError(ctx, http.StatusBadRequest, "商品不存在", &logData, startTime)
		case strings.Contains(errorMsg, "商品已下架"):
			c.respondError(ctx, http.StatusBadRequest, "商品已下架", &logData, startTime)
		case strings.Contains(errorMsg, "创建订单失败"):
			c.respondError(ctx, http.StatusInternalServerError, "订单创建失败，请稍后重试", &logData, startTime)
		case strings.Contains(errorMsg, "开启事务失败"):
			c.respondError(ctx, http.StatusInternalServerError, "系统繁忙，请稍后重试", &logData, startTime)
		default:
			c.respondError(ctx, http.StatusInternalServerError, "订单处理失败，请稍后重试", &logData, startTime)
		}
		return
	}

	// 更新日志信息
	logData.OrderID = strconv.FormatInt(order.ID, 10)
	logData.GoodsID = order.ProductID
	logData.Amount = order.TotalPrice
	logData.Status = 1

	// 记录成功日志到数据库，但只有当order_id不为空时才插入
	if logData.OrderID != "" {
		if err := c.logRepo.Create(ctx, &logData); err != nil {
			logger.Error("Failed to create external order log", "error", err)
		}
	}

	// 构建响应
	response := &ExternalOrderCreateResponse{
		Code:      200,
		Message:   "Success",
		Timestamp: time.Now().Unix(),
		Data: &ExternalOrderData{

			OrderNumber: order.OrderNumber,
			OutTradeNum: order.OutTradeNum,
			Status:      int(order.Status),
			StatusDesc:  c.getStatusDesc(int(order.Status)),
			Amount:      order.TotalPrice,
			CreateTime:  order.CreateTime.Unix(),
		},
	}

	// 记录成功响应

	// 记录日志（这里应该调用日志服务）
	// TODO: 记录到数据库

	ctx.JSON(http.StatusOK, response)
}

// GetOrder 查询订单
func (c *ExternalOrderController) GetOrder(ctx *gin.Context) {
	startTime := time.Now()
	var logData model.ExternalOrderLog

	// 获取API Key信息
	_, exists := ctx.Get("api_key_info")
	if !exists {
		c.respondError(ctx, http.StatusUnauthorized, "API Key information not found", &logData, startTime)
		return
	}

	// 初始化日志
	logData = model.ExternalOrderLog{
		Platform:  "external_api",
		BizType:   "query_order",
		Status:    0, // 默认失败
		Timestamp: time.Now().Unix(),
	}

	// 获取查询参数
	outTradeNum := ctx.Query("out_trade_num")
	orderNumber := ctx.Query("order_number")

	if outTradeNum == "" && orderNumber == "" {
		logData.ErrorMsg = "out_trade_num or order_number is required"
		c.respondError(ctx, http.StatusBadRequest, "out_trade_num or order_number is required", &logData, startTime)
		return
	}

	logData.OrderID = outTradeNum
	if orderNumber != "" {
		logData.OrderID = orderNumber
	}

	// 查询订单
	var order *model.Order
	var err error

	if outTradeNum != "" {
		order, err = c.orderService.GetOrderByOutTradeNum(ctx, outTradeNum)
	} else {
		order, err = c.orderService.GetOrderByOrderNumber(ctx, orderNumber)
	}

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			logData.ErrorMsg = "Order not found"
			c.respondError(ctx, http.StatusNotFound, "Order not found", &logData, startTime)
		} else {
			logData.ErrorMsg = fmt.Sprintf("Database error: %v", err)
			c.respondError(ctx, http.StatusInternalServerError, "Database error", &logData, startTime)
		}
		return
	}

	// 更新日志信息
	logData.OrderID = strconv.FormatInt(order.ID, 10)
	logData.GoodsID = order.ProductID
	logData.Amount = order.TotalPrice
	logData.Status = 1

	// 记录成功日志到数据库，但只有当order_id不为空时才插入
	if logData.OrderID != "" {
		if err := c.logRepo.Create(ctx, &logData); err != nil {
			logger.Error("Failed to create external order log", "error", err)
		}
	}

	// 构建响应
	response := &ExternalOrderCreateResponse{
		Code:      200,
		Message:   "Success",
		Timestamp: time.Now().Unix(),
		Data: &ExternalOrderData{

			OrderNumber: order.OrderNumber,
			OutTradeNum: order.OutTradeNum,
			Status:      int(order.Status),
			StatusDesc:  c.getStatusDesc(int(order.Status)),
			Amount:      order.TotalPrice,
			CreateTime:  order.CreateTime.Unix(),
		},
	}

	// 记录日志（这里应该调用日志服务）
	// TODO: 记录到数据库

	ctx.JSON(http.StatusOK, response)
}

// respondError 统一错误响应
func (c *ExternalOrderController) respondError(ctx *gin.Context, statusCode int, message string, logData *model.ExternalOrderLog, startTime time.Time) {
	logData.Status = 0
	if logData.ErrorMsg == "" {
		logData.ErrorMsg = message
	}

	response := &ExternalOrderCreateResponse{
		Code:      statusCode,
		Message:   message,
		Timestamp: time.Now().Unix(),
	}

	// 记录错误日志到数据库，但只有当order_id不为空时才插入
	if logData.OrderID != "" {
		if err := c.logRepo.Create(ctx, logData); err != nil {
			logger.Error("Failed to create external order error log", "error", err)
		}
	}

	ctx.JSON(statusCode, response)
}

// getStatusDesc 获取状态描述
func (c *ExternalOrderController) getStatusDesc(status int) string {
	switch model.OrderStatus(status) {
	case model.OrderStatusPendingPayment:
		return "待支付"
	case model.OrderStatusPendingRecharge:
		return "待充值"
	case model.OrderStatusRecharging:
		return "充值中"
	case model.OrderStatusSuccess:
		return "成功"
	case model.OrderStatusFailed:
		return "失败"
	case model.OrderStatusCancelled:
		return "已取消"
	case model.OrderStatusRefunded:
		return "已退款"
	default:
		return "未知状态"
	}
}
