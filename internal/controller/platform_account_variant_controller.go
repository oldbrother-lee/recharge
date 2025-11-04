package controller

import (
	"strconv"
	"recharge-go/internal/model"
	"recharge-go/internal/service"
	"recharge-go/internal/utils"

	"github.com/gin-gonic/gin"
)

type PlatformAccountVariantController struct {
	svc *service.PlatformAccountVariantService
}

func NewPlatformAccountVariantController(svc *service.PlatformAccountVariantService) *PlatformAccountVariantController {
	return &PlatformAccountVariantController{svc: svc}
}

// Create 创建变体
func (c *PlatformAccountVariantController) Create(ctx *gin.Context) {
	var req model.PlatformAccountVariant
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.Error(ctx, 1, "参数错误: "+err.Error())
		return
	}
	
	if err := c.svc.Create(ctx, &req); err != nil {
		utils.Error(ctx, 1, "创建变体失败: "+err.Error())
		return
	}
	
	utils.Success(ctx, req)
}

// Update 更新变体
func (c *PlatformAccountVariantController) Update(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		utils.Error(ctx, 1, "参数错误: "+err.Error())
		return
	}
	
	var req model.PlatformAccountVariant
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.Error(ctx, 1, "参数错误: "+err.Error())
		return
	}
	
	req.ID = id
	if err := c.svc.Update(ctx, &req); err != nil {
		utils.Error(ctx, 1, "更新变体失败: "+err.Error())
		return
	}
	
	utils.Success(ctx, req)
}

// Delete 删除变体
func (c *PlatformAccountVariantController) Delete(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		utils.Error(ctx, 1, "参数错误: "+err.Error())
		return
	}
	
	if err := c.svc.Delete(ctx, id); err != nil {
		utils.Error(ctx, 1, "删除变体失败: "+err.Error())
		return
	}
	
	utils.Success(ctx, nil)
}

// GetByID 获取变体详情
func (c *PlatformAccountVariantController) GetByID(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		utils.Error(ctx, 1, "参数错误: "+err.Error())
		return
	}
	
	variant, err := c.svc.GetByID(ctx, id)
	if err != nil {
		utils.Error(ctx, 1, "获取变体详情失败: "+err.Error())
		return
	}
	if variant == nil {
		utils.Error(ctx, 1, "变体不存在")
		return
	}
	
	utils.Success(ctx, variant)
}

// List 获取变体列表
func (c *PlatformAccountVariantController) List(ctx *gin.Context) {
	var req struct {
		PlatformAccountID *int64 `form:"platform_account_id"`
		ISP               *int   `form:"isp"`
		Enabled           *bool  `form:"enabled"`
		Page              int    `form:"page" binding:"min=1"`
		PageSize          int    `form:"page_size" binding:"min=1,max=100"`
	}
	
	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.Error(ctx, 1, "参数错误: "+err.Error())
		return
	}
	
	// 设置默认值
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 20
	}
	
	offset := (req.Page - 1) * req.PageSize
	variants, total, err := c.svc.List(ctx, req.PlatformAccountID, req.ISP, req.Enabled, offset, req.PageSize)
	if err != nil {
		utils.Error(ctx, 1, "获取变体列表失败: "+err.Error())
		return
	}
	
	utils.Success(ctx, gin.H{
		"items": variants,
		"total": total,
		"page":  req.Page,
		"page_size": req.PageSize,
	})
}

// GetByPlatformAccount 根据平台账号获取变体列表
func (c *PlatformAccountVariantController) GetByPlatformAccount(ctx *gin.Context) {
	platformAccountID, err := strconv.ParseInt(ctx.Param("platform_account_id"), 10, 64)
	if err != nil {
		utils.Error(ctx, 1, "参数错误: "+err.Error())
		return
	}
	
	variants, err := c.svc.GetByPlatformAccountID(ctx, platformAccountID)
	if err != nil {
		utils.Error(ctx, 1, "获取变体列表失败: "+err.Error())
		return
	}
	
	utils.Success(ctx, variants)
}