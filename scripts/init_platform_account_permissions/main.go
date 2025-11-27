package main

import (
	"log"
	"recharge-go/internal/model"
	"recharge-go/internal/repository"
	"recharge-go/pkg/database"
	"time"

	"github.com/spf13/viper"
)

func main() {
	// 初始化数据库连接
	viper.SetConfigFile("configs/config.yaml")
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("读取配置失败: %v", err)
	}
	dbm, err := database.NewDatabaseManager(&database.DatabaseConfig{
		Host:     viper.GetString("database.host"),
		Port:     viper.GetInt("database.port"),
		User:     viper.GetString("database.user"),
		Password: viper.GetString("database.password"),
		Name:     viper.GetString("database.dbname"),
		Charset:  "utf8mb4",
	})
	if err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}
	db := dbm.GetDB()

	// 创建权限仓库
	permissionRepo := repository.NewPermissionRepository(db)
	roleRepo := repository.NewRoleRepository(db)

	// 定义平台账号管理权限
	permissions := []model.Permission{
		{
			Code:        "platform:account",
			Name:        "平台账号管理",
			Type:        "MENU",
			Path:        "/platform",
			Component:   "/src/views/platform/index.vue",
			Icon:        "mdi:account-multiple",
			Description: "平台账号管理菜单",
			Show:        1,
			Enable:      1,
			Order:       9,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			Code:        "platform:account:list",
			Name:        "查看平台账号",
			Type:        "BUTTON",
			Description: "查看平台账号列表权限",
			Show:        1,
			Enable:      1,
			Order:       1,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			Code:        "platform:account:add",
			Name:        "新增平台账号",
			Type:        "BUTTON",
			Description: "新增平台账号权限",
			Show:        1,
			Enable:      1,
			Order:       2,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			Code:        "platform:account:edit",
			Name:        "编辑平台账号",
			Type:        "BUTTON",
			Description: "编辑平台账号权限",
			Show:        1,
			Enable:      1,
			Order:       3,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			Code:        "platform:account:delete",
			Name:        "删除平台账号",
			Type:        "BUTTON",
			Description: "删除平台账号权限",
			Show:        1,
			Enable:      1,
			Order:       4,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			Code:        "platform:account:variant:list",
			Name:        "查看变体配置",
			Type:        "BUTTON",
			Description: "查看平台账号变体配置权限",
			Show:        1,
			Enable:      1,
			Order:       5,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			Code:        "platform:account:variant:add",
			Name:        "新增变体配置",
			Type:        "BUTTON",
			Description: "新增平台账号变体配置权限",
			Show:        1,
			Enable:      1,
			Order:       6,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			Code:        "platform:account:variant:edit",
			Name:        "编辑变体配置",
			Type:        "BUTTON",
			Description: "编辑平台账号变体配置权限",
			Show:        1,
			Enable:      1,
			Order:       7,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			Code:        "platform:account:variant:delete",
			Name:        "删除变体配置",
			Type:        "BUTTON",
			Description: "删除平台账号变体配置权限",
			Show:        1,
			Enable:      1,
			Order:       8,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}

	// 创建权限并分配给超级管理员
	for _, permission := range permissions {
		// 检查权限是否已存在
		existingPermission, err := permissionRepo.GetByCode(permission.Code)
		if err == nil && existingPermission != nil {
			log.Printf("权限已存在，跳过创建 (权限代码: %s)", permission.Code)
			continue
		}

		// 创建权限
		err = permissionRepo.Create(&permission)
		if err != nil {
			log.Printf("创建权限失败 (权限代码: %s): %v", permission.Code, err)
			continue
		}
		log.Printf("创建权限成功 (权限代码: %s)", permission.Code)
	}

	// 查找超级管理员角色
	superAdmin, err := roleRepo.GetByCode("SUPER_ADMIN")
	if err != nil {
		log.Printf("查找超级管理员角色失败: %v", err)
		return
	}

	// 为超级管理员分配所有拉单源管理权限
	for _, permission := range permissions {
		// 重新获取权限ID
		createdPermission, err := permissionRepo.GetByCode(permission.Code)
		if err != nil || createdPermission == nil {
			log.Printf("获取权限失败 (%s): %v", permission.Code, err)
			continue
		}

		// 检查是否已经分配了该权限
		var existingRolePermission model.RolePermission
		err = db.Where("role_id = ? AND permission_id = ?", superAdmin.ID, createdPermission.ID).First(&existingRolePermission).Error
		if err == nil {
			log.Printf("超级管理员已拥有权限: %s", permission.Code)
			continue
		}

		// 创建角色权限关联
		rolePermission := &model.RolePermission{
			RoleID:       superAdmin.ID,
			PermissionID: createdPermission.ID,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		if err := db.Create(rolePermission).Error; err != nil {
			log.Printf("为超级管理员分配权限失败 (权限: %s): %v", permission.Code, err)
			continue
		}
		log.Printf("为超级管理员分配权限成功: %s", permission.Code)
	}

	log.Println("拉单源管理权限初始化完成")
}
