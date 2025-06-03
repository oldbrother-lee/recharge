package migrations

import (
	"recharge-go/internal/model"
	"time"

	"gorm.io/gorm"
)

func InitRoles(db *gorm.DB) error {
	// 检查是否已存在超级管理员角色
	var count int64
	if err := db.Model(&model.Role{}).Where("code = ?", "SUPER_ADMIN").Count(&count).Error; err != nil {
		return err
	}

	// 如果不存在超级管理员角色，则创建
	if count == 0 {
		// 创建超级管理员角色
		superAdmin := &model.Role{
			Name:        "超级管理员",
			Code:        "SUPER_ADMIN",
			Description: "系统超级管理员，拥有所有权限",
			Status:      1,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		if err := db.Create(superAdmin).Error; err != nil {
			return err
		}

		// 创建基础权限
		permissions := []*model.Permission{
			{
				Code:        "system",
				Name:        "系统管理",
				Type:        "MENU",
				Path:        "/system",
				Component:   "/src/views/system/index.vue",
				Icon:        "i-fe:settings",
				Description: "系统管理菜单",
				Show:        1,
				Enable:      1,
				Order:       1,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
			{
				Code:        "user",
				Name:        "用户管理",
				Type:        "MENU",
				Path:        "/system/user",
				Component:   "/src/views/system/user/index.vue",
				Icon:        "i-fe:user",
				Description: "用户管理菜单",
				Show:        1,
				Enable:      1,
				Order:       2,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
			{
				Code:        "role",
				Name:        "角色管理",
				Type:        "MENU",
				Path:        "/system/role",
				Component:   "/src/views/system/role/index.vue",
				Icon:        "i-fe:user-check",
				Description: "角色管理菜单",
				Show:        1,
				Enable:      1,
				Order:       3,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
			{
				Code:        "permission",
				Name:        "权限管理",
				Type:        "MENU",
				Path:        "/system/permission",
				Component:   "/src/views/system/permission/index.vue",
				Icon:        "i-fe:lock",
				Description: "权限管理菜单",
				Show:        1,
				Enable:      1,
				Order:       4,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
		}

		for _, permission := range permissions {
			if err := db.Create(permission).Error; err != nil {
				return err
			}

			// 创建角色权限关联
			rolePermission := &model.RolePermission{
				RoleID:       superAdmin.ID,
				PermissionID: permission.ID,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			}
			if err := db.Create(rolePermission).Error; err != nil {
				return err
			}
		}
	}

	return nil
}
