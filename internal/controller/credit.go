package controller

import (
	"recharge-go/internal/model"
	"recharge-go/internal/service"
	"recharge-go/internal/utils"

	"github.com/gin-gonic/gin"
)

// CreditController 授信控制器
type CreditController struct {
	creditService *service.CreditService
}

// NewCreditController 创建授信控制器
func NewCreditController(creditService *service.CreditService) *CreditController {
	return &CreditController{
		creditService: creditService,
	}
}

// SetCredit 设置授信额度
func (c *CreditController) SetCredit(ctx *gin.Context) {
	var req model.CreditLogRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.Error(ctx, 400, err.Error())
		return
	}

	if err := c.creditService.SetCredit(ctx, &req); err != nil {
		utils.Error(ctx, 500, err.Error())
		return
	}

	utils.Success(ctx, nil)
}

// GetCreditLogs 获取授信日志列表
func (c *CreditController) GetCreditLogs(ctx *gin.Context) {
	var req model.CreditLogListRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.Error(ctx, 400, err.Error())
		return
	}

	if req.Current == 0 {
		req.Current = 1
	}
	if req.Size == 0 {
		req.Size = 10
	}

	resp, err := c.creditService.GetCreditLogs(ctx, &req)
	if err != nil {
		utils.Error(ctx, 500, err.Error())
		return
	}

	utils.Success(ctx, resp)
}

// GetUserCreditStats 获取用户授信统计
func (c *CreditController) GetUserCreditStats(ctx *gin.Context) {
	userID := ctx.GetInt64("user_id")
	if userID == 0 {
		utils.Error(ctx, 400, "invalid user id")
		return
	}

	totalUsed, totalRestored, err := c.creditService.GetUserCreditStats(ctx, userID)
	if err != nil {
		utils.Error(ctx, 500, err.Error())
		return
	}

	utils.Success(ctx, gin.H{
		"total_used":     totalUsed,
		"total_restored": totalRestored,
	})
}
