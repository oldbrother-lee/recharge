package handler

import (
	"net/http"
	"strconv"

	"recharge-go/internal/model"
	"recharge-go/internal/service"

	"github.com/gin-gonic/gin"
)

type PlatformAPIController struct {
	platformAPIService service.PlatformAPIService
}

func NewPlatformAPIController(platformAPIService service.PlatformAPIService) *PlatformAPIController {
	return &PlatformAPIController{
		platformAPIService: platformAPIService,
	}
}

// Create 创建平台API
func (c *PlatformAPIController) Create(ctx *gin.Context) {
	var api model.PlatformAPI
	if err := ctx.ShouldBindJSON(&api); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 设置默认值
	if api.Timeout == 0 {
		api.Timeout = 30
	}
	if api.Status == 0 {
		api.Status = 1
	}
	if api.RetryTimes == 0 {
		api.RetryTimes = 3
	}
	if api.RetryDelay == 0 {
		api.RetryDelay = 5
	}

	if err := c.platformAPIService.CreateAPI(ctx, &api); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, api)
}

// Update 更新平台API
func (c *PlatformAPIController) Update(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var api model.PlatformAPI
	if err := ctx.ShouldBindJSON(&api); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	api.ID = id
	if err := c.platformAPIService.UpdateAPI(ctx, &api); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, api)
}

// Delete 删除平台API
func (c *PlatformAPIController) Delete(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := c.platformAPIService.DeleteAPI(ctx, id); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "success"})
}

// Get 获取平台API
func (c *PlatformAPIController) Get(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	api, err := c.platformAPIService.GetAPI(ctx, id)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, api)
}

// List 获取平台API列表
func (c *PlatformAPIController) List(ctx *gin.Context) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "10"))

	apis, total, err := c.platformAPIService.ListAPIs(ctx, page, pageSize)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"total": total,
		"items": apis,
	})
}

// RegisterRoutes 注册路由
func (c *PlatformAPIController) RegisterRoutes(router *gin.RouterGroup) {
	api := router.Group("/platform-apis")
	{
		api.POST("", c.Create)
		api.PUT("/:id", c.Update)
		api.DELETE("/:id", c.Delete)
		api.GET("/:id", c.Get)
		api.GET("", c.List)
	}
}
