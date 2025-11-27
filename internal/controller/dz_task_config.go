package controller

import (
    "net/http"
    "recharge-go/internal/model"
    "recharge-go/internal/service"
    resp "recharge-go/pkg/utils/response"
    "strconv"

    "github.com/gin-gonic/gin"
)

type DzTaskConfigController struct {
	dzTaskConfigService *service.DzTaskConfigService
	notifier            *service.TaskConfigNotifier
}

func NewDzTaskConfigController(dzTaskConfigService *service.DzTaskConfigService, notifier *service.TaskConfigNotifier) *DzTaskConfigController {
	return &DzTaskConfigController{
		dzTaskConfigService: dzTaskConfigService,
		notifier:            notifier,
	}
}

// Create 创建得众任务配置
func (c *DzTaskConfigController) Create(ctx *gin.Context) {
	var configs []model.DzTaskConfig
    if err := ctx.ShouldBindJSON(&configs); err != nil {
        resp.Error(ctx, http.StatusBadRequest, "无效的参数")
        return
    }

	// 转为 []*model.DzTaskConfig
	configPtrs := make([]*model.DzTaskConfig, len(configs))
	for i := range configs {
		configPtrs[i] = &configs[i]
	}

    if err := c.dzTaskConfigService.BatchCreate(ctx, configPtrs); err != nil {
        resp.Error(ctx, http.StatusInternalServerError, "批量创建得众任务配置失败")
        return
    }

	// 通知任务配置变更（批量创建时通知每个配置）
	for _, config := range configPtrs {
		if err := c.notifier.NotifyConfigCreate(ctx.Request.Context(), config.ID); err != nil {
			// 记录错误但不影响响应，因为配置已经创建成功
            resp.Error(ctx, http.StatusInternalServerError, "配置创建成功但通知失败")
            return
        }
    }

    resp.Success(ctx, nil)
}

// Update 更新得众任务配置
func (c *DzTaskConfigController) Update(ctx *gin.Context) {
	var req model.UpdateDzTaskConfigRequest
    if err := ctx.ShouldBindJSON(&req); err != nil {
        resp.Error(ctx, http.StatusBadRequest, "无效的参数")
        return
    }

    if err := c.dzTaskConfigService.UpdatePartial(ctx, &req); err != nil {
        resp.Error(ctx, http.StatusInternalServerError, "更新得众任务配置失败")
        return
    }

	// 通知任务配置变更
	if req.ID != nil {
		if err := c.notifier.NotifyConfigUpdate(ctx.Request.Context(), *req.ID); err != nil {
			// 记录错误但不影响响应，因为配置已经更新成功
            resp.Error(ctx, http.StatusInternalServerError, "配置更新成功但通知失败")
            return
        }
    }

    resp.Success(ctx, nil)
}

// Delete 删除得众任务配置
func (c *DzTaskConfigController) Delete(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
    if err != nil {
        resp.Error(ctx, http.StatusBadRequest, "无效的ID")
        return
    }

    if err := c.dzTaskConfigService.Delete(ctx, id); err != nil {
        resp.Error(ctx, http.StatusInternalServerError, "删除得众任务配置失败")
        return
    }

	// 通知任务配置变更
	if err := c.notifier.NotifyConfigDelete(ctx.Request.Context(), id); err != nil {
		// 记录错误但不影响响应，因为配置已经删除成功
        resp.Error(ctx, http.StatusInternalServerError, "配置删除成功但通知失败")
        return
    }

    resp.Success(ctx, nil)
}

// Get 获取得众任务配置
func (c *DzTaskConfigController) Get(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
    if err != nil {
        resp.Error(ctx, http.StatusBadRequest, "无效的ID")
        return
    }

	config, err := c.dzTaskConfigService.GetByID(ctx, id)
    if err != nil {
        resp.Error(ctx, http.StatusInternalServerError, "获取得众任务配置失败")
        return
    }

    resp.Success(ctx, config)
}

// List 获取得众任务配置列表
func (c *DzTaskConfigController) List(ctx *gin.Context) {
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

	configs, total, err := c.dzTaskConfigService.List(ctx, page, pageSize, platformAccountID)
    if err != nil {
        resp.Error(ctx, http.StatusInternalServerError, "获取得众任务配置列表失败")
        return
    }

    resp.Success(ctx, gin.H{
        "list":  configs,
        "total": total,
    })
}

// GetByID 根据ID获取得众任务配置
func (c *DzTaskConfigController) GetByID(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
    if err != nil {
        resp.Error(ctx, 400, "参数错误")
        return
    }
    config, err := c.dzTaskConfigService.GetByID(ctx, id)
    if err != nil {
        resp.Error(ctx, 500, "获取得众任务配置失败")
        return
    }
    resp.Success(ctx, config)
}
