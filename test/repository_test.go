package test

import (
	"recharge-go/internal/model"
	"recharge-go/internal/repository"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestRepositoryGetByID 测试repository的GetByID方法
func TestRepositoryGetByID(t *testing.T) {
	// 1. 初始化测试数据库
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		t.Fatalf("Failed to connect database: %v", err)
	}

	// 2. 自动迁移表结构
	err = db.AutoMigrate(&model.User{}, &model.Platform{}, &model.PlatformAccount{}, &model.BalanceLog{})
	if err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// 3. 创建测试数据
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
	if err := db.Create(account).Error; err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	// 4. 测试repository
	platformAccountRepo := repository.NewPlatformAccountRepository(db)

	// 5. 测试GetByID方法
	t.Log("测试GetByID方法...")
	retrievedAccount, err := platformAccountRepo.GetByID(1)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	t.Logf("GetByID成功: %+v", retrievedAccount)

	// 6. 测试不使用Preload的查询
	t.Log("测试不使用Preload的查询...")
	var directAccount model.PlatformAccount
	err = db.Where("id = ?", 1).First(&directAccount).Error
	if err != nil {
		t.Fatalf("Direct query failed: %v", err)
	}

	t.Logf("直接查询成功: %+v", directAccount)
	t.Log("✅ Repository测试通过")
}