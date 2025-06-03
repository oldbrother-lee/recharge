package controller

import (
	"net/http"
	"recharge-go/internal/service"
	"recharge-go/internal/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

// BalanceController 余额相关接口

type BalanceController struct {
	service *service.BalanceService
}

func NewBalanceController(service *service.BalanceService) *BalanceController {
	return &BalanceController{service: service}
}

// Recharge 余额充值接口
func (c *BalanceController) Recharge(ctx *gin.Context) {
	var req struct {
		UserID   int64   `json:"user_id" binding:"required"`
		Amount   float64 `json:"amount" binding:"required,gt=0"`
		Remark   string  `json:"remark"`
		Operator string  `json:"operator"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}
	if err := c.service.Recharge(ctx, req.UserID, req.Amount, req.Remark, req.Operator); err != nil {
		utils.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	utils.Success(ctx, nil)
}

// Deduct 余额扣款接口
func (c *BalanceController) Deduct(ctx *gin.Context) {
	var req struct {
		UserID   int64   `json:"user_id" binding:"required"`
		Amount   float64 `json:"amount" binding:"required,gt=0"`
		Style    int     `json:"style" binding:"required"`
		Remark   string  `json:"remark"`
		Operator string  `json:"operator"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}
	if err := c.service.Deduct(ctx, req.UserID, req.Amount, req.Style, req.Remark, req.Operator); err != nil {
		utils.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	utils.Success(ctx, nil)
}

// ListLogs 余额流水查询接口
func (c *BalanceController) ListLogs(ctx *gin.Context) {
	userIDStr := ctx.Query("user_id")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil || userID <= 0 {
		utils.Error(ctx, http.StatusBadRequest, "invalid user_id")
		return
	}
	offset, _ := strconv.Atoi(ctx.DefaultQuery("offset", "0"))
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "20"))
	logs, total, err := c.service.ListLogs(ctx, userID, offset, limit)
	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	utils.Success(ctx, gin.H{"list": logs, "total": total})
}
