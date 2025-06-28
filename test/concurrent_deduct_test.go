package test

// 简化的并发测试，专注于验证事务和锁的有效性

import (
	"context"
	"fmt"
	"recharge-go/internal/model"
	"recharge-go/internal/repository"
	"recharge-go/internal/service"
	"sync"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestConcurrentDeductBalance 测试并发扣款的安全性
func TestConcurrentDeductBalance(t *testing.T) {
	// 1. 初始化测试数据库
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("Failed to connect database: %v", err)
	}

	// 2. 自动迁移表结构
	err = db.AutoMigrate(&model.User{}, &model.Platform{}, &model.PlatformAccount{}, &model.BalanceLog{})
	if err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// 验证表是否创建成功
	var tableCount int64
	db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='balance_logs'").Scan(&tableCount)
	if tableCount == 0 {
		t.Fatal("balance_logs table was not created")
	}

	// 3. 创建测试数据
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

	// 创建用户，初始余额1000元
	user := &model.User{
		ID:      userID,
		Balance: 1000.0,
		Credit:  0.0,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// 创建平台账号
	account := &model.PlatformAccount{
		ID:         accountID,
		PlatformID: platformID,
		BindUserID: &userID,
	}
	if err := db.Create(account).Error; err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	// 4. 初始化服务
	platformAccountRepo := repository.NewPlatformAccountRepository(db)
	userRepo := repository.NewUserRepository(db)
	balanceLogRepo := repository.NewBalanceLogRepository(db)
	balanceService := service.NewPlatformAccountBalanceService(db, platformAccountRepo, userRepo, balanceLogRepo)

	// 5. 并发测试参数
	concurrentCount := 20 // 并发数
	deductAmount := 100.0 // 每次扣款金额
	expectedSuccessCount := 10 // 预期成功次数（1000/100=10）

	// 6. 执行并发扣款测试
	var wg sync.WaitGroup
	var successCount int64
	var failureCount int64
	var mu sync.Mutex

	for i := 0; i < concurrentCount; i++ {
		wg.Add(1)
		go func(orderID int) {
			defer wg.Done()
			err := balanceService.DeductBalance(ctx, accountID, deductAmount, int64(orderID), fmt.Sprintf("测试订单%d", orderID))
			mu.Lock()
			if err != nil {
				failureCount++
				t.Logf("订单%d扣款失败: %v", orderID, err)
			} else {
				successCount++
				t.Logf("订单%d扣款成功", orderID)
			}
			mu.Unlock()
		}(i + 1)
	}

	// 等待所有goroutine完成
	wg.Wait()

	// 7. 验证结果
	t.Logf("并发扣款测试完成: 成功%d次, 失败%d次", successCount, failureCount)

	// 检查成功次数是否符合预期
	if successCount != int64(expectedSuccessCount) {
		t.Errorf("预期成功%d次，实际成功%d次", expectedSuccessCount, successCount)
	}

	// 检查用户最终余额
	var finalUser model.User
	db.First(&finalUser, userID)
	expectedBalance := 1000.0 - float64(successCount)*deductAmount
	if finalUser.Balance != expectedBalance {
		t.Errorf("预期最终余额%.2f，实际余额%.2f", expectedBalance, finalUser.Balance)
	}

	// 检查余额日志数量
	var logCount int64
	db.Model(&model.BalanceLog{}).Where("user_id = ? AND style = ?", userID, model.BalanceStyleOrderDeduct).Count(&logCount)
	if logCount != successCount {
		t.Errorf("预期日志数量%d，实际日志数量%d", successCount, logCount)
	}

	// 8. 测试幂等性
	t.Log("开始测试幂等性...")
	// 重复执行第一个订单的扣款，应该被跳过
	err = balanceService.DeductBalance(ctx, accountID, deductAmount, 1, "重复测试订单1")
	if err != nil {
		t.Errorf("幂等性测试失败: %v", err)
	}

	// 验证余额没有变化
	var idempotentUser model.User
	db.First(&idempotentUser, userID)
	if idempotentUser.Balance != finalUser.Balance {
		t.Errorf("幂等性测试失败，余额发生了变化: %.2f -> %.2f", finalUser.Balance, idempotentUser.Balance)
	}

	t.Log("并发扣款安全性测试通过！")
}

// TestConcurrentDeductWithCredit 测试包含授信额度的并发扣款
func TestConcurrentDeductWithCredit(t *testing.T) {
	// 1. 初始化测试数据库
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
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
	ctx := context.Background()
	userID := int64(2)
	accountID := int64(2)
	platformID := int64(2)

	// 创建平台
	platform := &model.Platform{
		ID:   platformID,
		Code: "test2",
		Name: "测试平台2",
	}
	if err := db.Create(platform).Error; err != nil {
		t.Fatalf("Failed to create platform: %v", err)
	}

	// 创建用户，余额500元，授信额度1000元
	user := &model.User{
		ID:      userID,
		Balance: 500.0,
		Credit:  1000.0,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// 创建平台账号
	account := &model.PlatformAccount{
		ID:         accountID,
		PlatformID: platformID,
		BindUserID: &userID,
	}
	if err := db.Create(account).Error; err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	// 4. 初始化服务
	platformAccountRepo := repository.NewPlatformAccountRepository(db)
	userRepo := repository.NewUserRepository(db)
	balanceLogRepo := repository.NewBalanceLogRepository(db)
	balanceService := service.NewPlatformAccountBalanceService(db, platformAccountRepo, userRepo, balanceLogRepo)

	// 5. 并发测试参数
	concurrentCount := 30 // 并发数
	deductAmount := 100.0 // 每次扣款金额
	expectedSuccessCount := 15 // 预期成功次数（(500+1000)/100=15）

	// 6. 执行并发扣款测试
	var wg sync.WaitGroup
	var successCount int64
	var failureCount int64
	var mu sync.Mutex

	for i := 0; i < concurrentCount; i++ {
		wg.Add(1)
		go func(orderID int) {
			defer wg.Done()
			err := balanceService.DeductBalance(ctx, accountID, deductAmount, int64(orderID+100), fmt.Sprintf("授信测试订单%d", orderID))
			mu.Lock()
			if err != nil {
				failureCount++
				t.Logf("订单%d扣款失败: %v", orderID+100, err)
			} else {
				successCount++
				t.Logf("订单%d扣款成功", orderID+100)
			}
			mu.Unlock()
		}(i + 1)
	}

	// 等待所有goroutine完成
	wg.Wait()

	// 7. 验证结果
	t.Logf("授信额度并发扣款测试完成: 成功%d次, 失败%d次", successCount, failureCount)

	// 检查成功次数是否符合预期
	if successCount != int64(expectedSuccessCount) {
		t.Errorf("预期成功%d次，实际成功%d次", expectedSuccessCount, successCount)
	}

	// 检查用户最终余额（应该为负数，使用了授信额度）
	var finalUser model.User
	db.First(&finalUser, userID)
	expectedBalance := 500.0 - float64(successCount)*deductAmount
	if finalUser.Balance != expectedBalance {
		t.Errorf("预期最终余额%.2f，实际余额%.2f", expectedBalance, finalUser.Balance)
	}

	// 验证授信额度使用情况
	if finalUser.Balance < 0 {
		creditUsed := -finalUser.Balance
		t.Logf("使用授信额度: %.2f", creditUsed)
		if creditUsed > finalUser.Credit {
			t.Errorf("使用的授信额度%.2f超过了可用额度%.2f", creditUsed, finalUser.Credit)
		}
	}

	t.Log("授信额度并发扣款安全性测试通过！")
}