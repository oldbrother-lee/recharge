package controller

import (
	"recharge-go/internal/model"
	"recharge-go/internal/service"
	"recharge-go/internal/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

type RoleController struct {
	roleService *service.RoleService
}

func NewRoleController(roleService *service.RoleService) *RoleController {
	return &RoleController{
		roleService: roleService,
	}
}

// List 获取角色列表
func (c *RoleController) List(ctx *gin.Context) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(ctx.DefaultQuery("size", "10"))

	roles, total, err := c.roleService.List(page, size)
	if err != nil {
		utils.Error(ctx, 500, err.Error())
		return
	}

	utils.Success(ctx, gin.H{
		"list":  roles,
		"total": total,
	})
}

// GetAll 获取所有角色
func (c *RoleController) GetAll(ctx *gin.Context) {
	roles, err := c.roleService.GetAll()
	if err != nil {
		utils.Error(ctx, 500, err.Error())
		return
	}
	utils.Success(ctx, roles)
}

// Update 更新角色
func (c *RoleController) Update(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		utils.Error(ctx, 400, "Invalid role ID")
		return
	}

	var req model.RoleRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.Error(ctx, 400, err.Error())
		return
	}

	role, err := c.roleService.Update(id, &req)
	if err != nil {
		utils.Error(ctx, 500, err.Error())
		return
	}
	utils.Success(ctx, role)
}

// Delete 删除角色
func (c *RoleController) Delete(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		utils.Error(ctx, 400, "Invalid role ID")
		return
	}

	err = c.roleService.Delete(id)
	if err != nil {
		utils.Error(ctx, 500, err.Error())
		return
	}
	utils.Success(ctx, nil)
}

// Create 创建角色
func (c *RoleController) Create(ctx *gin.Context) {
	var req model.RoleRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.Error(ctx, 400, err.Error())
		return
	}

	role, err := c.roleService.Create(&req)
	if err != nil {
		utils.Error(ctx, 500, err.Error())
		return
	}
	utils.Success(ctx, role)
}

// GetByID 获取角色详情
func (c *RoleController) GetByID(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		utils.Error(ctx, 400, "Invalid role ID")
		return
	}

	role, err := c.roleService.GetByID(id)
	if err != nil {
		utils.Error(ctx, 500, err.Error())
		return
	}
	utils.Success(ctx, role)
}

// AddPermission 添加角色权限
func (c *RoleController) AddPermission(ctx *gin.Context) {
	roleID, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		utils.Error(ctx, 400, "Invalid role ID")
		return
	}

	permissionID, err := strconv.ParseInt(ctx.Param("permission_id"), 10, 64)
	if err != nil {
		utils.Error(ctx, 400, "Invalid permission ID")
		return
	}

	err = c.roleService.AddPermission(roleID, permissionID)
	if err != nil {
		utils.Error(ctx, 500, err.Error())
		return
	}
	utils.Success(ctx, nil)
}

// RemovePermission 移除角色权限
func (c *RoleController) RemovePermission(ctx *gin.Context) {
	roleID, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		utils.Error(ctx, 400, "Invalid role ID")
		return
	}

	permissionID, err := strconv.ParseInt(ctx.Param("permission_id"), 10, 64)
	if err != nil {
		utils.Error(ctx, 400, "Invalid permission ID")
		return
	}

	err = c.roleService.RemovePermission(roleID, permissionID)
	if err != nil {
		utils.Error(ctx, 500, err.Error())
		return
	}
	utils.Success(ctx, nil)
}

// RemoveAllPermissions 移除角色的所有权限
func (c *RoleController) RemoveAllPermissions(ctx *gin.Context) {
	roleID, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		utils.Error(ctx, 400, "Invalid role ID")
		return
	}

	err = c.roleService.RemoveAllPermissions(roleID)
	if err != nil {
		utils.Error(ctx, 500, err.Error())
		return
	}
	utils.Success(ctx, nil)
}
