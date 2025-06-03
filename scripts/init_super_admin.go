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
	roleRepo := repository.NewRoleRepository(database.DB)
	permissionRepo := repository.NewPermissionRepository(database.DB)
	userRepo := repository.NewUserRepository(database.DB)

	// 创建超级管理员角色
	superAdmin := &model.Role{
		Name:        "超级管理员",
		Code:        "SUPER_ADMIN",
		Description: "系统超级管理员，拥有所有权限",
		Status:      1, // 1: enabled
	}

	// 检查角色是否已存在
	existingRole, err := roleRepo.GetByCode(superAdmin.Code)
	if err == nil {
		log.Printf("超级管理员角色已存在，ID: %d", existingRole.ID)
		superAdmin = existingRole
	} else {
		// 创建超级管理员角色
		err = roleRepo.Create(superAdmin)
		if err != nil {
			log.Fatalf("创建超级管理员角色失败: %v", err)
		}
		log.Printf("创建超级管理员角色成功，ID: %d", superAdmin.ID)
	}

	// 获取所有权限
	permissions, err := permissionRepo.GetAll()
	if err != nil {
		log.Fatalf("获取权限列表失败: %v", err)
	}

	// 先移除所有权限
	err = roleRepo.RemoveAllRolePermissions(superAdmin.ID)
	if err != nil {
		log.Printf("移除超级管理员权限失败: %v", err)
	}

	// 为超级管理员分配所有权限
	for _, permission := range permissions {
		err = roleRepo.AddRolePermission(superAdmin.ID, permission.ID)
		if err != nil {
			log.Printf("为超级管理员分配权限失败 (权限ID: %d): %v", permission.ID, err)
			continue
		}
		log.Printf("为超级管理员分配权限成功 (权限ID: %d, 权限名称: %s)", permission.ID, permission.Name)
	}

	// 查找 admin 用户
	adminUser, err := userRepo.GetByUsername("admin")
	if err != nil {
		log.Fatalf("查找 admin 用户失败: %v", err)
	}

	// 将 admin 用户添加到超级管理员角色
	err = userRepo.AddUserRole(adminUser.ID, superAdmin.ID)
	if err != nil {
		log.Fatalf("将 admin 用户添加到超级管理员角色失败: %v", err)
	}
	log.Printf("成功将 admin 用户添加到超级管理员角色")

	log.Println("超级管理员角色初始化完成")
}
