package main

import (
	"log"
	"recharge-go/internal/model"
	"recharge-go/internal/repository"
	"recharge-go/pkg/database"
)

func main() {
	// 初始化数据库连接
	err := database.InitDB()
	if err != nil {
		log.Fatalf("初始化数据库连接失败: %v", err)
	}

	// 创建仓库
	permissionRepo := repository.NewPermissionRepository(database.DB)

	// 定义系统管理权限
	sysMgt := model.Permission{
		ID:     2,
		Name:   "系统管理",
		Code:   "SysMgt",
		Type:   "MENU",
		Icon:   "i-fe:grid",
		Show:   1,
		Enable: 1,
		Order:  2,
	}

	// 创建系统管理权限
	err = permissionRepo.Create(&sysMgt)
	if err != nil {
		log.Printf("创建系统管理权限失败: %v", err)
	} else {
		log.Printf("创建系统管理权限成功 (ID: %d)", sysMgt.ID)
	}

	// 定义子权限
	permissions := []model.Permission{
		{
			ID:        1,
			Name:      "资源管理",
			Code:      "Resource_Mgt",
			Type:      "MENU",
			ParentID:  &sysMgt.ID,
			Path:      "/pms/resource",
			Icon:      "i-fe:list",
			Component: "/src/views/pms/resource/index.vue",
			Show:      1,
			Enable:    1,
			Order:     1,
		},
		{
			ID:        3,
			Name:      "角色管理",
			Code:      "RoleMgt",
			Type:      "MENU",
			ParentID:  &sysMgt.ID,
			Path:      "/pms/role",
			Icon:      "i-fe:user-check",
			Component: "/src/views/pms/role/index.vue",
			Show:      1,
			Enable:    1,
			Order:     2,
		},
		{
			ID:        5,
			Name:      "分配用户",
			Code:      "RoleUser",
			Type:      "MENU",
			ParentID:  int64Ptr(3),
			Path:      "/pms/role/user/:roleId",
			Icon:      "i-fe:user-plus",
			Component: "/src/views/pms/role/role-user.vue",
			Layout:    "full",
			Show:      0,
			Enable:    1,
			Order:     1,
		},
		{
			ID:        4,
			Name:      "用户管理",
			Code:      "UserMgt",
			Type:      "MENU",
			ParentID:  &sysMgt.ID,
			Path:      "/pms/user",
			Icon:      "i-fe:user",
			Component: "/src/views/pms/user/index.vue",
			KeepAlive: 1,
			Show:      1,
			Enable:    1,
			Order:     3,
		},
		{
			ID:       13,
			Name:     "创建新用户",
			Code:     "AddUser",
			Type:     "BUTTON",
			ParentID: int64Ptr(4),
			Show:     1,
			Enable:   1,
			Order:    1,
		},
	}

	// 创建子权限
	for _, permission := range permissions {
		err = permissionRepo.Create(&permission)
		if err != nil {
			log.Printf("创建权限失败 (权限代码: %s): %v", permission.Code, err)
			continue
		}
		log.Printf("创建权限成功 (权限代码: %s)", permission.Code)
	}

	log.Println("权限初始化完成")
}

func stringPtr(s string) *string {
	return &s
}

func int64Ptr(i int64) *int64 {
	return &i
}

func boolPtr(b bool) *bool {
	return &b
}
