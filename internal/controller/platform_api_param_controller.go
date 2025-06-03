package controller

import (
	"recharge-go/internal/model"
	"recharge-go/internal/service"
	"recharge-go/internal/utils"
	"recharge-go/internal/validator"
	"recharge-go/pkg/logger"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// PlatformAPIParamController 平台接口参数控制器
type PlatformAPIParamController struct {
	service service.PlatformAPIParamService
}

// NewPlatformAPIParamController 创建平台接口参数控制器实例
func NewPlatformAPIParamController(service service.PlatformAPIParamService) *PlatformAPIParamController {
	return &PlatformAPIParamController{
		service: service,
	}
}

// CreateParam 创建平台接口参数
func (c *PlatformAPIParamController) CreateParam(ctx *gin.Context) {
	logger.Log.Info("开始创建平台接口参数",
		zap.String("method", "CreateParam"),
		zap.String("path", ctx.Request.URL.Path),
	)

	var param model.PlatformAPIParam
	if err := ctx.ShouldBindJSON(&param); err != nil {
		logger.Log.Error("参数绑定失败", zap.Error(err))
		utils.Error(ctx, 400, "参数格式错误1")
		return
	}

	// 参数验证
	if err := validator.ValidatePlatformAPIParam(&param); err != nil {
		logger.Log.Error("参数验证失败", zap.Error(err))
		utils.Error(ctx, 400, err.Error())
		return
	}

	if err := c.service.CreateParam(ctx, &param); err != nil {
		logger.Log.Error("创建平台接口参数失败", zap.Error(err))
		utils.Error(ctx, 500, "创建平台接口参数失败")
		return
	}

	logger.Log.Info("创建平台接口参数成功", zap.Any("param", param))
	utils.Success(ctx, param)
}

// UpdateParam 更新平台接口参数
func (c *PlatformAPIParamController) UpdateParam(ctx *gin.Context) {
	logger.Log.Info("开始更新平台接口参数",
		zap.String("method", "UpdateParam"),
		zap.String("path", ctx.Request.URL.Path),
	)

	var param model.PlatformAPIParam
	if err := ctx.ShouldBindJSON(&param); err != nil {
		logger.Log.Error("参数绑定失败", zap.Error(err))
		utils.Error(ctx, 400, "参数格式错误")
		return
	}

	// 参数验证
	if err := validator.ValidatePlatformAPIParam(&param); err != nil {
		logger.Log.Error("参数验证失败", zap.Error(err))
		utils.Error(ctx, 400, err.Error())
		return
	}

	if err := c.service.UpdateParam(ctx, &param); err != nil {
		logger.Log.Error("更新平台接口参数失败", zap.Error(err))
		utils.Error(ctx, 500, "更新平台接口参数失败")
		return
	}

	logger.Log.Info("更新平台接口参数成功", zap.Any("param", param))
	utils.Success(ctx, param)
}

// DeleteParam 删除平台接口参数
func (c *PlatformAPIParamController) DeleteParam(ctx *gin.Context) {
	logger.Log.Info("开始删除平台接口参数",
		zap.String("method", "DeleteParam"),
		zap.String("path", ctx.Request.URL.Path),
	)

	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		logger.Log.Error("参数ID格式错误", zap.Error(err))
		utils.Error(ctx, 400, "无效的参数ID")
		return
	}

	if err := c.service.DeleteParam(ctx, id); err != nil {
		logger.Log.Error("删除平台接口参数失败", zap.Error(err))
		utils.Error(ctx, 500, "删除平台接口参数失败")
		return
	}

	logger.Log.Info("删除平台接口参数成功", zap.Int64("id", id))
	utils.Success(ctx, gin.H{"message": "删除成功"})
}

// GetParam 获取平台接口参数详情
func (c *PlatformAPIParamController) GetParam(ctx *gin.Context) {
	logger.Log.Info("开始获取平台接口参数详情",
		zap.String("method", "GetParam"),
		zap.String("path", ctx.Request.URL.Path),
	)

	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		logger.Log.Error("参数ID格式错误", zap.Error(err))
		utils.Error(ctx, 400, "无效的参数ID")
		return
	}

	param, err := c.service.GetParam(ctx, id)
	if err != nil {
		logger.Log.Error("获取平台接口参数详情失败", zap.Error(err))
		utils.Error(ctx, 500, "获取平台接口参数详情失败")
		return
	}

	if param == nil {
		logger.Log.Warn("平台接口参数不存在", zap.Int64("id", id))
		utils.Error(ctx, 404, "平台接口参数不存在")
		return
	}

	logger.Log.Info("获取平台接口参数详情成功", zap.Any("param", param))
	utils.Success(ctx, param)
}

// ListParams 获取平台接口参数列表
func (c *PlatformAPIParamController) ListParams(ctx *gin.Context) {
	apiID, _ := strconv.ParseInt(ctx.Query("api_id"), 10, 64)
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "10"))

	params, total, err := c.service.ListParams(ctx, apiID, page, pageSize)
	if err != nil {
		logger.Log.Error("获取平台接口参数列表失败", zap.Error(err))
		utils.Error(ctx, 500, "获取平台接口参数列表失败")
		return
	}

	logger.Log.Info("获取平台接口参数列表成功",
		zap.Int64("total", total),
		zap.Int("page", page),
		zap.Int("size", pageSize),
	)

	utils.Success(ctx, gin.H{
		"list":  params,
		"total": total,
	})
}
