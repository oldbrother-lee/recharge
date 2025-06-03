package controller

import (
	"net/http"
	"recharge-go/internal/model"
	"recharge-go/internal/service"
	"recharge-go/internal/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

// UserLogController 用户日志控制器
type UserLogController struct {
	service *service.UserLogService
}

// NewUserLogController 创建用户日志控制器实例
func NewUserLogController(service *service.UserLogService) *UserLogController {
	return &UserLogController{
		service: service,
	}
}

// CreateLog 创建用户日志
// @Summary 创建用户日志
// @Description 创建新的用户日志记录
// @Tags 用户日志
// @Accept json
// @Produce json
// @Param log body model.UserLogRequest true "用户日志信息"
// @Success 200 {object} model.UserLog
// @Router /user/logs [post]
func (c *UserLogController) CreateLog(ctx *gin.Context) {
	var req model.UserLogRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.Error(ctx, http.StatusBadRequest, "参数格式错误")
		return
	}

	log, err := c.service.CreateLog(ctx, &req)
	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, "创建日志失败")
		return
	}

	utils.Success(ctx, log)
}

// GetLogByID 获取用户日志详情
// @Summary 获取用户日志详情
// @Description 根据ID获取用户日志详情
// @Tags 用户日志
// @Accept json
// @Produce json
// @Param id path int true "日志ID"
// @Success 200 {object} model.UserLog
// @Router /user/logs/{id} [get]
func (c *UserLogController) GetLogByID(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		utils.Error(ctx, http.StatusBadRequest, "无效的日志ID")
		return
	}

	log, err := c.service.GetLogByID(ctx, id)
	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, "获取日志失败")
		return
	}

	utils.Success(ctx, log)
}

// ListLogs 获取用户日志列表
// @Summary 获取用户日志列表
// @Description 获取用户日志列表，支持分页和筛选
// @Tags 用户日志
// @Accept json
// @Produce json
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Param user_id query int false "用户ID"
// @Success 200 {object} model.UserLogListResponse
// @Router /user/logs [get]
func (c *UserLogController) ListLogs(ctx *gin.Context) {
	var req model.UserLogListRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.Error(ctx, http.StatusBadRequest, "参数格式错误")
		return
	}

	resp, err := c.service.ListLogs(ctx, &req)
	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, "获取日志列表失败")
		return
	}

	utils.Success(ctx, resp)
}
