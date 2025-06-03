package controller

import (
	"net/http"
	"recharge-go/internal/service"
	"recharge-go/internal/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

type RebateController struct {
	rebateService service.RebateService
}

func NewRebateController(rebateService service.RebateService) *RebateController {
	return &RebateController{
		rebateService: rebateService,
	}
}

// GetRebates 获取返利列表
func (c *RebateController) GetRebates(ctx *gin.Context) {
	userID, _ := ctx.Get("user_id")
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "10"))

	rebates, total, err := c.rebateService.GetRebatesByUserID(ctx, userID.(int64), page, pageSize)
	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(ctx, gin.H{
		"list":  rebates,
		"total": total,
	})
}

// GetRebate 获取返利详情
func (c *RebateController) GetRebate(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		utils.Error(ctx, http.StatusBadRequest, "Invalid ID")
		return
	}

	rebate, err := c.rebateService.GetRebateByID(ctx, id)
	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(ctx, rebate)
}

// UpdateRebateStatus 更新返利状态
func (c *RebateController) UpdateRebateStatus(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		utils.Error(ctx, http.StatusBadRequest, "Invalid ID")
		return
	}

	var req struct {
		Status int `json:"status" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}

	err = c.rebateService.UpdateRebateStatus(ctx, id, req.Status)
	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(ctx, nil)
}
