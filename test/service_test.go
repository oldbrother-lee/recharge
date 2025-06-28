package test

import (
	"context"
	"recharge-go/internal/model"
	"recharge-go/internal/repository"
	"recharge-go/internal/service"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestServiceDeductBalance 测试服务层的扣款功能
func TestServiceDeductBalance(t *testing.T) {
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

	// 3. 验证表创建
	var tables []string
	db.Raw("SELECT name FROM sqlite_master WHERE type='table'").Scan(&tables)
	t.Logf("数据库中的表: %v", tables)

	// 4. 创建测试数据
	ctx := context.Background()
	userID := int64(1)
	accountID := int64(1)
	platformID := int64(1)

	// 创建平台
	platform := &model.Platform{
		ID:   platformID,
		Code: "test",
		Name: "测试平台",
	}
	if err := db.Create(platform).Error; err != nil {
		t.Fatalf("Failed to create platform: %v", err)
	}

	// 创建用户，初始余额100元
	user := &model.User{
		ID:      userID,
		Balance: 100.0,
		Credit:  0.0,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// 创建平台账号
	account := &model.PlatformAccount{
		ID:           accountID,
		PlatformID:   platformID,
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

	// 5. 验证数据创建成功
	var createdAccount model.PlatformAccount
	if err := db.First(&createdAccount, accountID).Error; err != nil {
		t.Fatalf("Failed to query created account: %v", err)
	}
	t.Logf("创建的账号: %+v", createdAccount)

	// 6. 初始化服务 - 确保所有repository使用同一个数据库连接
	platformAccountRepo := repository.NewPlatformAccountRepository(db)
	userRepo := repository.NewUserRepository(db)
	balanceLogRepo := repository.NewBalanceLogRepository(db)
	balanceService := service.NewPlatformAccountBalanceService(db, platformAccountRepo, userRepo, balanceLogRepo)

	// 7. 测试repository的GetByID方法
	t.Log("测试repository GetByID...")
	retrievedAccount, err := platformAccountRepo.GetByID(accountID)
	if err != nil {
		t.Fatalf("Repository GetByID failed: %v", err)
	}
	t.Logf("Repository查询成功: ID=%d, Name=%s", retrievedAccount.ID, retrievedAccount.AccountName)

	// 8. 测试服务层扣款
	t.Log("测试服务层扣款...")
	deductAmount := 50.0
	orderID := int64(100)
	err = balanceService.DeductBalance(ctx, accountID, deductAmount, orderID, "测试扣款")
	if err != nil {
		t.Fatalf("服务层扣款失败: %v", err)
	}

	// 9. 验证扣款结果
	var userAfterDeduct model.User
	db.First(&userAfterDeduct, userID)
	expectedBalance := 50.0 // 100 - 50
	if userAfterDeduct.Balance != expectedBalance {
		t.Errorf("余额不正确！预期%.2f，实际%.2f", expectedBalance, userAfterDeduct.Balance)
	} else {
		t.Logf("✅ 扣款成功，余额从100.00变为%.2f", userAfterDeduct.Balance)
	}

	// 10. 检查扣款日志
	var deductLog model.BalanceLog
	db.Where("user_id = ? AND order_id = ? AND style = ?", userID, orderID, model.BalanceStyleOrderDeduct).First(&deductLog)
	if deductLog.ID == 0 {
		t.Error("❌ 扣款日志未创建")
	} else {
		t.Logf("✅ 扣款日志创建成功: 金额%.2f, 余额变化%.2f->%.2f", deductLog.Amount, deductLog.BalanceBefore, deductLog.Balance)
	}
}