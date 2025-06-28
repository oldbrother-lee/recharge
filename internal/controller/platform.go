package controller

import (
	"fmt"
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
	beeService  *service.BeeService
}

func NewPlatformController(platformService *service.PlatformService, platformSvc *platform.Service) *PlatformController {
	return &PlatformController{
		service:     platformService,
		platformSvc: platformSvc,
		beeService:  service.NewBeeService(),
	}
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
	//从 url 获取参数 name
	name := ctx.Query("name")

	// 通过这个platform_accounts account_name自动匹配 name 表获取 appkey 和 appsecret
	var appKey, apiURL string
	if name != "" {
		account, err := c.service.GetPlatformAccountByAccountName(name)
		if err != nil {
			utils.Error(ctx, http.StatusBadRequest, fmt.Sprintf("未找到账号名为 %s 的平台账号: %v", name, err))
			return
		}
		// 获取 appkey 和 appsecret
		appKey = account.AppKey
		platformId := account.PlatformID
		// 通过 platformId 从 platforms 表获取 apiurl
		platform, err := c.service.GetPlatformByID(platformId)
		if err != nil {
			utils.Error(ctx, http.StatusInternalServerError, fmt.Sprintf("获取平台信息失败: %v", err))
			return
		}
		apiURL = platform.ApiURL
		// 设置 API URL 到响应头

	}

	channels, err := c.platformSvc.GetChannelList(appKey, name, apiURL)
	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	utils.Success(ctx, channels)
}

// GetBeeProductList 获取蜜蜂平台商品列表
func (c *PlatformController) GetBeeProductList(ctx *gin.Context) {
	accountID, err := strconv.ParseInt(ctx.Param("accountId"), 10, 64)
	if err != nil {
		utils.Error(ctx, http.StatusBadRequest, "invalid account id")
		return
	}

	// 获取平台账号信息
	account, err := c.service.GetPlatformAccountByID(accountID)
	if err != nil {
		utils.Error(ctx, http.StatusNotFound, "平台账号不存在")
		return
	}

	// 验证是否为蜜蜂平台
	platform, err := c.service.GetPlatformByID(account.PlatformID)
	if err != nil || platform.Code != "mifeng" {
		utils.Error(ctx, http.StatusBadRequest, "该账号不是蜜蜂平台账号")
		return
	}

	// 获取分页参数
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "20"))

	// 调用蜜蜂平台API
	result, err := c.beeService.GetProductList(account, page, pageSize)
	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, fmt.Sprintf("获取商品列表失败: %v", err))
		return
	}

	// 解析数据，处理不同的返回格式
	parsedData, err := service.ParseProductListData(result.Data)
	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, fmt.Sprintf("解析商品数据失败: %v", err))
		return
	}

	// 转换为前端期望的格式
	response := map[string]interface{}{
		"list":  parsedData.GoodsInfo,
		"total": parsedData.StatInfo.Total,
	}

	utils.Success(ctx, response)
}

// UpdateBeeProductPrice 更新蜜蜂平台商品价格
func (c *PlatformController) UpdateBeeProductPrice(ctx *gin.Context) {
	accountID, err := strconv.ParseInt(ctx.Param("accountId"), 10, 64)
	if err != nil {
		utils.Error(ctx, http.StatusBadRequest, "invalid account id")
		return
	}

	// 获取平台账号信息
	account, err := c.service.GetPlatformAccountByID(accountID)
	if err != nil {
		utils.Error(ctx, http.StatusNotFound, "平台账号不存在")
		return
	}

	// 验证是否为蜜蜂平台
	platform, err := c.service.GetPlatformByID(account.PlatformID)
	if err != nil || platform.Code != "mifeng" {
		utils.Error(ctx, http.StatusBadRequest, "该账号不是蜜蜂平台账号")
		return
	}

	// 解析请求参数
	var req service.BeeUpdatePriceRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}

	// 调用蜜蜂平台API
	err = c.beeService.UpdateProductPrice(account, &req)
	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, fmt.Sprintf("更新商品价格失败: %v", err))
		return
	}

	utils.Success(ctx, nil)
}

// UpdateBeeProductProvince 更新蜜蜂平台商品省份配置
func (c *PlatformController) UpdateBeeProductProvince(ctx *gin.Context) {
	accountID, err := strconv.ParseInt(ctx.Param("accountId"), 10, 64)
	if err != nil {
		utils.Error(ctx, http.StatusBadRequest, "invalid account id")
		return
	}

	// 获取平台账号信息
	account, err := c.service.GetPlatformAccountByID(accountID)
	if err != nil {
		utils.Error(ctx, http.StatusNotFound, "平台账号不存在")
		return
	}

	// 验证是否为蜜蜂平台
	platform, err := c.service.GetPlatformByID(account.PlatformID)
	if err != nil || platform.Code != "mifeng" {
		utils.Error(ctx, http.StatusBadRequest, "该账号不是蜜蜂平台账号")
		return
	}

	// 解析请求参数
	var req service.BeeUpdateProvinceRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}

	// 调用蜜蜂平台API
	err = c.beeService.UpdateProductProvince(account, &req)
	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, fmt.Sprintf("更新商品省份配置失败: %v", err))
		return
	}

	utils.Success(ctx, nil)
}
