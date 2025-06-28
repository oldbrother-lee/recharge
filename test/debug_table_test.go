package test

import (
	"recharge-go/internal/model"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestTableNameDebug 调试表名问题
func TestTableNameDebug(t *testing.T) {
	// 1. 初始化测试数据库
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info), // 开启详细日志
	})
	if err != nil {
		t.Fatalf("Failed to connect database: %v", err)
	}

	// 2. 检查表名
	var platformAccount model.PlatformAccount
	tableName := platformAccount.TableName()
	t.Logf("PlatformAccount表名: %s", tableName)

	// 3. 自动迁移表结构
	t.Log("开始迁移表结构...")
	err = db.AutoMigrate(&model.User{}, &model.Platform{}, &model.PlatformAccount{}, &model.BalanceLog{})
	if err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// 4. 检查表是否存在
	var tables []string
	db.Raw("SELECT name FROM sqlite_master WHERE type='table'").Scan(&tables)
	t.Logf("数据库中的表: %v", tables)

	// 5. 验证platform_accounts表
	var tableCount int64
	db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='platform_accounts'").Scan(&tableCount)
	t.Logf("platform_accounts表数量: %d", tableCount)

	// 6. 创建测试数据
	platform := &model.Platform{
		ID:   1,
		Code: "test",
		Name: "测试平台",
	}
	if err := db.Create(platform).Error; err != nil {
		t.Fatalf("Failed to create platform: %v", err)
	}

	userID := int64(1)
	account := &model.PlatformAccount{
		ID:           1,
		PlatformID:   1,
		BindUserID:   &userID,
		AccountName:  "测试账号",
		Type:         1,
		AppKey:       "test_key",
		AppSecret:    "test_secret",
		Description:  "测试账号",
		DailyLimit:   1000.0,
		MonthlyLimit: 30000.0,
		Balance:      0.0,
		Priority:     1,
		Status:       1,
	}

	t.Log("创建PlatformAccount...")
	if err := db.Create(account).Error; err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	// 7. 查询测试
	t.Log("查询PlatformAccount...")
	var retrievedAccount model.PlatformAccount
	if err := db.First(&retrievedAccount, 1).Error; err != nil {
		t.Fatalf("Failed to query account: %v", err)
	}

	t.Logf("查询成功: %+v", retrievedAccount)
	t.Log("✅ 表名和查询都正常工作")
}