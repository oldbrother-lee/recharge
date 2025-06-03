package controller

import (
	"net/http"
	"recharge-go/internal/service"
	"recharge-go/internal/utils"
	"time"

	"github.com/gin-gonic/gin"
)

type StatisticsController struct {
	statisticsSvc service.StatisticsService
}

func NewStatisticsController(statisticsSvc service.StatisticsService) *StatisticsController {
	return &StatisticsController{
		statisticsSvc: statisticsSvc,
	}
}

// GetOrderOverview 获取订单统计概览
// @Summary 获取订单统计概览
// @Description 获取订单总数、昨日订单、今日订单等统计信息
// @Tags 订单统计
// @Accept json
// @Produce json
// @Success 200 {object} model.OrderStatisticsOverview
// @Router /api/v1/statistics/order/overview [get]
func (c *StatisticsController) GetOrderOverview(ctx *gin.Context) {
	result, err := c.statisticsSvc.GetOrderOverview(ctx)
	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	utils.Success(ctx, result)
}

// GetOperatorStatistics 获取运营商统计
// @Summary 获取运营商统计
// @Description 获取各运营商的订单统计信息（当天）
// @Tags 订单统计
// @Accept json
// @Produce json
// @Success 200 {array} model.OrderStatisticsOperator
// @Router /api/v1/statistics/order/operator [get]
func (c *StatisticsController) GetOperatorStatistics(ctx *gin.Context) {
	today := time.Now()
	start := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())
	end := start.Add(24 * time.Hour).Add(-time.Nanosecond)

	result, err := c.statisticsSvc.GetOperatorStatistics(ctx, start, end)
	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	utils.Success(ctx, result)
}

// GetDailyStatistics 获取每日统计
// @Summary 获取每日统计
// @Description 获取每日订单统计信息
// @Tags 订单统计
// @Accept json
// @Produce json
// @Param startDate query string true "开始日期"
// @Param endDate query string true "结束日期"
// @Success 200 {array} model.OrderStatisticsDaily
// @Router /api/v1/statistics/order/daily [get]
func (c *StatisticsController) GetDailyStatistics(ctx *gin.Context) {
	startDate := ctx.Query("startDate")
	endDate := ctx.Query("endDate")

	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		utils.Error(ctx, http.StatusBadRequest, "Invalid start date format")
		return
	}

	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		utils.Error(ctx, http.StatusBadRequest, "Invalid end date format")
		return
	}

	result, err := c.statisticsSvc.GetDailyStatistics(ctx, start, end)
	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	utils.Success(ctx, result)
}

// GetTrendStatistics 获取趋势统计
// @Summary 获取趋势统计
// @Description 获取订单趋势统计信息
// @Tags 订单统计
// @Accept json
// @Produce json
// @Param startDate query string true "开始日期"
// @Param endDate query string true "结束日期"
// @Param operator query string false "运营商"
// @Success 200 {array} model.OrderStatisticsTrend
// @Router /api/v1/statistics/order/trend [get]
func (c *StatisticsController) GetTrendStatistics(ctx *gin.Context) {
	startDate := ctx.Query("startDate")
	endDate := ctx.Query("endDate")
	operator := ctx.Query("operator")

	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		utils.Error(ctx, http.StatusBadRequest, "Invalid start date format")
		return
	}

	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		utils.Error(ctx, http.StatusBadRequest, "Invalid end date format")
		return
	}

	result, err := c.statisticsSvc.GetTrendStatistics(ctx, start, end, operator)
	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	utils.Success(ctx, result)
}

// TriggerStatistics 手动触发统计任务
func (c *StatisticsController) TriggerStatistics(ctx *gin.Context) {
	if err := c.statisticsSvc.UpdateStatistics(ctx); err != nil {
		utils.Error(ctx, http.StatusInternalServerError, "Failed to update statistics: "+err.Error())
		return
	}

	utils.Success(ctx, gin.H{
		"message": "Statistics update triggered successfully",
	})
}

// GetOrderRealtimeStatistics 获取实时订单统计
// @Summary 获取实时订单统计
// @Description 实时获取订单统计概览
// @Tags 订单统计
// @Accept json
// @Produce json
// @Success 200 {object} model.OrderStatisticsOverview
// @Router /api/v1/statistics/order/realtime [get]
func (c *StatisticsController) GetOrderRealtimeStatistics(ctx *gin.Context) {
	roles, _ := ctx.Get("roles")
	userId := ctx.GetInt64("user_id")

	var result interface{}
	var err error

	if utils.HasRole(roles.([]string), "SUPER_ADMIN") {
		// 管理员可以查看所有订单统计
		result, err = c.statisticsSvc.GetOrderRealtimeStatistics(ctx, 0)
	} else {
		// 代理商只能查看自己的订单统计
		result, err = c.statisticsSvc.GetOrderRealtimeStatistics(ctx, userId)
	}

	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(ctx, result)
}

// GetOperatorOrderCount 获取各运营商订单总数
// @Summary 获取各运营商订单总数
// @Description 获取各运营商订单总数（当天）
// @Tags 订单统计
// @Accept json
// @Produce json
// @Success 200 {array} object
// @Router /api/v1/statistics/order/isp-count [get]
func (c *StatisticsController) GetOperatorOrderCount(ctx *gin.Context) {
	today := time.Now()
	start := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())
	end := start.Add(24 * time.Hour).Add(-time.Nanosecond)

	result, err := c.statisticsSvc.GetOperatorOrderCount(ctx, start, end)
	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	utils.Success(ctx, result)
}
