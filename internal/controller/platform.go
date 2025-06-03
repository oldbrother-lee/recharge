package controller

import (
	"net/http"
	"recharge-go/internal/model"
	"recharge-go/internal/service"
	"recharge-go/internal/service/platform"
	"recharge-go/internal/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

type PlatformController struct {
	service     *service.PlatformService
	platformSvc *platform.Service
}

func NewPlatformController(service *service.PlatformService, platformSvc *platform.Service) *PlatformController {
	return &PlatformController{service: service, platformSvc: platformSvc}
}

// ListPlatforms 获取平台列表
func (c *PlatformController) ListPlatforms(ctx *gin.Context) {
	var req model.PlatformListRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}

	platforms, total := c.service.ListPlatforms(&req)

	resp := gin.H{
		"list":  platforms,
		"total": total,
	}

	utils.Success(ctx, resp)
}

// CreatePlatform 创建平台
func (c *PlatformController) CreatePlatform(ctx *gin.Context) {
	var req model.PlatformCreateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}

	if err := c.service.CreatePlatform(&req); err != nil {
		utils.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(ctx, nil)
}

// UpdatePlatform 更新平台
func (c *PlatformController) UpdatePlatform(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		utils.Error(ctx, http.StatusBadRequest, "invalid platform id")
		return
	}

	var req model.PlatformUpdateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}

	err = c.service.UpdatePlatform(id, &req)
	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(ctx, nil)
}

// DeletePlatform 删除平台
func (c *PlatformController) DeletePlatform(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		utils.Error(ctx, http.StatusBadRequest, "invalid platform id")
		return
	}

	err = c.service.DeletePlatform(id)
	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(ctx, nil)
}

// GetPlatform 获取平台详情
func (c *PlatformController) GetPlatform(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		utils.Error(ctx, http.StatusBadRequest, "invalid platform id")
		return
	}

	platform, err := c.service.GetPlatform(id)
	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(ctx, platform)
}

// ListPlatformAccounts 获取平台账号列表
func (c *PlatformController) ListPlatformAccounts(ctx *gin.Context) {
	var req model.PlatformAccountListRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := c.service.ListPlatformAccounts(&req)
	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(ctx, resp)
}

// CreatePlatformAccount 创建平台账号
func (c *PlatformController) CreatePlatformAccount(ctx *gin.Context) {
	var req model.PlatformAccountCreateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}

	if err := c.service.CreatePlatformAccount(&req); err != nil {
		utils.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(ctx, nil)
}

// UpdatePlatformAccount 更新平台账号
func (c *PlatformController) UpdatePlatformAccount(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		utils.Error(ctx, http.StatusBadRequest, "invalid account id")
		return
	}

	var req model.PlatformAccountUpdateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}

	if err := c.service.UpdatePlatformAccount(ctx, id, &req); err != nil {
		utils.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(ctx, nil)
}

// DeletePlatformAccount 删除平台账号
func (c *PlatformController) DeletePlatformAccount(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		utils.Error(ctx, http.StatusBadRequest, "invalid account id")
		return
	}

	if err := c.service.DeletePlatformAccount(id); err != nil {
		utils.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(ctx, nil)
}

// GetPlatformAccount 获取平台账号详情
func (c *PlatformController) GetPlatformAccount(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		utils.Error(ctx, http.StatusBadRequest, "invalid account id")
		return
	}

	account, err := c.service.GetPlatformAccount(id)
	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(ctx, account)
}

// GetChannelList 获取渠道列表
// @Summary 获取渠道列表
// @Description 获取所有渠道及对应运营商编码
// @Tags 平台接口
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=[]platform.Channel}
// @Router /api/platform/channels [get]
func (c *PlatformController) GetChannelList(ctx *gin.Context) {
	channels, err := c.platformSvc.GetChannelList()
	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	utils.Success(ctx, channels)
}
