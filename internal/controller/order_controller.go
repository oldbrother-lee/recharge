package controller

import (
	"net/http"
	"recharge-go/internal/utils"
	"recharge-go/pkg/logger"
	"strconv"

	"github.com/gin-gonic/gin"
)

// GetOrderStatistics 实时统计订单接口
func (c *OrderController) GetOrderStatistics(ctx *gin.Context) {
	customerIDStr := ctx.Query("customer_id")
	customerID, err := strconv.ParseInt(customerIDStr, 10, 64)
	if err != nil {
		utils.Error(ctx, http.StatusBadRequest, "invalid customer_id")
		return
	}
	stats, err := c.orderService.GetOrderStatistics(ctx, customerID)
	logger.WithContextCategory(ctx.Request.Context(), "order").Info("订单统计结果", logger.AnyV2("stats", stats), logger.Int64V2("customer_id", customerID))
	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	utils.Success(ctx, stats)
}
