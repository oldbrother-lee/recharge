package controller

import (
	"recharge-go/internal/model"
	"recharge-go/internal/service"
	"recharge-go/pkg/utils/response"
	"strconv"

	"github.com/gin-gonic/gin"
)

type UserGradeController struct {
	service *service.UserGradeService
}

func NewUserGradeController(service *service.UserGradeService) *UserGradeController {
	return &UserGradeController{service: service}
}

// List 获取用户等级列表
func (c *UserGradeController) List(ctx *gin.Context) {
	grades, err := c.service.ListGrades(ctx)
	if err != nil {
		response.Error(ctx, 500, err.Error())
		return
	}
	response.Success(ctx, grades)
}

// Create 创建用户等级
func (c *UserGradeController) Create(ctx *gin.Context) {
	var grade model.UserGrade
	if err := ctx.ShouldBindJSON(&grade); err != nil {
		response.Error(ctx, 400, "参数错误")
		return
	}

	if err := c.service.CreateGrade(ctx, &grade); err != nil {
		response.Error(ctx, 500, err.Error())
		return
	}
	response.Success(ctx, grade)
}

// Update 更新用户等级
func (c *UserGradeController) Update(ctx *gin.Context) {
	var grade model.UserGrade
	if err := ctx.ShouldBindJSON(&grade); err != nil {
		response.Error(ctx, 400, "参数错误")
		return
	}

	if err := c.service.UpdateGrade(ctx, &grade); err != nil {
		response.Error(ctx, 500, err.Error())
		return
	}
	response.Success(ctx, grade)
}

// Delete 删除用户等级
func (c *UserGradeController) Delete(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		response.Error(ctx, 400, "参数错误")
		return
	}
	if err := c.service.DeleteGrade(ctx, id); err != nil {
		response.Error(ctx, 500, err.Error())
		return
	}
	response.Success(ctx, nil)
}

// Get 获取用户等级
func (c *UserGradeController) Get(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		response.Error(ctx, 400, "参数错误")
		return
	}
	grade, err := c.service.GetGrade(ctx, id)
	if err != nil {
		response.Error(ctx, 500, err.Error())
		return
	}
	response.Success(ctx, grade)
}

// AssignUserGrade 分配用户等级
func (c *UserGradeController) AssignUserGrade(ctx *gin.Context) {
	var req struct {
		UserID  int64 `json:"user_id" binding:"required"`
		GradeID int64 `json:"grade_id" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.Error(ctx, 400, "参数错误")
		return
	}

	if err := c.service.AssignUserGrade(ctx, req.UserID, req.GradeID); err != nil {
		response.Error(ctx, 500, err.Error())
		return
	}
	response.Success(ctx, nil)
}

// GetUserGrade 获取用户的等级
func (c *UserGradeController) GetUserGrade(ctx *gin.Context) {
	userID, err := strconv.ParseInt(ctx.Param("user_id"), 10, 64)
	if err != nil {
		response.Error(ctx, 400, "参数错误")
		return
	}
	grade, err := c.service.GetUserGrade(ctx, userID)
	if err != nil {
		response.Error(ctx, 500, err.Error())
		return
	}
	response.Success(ctx, grade)
}

// RemoveUserGrade 移除用户等级
func (c *UserGradeController) RemoveUserGrade(ctx *gin.Context) {
	var req struct {
		UserID  int64 `json:"user_id" binding:"required"`
		GradeID int64 `json:"grade_id" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.Error(ctx, 400, "参数错误")
		return
	}

	if err := c.service.RemoveUserGrade(ctx, req.UserID, req.GradeID); err != nil {
		response.Error(ctx, 500, err.Error())
		return
	}
	response.Success(ctx, nil)
}

// UpdateGradeStatus 更新等级状态
func (c *UserGradeController) UpdateGradeStatus(ctx *gin.Context) {
	var req struct {
		ID     int64 `json:"id" binding:"required"`
		Status int   `json:"status" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.Error(ctx, 400, "参数错误")
		return
	}

	if err := c.service.UpdateGradeStatus(ctx, req.ID, req.Status); err != nil {
		response.Error(ctx, 500, err.Error())
		return
	}
	response.Success(ctx, nil)
}
