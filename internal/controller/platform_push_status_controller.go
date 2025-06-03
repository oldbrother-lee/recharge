package controller

import (
	"fmt"
	"net/http"
	"recharge-go/internal/service/platform"
	"recharge-go/internal/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

// PlatformPushStatusController 推单状态控制器
type PlatformPushStatusController struct {
	pushStatusService *platform.PushStatusService
}

// NewPlatformPushStatusController 创建推单状态控制器
func NewPlatformPushStatusController(pushStatusService *platform.PushStatusService) *PlatformPushStatusController {
	return &PlatformPushStatusController{
		pushStatusService: pushStatusService,
	}
}

// GetPushStatus 获取推单状态
func (c *PlatformPushStatusController) GetPushStatus(ctx *gin.Context) {
	// 获取账号ID
	accountIDStr := ctx.Param("account_id")
	if accountIDStr == "" {
		utils.Error(ctx, http.StatusBadRequest, "account_id is required")
		return
	}

	accountID, err := strconv.ParseInt(accountIDStr, 10, 64)
	if err != nil {
		utils.Error(ctx, http.StatusBadRequest, "invalid account_id")
		return
	}

	// 获取账号信息
	account, err := c.pushStatusService.AccountRepo.GetByID(accountID)
	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, "failed to get account: "+err.Error())
		return
	}

	// 获取推单状态
	status, err := c.pushStatusService.GetPushStatus(account)
	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, "failed to get push status: "+err.Error())
		return
	}

	utils.Success(ctx, gin.H{
		"status": status,
	})
}

// UpdatePushStatus 更新推单状态
func (c *PlatformPushStatusController) UpdatePushStatus(ctx *gin.Context) {
	// 获取账号ID
	accountIDStr := ctx.Param("account_id")
	if accountIDStr == "" {
		utils.Error(ctx, http.StatusBadRequest, "account_id is required")
		return
	}

	accountID, err := strconv.ParseInt(accountIDStr, 10, 64)
	if err != nil {
		utils.Error(ctx, http.StatusBadRequest, "invalid account_id")
		return
	}

	// 获取请求参数
	var req struct {
		Status int `json:"status" binding:"required,oneof=1 2"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.Error(ctx, http.StatusBadRequest, "invalid request parameters: status 只能为 1(开启) 或 2(关闭)")
		return
	}
	fmt.Printf("Bind result: %+v\n", req)

	// 获取账号信息
	account, err := c.pushStatusService.AccountRepo.GetByID(accountID)
	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, "failed to get account: "+err.Error())
		return
	}

	// 更新推单状态
	if err := c.pushStatusService.UpdatePushStatus(account, req.Status); err != nil {
		utils.Error(ctx, http.StatusInternalServerError, "failed to update push status: "+err.Error())
		return
	}

	utils.Success(ctx, nil)
}
