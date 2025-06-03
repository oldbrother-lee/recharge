package controller

import (
	"recharge-go/internal/model"
	"recharge-go/internal/service"
	"recharge-go/internal/utils"
	"recharge-go/pkg/logger"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// PlatformAPIController 平台接口控制器
type PlatformAPIController struct {
	service         service.PlatformAPIService
	platformService *service.PlatformService
}

// NewPlatformAPIController 创建平台接口控制器实例
func NewPlatformAPIController(service service.PlatformAPIService, platformService *service.PlatformService) *PlatformAPIController {
	return &PlatformAPIController{
		service:         service,
		platformService: platformService,
	}
}

// CreateAPI 创建平台接口
func (c *PlatformAPIController) CreateAPI(ctx *gin.Context) {
	logger.Log.Info("开始创建平台接口",
		zap.String("method", "CreateAPI"),
		zap.String("path", ctx.Request.URL.Path),
	)

	var api model.PlatformAPI
	if err := ctx.ShouldBindJSON(&api); err != nil {
		logger.Log.Error("参数绑定失败", zap.Error(err))
		utils.Error(ctx, 400, "参数格式错误")
		return
	}

	// 参数验证
	if api.PlatformID == 0 {
		utils.Error(ctx, 400, "平台ID不能为空")
		return
	}
	if api.Name == "" {
		utils.Error(ctx, 400, "接口名称不能为空")
		return
	}

	if api.URL == "" {
		utils.Error(ctx, 400, "接口地址不能为空")
		return
	}

	// 检查平台是否存在
	platform, err := c.platformService.GetPlatform(api.PlatformID)
	if err != nil {
		logger.Log.Error("获取平台信息失败", zap.Error(err))
		utils.Error(ctx, 500, "获取平台信息失败")
		return
	}
	if platform == nil {
		utils.Error(ctx, 400, "平台不存在")
		return
	}

	// 设置默认值
	if api.Method == "" {
		api.Method = "POST"
	}
	api.Status = 1
	api.Timeout = 30
	api.RetryTimes = 3

	if err := c.service.CreateAPI(ctx, &api); err != nil {
		logger.Log.Error("创建平台接口失败", zap.Error(err))
		utils.Error(ctx, 500, err.Error())
		return
	}

	logger.Log.Info("创建平台接口成功", zap.Any("api", api))
	utils.Success(ctx, api)
}

// UpdateAPI 更新平台接口
func (c *PlatformAPIController) UpdateAPI(ctx *gin.Context) {
	logger.Log.Info("开始更新平台接口",
		zap.String("method", "UpdateAPI"),
		zap.String("path", ctx.Request.URL.Path),
	)

	var api model.PlatformAPI
	if err := ctx.ShouldBindJSON(&api); err != nil {
		logger.Log.Error("参数绑定失败", zap.Error(err))
		utils.Error(ctx, 400, "参数格式错误")
		return
	}

	// 参数验证
	if api.Name == "" {
		utils.Error(ctx, 400, "接口名称不能为空")
		return
	}

	if api.URL == "" {
		utils.Error(ctx, 400, "接口地址不能为空")
		return
	}

	if err := c.service.UpdateAPI(ctx, &api); err != nil {
		logger.Log.Error("更新平台接口失败", zap.Error(err))
		utils.Error(ctx, 500, err.Error())
		return
	}

	logger.Log.Info("更新平台接口成功", zap.Any("api", api))
	utils.Success(ctx, api)
}

// DeleteAPI 删除平台接口
func (c *PlatformAPIController) DeleteAPI(ctx *gin.Context) {
	logger.Log.Info("开始删除平台接口",
		zap.String("method", "DeleteAPI"),
		zap.String("path", ctx.Request.URL.Path),
	)

	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		logger.Log.Error("参数ID格式错误", zap.Error(err))
		utils.Error(ctx, 400, "无效的参数ID")
		return
	}

	if err := c.service.DeleteAPI(ctx, id); err != nil {
		logger.Log.Error("删除平台接口失败", zap.Error(err))
		utils.Error(ctx, 500, err.Error())
		return
	}

	logger.Log.Info("删除平台接口成功", zap.Int64("id", id))
	utils.Success(ctx, gin.H{"message": "删除成功"})
}

// GetAPI 获取平台接口详情
func (c *PlatformAPIController) GetAPI(ctx *gin.Context) {
	logger.Log.Info("开始获取平台接口详情",
		zap.String("method", "GetAPI"),
		zap.String("path", ctx.Request.URL.Path),
	)

	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		logger.Log.Error("参数ID格式错误", zap.Error(err))
		utils.Error(ctx, 400, "无效的参数ID")
		return
	}

	api, err := c.service.GetAPI(ctx, id)
	if err != nil {
		logger.Log.Error("获取平台接口详情失败", zap.Error(err))
		utils.Error(ctx, 500, err.Error())
		return
	}

	if api == nil {
		logger.Log.Warn("平台接口不存在", zap.Int64("id", id))
		utils.Error(ctx, 404, "平台接口不存在")
		return
	}

	logger.Log.Info("获取平台接口详情成功", zap.Any("api", api))
	utils.Success(ctx, api)
}

// ListAPIs 获取平台接口列表
func (c *PlatformAPIController) ListAPIs(ctx *gin.Context) {

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "10"))

	apis, total, err := c.service.ListAPIs(ctx, page, pageSize)
	if err != nil {
		logger.Log.Error("获取平台接口列表失败", zap.Error(err))
		utils.Error(ctx, 500, err.Error())
		return
	}

	utils.Success(ctx, gin.H{
		"list":  apis,
		"total": total,
	})
}
