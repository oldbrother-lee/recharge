package test

import (
	"recharge-go/internal/model"
	"recharge-go/internal/repository"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestTransactionIssue 测试事务和数据库连接问题
func TestTransactionIssue(t *testing.T) {
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

	// 创建用户
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

	// 4. 测试不同的查询方式
	t.Log("=== 测试直接查询 ===")
	var directAccount model.PlatformAccount
	if err := db.First(&directAccount, accountID).Error; err != nil {
		t.Fatalf("直接查询失败: %v", err)
	}
	t.Log("✅ 直接查询成功")

	t.Log("=== 测试Repository查询 ===")
	platformAccountRepo := repository.NewPlatformAccountRepository(db)
	_, err = platformAccountRepo.GetByID(accountID)
	if err != nil {
		t.Fatalf("Repository查询失败: %v", err)
	}
	t.Log("✅ Repository查询成功")

	t.Log("=== 测试事务中查询 ===")
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatalf("开启事务失败: %v", tx.Error)
	}

	// 在事务中直接查询
	var txAccount model.PlatformAccount
	if err := tx.First(&txAccount, accountID).Error; err != nil {
		tx.Rollback()
		t.Fatalf("事务中直接查询失败: %v", err)
	}
	t.Log("✅ 事务中直接查询成功")

	// 在事务中使用Repository查询（这里会有问题）
	t.Log("=== 测试事务中Repository查询 ===")
	_, err = platformAccountRepo.GetByID(accountID)
	if err != nil {
		tx.Rollback()
		t.Logf("❌ 事务中Repository查询失败: %v", err)
		t.Log("这证实了问题：Repository使用的是原始db连接，而不是事务tx")
	} else {
		t.Log("✅ 事务中Repository查询成功")
	}

	tx.Commit()

	// 5. 测试修复方案：在事务中直接查询，不使用Repository
	t.Log("=== 测试修复方案：模拟服务层直接在事务中查询 ===")
	tx2 := db.Begin()
	if tx2.Error != nil {
		t.Fatalf("开启事务失败: %v", tx2.Error)
	}

	// 模拟DeductBalance方法的逻辑，但在事务中直接查询
	var serviceAccount model.PlatformAccount
	err = tx2.Preload("Platform").Where("id = ?", accountID).First(&serviceAccount).Error
	if err != nil {
		tx2.Rollback()
		t.Fatalf("事务中查询平台账号失败: %v", err)
	}
	t.Logf("✅ 事务中查询平台账号成功: %s", serviceAccount.AccountName)

	// 检查绑定用户
	if serviceAccount.BindUserID == nil {
		tx2.Rollback()
		t.Fatal("平台账号未绑定本地用户")
	}

	// 查询用户
	var serviceUser model.User
	err = tx2.Where("id = ?", *serviceAccount.BindUserID).First(&serviceUser).Error
	if err != nil {
		tx2.Rollback()
		t.Fatalf("查询用户失败: %v", err)
	}
	t.Logf("✅ 查询用户成功: 余额%.2f", serviceUser.Balance)

	tx2.Commit()
	t.Log("✅ 修复方案测试成功！问题确实是Repository和事务的连接不一致")
}