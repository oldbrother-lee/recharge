package controller

import (
	"net/http"
	"recharge-go/internal/model"
	"recharge-go/internal/service"
	"recharge-go/internal/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

type DistributionController struct {
	service *service.DistributionService
}

func NewDistributionController(service *service.DistributionService) *DistributionController {
	return &DistributionController{service: service}
}

// CreateGrade 创建分销等级
// @Summary 创建分销等级
// @Description 创建新的分销等级
// @Tags 分销管理
// @Accept json
// @Produce json
// @Param grade body model.DistributionGrade true "分销等级信息"
// @Success 200 {object} response.Response
// @Router /distribution/grades [post]
func (c *DistributionController) CreateGrade(ctx *gin.Context) {
	var grade model.DistributionGrade
	if err := ctx.ShouldBindJSON(&grade); err != nil {
		utils.Error(ctx, http.StatusBadRequest, "参数格式错误")
		return
	}

	if err := c.service.CreateGrade(&grade); err != nil {
		utils.Error(ctx, http.StatusInternalServerError, "创建分销等级失败")
		return
	}

	utils.Success(ctx, grade)
}

// UpdateGrade 更新分销等级
// @Summary 更新分销等级
// @Description 更新分销等级信息
// @Tags 分销管理
// @Accept json
// @Produce json
// @Param id path int true "分销等级ID"
// @Param grade body model.DistributionGrade true "分销等级信息"
// @Success 200 {object} response.Response
// @Router /distribution/grades/{id} [put]
func (c *DistributionController) UpdateGrade(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		utils.Error(ctx, http.StatusBadRequest, "无效的分销等级ID")
		return
	}

	var grade model.DistributionGrade
	if err := ctx.ShouldBindJSON(&grade); err != nil {
		utils.Error(ctx, http.StatusBadRequest, "参数格式错误")
		return
	}
	grade.ID = id

	if err := c.service.UpdateGrade(&grade); err != nil {
		utils.Error(ctx, http.StatusInternalServerError, "更新分销等级失败")
		return
	}

	utils.Success(ctx, grade)
}

// ListGrades 获取分销等级列表
// @Summary 获取分销等级列表
// @Description 获取所有分销等级
// @Tags 分销管理
// @Produce json
// @Success 200 {object} response.Response
// @Router /distribution/grades [get]
func (c *DistributionController) ListGrades(ctx *gin.Context) {
	grades, err := c.service.ListGrades()
	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, "获取分销等级列表失败")
		return
	}

	utils.Success(ctx, grades)
}

// CreateRule 创建分销规则
// @Summary 创建分销规则
// @Description 创建新的分销规则
// @Tags 分销管理
// @Accept json
// @Produce json
// @Param rule body model.DistributionRule true "分销规则信息"
// @Success 200 {object} response.Response
// @Router /distribution/rules [post]
func (c *DistributionController) CreateRule(ctx *gin.Context) {
	var rule model.DistributionRule
	if err := ctx.ShouldBindJSON(&rule); err != nil {
		utils.Error(ctx, http.StatusBadRequest, "参数格式错误")
		return
	}

	if err := c.service.CreateRule(&rule); err != nil {
		utils.Error(ctx, http.StatusInternalServerError, "创建分销规则失败")
		return
	}

	utils.Success(ctx, rule)
}

// ListRules 获取分销规则列表
// @Summary 获取分销规则列表
// @Description 获取指定等级的分销规则
// @Tags 分销管理
// @Produce json
// @Param grade_id query int true "分销等级ID"
// @Success 200 {object} response.Response
// @Router /distribution/rules [get]
func (c *DistributionController) ListRules(ctx *gin.Context) {
	gradeID, err := strconv.ParseInt(ctx.Query("grade_id"), 10, 64)
	if err != nil {
		utils.Error(ctx, http.StatusBadRequest, "无效的分销等级ID")
		return
	}

	rules, err := c.service.ListRules(gradeID)
	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, "获取分销规则列表失败")
		return
	}

	utils.Success(ctx, rules)
}

// CreateWithdrawal 创建提现申请
// @Summary 创建提现申请
// @Description 创建新的提现申请
// @Tags 分销管理
// @Accept json
// @Produce json
// @Param withdrawal body model.DistributionWithdrawal true "提现申请信息"
// @Success 200 {object} response.Response
// @Router /distribution/withdrawals [post]
func (c *DistributionController) CreateWithdrawal(ctx *gin.Context) {
	var withdrawal model.DistributionWithdrawal
	if err := ctx.ShouldBindJSON(&withdrawal); err != nil {
		utils.Error(ctx, http.StatusBadRequest, "参数格式错误")
		return
	}

	if err := c.service.CreateWithdrawal(&withdrawal); err != nil {
		utils.Error(ctx, http.StatusInternalServerError, "创建提现申请失败")
		return
	}

	utils.Success(ctx, withdrawal)
}

// ListWithdrawals 获取提现记录列表
// @Summary 获取提现记录列表
// @Description 获取用户的提现记录
// @Tags 分销管理
// @Produce json
// @Param user_id query int true "用户ID"
// @Param status query int false "状态(0:待审核 1:已通过 2:已拒绝)"
// @Success 200 {object} response.Response
// @Router /distribution/withdrawals [get]
func (c *DistributionController) ListWithdrawals(ctx *gin.Context) {
	userID, err := strconv.ParseInt(ctx.Query("user_id"), 10, 64)
	if err != nil {
		utils.Error(ctx, http.StatusBadRequest, "无效的用户ID")
		return
	}

	status := -1
	if statusStr := ctx.Query("status"); statusStr != "" {
		status, err = strconv.Atoi(statusStr)
		if err != nil {
			utils.Error(ctx, http.StatusBadRequest, "无效的状态值")
			return
		}
	}

	withdrawals, err := c.service.ListWithdrawals(userID, status)
	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, "获取提现记录列表失败")
		return
	}

	utils.Success(ctx, withdrawals)
}

// GetStatistics 获取分销统计
// @Summary 获取分销统计
// @Description 获取用户的分销统计数据
// @Tags 分销管理
// @Produce json
// @Param user_id query int true "用户ID"
// @Success 200 {object} response.Response
// @Router /distribution/statistics [get]
func (c *DistributionController) GetStatistics(ctx *gin.Context) {
	userID, err := strconv.ParseInt(ctx.Query("user_id"), 10, 64)
	if err != nil {
		utils.Error(ctx, http.StatusBadRequest, "无效的用户ID")
		return
	}

	statistics, err := c.service.GetStatistics(userID)
	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, "获取分销统计数据失败")
		return
	}

	utils.Success(ctx, statistics)
}

// CreateDistributor 创建分销商
// @Summary 创建分销商
// @Description 创建新的分销商
// @Tags 分销管理
// @Accept json
// @Produce json
// @Param distributor body model.DistributorRequest true "分销商信息"
// @Success 200 {object} response.Response
// @Router /distribution/distributors [post]
func (c *DistributionController) CreateDistributor(ctx *gin.Context) {
	var req model.DistributorRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.Error(ctx, http.StatusBadRequest, "参数格式错误")
		return
	}

	if err := c.service.CreateDistributor(&req); err != nil {
		utils.Error(ctx, http.StatusInternalServerError, "创建分销商失败")
		return
	}

	utils.Success(ctx, nil)
}

// GetDistributor 获取分销商详情
// @Summary 获取分销商详情
// @Description 获取分销商的详细信息
// @Tags 分销管理
// @Produce json
// @Param id path uint true "分销商ID"
// @Success 200 {object} response.Response
// @Router /distribution/distributors/{id} [get]
func (c *DistributionController) GetDistributor(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		utils.Error(ctx, http.StatusBadRequest, "无效的分销商ID")
		return
	}

	distributor, err := c.service.GetDistributor(id)
	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, "获取分销商详情失败")
		return
	}

	utils.Success(ctx, distributor)
}

// UpdateDistributor 更新分销商信息
// @Summary 更新分销商信息
// @Description 更新分销商的详细信息
// @Tags 分销管理
// @Accept json
// @Produce json
// @Param id path uint true "分销商ID"
// @Param distributor body model.DistributorRequest true "分销商信息"
// @Success 200 {object} response.Response
// @Router /distribution/distributors/{id} [put]
func (c *DistributionController) UpdateDistributor(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		utils.Error(ctx, http.StatusBadRequest, "无效的分销商ID")
		return
	}

	var req model.DistributorRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.Error(ctx, http.StatusBadRequest, "参数格式错误")
		return
	}

	if err := c.service.UpdateDistributor(id, &req); err != nil {
		utils.Error(ctx, http.StatusInternalServerError, "更新分销商失败")
		return
	}

	utils.Success(ctx, nil)
}

// DeleteDistributor 删除分销商
// @Summary 删除分销商
// @Description 删除指定的分销商
// @Tags 分销管理
// @Param id path uint true "分销商ID"
// @Success 200 {object} response.Response
// @Router /distribution/distributors/{id} [delete]
func (c *DistributionController) DeleteDistributor(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		utils.Error(ctx, http.StatusBadRequest, "无效的分销商ID")
		return
	}

	if err := c.service.DeleteDistributor(id); err != nil {
		utils.Error(ctx, http.StatusInternalServerError, "删除分销商失败")
		return
	}

	utils.Success(ctx, nil)
}

// ListDistributors 获取分销商列表
// @Summary 获取分销商列表
// @Description 获取分销商列表
// @Tags 分销管理
// @Produce json
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Param status query string false "状态"
// @Success 200 {object} response.Response
// @Router /distribution/distributors [get]
func (c *DistributionController) ListDistributors(ctx *gin.Context) {
	var req model.DistributorListRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.Error(ctx, http.StatusBadRequest, "参数格式错误")
		return
	}

	resp, err := c.service.ListDistributors(&req)
	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, "获取分销商列表失败")
		return
	}

	utils.Success(ctx, resp)
}

// GetDistributorStatistics 获取分销商统计信息
// @Summary 获取分销商统计信息
// @Description 获取指定分销商的统计信息
// @Tags 分销管理
// @Produce json
// @Param id path uint true "分销商ID"
// @Success 200 {object} response.Response
// @Router /distribution/distributors/{id}/statistics [get]
func (c *DistributionController) GetDistributorStatistics(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		utils.Error(ctx, http.StatusBadRequest, "无效的分销商ID")
		return
	}

	statistics, err := c.service.GetDistributorStatistics(id)
	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, "获取分销商统计信息失败")
		return
	}

	utils.Success(ctx, statistics)
}
