package controller

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"recharge-go/internal/model"
	"recharge-go/internal/service"
	"recharge-go/pkg/log"
	logger "recharge-go/pkg/log"
	resp "recharge-go/pkg/utils/response"
	"strconv"

	"go.uber.org/zap"

	"github.com/gin-gonic/gin"
)

type OrderController struct {
	orderService service.OrderService
}

// NewOrderController 创建订单控制器
func NewOrderController(orderService service.OrderService) *OrderController {
	return &OrderController{orderService: orderService}
}

// CreateOrder 创建订单
func (c *OrderController) CreateOrder(ctx *gin.Context) {
	var order model.Order
	if b, err := ctx.GetRawData(); err == nil {
		log.WithContextCategory(ctx, "server").Info("创建订单请求",
			log.StringV2("stage", "intake"),
			log.StringV2("raw_body", string(b)),
			log.StringV2("content_type", ctx.ContentType()),
			log.StringV2("user_agent", ctx.Request.UserAgent()),
		)
		ctx.Request.Body = io.NopCloser(bytes.NewBuffer(b))
	}
	if err := ctx.ShouldBindJSON(&order); err != nil {
		resp.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}

	if err := c.orderService.CreateOrder(ctx, &order); err != nil {
		resp.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	log.WithContextCategory(ctx, "server").Info("创建订单落库",
		log.StringV2("stage", "persist"),
		log.Int64V2("order_id", order.ID),
		log.StringV2("order_number", order.OrderNumber),
		log.Int64V2("customer_id", order.CustomerID),
		log.StringV2("mobile", order.Mobile),
		log.IntV2("status", int(order.Status)),
	)
	resp.Success(ctx, order)
}

// CreateAgentManualOrder 代理商手动下单（扣款与外部 API 下单一致）
func (c *OrderController) CreateAgentManualOrder(ctx *gin.Context) {
	roles, _ := ctx.Get("roles")
	var isAgent bool
	if rs, ok := roles.([]string); ok {
		for _, r := range rs {
			if r == "AGENT" {
				isAgent = true
				break
			}
		}
	}
	if !isAgent {
		resp.Error(ctx, http.StatusForbidden, "仅代理商可使用手动下单")
		return
	}

	var req struct {
		Mobile      string `json:"mobile" binding:"required"`
		ProductID   int64  `json:"product_id" binding:"required"`
		OutTradeNum string `json:"out_trade_num" binding:"required"`
		ISP         int    `json:"isp"`
		Remark      string `json:"remark"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		resp.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}

	userID := ctx.GetInt64("user_id")
	order, err := c.orderService.CreateManualAgentOrder(ctx.Request.Context(), userID, req.Mobile, req.ProductID, req.OutTradeNum, req.ISP, req.Remark)
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "余额") || strings.Contains(msg, "授信") || strings.Contains(msg, "不足") {
			resp.Error(ctx, http.StatusPaymentRequired, msg)
			return
		}
		if strings.Contains(msg, "运营商") || strings.Contains(msg, "无效的运营商") {
			resp.Error(ctx, http.StatusBadRequest, msg)
			return
		}
		resp.Error(ctx, http.StatusInternalServerError, msg)
		return
	}
	resp.Success(ctx, order)
}

// GetOrderByID 根据ID获取订单
func (c *OrderController) GetOrderByID(ctx *gin.Context) {
	id := ctx.Param("id")
	orderID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		resp.Error(ctx, http.StatusBadRequest, "invalid order id")
		return
	}

	order, err := c.orderService.GetOrderByID(ctx, orderID)
	if err != nil {
		resp.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	resp.Success(ctx, order)
}

// GetOrderByOrderNumber 根据订单号获取订单
func (c *OrderController) GetOrderByOrderNumber(ctx *gin.Context) {
	orderNumber := ctx.Param("order_number")
	if orderNumber == "" {
		resp.Error(ctx, http.StatusBadRequest, "order number is required")
		return
	}

	order, err := c.orderService.GetOrderByOrderNumber(ctx, orderNumber)
	if err != nil {
		resp.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	resp.Success(ctx, order)
}

// GetOrdersByCustomerID 根据客户ID获取订单列表
func (c *OrderController) GetOrdersByCustomerID(ctx *gin.Context) {
	customerID := ctx.Param("customer_id")
	customerIDInt, err := strconv.ParseInt(customerID, 10, 64)
	if err != nil {
		resp.Error(ctx, http.StatusBadRequest, "invalid customer id")
		return
	}

	page := ctx.DefaultQuery("page", "1")
	pageSize := ctx.DefaultQuery("page_size", "10")
	pageInt, err := strconv.Atoi(page)
	if err != nil {
		resp.Error(ctx, http.StatusBadRequest, "invalid page")
		return
	}
	pageSizeInt, err := strconv.Atoi(pageSize)
	if err != nil {
		resp.Error(ctx, http.StatusBadRequest, "invalid page size")
		return
	}

	orders, total, err := c.orderService.GetOrdersByCustomerID(ctx, customerIDInt, pageInt, pageSizeInt)
	if err != nil {
		resp.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	resp.Success(ctx, gin.H{
		"list":  orders,
		"total": total,
	})
}

// UpdateOrderStatus 更新订单状态
func (c *OrderController) UpdateOrderStatus(ctx *gin.Context) {
	id := ctx.Param("id")
	orderID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		resp.Error(ctx, http.StatusBadRequest, "invalid order id")
		return
	}

	var req struct {
		Status model.OrderStatus `json:"status" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		resp.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}

	if err := c.orderService.UpdateOrderStatus(ctx, orderID, req.Status); err != nil {
		resp.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	resp.Success(ctx, nil)
}

// ProcessOrderPayment 处理订单支付
func (c *OrderController) ProcessOrderPayment(ctx *gin.Context) {
	id := ctx.Param("id")
	orderID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		resp.Error(ctx, http.StatusBadRequest, "invalid order id")
		return
	}

	var req struct {
		PayWay       int    `json:"pay_way" binding:"required"`
		SerialNumber string `json:"serial_number" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		resp.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}

	if err := c.orderService.ProcessOrderPayment(ctx, orderID, req.PayWay, req.SerialNumber); err != nil {
		resp.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	resp.Success(ctx, nil)
}

// ProcessOrderRecharge 处理订单充值
func (c *OrderController) ProcessOrderRecharge(ctx *gin.Context) {
	id := ctx.Param("id")
	orderID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		resp.Error(ctx, http.StatusBadRequest, "invalid order id")
		return
	}

	var req struct {
		APIID          int64  `json:"api_id" binding:"required"`
		APIOrderNumber string `json:"api_order_number" binding:"required"`
		APITradeNum    string `json:"api_trade_num" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		resp.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}

	if err := c.orderService.ProcessOrderRecharge(ctx, orderID, req.APIID, req.APIOrderNumber, req.APITradeNum); err != nil {
		resp.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	resp.Success(ctx, nil)
}

// ProcessOrderSuccess 处理订单成功
func (c *OrderController) ProcessOrderSuccess(ctx *gin.Context) {
	id := ctx.Param("id")
	orderID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		resp.Error(ctx, http.StatusBadRequest, "invalid order id")
		return
	}

	if err := c.orderService.ProcessOrderSuccess(ctx, orderID); err != nil {
		resp.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	resp.Success(ctx, nil)
}

// ProcessOrderFail 处理订单失败
func (c *OrderController) ProcessOrderFail(ctx *gin.Context) {
	id := ctx.Param("id")
	orderID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		resp.Error(ctx, http.StatusBadRequest, "invalid order id")
		return
	}

	var req struct {
		Remark string `json:"remark" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		resp.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}

	if err := c.orderService.ProcessOrderFail(ctx, orderID, req.Remark); err != nil {
		resp.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	resp.Success(ctx, nil)
}

// ProcessOrderRefund 处理订单退款
func (c *OrderController) ProcessOrderRefund(ctx *gin.Context) {
	id := ctx.Param("id")
	orderID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		resp.Error(ctx, http.StatusBadRequest, "invalid order id")
		return
	}

	var req struct {
		Remark string `json:"remark" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		resp.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}

	if err := c.orderService.ProcessOrderRefund(ctx, orderID, req.Remark); err != nil {
		resp.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	resp.Success(ctx, nil)
}

// ProcessOrderRefundReject 拒绝退款申请（仅待退款审核订单，恢复为待充值）
func (c *OrderController) ProcessOrderRefundReject(ctx *gin.Context) {
	id := ctx.Param("id")
	orderID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		resp.Error(ctx, http.StatusBadRequest, "invalid order id")
		return
	}

	var req struct {
		Remark string `json:"remark"`
	}
	_ = ctx.ShouldBindJSON(&req)

	if err := c.orderService.ProcessOrderRefundReject(ctx, orderID, req.Remark); err != nil {
		resp.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	resp.Success(ctx, nil)
}

// ProcessOrderCancel 处理订单取消
func (c *OrderController) ProcessOrderCancel(ctx *gin.Context) {
	id := ctx.Param("id")
	orderID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		resp.Error(ctx, http.StatusBadRequest, "invalid order id")
		return
	}

	var req struct {
		Remark string `json:"remark" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		resp.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}

	if err := c.orderService.ProcessOrderCancel(ctx, orderID, req.Remark); err != nil {
		resp.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	resp.Success(ctx, nil)
}

// ProcessOrderSplit 处理订单拆单
func (c *OrderController) ProcessOrderSplit(ctx *gin.Context) {
	id := ctx.Param("id")
	orderID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		resp.Error(ctx, http.StatusBadRequest, "invalid order id")
		return
	}

	var req struct {
		Remark string `json:"remark" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		resp.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}

	if err := c.orderService.ProcessOrderSplit(ctx, orderID, req.Remark); err != nil {
		resp.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	resp.Success(ctx, nil)
}

// ProcessOrderPartial 处理订单部分充值
func (c *OrderController) ProcessOrderPartial(ctx *gin.Context) {
	id := ctx.Param("id")
	orderID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		resp.Error(ctx, http.StatusBadRequest, "invalid order id")
		return
	}

	var req struct {
		Remark string `json:"remark" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		resp.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}

	if err := c.orderService.ProcessOrderPartial(ctx, orderID, req.Remark); err != nil {
		resp.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	resp.Success(ctx, nil)
}

// GetOrders 获取订单列表（管理员接口）
func (c *OrderController) GetOrders(ctx *gin.Context) {
	// 获取当前用户信息
	userID := ctx.GetInt64("user_id")
	roles, _ := ctx.Get("roles")
	var userRole string
	if rolesSlice, ok := roles.([]string); ok && len(rolesSlice) > 0 {
		userRole = rolesSlice[0]
	}

	// 获取分页参数
	page := ctx.DefaultQuery("page", "1")
	pageSize := ctx.DefaultQuery("page_size", "10")
	pageInt, _ := strconv.Atoi(page)
	pageSizeInt, _ := strconv.Atoi(pageSize)

	// 获取查询参数
	params := make(map[string]interface{})
	// 支持按面值过滤（denom）
	queryParams := []string{"order_number", "mobile", "status", "client", "platform_code", "start_time", "end_time", "isp", "out_trade_num", "denom"}
	for _, param := range queryParams {
		if value := ctx.Query(param); value != "" {
			params[param] = value
		}
	}

	// 如果是代理商，只查询自己的订单
	if userRole == "AGENT" {
		params["user_id"] = userID
	} else if userRole == "" {
		resp.Error(ctx, http.StatusBadRequest, "没有权限，联系管理员")
		return
	}

	// 诊断日志：记录角色、用户ID、分页与查询参数
	logger.WithContext(ctx).Info("OrderController.GetOrders",
		logger.StringV2("role", userRole),
		logger.Int64V2("user_id", userID),
		logger.AnyV2("params", params),
		logger.IntV2("page", pageInt),
		logger.IntV2("page_size", pageSizeInt),
	)

	// 使用包含通知信息的查询方法
	orders, total, err := c.orderService.GetOrdersWithNotification(ctx, params, pageInt, pageSizeInt)
	if err != nil {
		logger.Log.Error("获取订单列表失败", zap.Error(err))
		resp.Error(ctx, http.StatusInternalServerError, "获取订单列表失败")
		return
	}

	resp.Success(ctx, gin.H{
		"list":  orders,
		"total": total,
	})
}

// GetSuccessStatsByIspAndDenom 获取按运营商与面值的成功订单统计
func (c *OrderController) GetSuccessStatsByIspAndDenom(ctx *gin.Context) {
	// 获取当前用户信息与角色
	userID := ctx.GetInt64("user_id")
	roles, _ := ctx.Get("roles")
	var userRole string
	if rolesSlice, ok := roles.([]string); ok && len(rolesSlice) > 0 {
		userRole = rolesSlice[0]
	}

	// 获取查询参数（与订单列表保持一致）
	params := make(map[string]interface{})
	queryParams := []string{"order_number", "mobile", "status", "client", "platform_code", "start_time", "end_time", "isp", "out_trade_num", "denom"}
	for _, param := range queryParams {
		if value := ctx.Query(param); value != "" {
			params[param] = value
		}
	}

	// 角色限制：代理商只能查询自己的订单统计
	if userRole == "AGENT" {
		params["user_id"] = userID
	} else if userRole == "" {
		resp.Error(ctx, http.StatusBadRequest, "没有权限，联系管理员")
		return
	}

	logger.WithContext(ctx).Info("OrderController.GetSuccessStatsByIspAndDenom",
		logger.StringV2("role", userRole),
		logger.Int64V2("user_id", userID),
		logger.AnyV2("params", params),
	)

	stats, err := c.orderService.GetSuccessStatsByIspDenom(ctx, params)
	if err != nil {
		logger.Log.Error("获取统计失败", zap.Error(err))
		resp.Error(ctx, http.StatusInternalServerError, "获取统计失败")
		return
	}

	resp.Success(ctx, gin.H{
		"list":  stats,
		"total": len(stats),
	})
}

// DeleteOrder 删除订单（软删除）
func (c *OrderController) DeleteOrder(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		resp.Error(ctx, 400, "缺少订单ID")
		return
	}
	if err := c.orderService.DeleteOrder(ctx, id); err != nil {
		logger.Log.Error("删除订单失败", zap.String("order_id", id), zap.Error(err))
		resp.Error(ctx, 500, "删除订单失败: "+err.Error())
		return
	}
	resp.Success(ctx, "删除订单成功")
}

// BatchDeleteOrders 批量删除订单
func (c *OrderController) BatchDeleteOrders(ctx *gin.Context) {
	var req struct {
		OrderIDs []int64 `json:"order_ids" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		resp.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}

	if len(req.OrderIDs) == 0 {
		resp.Error(ctx, http.StatusBadRequest, "订单ID列表不能为空")
		return
	}

	successCount := 0
	failedCount := 0
	var errors []string

	for _, orderID := range req.OrderIDs {
		orderIDStr := strconv.FormatInt(orderID, 10)
		if err := c.orderService.DeleteOrder(ctx, orderIDStr); err != nil {
			failedCount++
			errors = append(errors, "订单"+orderIDStr+"删除失败: "+err.Error())
			logger.Log.Error("批量删除订单失败", zap.Int64("order_id", orderID), zap.Error(err))
		} else {
			successCount++
		}
	}

	result := gin.H{
		"success_count": successCount,
		"failed_count":  failedCount,
		"total_count":   len(req.OrderIDs),
	}

	if len(errors) > 0 {
		result["errors"] = errors
	}

	resp.Success(ctx, result)
}

// BatchProcessOrderSuccess 批量设置订单成功
func (c *OrderController) BatchProcessOrderSuccess(ctx *gin.Context) {
	var req struct {
		OrderIDs []int64 `json:"order_ids" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		resp.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}

	if len(req.OrderIDs) == 0 {
		resp.Error(ctx, http.StatusBadRequest, "订单ID列表不能为空")
		return
	}

	successCount := 0
	failedCount := 0
	var errors []string

	for _, orderID := range req.OrderIDs {
		if err := c.orderService.ProcessOrderSuccess(ctx, orderID); err != nil {
			failedCount++
			errors = append(errors, "订单"+strconv.FormatInt(orderID, 10)+"设置成功失败: "+err.Error())
			logger.Log.Error("批量设置订单成功失败", zap.Int64("order_id", orderID), zap.Error(err))
		} else {
			successCount++
		}
	}

	result := gin.H{
		"success_count": successCount,
		"failed_count":  failedCount,
		"total_count":   len(req.OrderIDs),
	}

	if len(errors) > 0 {
		result["errors"] = errors
	}

	resp.Success(ctx, result)
}

// BatchProcessOrderFail 批量设置订单失败
func (c *OrderController) BatchProcessOrderFail(ctx *gin.Context) {
	var req struct {
		OrderIDs []int64 `json:"order_ids" binding:"required"`
		Remark   string  `json:"remark" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		resp.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}

	if len(req.OrderIDs) == 0 {
		resp.Error(ctx, http.StatusBadRequest, "订单ID列表不能为空")
		return
	}

	successCount := 0
	failedCount := 0
	var errors []string

	for _, orderID := range req.OrderIDs {
		if err := c.orderService.ProcessOrderFail(ctx, orderID, req.Remark); err != nil {
			failedCount++
			errors = append(errors, "订单"+strconv.FormatInt(orderID, 10)+"设置失败失败: "+err.Error())
			logger.Log.Error("批量设置订单失败失败", zap.Int64("order_id", orderID), zap.Error(err))
		} else {
			successCount++
		}
	}

	result := gin.H{
		"success_count": successCount,
		"failed_count":  failedCount,
		"total_count":   len(req.OrderIDs),
	}

	if len(errors) > 0 {
		result["errors"] = errors
	}

	resp.Success(ctx, result)
}

// BatchSendNotification 批量发送回调通知
func (c *OrderController) BatchSendNotification(ctx *gin.Context) {
	var req struct {
		OrderIDs []int64 `json:"order_ids" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		resp.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}

	if len(req.OrderIDs) == 0 {
		resp.Error(ctx, http.StatusBadRequest, "订单ID列表不能为空")
		return
	}

	successCount := 0
	failedCount := 0
	var errors []string

	for _, orderID := range req.OrderIDs {
		if err := c.orderService.SendNotification(ctx, orderID); err != nil {
			failedCount++
			errors = append(errors, "订单"+strconv.FormatInt(orderID, 10)+"发送通知失败: "+err.Error())
			logger.Log.Error("批量发送回调通知失败", zap.Int64("order_id", orderID), zap.Error(err))
		} else {
			successCount++
		}
	}

	result := gin.H{
		"success_count": successCount,
		"failed_count":  failedCount,
		"total_count":   len(req.OrderIDs),
	}

	if len(errors) > 0 {
		result["errors"] = errors
	}

	resp.Success(ctx, result)
}

// CleanupOrders 清理指定时间范围的订单及相关日志
func (c *OrderController) CleanupOrders(ctx *gin.Context) {
	start := ctx.Query("start")
	end := ctx.Query("end")

	logger.Log.Info("CleanupOrders", zap.String("start", start), zap.String("end", end))
	if start == "" || end == "" {
		resp.ErrorWithCode(ctx, http.StatusBadRequest, 1, "请提供开始和结束时间!", nil)
		return
	}
	count, err := c.orderService.CleanupOrders(ctx.Request.Context(), start, end)
	if err != nil {
		resp.ErrorWithCode(ctx, http.StatusInternalServerError, 1, "清理失败: "+err.Error(), nil)
		return
	}
	resp.Success(ctx, gin.H{
		"message": "清理成功",
		"deleted": count,
	})
}
