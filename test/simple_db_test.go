package test

import (
	"recharge-go/internal/model"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestSimpleDBCreation 测试数据库表创建
func TestSimpleDBCreation(t *testing.T) {
	// 1. 初始化测试数据库
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info), // 开启详细日志
	})
	if err != nil {
		t.Fatalf("Failed to connect database: %v", err)
	}

	// 2. 自动迁移表结构
	t.Log("开始迁移表结构...")
	err = db.AutoMigrate(&model.User{}, &model.Platform{}, &model.PlatformAccount{}, &model.BalanceLog{})
	if err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}
	t.Log("表结构迁移完成")

	// 3. 列出所有表
	var tables []string
	db.Raw("SELECT name FROM sqlite_master WHERE type='table'").Scan(&tables)
	t.Logf("数据库中的表: %v", tables)

	// 4. 检查platform_accounts表是否存在
	var tableCount int64
	db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='platform_accounts'").Scan(&tableCount)
	t.Logf("platform_accounts表数量: %d", tableCount)

	// 5. 创建测试数据
	platform := &model.Platform{
		ID:   1,
		Code: "test",
		Name: "测试平台",
	}
	if err := db.Create(platform).Error; err != nil {
		t.Fatalf("Failed to create platform: %v", err)
	}
	t.Log("平台创建成功")

	user := &model.User{
		ID:      1,
		Balance: 100.0,
		Credit:  0.0,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}
	t.Log("用户创建成功")

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
	if err := db.Create(account).Error; err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}
	t.Log("平台账号创建成功")

	// 6. 测试查询
	var retrievedAccount model.PlatformAccount
	err = db.Where("id = ?", 1).First(&retrievedAccount).Error
	if err != nil {
		t.Fatalf("Failed to query account: %v", err)
	}
	t.Logf("查询到的账号: ID=%d, Name=%s", retrievedAccount.ID, retrievedAccount.AccountName)

	// 7. 测试带Preload的查询（模拟repository的查询方式）
	var accountWithPlatform model.PlatformAccount
	err = db.Preload("Platform").Where("id = ?", 1).First(&accountWithPlatform).Error
	if err != nil {
		t.Fatalf("Failed to query account with preload: %v", err)
	}
	t.Logf("带Preload查询成功: ID=%d, Name=%s", accountWithPlatform.ID, accountWithPlatform.AccountName)

	t.Log("✅ 数据库表创建和查询测试通过")
}