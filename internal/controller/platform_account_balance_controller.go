package controller

import (
	"net/http"
	"recharge-go/internal/service"
	"strconv"

	"github.com/gin-gonic/gin"
)

// PlatformAccountBalanceController 平台账号余额控制器
type PlatformAccountBalanceController struct {
	balanceService *service.PlatformAccountBalanceService
}

// NewPlatformAccountBalanceController 创建平台账号余额控制器实例
func NewPlatformAccountBalanceController(balanceService *service.PlatformAccountBalanceService) *PlatformAccountBalanceController {
	return &PlatformAccountBalanceController{
		balanceService: balanceService,
	}
}

// AdjustBalance 手动调整余额
func (c *PlatformAccountBalanceController) AdjustBalance(ctx *gin.Context) {
	var req struct {
		AccountID int64   `json:"account_id" binding:"required"`
		Amount    float64 `json:"amount" binding:"required"`
		Style     int     `json:"style" binding:"required"`
		Remark    string  `json:"remark"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	// 获取当前用户
	operator := ctx.GetString("username")
	if operator == "" {
		operator = "system"
	}

	if err := c.balanceService.AdjustBalance(ctx, req.AccountID, req.Amount, req.Style, req.Remark, operator); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "调整成功"})
}

// GetBalanceLogs 获取余额变动记录
func (c *PlatformAccountBalanceController) GetBalanceLogs(ctx *gin.Context) {
	accountID, err := strconv.ParseInt(ctx.Query("account_id"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	offset, err := strconv.Atoi(ctx.DefaultQuery("offset", "0"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	limit, err := strconv.Atoi(ctx.DefaultQuery("limit", "20"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	logs, total, err := c.balanceService.GetBalanceLogs(ctx, accountID, offset, limit)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"data":  logs,
		"total": total,
	})
}

// RegisterRoutes 注册路由
func (c *PlatformAccountBalanceController) RegisterRoutes(r *gin.RouterGroup) {
	balanceGroup := r.Group("/balance")
	{
		balanceGroup.POST("/adjust", c.AdjustBalance)
		balanceGroup.GET("/logs", c.GetBalanceLogs)
	}
}
