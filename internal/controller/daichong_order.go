package controller

import (
	"net/http"
	"recharge-go/internal/model"
	"recharge-go/internal/service"
	"recharge-go/internal/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

type DaichongOrderController struct {
	service *service.DaichongOrderService
}

func NewDaichongOrderController(service *service.DaichongOrderService) *DaichongOrderController {
	return &DaichongOrderController{service: service}
}

// Create 新增订单
func (c *DaichongOrderController) Create(ctx *gin.Context) {
	var order model.DaichongOrder
	if err := ctx.ShouldBindJSON(&order); err != nil {
		utils.Error(ctx, http.StatusBadRequest, "参数错误")
		return
	}
	if err := c.service.Create(ctx, &order); err != nil {
		utils.Error(ctx, http.StatusInternalServerError, "创建订单失败")
		return
	}
	utils.Success(ctx, order)
}

// GetByID 查询订单
func (c *DaichongOrderController) GetByID(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.Error(ctx, http.StatusBadRequest, "ID参数错误")
		return
	}
	order, err := c.service.GetByID(ctx, id)
	if err != nil {
		utils.Error(ctx, http.StatusNotFound, "订单不存在")
		return
	}
	utils.Success(ctx, order)
}

// Update 更新订单
func (c *DaichongOrderController) Update(ctx *gin.Context) {
	var order model.DaichongOrder
	if err := ctx.ShouldBindJSON(&order); err != nil {
		utils.Error(ctx, http.StatusBadRequest, "参数错误")
		return
	}
	if err := c.service.Update(ctx, &order); err != nil {
		utils.Error(ctx, http.StatusInternalServerError, "更新订单失败")
		return
	}
	utils.Success(ctx, order)
}

// Delete 删除订单
func (c *DaichongOrderController) Delete(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.Error(ctx, http.StatusBadRequest, "ID参数错误")
		return
	}
	if err := c.service.Delete(ctx, id); err != nil {
		utils.Error(ctx, http.StatusInternalServerError, "删除订单失败")
		return
	}
	utils.Success(ctx, nil)
}

// List 获取订单列表
func (c *DaichongOrderController) List(ctx *gin.Context) {
	// 获取分页参数
	page := utils.GetIntQuery(ctx, "page", 1)
	pageSize := utils.GetIntQuery(ctx, "page_size", 10)

	// 获取过滤条件
	query := map[string]interface{}{
		"page":      page,
		"page_size": pageSize,
	}

	// 手机号
	if account := ctx.Query("account"); account != "" {
		query["account"] = account
	}

	// 订单号
	if orderID := ctx.Query("order_id"); orderID != "" {
		query["order_id"] = orderID
	}

	// 状态
	if status := utils.GetIntQuery(ctx, "status", 0); status > 0 {
		query["status"] = status
	}

	// 时间范围
	if startTime := utils.GetInt64Query(ctx, "start_time", 0); startTime > 0 {
		query["start_time"] = startTime
	}
	if endTime := utils.GetInt64Query(ctx, "end_time", 0); endTime > 0 {
		query["end_time"] = endTime
	}

	// 调用服务层
	orders, total, err := c.service.List(query)
	if err != nil {
		utils.Error(ctx, 500, err.Error())
		return
	}

	utils.Success(ctx, gin.H{
		"list":  orders,
		"total": total,
	})
}
