package controller

import (
	"net/http"
	"recharge-go/internal/model"
	"recharge-go/internal/service"
	"recharge-go/internal/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

type TaskConfigController struct {
	taskConfigService *service.TaskConfigService
	notifier          *service.TaskConfigNotifier
}

func NewTaskConfigController(taskConfigService *service.TaskConfigService, notifier *service.TaskConfigNotifier) *TaskConfigController {
	return &TaskConfigController{
		taskConfigService: taskConfigService,
		notifier:          notifier,
	}
}

// Create 创建任务配置
func (c *TaskConfigController) Create(ctx *gin.Context) {
	var configs []model.TaskConfig
	if err := ctx.ShouldBindJSON(&configs); err != nil {
		utils.Error(ctx, http.StatusBadRequest, "无效的参数")
		return
	}

	// 转为 []*model.TaskConfig
	configPtrs := make([]*model.TaskConfig, len(configs))
	for i := range configs {
		configPtrs[i] = &configs[i]
	}

	if err := c.taskConfigService.BatchCreate(ctx, configPtrs); err != nil {
		utils.Error(ctx, http.StatusInternalServerError, "批量创建任务配置失败")
		return
	}

	// 通知任务配置变更（批量创建时通知每个配置）
	for _, config := range configPtrs {
		if err := c.notifier.NotifyConfigCreate(config.ID); err != nil {
			// 记录错误但不影响响应，因为配置已经创建成功
			utils.Error(ctx, http.StatusInternalServerError, "配置创建成功但通知失败")
			return
		}
	}

	utils.Success(ctx, nil)
}

// Update 更新任务配置
func (c *TaskConfigController) Update(ctx *gin.Context) {
	var req model.UpdateTaskConfigRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.Error(ctx, http.StatusBadRequest, "无效的参数")
		return
	}

	if err := c.taskConfigService.UpdatePartial(ctx, &req); err != nil {
		utils.Error(ctx, http.StatusInternalServerError, "更新任务配置失败")
		return
	}

	// 通知任务配置变更
	if err := c.notifier.NotifyConfigUpdate(*req.ID); err != nil {
		// 记录错误但不影响响应，因为配置已经更新成功
		utils.Error(ctx, http.StatusInternalServerError, "配置更新成功但通知失败")
		return
	}

	// 获取更新后的完整配置
	config, err := c.taskConfigService.GetByID(ctx, *req.ID)
	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, "获取更新后的配置失败")
		return
	}

	utils.Success(ctx, config)
}

// Delete 删除任务配置
func (c *TaskConfigController) Delete(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		utils.Error(ctx, http.StatusBadRequest, "无效的ID")
		return
	}

	if err := c.taskConfigService.Delete(ctx, id); err != nil {
		utils.Error(ctx, http.StatusInternalServerError, "删除任务配置失败")
		return
	}

	// 通知任务配置变更
	if err := c.notifier.NotifyConfigDelete(id); err != nil {
		// 记录错误但不影响响应，因为配置已经删除成功
		utils.Error(ctx, http.StatusInternalServerError, "配置删除成功但通知失败")
		return
	}

	utils.Success(ctx, nil)
}

// Get 获取任务配置
func (c *TaskConfigController) Get(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		utils.Error(ctx, http.StatusBadRequest, "无效的ID")
		return
	}

	config, err := c.taskConfigService.GetByID(ctx, id)
	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, "获取任务配置失败")
		return
	}

	utils.Success(ctx, config)
}

// List 获取任务配置列表
func (c *TaskConfigController) List(ctx *gin.Context) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "10"))
	platformAccountIDStr := ctx.Query("platform_account_id")
	var platformAccountID *int64
	if platformAccountIDStr != "" {
		id, err := strconv.ParseInt(platformAccountIDStr, 10, 64)
		if err == nil {
			platformAccountID = &id
		}
	}

	configs, total, err := c.taskConfigService.List(ctx, page, pageSize, platformAccountID)
	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, "获取任务配置列表失败")
		return
	}

	utils.Success(ctx, gin.H{
		"list":  configs,
		"total": total,
	})
}

// GetByID 根据ID获取任务配置
func (c *TaskConfigController) GetByID(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.Error(ctx, 400, "参数错误")
		return
	}
	config, err := c.taskConfigService.GetByID(ctx, id)
	if err != nil {
		utils.Error(ctx, 500, "获取任务配置失败")
		return
	}
	utils.Success(ctx, config)
}
