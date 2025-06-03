package controller

import (
	"net/http"
	"strconv"

	"recharge-go/internal/model"
	"recharge-go/internal/service"
	"recharge-go/internal/utils"

	"github.com/gin-gonic/gin"
)

// PermissionController handles permission-related HTTP requests
type PermissionController struct {
	service *service.PermissionService
}

// NewPermissionController creates a new PermissionController
func NewPermissionController(service *service.PermissionService) *PermissionController {
	return &PermissionController{
		service: service,
	}
}

// List 获取权限列表
func (c *PermissionController) List(ctx *gin.Context) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "10"))

	permissions, total, err := c.service.List(page, pageSize)
	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, "获取权限列表失败")
		return
	}
	utils.Success(ctx, gin.H{
		"total": total,
		"list":  permissions,
	})
}

// Create 创建权限
func (c *PermissionController) Create(ctx *gin.Context) {
	var req model.PermissionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.Error(ctx, http.StatusBadRequest, "无效的输入数据")
		return
	}

	if err := c.service.CreatePermission(&req); err != nil {
		utils.Error(ctx, http.StatusInternalServerError, "创建权限失败")
		return
	}

	utils.Success(ctx, nil)
}

// Update 更新权限
func (c *PermissionController) Update(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		utils.Error(ctx, http.StatusBadRequest, "无效的ID")
		return
	}

	var req model.PermissionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.Error(ctx, http.StatusBadRequest, "无效的输入数据")
		return
	}

	if err := c.service.Update(id, &req); err != nil {
		utils.Error(ctx, http.StatusInternalServerError, "更新权限失败")
		return
	}

	utils.Success(ctx, nil)
}

// Delete 删除权限
func (c *PermissionController) Delete(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		utils.Error(ctx, http.StatusBadRequest, "无效的ID")
		return
	}

	if err := c.service.DeletePermission(id); err != nil {
		utils.Error(ctx, http.StatusInternalServerError, "删除权限失败")
		return
	}

	utils.Success(ctx, nil)
}

// GetByID 根据ID获取权限
func (c *PermissionController) GetByID(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		utils.Error(ctx, http.StatusBadRequest, "无效的ID")
		return
	}

	permission, err := c.service.GetByID(id)
	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, "获取权限失败")
		return
	}

	utils.Success(ctx, permission)
}

// GetByRoleID 根据角色ID获取权限
func (c *PermissionController) GetByRoleID(ctx *gin.Context) {
	roleID, err := strconv.ParseInt(ctx.Param("role_id"), 10, 64)
	if err != nil {
		utils.Error(ctx, http.StatusBadRequest, "无效的角色ID")
		return
	}

	permissions, err := c.service.GetByRoleID(roleID)
	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, "获取权限失败")
		return
	}

	utils.Success(ctx, permissions)
}

// AssignToRole 为角色分配权限
func (c *PermissionController) AssignToRole(ctx *gin.Context) {
	roleID, err := strconv.ParseInt(ctx.Param("role_id"), 10, 64)
	if err != nil {
		utils.Error(ctx, http.StatusBadRequest, "无效的角色ID")
		return
	}

	var permissionIDs []int64
	if err := ctx.ShouldBindJSON(&permissionIDs); err != nil {
		utils.Error(ctx, http.StatusBadRequest, "无效的输入数据")
		return
	}

	if err := c.service.AssignToRole(roleID, permissionIDs); err != nil {
		utils.Error(ctx, http.StatusInternalServerError, "分配权限失败")
		return
	}

	utils.Success(ctx, "分配权限成功")
}

// GetTree 获取权限树
func (c *PermissionController) GetTree(ctx *gin.Context) {
	tree, err := c.service.GetPermissionTree()
	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, "获取权限树失败")
		return
	}

	utils.Success(ctx, tree)
}

// GetMenuPermissions 获取菜单权限
func (c *PermissionController) GetMenuPermissions(ctx *gin.Context) {
	menus, err := c.service.GetMenuPermissions()
	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, "获取菜单权限失败")
		return
	}

	utils.Success(ctx, menus)
}

// GetAllPermissions 获取所有权限
func (c *PermissionController) GetAllPermissions(ctx *gin.Context) {
	permissions, err := c.service.GetAllPermissions()
	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, "获取所有权限失败")
		return
	}
	utils.Success(ctx, permissions)
}

// GetButtonPermissions 获取按钮权限
func (c *PermissionController) GetButtonPermissions(ctx *gin.Context) {
	buttons, err := c.service.GetButtonPermissions()
	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, "获取按钮权限失败")
		return
	}

	utils.Success(ctx, buttons)
}
