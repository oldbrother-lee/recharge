package controller

import (
	"net/http"
	"recharge-go/internal/model"
	"recharge-go/internal/repository"
	"strconv"

	"github.com/gin-gonic/gin"
)

type PullTaskConfigController struct {
	repo repository.PullTaskConfigRepository
}

func NewPullTaskConfigController(repo repository.PullTaskConfigRepository) *PullTaskConfigController {
	return &PullTaskConfigController{repo: repo}
}

// Create 创建配置
func (c *PullTaskConfigController) Create(ctx *gin.Context) {
	var req model.PullTaskConfig
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := c.repo.Create(ctx, &req); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"id": req.ID})
}

// Update 更新配置
func (c *PullTaskConfigController) Update(ctx *gin.Context) {
	id, _ := strconv.ParseInt(ctx.Param("id"), 10, 64)
	var req model.PullTaskConfig
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.ID = id
	if err := c.repo.Update(ctx, &req); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"ok": true})
}

// Delete 删除配置
func (c *PullTaskConfigController) Delete(ctx *gin.Context) {
	id, _ := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err := c.repo.Delete(ctx, id); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"ok": true})
}

// List 列表
func (c *PullTaskConfigController) List(ctx *gin.Context) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "10"))
	platformAccountID, _ := strconv.ParseInt(ctx.DefaultQuery("platform_account_id", "0"), 10, 64)
	platformCode := ctx.DefaultQuery("platform_code", "")
	list, total, err := c.repo.List(ctx, platformAccountID, platformCode, page, pageSize)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"total": total, "items": list})
}