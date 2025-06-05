package controller

import (
	"net/http"
	"recharge-go/internal/model"
	"recharge-go/internal/service"
	"recharge-go/internal/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

type SystemConfigController struct {
	systemConfigService *service.SystemConfigService
}

func NewSystemConfigController(systemConfigService *service.SystemConfigService) *SystemConfigController {
	return &SystemConfigController{
		systemConfigService: systemConfigService,
	}
}

// Create 创建系统配置
func (c *SystemConfigController) Create(ctx *gin.Context) {
	var req model.SystemConfigRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.Error(ctx, http.StatusBadRequest, "无效的参数")
		return
	}

	if err := c.systemConfigService.Create(ctx, &req); err != nil {
		utils.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(ctx, nil)
}

// Update 更新系统配置
func (c *SystemConfigController) Update(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.Error(ctx, http.StatusBadRequest, "无效的ID")
		return
	}

	var req model.SystemConfigRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.Error(ctx, http.StatusBadRequest, "无效的参数")
		return
	}

	if err := c.systemConfigService.Update(ctx, id, &req); err != nil {
		utils.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(ctx, nil)
}

// Delete 删除系统配置
func (c *SystemConfigController) Delete(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.Error(ctx, http.StatusBadRequest, "无效的ID")
		return
	}

	if err := c.systemConfigService.Delete(ctx, id); err != nil {
		utils.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(ctx, nil)
}

// GetByID 根据ID获取系统配置
func (c *SystemConfigController) GetByID(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.Error(ctx, http.StatusBadRequest, "无效的ID")
		return
	}

	config, err := c.systemConfigService.GetByID(ctx, id)
	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(ctx, config)
}

// GetByKey 根据配置键获取系统配置
func (c *SystemConfigController) GetByKey(ctx *gin.Context) {
	key := ctx.Param("key")
	if key == "" {
		utils.Error(ctx, http.StatusBadRequest, "配置键不能为空")
		return
	}

	config, err := c.systemConfigService.GetByKey(ctx, key)
	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(ctx, config)
}

// GetList 获取系统配置列表
func (c *SystemConfigController) GetList(ctx *gin.Context) {
	pageStr := ctx.DefaultQuery("page", "1")
	pageSizeStr := ctx.DefaultQuery("page_size", "10")
	configKey := ctx.Query("config_key")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 {
		pageSize = 10
	}

	configs, total, err := c.systemConfigService.GetList(ctx, page, pageSize, configKey)
	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	response := map[string]interface{}{
		"list":      configs,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}

	utils.Success(ctx, response)
}

// UpdateSystemName 更新系统名称
func (c *SystemConfigController) UpdateSystemName(ctx *gin.Context) {
	var req struct {
		SystemName string `json:"system_name" binding:"required"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.Error(ctx, http.StatusBadRequest, "无效的参数")
		return
	}

	if err := c.systemConfigService.UpdateSystemName(ctx, req.SystemName); err != nil {
		utils.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(ctx, nil)
}

// GetSystemName 获取系统名称
func (c *SystemConfigController) GetSystemName(ctx *gin.Context) {
	systemName, err := c.systemConfigService.GetSystemName(ctx)
	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	response := map[string]string{
		"system_name": systemName,
	}

	utils.Success(ctx, response)
}

// BatchUpdate 批量更新配置
func (c *SystemConfigController) BatchUpdate(ctx *gin.Context) {
	var req map[string]string
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.Error(ctx, http.StatusBadRequest, "无效的参数")
		return
	}

	if len(req) == 0 {
		utils.Error(ctx, http.StatusBadRequest, "配置不能为空")
		return
	}

	if err := c.systemConfigService.BatchUpdate(ctx, req); err != nil {
		utils.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(ctx, nil)
}

// GetSystemInfo 获取系统信息
func (c *SystemConfigController) GetSystemInfo(ctx *gin.Context) {
	// 获取所有系统配置
	configs, err := c.systemConfigService.GetAllAsMap(ctx)
	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	// 获取系统运行时间信息
	uptimeManager := utils.GetUptimeManager()
	systemInfo := uptimeManager.GetSystemInfo()

	// 构建返回数据结构，符合前端期望的格式
	response := map[string]interface{}{
		"configs":     configs,
		"system_info": systemInfo,
	}

	utils.Success(ctx, response)
}
