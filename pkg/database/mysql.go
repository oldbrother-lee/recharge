package database

import (
	"fmt"
	"recharge-go/configs"
	"recharge-go/internal/model"
	"recharge-go/internal/model/notification"
	"recharge-go/pkg/database/migrations"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Init(cfg *configs.Config) error {
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

	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %v", err)
	}
	if cfg.DB.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(cfg.DB.MaxIdleConns)
	} else {
		sqlDB.SetMaxIdleConns(10)
	}
	if cfg.DB.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(cfg.DB.MaxOpenConns)
	} else {
		sqlDB.SetMaxOpenConns(100)
	}
	if cfg.DB.ConnMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(time.Duration(cfg.DB.ConnMaxLifetime) * time.Second)
	} else {
		sqlDB.SetConnMaxLifetime(time.Hour)
	}

	if err := DB.Exec("SET FOREIGN_KEY_CHECKS = 0").Error; err != nil {
		return fmt.Errorf("failed to disable foreign key checks: %v", err)
	}

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
		&model.BalanceQueryRecord{},
		&model.OrderException{},
		&model.BalanceQueryRecord{},
		&model.OrderRetryRecord{},
		&model.Platform{},
		&model.PlatformAccount{},
		&model.PlatformAccountVariant{},
		&model.PullTaskConfig{},
	); err != nil {
		return fmt.Errorf("failed to migrate tables: %v", err)
	}

	if err := DB.Exec("SET FOREIGN_KEY_CHECKS = 1").Error; err != nil {
		return fmt.Errorf("failed to enable foreign key checks: %v", err)
	}

	if err := migrations.InitRoles(DB); err != nil {
		return fmt.Errorf("failed to init roles: %v", err)
	}

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
