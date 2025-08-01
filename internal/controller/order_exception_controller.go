package controller

import (
	"net/http"
	"strconv"
	"time"

	"recharge-go/internal/model"
	"recharge-go/internal/service"
	"recharge-go/internal/utils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// OrderExceptionController 订单异常控制器
type OrderExceptionController struct {
	orderExceptionService service.OrderExceptionService
	logger                *zap.Logger
}

// NewOrderExceptionController 创建订单异常控制器实例
func NewOrderExceptionController(
	orderExceptionService service.OrderExceptionService,
	logger *zap.Logger,
) *OrderExceptionController {
	return &OrderExceptionController{
		orderExceptionService: orderExceptionService,
		logger:                logger,
	}
}

// List 获取订单异常列表
// @Summary 获取订单异常列表
// @Description 分页获取订单异常记录列表
// @Tags 订单异常
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Param order_number query string false "订单号"
// @Param exception_type query string false "异常类型"
// @Param status query string false "处理状态"
// @Param start_date query string false "开始日期(YYYY-MM-DD)"
// @Param end_date query string false "结束日期(YYYY-MM-DD)"
// @Success 200 {object} model.OrderExceptionListResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/order-exceptions [get]
func (c *OrderExceptionController) List(ctx *gin.Context) {
	req := &model.OrderExceptionListRequest{}

	// 绑定查询参数
	if err := ctx.ShouldBindQuery(req); err != nil {
		c.logger.Error("绑定查询参数失败", zap.Error(err))
		utils.Error(ctx, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	// 获取异常列表
	response, err := c.orderExceptionService.List(ctx.Request.Context(), req)
	if err != nil {
		c.logger.Error("获取订单异常列表失败", zap.Error(err))
		utils.Error(ctx, http.StatusInternalServerError, "获取异常列表失败")
		return
	}

	utils.Success(ctx, response)
}

// GetByID 根据ID获取订单异常详情
// @Summary 获取订单异常详情
// @Description 根据异常ID获取详细信息
// @Tags 订单异常
// @Accept json
// @Produce json
// @Param id path int true "异常ID"
// @Success 200 {object} model.OrderException
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/order-exceptions/{id} [get]
func (c *OrderExceptionController) GetByID(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.Error(ctx, http.StatusBadRequest, "无效的异常ID")
		return
	}

	exception, err := c.orderExceptionService.GetByID(ctx.Request.Context(), id)
	if err != nil {
		c.logger.Error("获取订单异常详情失败", zap.Int64("id", id), zap.Error(err))
		utils.Error(ctx, http.StatusInternalServerError, "获取异常详情失败")
		return
	}

	if exception == nil {
		utils.Error(ctx, http.StatusNotFound, "异常记录不存在")
		return
	}

	utils.Success(ctx, exception)
}

// GetByOrderID 根据订单ID获取异常记录列表
// @Summary 根据订单ID获取异常记录
// @Description 获取指定订单的所有异常记录
// @Tags 订单异常
// @Accept json
// @Produce json
// @Param order_id path int true "订单ID"
// @Success 200 {object} []model.OrderException
// @Failure 400 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/orders/{order_id}/exceptions [get]
func (c *OrderExceptionController) GetByOrderID(ctx *gin.Context) {
	orderIDStr := ctx.Param("order_id")
	orderID, err := strconv.ParseInt(orderIDStr, 10, 64)
	if err != nil {
		utils.Error(ctx, http.StatusBadRequest, "无效的订单ID")
		return
	}

	exceptions, err := c.orderExceptionService.GetByOrderID(ctx.Request.Context(), orderID)
	if err != nil {
		c.logger.Error("获取订单异常记录失败", zap.Int64("order_id", orderID), zap.Error(err))
		utils.Error(ctx, http.StatusInternalServerError, "获取异常记录失败")
		return
	}

	utils.Success(ctx, exceptions)
}

// UpdateStatus 更新异常状态
// @Summary 更新异常状态
// @Description 更新订单异常的处理状态
// @Tags 订单异常
// @Accept json
// @Produce json
// @Param id path int true "异常ID"
// @Param request body model.UpdateOrderExceptionRequest true "更新请求"
// @Success 200 {object} utils.SuccessResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/order-exceptions/{id}/status [put]
func (c *OrderExceptionController) UpdateStatus(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.Error(ctx, http.StatusBadRequest, "无效的异常ID")
		return
	}

	req := &model.UpdateOrderExceptionRequest{}
	if err := ctx.ShouldBindJSON(req); err != nil {
		c.logger.Error("绑定请求参数失败", zap.Error(err))
		utils.Error(ctx, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	// 获取操作员ID（从认证中间件中获取）
	operatorID, exists := ctx.Get("user_id")
	if !exists {
		utils.Error(ctx, http.StatusUnauthorized, "未找到操作员信息")
		return
	}

	operatorIDInt64, ok := operatorID.(int64)
	if !ok {
		utils.Error(ctx, http.StatusUnauthorized, "操作员ID格式错误")
		return
	}

	// 更新异常状态
	err = c.orderExceptionService.UpdateStatus(ctx.Request.Context(), id, req, operatorIDInt64)
	if err != nil {
		c.logger.Error("更新异常状态失败", zap.Int64("id", id), zap.Error(err))
		utils.Error(ctx, http.StatusInternalServerError, "更新状态失败")
		return
	}

	utils.Success(ctx, gin.H{"message": "状态更新成功"})
}

// GetPendingCount 获取待处理异常数量
// @Summary 获取待处理异常数量
// @Description 获取状态为待处理的异常记录数量
// @Tags 订单异常
// @Accept json
// @Produce json
// @Success 200 {object} map[string]int64
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/order-exceptions/pending-count [get]
func (c *OrderExceptionController) GetPendingCount(ctx *gin.Context) {
	count, err := c.orderExceptionService.GetPendingCount(ctx.Request.Context())
	if err != nil {
		c.logger.Error("获取待处理异常数量失败", zap.Error(err))
		utils.Error(ctx, http.StatusInternalServerError, "获取数量失败")
		return
	}

	utils.Success(ctx, gin.H{"pending_count": count})
}

// GetStatistics 获取异常统计信息
// @Summary 获取异常统计信息
// @Description 获取指定时间范围内的异常统计数据
// @Tags 订单异常
// @Accept json
// @Produce json
// @Param start_date query string true "开始日期(YYYY-MM-DD)"
// @Param end_date query string true "结束日期(YYYY-MM-DD)"
// @Success 200 {object} map[string]int64
// @Failure 400 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/order-exceptions/statistics [get]
func (c *OrderExceptionController) GetStatistics(ctx *gin.Context) {
	startDateStr := ctx.Query("start_date")
	endDateStr := ctx.Query("end_date")

	if startDateStr == "" || endDateStr == "" {
		utils.Error(ctx, http.StatusBadRequest, "开始日期和结束日期不能为空")
		return
	}

	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		utils.Error(ctx, http.StatusBadRequest, "开始日期格式错误")
		return
	}

	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		utils.Error(ctx, http.StatusBadRequest, "结束日期格式错误")
		return
	}

	// 结束日期加一天，包含当天的所有记录
	endDate = endDate.AddDate(0, 0, 1)

	stats, err := c.orderExceptionService.GetStatistics(ctx.Request.Context(), startDate, endDate)
	if err != nil {
		c.logger.Error("获取异常统计信息失败", zap.Error(err))
		utils.Error(ctx, http.StatusInternalServerError, "获取统计信息失败")
		return
	}

	utils.Success(ctx, stats)
}