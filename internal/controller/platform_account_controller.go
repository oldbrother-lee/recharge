package controller

import (
	"recharge-go/internal/model"
	"recharge-go/internal/service"
	"recharge-go/internal/utils"

	"github.com/gin-gonic/gin"
)

type PlatformAccountController struct {
	svc *service.PlatformAccountService
}

func NewPlatformAccountController(svc *service.PlatformAccountService) *PlatformAccountController {
	return &PlatformAccountController{svc: svc}
}

// 绑定本地用户
func (c *PlatformAccountController) BindUser(ctx *gin.Context) {
	var req struct {
		PlatformAccountID int64 `json:"platform_account_id" binding:"required"`
		UserID            int64 `json:"user_id" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.Error(ctx, 1, "参数错误: "+err.Error())
		return
	}
	if err := c.svc.BindUser(req.PlatformAccountID, req.UserID); err != nil {
		utils.Error(ctx, 1, "绑定失败: "+err.Error())
		return
	}
	utils.Success(ctx, nil)
}

// 查询账号列表（带本地用户名）
func (c *PlatformAccountController) List(ctx *gin.Context) {
	var req model.PlatformAccountListRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.Error(ctx, 1, "参数错误: "+err.Error())
		return
	}
	total, list, err := c.svc.GetListWithUserName(&req)
	if err != nil {
		utils.Error(ctx, 1, "查询失败: "+err.Error())
		return
	}
	utils.Success(ctx, gin.H{"total": total, "items": list})
}
