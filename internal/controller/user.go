package controller

import (
	"fmt"
	"recharge-go/internal/model"
	"recharge-go/internal/service"
	"recharge-go/internal/utils"
	"recharge-go/pkg/utils/response"
	"strconv"

	"github.com/gin-gonic/gin"
)

// UserController 用户控制器
type UserController struct {
	userService      *service.UserService
	userGradeService *service.UserGradeService
	userTagService   *service.UserTagService
}

// NewUserController 创建用户控制器
func NewUserController(
	userService *service.UserService,
	userGradeService *service.UserGradeService,
	userTagService *service.UserTagService,
) *UserController {
	return &UserController{
		userService:      userService,
		userGradeService: userGradeService,
		userTagService:   userTagService,
	}
}

// Register 用户注册
// @Summary 用户注册
// @Description 创建新用户账号
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param request body model.UserRegisterRequest true "注册信息"
// @Success 200 {object} response.Response{data=model.UserResponse}
// @Router /api/v1/users/register [post]
func (c *UserController) Register(ctx *gin.Context) {
	var req model.UserRegisterRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.Error(ctx, 400, "参数错误")
		return
	}

	if err := c.userService.Register(ctx, &req); err != nil {
		response.Error(ctx, 500, err.Error())
		return
	}

	response.Success(ctx, nil)
}

// Login 用户登录
// @Summary 用户登录
// @Description 用户登录接口
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param request body model.UserLoginRequest true "登录信息"
// @Success 200 {object} response.Response{data=model.UserLoginResponse}
// @Router /api/v1/user/login [post]
func (c *UserController) Login(ctx *gin.Context) {
	var req model.UserLoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.Error(ctx, 400, "参数错误")
		return
	}

	loginResp, err := c.userService.Login(ctx, &req)
	if err != nil {
		response.Error(ctx, 401, err.Error())
		return
	}

	response.Success(ctx, loginResp)
}

// RefreshToken 刷新访问令牌
// @Summary 刷新访问令牌
// @Description 使用刷新令牌获取新的访问令牌
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param refreshToken query string true "刷新令牌"
// @Success 200 {object} response.Response{data=model.UserLoginResponse}
// @Router /api/v1/user/refresh-token [post]
func (c *UserController) RefreshToken(ctx *gin.Context) {
	refreshToken := ctx.Query("refreshToken")
	if refreshToken == "" {
		response.Error(ctx, 400, "刷新令牌不能为空")
		return
	}

	loginResp, err := c.userService.RefreshToken(ctx, refreshToken)
	if err != nil {
		response.Error(ctx, 401, err.Error())
		return
	}

	response.Success(ctx, loginResp)
}

// GetUserInfo 获取用户信息
// @Summary 获取用户信息
// @Description 获取指定用户的详细信息
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param id path int true "用户ID"
// @Success 200 {object} response.Response{data=model.UserResponse}
// @Router /api/v1/users/{id} [get]
func (c *UserController) GetUserInfo(ctx *gin.Context) {

	userID, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		response.Error(ctx, 400, "无效的用户ID")
		return
	}

	user, err := c.userService.GetUserInfo(ctx, userID)
	if err != nil {
		response.Error(ctx, 404, "用户不存在")
		return
	}

	response.Success(ctx, user)
}

// UpdateProfile 更新用户信息
// @Summary 更新用户信息
// @Description 更新用户的基本信息
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param id path int true "用户ID"
// @Param request body model.UserUpdateRequest true "更新信息"
// @Success 200 {object} response.Response{data=model.UserResponse}
// @Router /api/v1/users/{id} [put]
func (c *UserController) UpdateProfile(ctx *gin.Context) {
	fmt.Println("UpdateProfile", ctx.Param("id"))
	userID, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		response.Error(ctx, 400, "无效的用户ID")
		return
	}

	var req model.UserUpdateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.Error(ctx, 400, "参数错误")
		return
	}

	if err := c.userService.UpdateUser(ctx, userID, &req); err != nil {
		response.Error(ctx, 500, err.Error())
		return
	}

	response.Success(ctx, nil)
}

// GetUserList 获取用户列表
// @Summary 获取用户列表
// @Description 获取用户列表
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param request query model.UserListRequest true "查询参数"
// @Success 200 {object} response.Response{data=model.UserListResponse}
// @Router /api/v1/users [get]
func (c *UserController) GetUserList(ctx *gin.Context) {
	var req model.UserListRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		response.Error(ctx, 400, "参数错误")
		return
	}

	// 设置默认值
	if req.Current == 0 {
		req.Current = 1
	}
	if req.Size == 0 {
		req.Size = 10
	}

	result, err := c.userService.GetUserList(ctx, &req)
	if err != nil {
		response.Error(ctx, 500, err.Error())
		return
	}

	response.Success(ctx, result)
}

// UpdateUserStatus 更新用户状态
// @Summary 更新用户状态
// @Description 启用或禁用用户
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param id path int true "用户ID"
// @Param status body int true "状态(0:禁用 1:启用)"
// @Success 200 {object} response.Response
// @Router /api/v1/users/{id}/status [put]
func (c *UserController) UpdateUserStatus(ctx *gin.Context) {
	userID, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		response.Error(ctx, 400, "无效的用户ID")
		return
	}

	var status int
	if err := ctx.ShouldBindJSON(&status); err != nil {
		response.Error(ctx, 400, "参数错误")
		return
	}

	if err := c.userService.UpdateUserStatus(ctx, userID, status); err != nil {
		response.Error(ctx, 500, err.Error())
		return
	}

	response.Success(ctx, nil)
}

// GetUserGrade 获取用户等级
// @Summary 获取用户等级
// @Description 获取指定用户的等级信息
// @Tags 用户等级
// @Accept json
// @Produce json
// @Param id path int true "用户ID"
// @Success 200 {object} response.Response{data=model.UserGradeResponse}
// @Router /api/v1/users/{id}/grade [get]
func (c *UserController) GetUserGrade(ctx *gin.Context) {
	userID, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		response.Error(ctx, 400, "无效的用户ID")
		return
	}

	grade, err := c.userGradeService.GetUserGrade(ctx, userID)
	if err != nil {
		response.Error(ctx, 404, "等级信息不存在")
		return
	}

	response.Success(ctx, grade)
}

// GetUserTags 获取用户标签
// @Summary 获取用户标签
// @Description 获取指定用户的所有标签
// @Tags 用户标签
// @Accept json
// @Produce json
// @Param id path int true "用户ID"
// @Success 200 {object} response.Response{data=[]model.UserTagResponse}
// @Router /api/v1/users/{id}/tags [get]
func (c *UserController) GetUserTags(ctx *gin.Context) {
	userID, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		response.Error(ctx, 400, "无效的用户ID")
		return
	}

	tags, err := c.userTagService.GetUserTags(ctx, userID)
	if err != nil {
		response.Error(ctx, 500, err.Error())
		return
	}

	response.Success(ctx, tags)
}

// CreateGrade 创建用户等级
// @Summary 创建用户等级
// @Description 创建新的用户等级
// @Tags 用户等级
// @Accept json
// @Produce json
// @Param request body model.UserGrade true "等级信息"
// @Success 200 {object} response.Response{data=model.UserGrade}
// @Router /api/v1/user-grades [post]
func (c *UserController) CreateGrade(ctx *gin.Context) {
	var grade model.UserGrade
	if err := ctx.ShouldBindJSON(&grade); err != nil {
		response.Error(ctx, 400, "参数错误")
		return
	}

	if err := c.userGradeService.CreateGrade(ctx, &grade); err != nil {
		response.Error(ctx, 500, err.Error())
		return
	}

	response.Success(ctx, grade)
}

// UpdateGrade 更新用户等级
// @Summary 更新用户等级
// @Description 更新用户等级信息
// @Tags 用户等级
// @Accept json
// @Produce json
// @Param id path int true "等级ID"
// @Param request body model.UserGrade true "等级信息"
// @Success 200 {object} response.Response{data=model.UserGrade}
// @Router /api/v1/user-grades/{id} [put]
func (c *UserController) UpdateGrade(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		response.Error(ctx, 400, "无效的等级ID")
		return
	}

	var grade model.UserGrade
	if err := ctx.ShouldBindJSON(&grade); err != nil {
		response.Error(ctx, 400, "参数错误")
		return
	}

	grade.ID = id
	if err := c.userGradeService.UpdateGrade(ctx, &grade); err != nil {
		response.Error(ctx, 500, err.Error())
		return
	}

	response.Success(ctx, grade)
}

// DeleteGrade 删除用户等级
// @Summary 删除用户等级
// @Description 删除指定的用户等级
// @Tags 用户等级
// @Accept json
// @Produce json
// @Param id path int true "等级ID"
// @Success 200 {object} response.Response
// @Router /api/v1/user-grades/{id} [delete]
func (c *UserController) DeleteGrade(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		response.Error(ctx, 400, "无效的等级ID")
		return
	}

	if err := c.userGradeService.DeleteGrade(ctx, id); err != nil {
		response.Error(ctx, 500, err.Error())
		return
	}

	response.Success(ctx, nil)
}

// ListGrades 获取用户等级列表
// @Summary 获取用户等级列表
// @Description 获取所有用户等级列表
// @Tags 用户等级
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=[]model.UserGrade}
// @Router /api/v1/user-grades [get]
func (c *UserController) ListGrades(ctx *gin.Context) {
	grades, err := c.userGradeService.ListGrades(ctx)
	if err != nil {
		response.Error(ctx, 500, err.Error())
		return
	}

	response.Success(ctx, grades)
}

// CreateTag 创建用户标签
// @Summary 创建用户标签
// @Description 创建新的用户标签
// @Tags 用户标签
// @Accept json
// @Produce json
// @Param request body model.UserTag true "标签信息"
// @Success 200 {object} response.Response{data=model.UserTag}
// @Router /api/v1/user-tags [post]
func (c *UserController) CreateTag(ctx *gin.Context) {
	var tag model.UserTag
	if err := ctx.ShouldBindJSON(&tag); err != nil {
		response.Error(ctx, 400, "参数错误")
		return
	}

	if err := c.userTagService.CreateTag(ctx, &tag); err != nil {
		response.Error(ctx, 500, err.Error())
		return
	}

	response.Success(ctx, tag)
}

// UpdateTag 更新用户标签
// @Summary 更新用户标签
// @Description 更新用户标签信息
// @Tags 用户标签
// @Accept json
// @Produce json
// @Param id path int true "标签ID"
// @Param request body model.UserTag true "标签信息"
// @Success 200 {object} response.Response{data=model.UserTag}
// @Router /api/v1/user-tags/{id} [put]
func (c *UserController) UpdateTag(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		response.Error(ctx, 400, "无效的标签ID")
		return
	}

	var tag model.UserTag
	if err := ctx.ShouldBindJSON(&tag); err != nil {
		response.Error(ctx, 400, "参数错误")
		return
	}

	tag.ID = id
	if err := c.userTagService.UpdateTag(ctx, &tag); err != nil {
		response.Error(ctx, 500, err.Error())
		return
	}

	response.Success(ctx, tag)
}

// DeleteTag 删除用户标签
// @Summary 删除用户标签
// @Description 删除指定的用户标签
// @Tags 用户标签
// @Accept json
// @Produce json
// @Param id path int true "标签ID"
// @Success 200 {object} response.Response
// @Router /api/v1/user-tags/{id} [delete]
func (c *UserController) DeleteTag(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		response.Error(ctx, 400, "无效的标签ID")
		return
	}

	if err := c.userTagService.DeleteTag(ctx, id); err != nil {
		response.Error(ctx, 500, err.Error())
		return
	}

	response.Success(ctx, nil)
}

// ListTags 获取用户标签列表
// @Summary 获取用户标签列表
// @Description 获取所有用户标签列表
// @Tags 用户标签
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=[]model.UserTag}
// @Router /api/v1/user-tags [get]
func (c *UserController) ListTags(ctx *gin.Context) {
	tags, err := c.userTagService.ListTags(ctx)
	if err != nil {
		response.Error(ctx, 500, err.Error())
		return
	}

	response.Success(ctx, tags)
}

// AssignUserTag 分配用户标签
// @Summary 分配用户标签
// @Description 为用户分配标签
// @Tags 用户标签
// @Accept json
// @Produce json
// @Param id path int true "用户ID"
// @Param tag_id body int true "标签ID"
// @Success 200 {object} response.Response
// @Router /api/v1/users/{id}/tags [post]
func (c *UserController) AssignUserTag(ctx *gin.Context) {
	userID, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		response.Error(ctx, 400, "无效的用户ID")
		return
	}

	var req struct {
		TagID int64 `json:"tag_id" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.Error(ctx, 400, "参数错误")
		return
	}

	if err := c.userTagService.AssignUserTag(ctx, userID, req.TagID); err != nil {
		response.Error(ctx, 500, err.Error())
		return
	}

	response.Success(ctx, nil)
}

// RemoveUserTag 移除用户标签
// @Summary 移除用户标签
// @Description 移除用户的指定标签
// @Tags 用户标签
// @Accept json
// @Produce json
// @Param id path int true "用户ID"
// @Param tag_id path int true "标签ID"
// @Success 200 {object} response.Response
// @Router /api/v1/users/{id}/tags/{tag_id} [delete]
func (c *UserController) RemoveUserTag(ctx *gin.Context) {
	userID, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		response.Error(ctx, 400, "无效的用户ID")
		return
	}

	tagID, err := strconv.ParseInt(ctx.Param("tag_id"), 10, 64)
	if err != nil {
		response.Error(ctx, 400, "无效的标签ID")
		return
	}

	if err := c.userTagService.RemoveUserTag(ctx, userID, tagID); err != nil {
		response.Error(ctx, 500, err.Error())
		return
	}

	response.Success(ctx, nil)
}

// AssignUserGrade 分配用户等级
// @Summary 分配用户等级
// @Description 为用户分配等级
// @Tags 用户等级
// @Accept json
// @Produce json
// @Param id path int true "用户ID"
// @Param grade_id body int true "等级ID"
// @Success 200 {object} response.Response
// @Router /api/v1/users/{id}/grade [post]
func (c *UserController) AssignUserGrade(ctx *gin.Context) {
	userID, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		response.Error(ctx, 400, "无效的用户ID")
		return
	}

	var req struct {
		GradeID int64 `json:"grade_id" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.Error(ctx, 400, "参数错误")
		return
	}

	if err := c.userGradeService.AssignUserGrade(ctx, userID, req.GradeID); err != nil {
		response.Error(ctx, 500, err.Error())
		return
	}

	response.Success(ctx, nil)
}

// RemoveUserGrade 移除用户等级
// @Summary 移除用户等级
// @Description 移除用户的等级
// @Tags 用户等级
// @Accept json
// @Produce json
// @Param id path int true "用户ID"
// @Param grade_id path int true "等级ID"
// @Success 200 {object} response.Response
// @Router /api/v1/users/{id}/grade/{grade_id} [delete]
func (c *UserController) RemoveUserGrade(ctx *gin.Context) {
	userID, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		response.Error(ctx, 400, "无效的用户ID")
		return
	}

	gradeID, err := strconv.ParseInt(ctx.Param("grade_id"), 10, 64)
	if err != nil {
		response.Error(ctx, 400, "无效的等级ID")
		return
	}

	if err := c.userGradeService.RemoveUserGrade(ctx, userID, gradeID); err != nil {
		response.Error(ctx, 500, err.Error())
		return
	}

	response.Success(ctx, nil)
}

// GetProfile 获取当前用户信息
// @Summary 获取当前用户信息
// @Description 获取当前登录用户的信息
// @Tags 用户管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=model.UserResponse}
// @Router /api/v1/user/profile [get]
func (c *UserController) GetProfile(ctx *gin.Context) {
	userID := ctx.GetInt64("user_id")
	user, err := c.userService.GetUserInfo(ctx, userID)
	if err != nil {
		response.Error(ctx, 404, "用户不存在")
		return
	}
	roles, _ := c.userService.GetUserRoles(userID)
	roleNames := make([]string, 0, len(roles))
	for _, r := range roles {
		roleNames = append(roleNames, r.Code)
	}
	resp := map[string]interface{}{
		"userId":   fmt.Sprintf("%d", user.ID),
		"userName": user.Username,
		"roles":    roleNames,
		"buttons":  []string{},
		"balance":  user.Balance,
		"credit":   user.Credit,
	}

	utils.Success(ctx, resp)

	// ctx.JSON(200, gin.H{
	// 	"userId":   fmt.Sprintf("%d", user.ID),
	// 	"userName": user.Username,
	// 	"roles":    roleNames,
	// 	"buttons":  []string{}, // 可根据权限系统动态生成
	// })
}

// ChangePassword 修改密码
// @Summary 修改密码
// @Description 修改当前用户的密码
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param request body model.UserChangePasswordRequest true "密码信息"
// @Success 200 {object} response.Response
// @Router /api/v1/user/password [put]
func (c *UserController) ChangePassword(ctx *gin.Context) {
	userID := ctx.GetInt64("user_id")
	var req model.UserChangePasswordRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.Error(ctx, 400, "参数错误")
		return
	}

	if err := c.userService.ChangePassword(ctx, userID, &req); err != nil {
		response.Error(ctx, 500, err.Error())
		return
	}

	response.Success(ctx, nil)
}

// ListUsers 获取用户列表（管理员）
// @Summary 获取用户列表
// @Description 获取所有用户列表（仅管理员可用）
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param request query model.UserListRequest true "查询参数"
// @Success 200 {object} response.Response{data=model.UserListResponse}
// @Router /api/v1/user/list [get]
func (c *UserController) ListUsers(ctx *gin.Context) {
	var req model.UserListRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		response.Error(ctx, 400, "参数错误")
		return
	}

	// 设置默认值
	if req.Current == 0 {
		req.Current = 1
	}
	if req.Size == 0 {
		req.Size = 10
	}

	result, err := c.userService.GetUserList(ctx, &req)
	if err != nil {
		response.Error(ctx, 500, err.Error())
		return
	}

	response.Success(ctx, result)
}

// CreateUser 创建用户
// @Summary 创建用户
// @Description 创建新用户
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param request body model.UserRegisterRequest true "用户信息"
// @Success 200 {object} response.Response{data=model.UserResponse}
// @Router /api/v1/users [post]
func (c *UserController) CreateUser(ctx *gin.Context) {
	var req model.UserRegisterRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.Error(ctx, 400, "参数错误")
		return
	}

	if err := c.userService.Register(ctx, &req); err != nil {
		response.Error(ctx, 500, err.Error())
		return
	}

	response.Success(ctx, nil)
}

// GetUser 获取用户信息
// @Summary 获取用户信息
// @Description 获取指定用户的详细信息
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param id path int true "用户ID"
// @Success 200 {object} response.Response{data=model.UserResponse}
// @Router /api/v1/users/{id} [get]
func (c *UserController) GetUser(ctx *gin.Context) {
	userID, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		response.Error(ctx, 400, "无效的用户ID")
		return
	}

	user, err := c.userService.GetUserInfo(ctx, userID)
	if err != nil {
		response.Error(ctx, 404, "用户不存在")
		return
	}

	response.Success(ctx, user)
}

// UpdateUser 更新用户信息
// @Summary 更新用户信息
// @Description 更新指定用户的信息
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param id path int true "用户ID"
// @Param request body model.UserUpdateRequest true "用户信息"
// @Success 200 {object} response.Response{data=model.UserResponse}
// @Router /api/v1/users/{id} [put]
func (c *UserController) UpdateUser(ctx *gin.Context) {
	fmt.Println("UpdateUser", ctx.Param("id"))
	userID, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		response.Error(ctx, 400, "无效的用户ID")
		return
	}

	var req model.UserUpdateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.Error(ctx, 400, "参数错误")
		return
	}

	if err := c.userService.UpdateUser(ctx, userID, &req); err != nil {
		response.Error(ctx, 500, err.Error())
		return
	}

	response.Success(ctx, nil)
}

// DeleteUser 删除用户
// @Summary 删除用户
// @Description 删除指定用户
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param id path int true "用户ID"
// @Success 200 {object} response.Response
// @Router /api/v1/users/{id} [delete]
func (c *UserController) DeleteUser(ctx *gin.Context) {
	userID, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		response.Error(ctx, 400, "无效的用户ID")
		return
	}

	if err := c.userService.DeleteUser(ctx, userID); err != nil {
		response.Error(ctx, 500, err.Error())
		return
	}

	response.Success(ctx, nil)
}

// ResetPassword 管理员重置用户密码
// @Summary 管理员重置用户密码
// @Description 管理员重置指定用户的密码为初始密码
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param id path int true "用户ID"
// @Success 200 {object} response.Response{data=string}
// @Router /api/v1/users/{id}/reset-password [post]
func (c *UserController) ResetPassword(ctx *gin.Context) {
	userID, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		response.Error(ctx, 400, "无效的用户ID")
		return
	}

	var req struct {
		NewPassword string `json:"newPassword"`
	}
	_ = ctx.ShouldBindJSON(&req)

	newPwd, err := c.userService.ResetPassword(ctx, userID, req.NewPassword)
	if err != nil {
		response.Error(ctx, 500, err.Error())
		return
	}

	response.Success(ctx, gin.H{"new_password": newPwd})
}

// AssignRoles 为用户分配角色
// @Summary 为用户分配角色
// @Description 为用户分配一个或多个角色
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param id path int true "用户ID"
// @Param request body []int64 true "角色ID列表"
// @Success 200 {object} response.Response
// @Router /api/v1/users/{id}/roles [post]
func (c *UserController) AssignRoles(ctx *gin.Context) {
	userID, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		response.Error(ctx, 400, "无效的用户ID")
		return
	}

	var roleIDs []int64
	if err := ctx.ShouldBindJSON(&roleIDs); err != nil {
		response.Error(ctx, 400, "参数错误")
		return
	}

	if err := c.userService.AssignRoles(ctx, userID, roleIDs); err != nil {
		response.Error(ctx, 500, err.Error())
		return
	}

	response.Success(ctx, nil)
}

// GetUserRoles 获取用户角色
// @Summary 获取用户角色
// @Description 获取指定用户的所有角色
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param id path int true "用户ID"
// @Success 200 {object} response.Response{data=[]model.Role}
// @Router /api/v1/users/{id}/roles [get]
func (c *UserController) GetUserRoles(ctx *gin.Context) {
	userID, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		response.Error(ctx, 400, "无效的用户ID")
		return
	}

	roles, err := c.userService.GetUserRoles(userID)
	if err != nil {
		response.Error(ctx, 500, err.Error())
		return
	}

	response.Success(ctx, roles)
}

// RemoveRole 移除用户角色
// @Summary 移除用户角色
// @Description 移除用户的指定角色
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param id path int true "用户ID"
// @Param role_id path int true "角色ID"
// @Success 200 {object} response.Response
// @Router /api/v1/users/{id}/roles/{role_id} [delete]
func (c *UserController) RemoveRole(ctx *gin.Context) {
	userID, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		response.Error(ctx, 400, "无效的用户ID")
		return
	}

	roleID, err := strconv.ParseInt(ctx.Param("role_id"), 10, 64)
	if err != nil {
		response.Error(ctx, 400, "无效的角色ID")
		return
	}

	if err := c.userService.RemoveRole(ctx, userID, roleID); err != nil {
		response.Error(ctx, 500, err.Error())
		return
	}

	response.Success(ctx, nil)
}
