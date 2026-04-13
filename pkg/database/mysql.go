package database

import (
	"fmt"
	"recharge-go/internal/model"
	"recharge-go/internal/model/notification"
	"recharge-go/pkg/database/migrations"

	"gorm.io/gorm"
)

// AutoMigrateDB 对传入的 *gorm.DB 执行模型迁移与基础数据初始化
func AutoMigrateDB(db *gorm.DB) error {
	if err := db.Exec("SET FOREIGN_KEY_CHECKS = 0").Error; err != nil {
		return fmt.Errorf("failed to disable foreign key checks: %v", err)
	}

	if err := db.AutoMigrate(
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
		&model.DaichongOrder{},
		&model.PlatformAccount{},
		&model.SystemConfig{},
		&model.Platform{},
		&model.ExternalAPIKey{},
		&model.ExternalOrderLog{},
		&model.BalanceQueryRecord{},
		&model.OrderException{},
		&model.OrderTraceEvent{},
		&model.OrderRetryRecord{},
		&model.PlatformAccountVariant{},
		&model.PullTaskConfig{},
	); err != nil {
		return fmt.Errorf("failed to migrate tables: %v", err)
	}

	if err := db.Exec("SET FOREIGN_KEY_CHECKS = 1").Error; err != nil {
		return fmt.Errorf("failed to enable foreign key checks: %v", err)
	}

	if err := migrations.InitRoles(db); err != nil {
		return fmt.Errorf("failed to init roles: %v", err)
	}

	if err := migrations.InitAdmin(db); err != nil {
		return fmt.Errorf("failed to init admin: %v", err)
	}

	return nil
}
