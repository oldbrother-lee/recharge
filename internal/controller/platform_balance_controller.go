package controller

import (
	"net/http"
	"recharge-go/internal/service/recharge"
	"recharge-go/internal/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

// PlatformBalanceController 平台余额控制器
type PlatformBalanceController struct {
	platformManager *recharge.Manager
}

// NewPlatformBalanceController 创建平台余额控制器
func NewPlatformBalanceController(platformManager *recharge.Manager) *PlatformBalanceController {
	return &PlatformBalanceController{
		platformManager: platformManager,
	}
}

// QueryBalance 查询平台余额
func (c *PlatformBalanceController) QueryBalance(ctx *gin.Context) {
	// 获取平台代码
	platformCode := ctx.Param("platform")
	if platformCode == "" {
		utils.Error(ctx, http.StatusBadRequest, "platform code is required")
		return
	}

	// 获取账号ID
	accountIDStr := ctx.Query("account_id")
	if accountIDStr == "" {
		utils.Error(ctx, http.StatusBadRequest, "account_id is required")
		return
	}

	accountID, err := strconv.ParseInt(accountIDStr, 10, 64)
	if err != nil {
		utils.Error(ctx, http.StatusBadRequest, "invalid account_id")
		return
	}

	// 获取平台实例
	platform, err := c.platformManager.GetPlatform(platformCode)
	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	// 查询余额
	balance, err := platform.QueryBalance(ctx, accountID)
	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(ctx, gin.H{
		"balance": balance,
	})
}
