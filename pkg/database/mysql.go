package database

import (
	"fmt"
	"recharge-go/configs"
	"recharge-go/internal/model"
	"recharge-go/internal/model/notification"
	"recharge-go/pkg/database/migrations"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

// InitDB 初始化数据库连接
func InitDB() error {
	cfg := configs.GetConfig()
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.DB.User,
		cfg.DB.Password,
		cfg.DB.Host,
		cfg.DB.Port,
		cfg.DB.Name,
	)

	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %v", err)
	}

	// 设置数据库连接池
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %v", err)
	}
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)

	// 禁用外键检查
	if err := DB.Exec("SET FOREIGN_KEY_CHECKS = 0").Error; err != nil {
		return fmt.Errorf("failed to disable foreign key checks: %v", err)
	}

	// 创建所有表
	if err := DB.AutoMigrate(
		&model.User{},
		&model.Role{},
		&model.Permission{},
		&model.UserRole{},
		&model.RolePermission{},
		&model.ProductCategory{},
		&model.Product{},
		&model.ProductSpec{},
		&model.MemberGrade{},
		&model.ProductGradePrice{},
		&model.PlatformAPI{},
		&model.PlatformAPIParam{},
		&model.ProductAPIRelation{},
		&model.APICallLog{},
		&model.DistributionGrade{},
		&model.DistributionRule{},
		&model.DistributionWithdrawal{},
		&model.Distributor{},
		&model.DistributorStatistics{},
		&model.Admin{},
		&model.UserLog{},
		&model.UserTagRelation{},
		&model.UserGradeRelation{},
		&model.UserTag{},
		&model.UserGrade{},
		&model.Order{},
		&model.RechargeTask{},
		&model.CallbackLog{},
		&notification.NotificationRecord{},
		&notification.Template{},
		&model.BalanceLog{},
		&model.CreditLog{},
		&model.TaskConfig{},
		&model.TaskOrder{},
		&model.OrderStatistics{},
		&model.PlatformToken{},
		&model.TaskOrder{},
		&model.DaichongOrder{},
		&model.PlatformAccount{},
		&model.SystemConfig{},
		&model.Platform{},
		&model.PlatformAccount{},
		&model.ExternalAPIKey{},
		&model.ExternalOrderLog{},
	); err != nil {
		return fmt.Errorf("failed to migrate tables: %v", err)
	}

	// 启用外键检查
	if err := DB.Exec("SET FOREIGN_KEY_CHECKS = 1").Error; err != nil {
		return fmt.Errorf("failed to enable foreign key checks: %v", err)
	}

	// 初始化角色和权限
	if err := migrations.InitRoles(DB); err != nil {
		return fmt.Errorf("failed to init roles: %v", err)
	}

	// 初始化管理员账号
	if err := migrations.InitAdmin(DB); err != nil {
		return fmt.Errorf("failed to init admin: %v", err)
	}

	return nil
}

// Close 关闭数据库连接
func Close() error {
	if DB != nil {
		sqlDB, err := DB.DB()
		if err != nil {
			return fmt.Errorf("failed to get database instance: %v", err)
		}
		return sqlDB.Close()
	}
	return nil
}
