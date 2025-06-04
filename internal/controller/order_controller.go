package controller

import (
	"fmt"
	"net/http"
	"recharge-go/internal/utils"
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
	fmt.Printf("stats: %+v", stats)
	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	utils.Success(ctx, stats)
}
