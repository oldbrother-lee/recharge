package controller

import (
    "strconv"
    "recharge-go/internal/model"
    "recharge-go/internal/service"
    resp "recharge-go/pkg/utils/response"

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
        resp.ErrorWithCode(ctx, 400, 1, "参数错误: "+err.Error(), nil)
        return
    }
    if err := c.svc.BindUser(req.PlatformAccountID, req.UserID); err != nil {
        resp.ErrorWithCode(ctx, 500, 1, "绑定失败: "+err.Error(), nil)
        return
    }
    resp.Success(ctx, nil)
}

// 查询账号列表（带本地用户名）
func (c *PlatformAccountController) List(ctx *gin.Context) {
	var req model.PlatformAccountListRequest
    if err := ctx.ShouldBindQuery(&req); err != nil {
        resp.ErrorWithCode(ctx, 400, 1, "参数错误: "+err.Error(), nil)
        return
    }
    total, list, err := c.svc.GetListWithUserName(&req)
    if err != nil {
        resp.ErrorWithCode(ctx, 500, 1, "查询失败: "+err.Error(), nil)
        return
    }
    resp.Success(ctx, gin.H{"total": total, "items": list})
}

// 拉单相关接口

// GetPullOrderAccounts 获取拉单账号列表
func (c *PlatformAccountController) GetPullOrderAccounts(ctx *gin.Context) {
    accounts, err := c.svc.GetPullOrderAccounts(ctx)
    if err != nil {
        resp.ErrorWithCode(ctx, 500, 1, "获取拉单账号失败: "+err.Error(), nil)
        return
    }
    resp.Success(ctx, accounts)
}

// GetPullOrderAccount 获取拉单账号详情
func (c *PlatformAccountController) GetPullOrderAccount(ctx *gin.Context) {
    id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
    if err != nil {
        resp.ErrorWithCode(ctx, 400, 1, "参数错误: "+err.Error(), nil)
        return
    }

    account, err := c.svc.GetPullOrderAccountByID(ctx, id)
    if err != nil {
        resp.ErrorWithCode(ctx, 500, 1, "获取账号详情失败: "+err.Error(), nil)
        return
    }
    if account == nil {
        resp.ErrorWithCode(ctx, 404, 1, "账号不存在或未启用拉单", nil)
        return
    }
    
    resp.Success(ctx, account)
}

// UpdatePullOrderConfig 更新拉单配置
func (c *PlatformAccountController) UpdatePullOrderConfig(ctx *gin.Context) {
    id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
    if err != nil {
        resp.ErrorWithCode(ctx, 400, 1, "参数错误: "+err.Error(), nil)
        return
    }

    var req model.PlatformAccountUpdateRequest
    if err := ctx.ShouldBindJSON(&req); err != nil {
        resp.ErrorWithCode(ctx, 400, 1, "参数错误: "+err.Error(), nil)
        return
    }
    
    if err := c.svc.UpdatePullOrderConfig(ctx, id, &req); err != nil {
        resp.ErrorWithCode(ctx, 500, 1, "更新拉单配置失败: "+err.Error(), nil)
        return
    }
    
    resp.Success(ctx, nil)
}
