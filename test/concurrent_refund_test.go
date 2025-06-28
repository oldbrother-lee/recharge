package test

// 测试退款并发安全性和幂等性

import (
	"context"
	"fmt"
	"recharge-go/internal/model"
	"recharge-go/internal/repository"
	"recharge-go/internal/service"
	"sync"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestConcurrentRefundBalance 测试并发退款的安全性
func TestConcurrentRefundBalance(t *testing.T) {
	// 1. 初始化测试数据库 - 使用文件数据库而不是内存数据库，避免连接隔离问题
	db, err := gorm.Open(sqlite.Open("test_concurrent.db"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("Failed to connect database: %v", err)
	}
	// 清理测试数据库
	defer func() {
		db.Exec("DROP TABLE IF EXISTS balance_logs")
		db.Exec("DROP TABLE IF EXISTS platform_accounts")
		db.Exec("DROP TABLE IF EXISTS platforms")
		db.Exec("DROP TABLE IF EXISTS users")
	}()

	// 2. 自动迁移表结构
	err = db.AutoMigrate(&model.User{}, &model.Platform{}, &model.PlatformAccount{}, &model.BalanceLog{})
	if err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// 3. 创建测试数据
	ctx := context.Background()
	userID := int64(1)
	accountID := int64(1)
	platformID := int64(1)
	orderID := int64(100) // 使用同一个订单ID测试并发退款

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

	// 4. 验证表是否正确创建
	var tables []string
	db.Raw("SELECT name FROM sqlite_master WHERE type='table'").Scan(&tables)
	t.Logf("数据库中的表: %v", tables)

	// 验证platform_accounts表
	var tableCount int64
	db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='platform_accounts'").Scan(&tableCount)
	if tableCount == 0 {
		t.Fatal("platform_accounts表未创建成功")
	}

	// 5. 初始化服务
	platformAccountRepo := repository.NewPlatformAccountRepository(db)
	userRepo := repository.NewUserRepository(db)
	balanceLogRepo := repository.NewBalanceLogRepository(db)
	balanceService := service.NewPlatformAccountBalanceService(db, platformAccountRepo, userRepo, balanceLogRepo)

	// 5. 先执行一次扣款，创建扣款记录
	deductAmount := 50.0
	err = balanceService.DeductBalance(ctx, accountID, deductAmount, orderID, "测试订单扣款")
	if err != nil {
		t.Fatalf("扣款失败: %v", err)
	}

	// 验证扣款后余额
	var userAfterDeduct model.User
	db.First(&userAfterDeduct, userID)
	t.Logf("扣款后余额: %.2f", userAfterDeduct.Balance)

	// 6. 并发退款测试参数
	concurrentCount := 10 // 并发数
	refundAmount := 50.0  // 退款金额

	// 7. 执行并发退款测试
	var wg sync.WaitGroup
	var successCount int64
	var failureCount int64
	var mu sync.Mutex

	for i := 0; i < concurrentCount; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			// 使用相同的订单ID进行退款，测试幂等性
			err := balanceService.RefundBalance(ctx, nil, accountID, refundAmount, orderID, fmt.Sprintf("并发退款测试-%d", goroutineID))
			mu.Lock()
			if err != nil {
				failureCount++
				t.Logf("Goroutine %d 退款失败: %v", goroutineID, err)
			} else {
				successCount++
				t.Logf("Goroutine %d 退款成功", goroutineID)
			}
			mu.Unlock()
		}(i + 1)
	}

	// 等待所有goroutine完成
	wg.Wait()

	// 8. 验证结果
	t.Logf("并发退款测试完成: 成功%d次, 失败%d次", successCount, failureCount)

	// 检查用户最终余额 - 应该只退款一次
	var finalUser model.User
	db.First(&finalUser, userID)
	expectedBalance := 100.0 // 初始余额100 - 扣款50 + 退款50 = 100
	t.Logf("最终余额: %.2f, 预期余额: %.2f", finalUser.Balance, expectedBalance)

	if finalUser.Balance != expectedBalance {
		t.Errorf("余额异常！预期最终余额%.2f，实际余额%.2f", expectedBalance, finalUser.Balance)
	}

	// 检查退款日志数量 - 应该只有一条退款记录
	var refundLogCount int64
	db.Model(&model.BalanceLog{}).Where("user_id = ? AND order_id = ? AND style = ?", userID, orderID, model.BalanceStyleRefund).Count(&refundLogCount)
	t.Logf("退款日志数量: %d", refundLogCount)

	if refundLogCount != 1 {
		t.Errorf("退款日志数量异常！预期1条，实际%d条", refundLogCount)
	}

	// 检查所有余额日志
	var allLogs []model.BalanceLog
	db.Where("user_id = ? AND order_id = ?", userID, orderID).Order("created_at").Find(&allLogs)
	t.Logf("订单%d的所有余额日志:", orderID)
	for _, log := range allLogs {
		logType := "扣款"
		if log.Style == model.BalanceStyleRefund {
			logType = "退款"
		}
		t.Logf("  %s: 金额%.2f, 余额变化%.2f->%.2f, 时间%v", logType, log.Amount, log.BalanceBefore, log.Balance, log.CreatedAt.Format("15:04:05.000"))
	}

	// 检查业务逻辑的正确性：余额正确且只有一条退款日志
	if finalUser.Balance == expectedBalance && refundLogCount == 1 {
		t.Log("✅ 并发退款安全性测试通过！幂等性正常工作，余额正确，只有一条退款记录")
	} else {
		t.Errorf("❌ 并发退款安全性测试失败！余额%.2f(期望%.2f)，退款日志%d条(期望1条)", finalUser.Balance, expectedBalance, refundLogCount)
	}
}

// TestRefundWithoutRowLock 测试退款逻辑是否存在行锁问题
func TestRefundWithoutRowLock(t *testing.T) {
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

	// 创建用户，初始余额200元
	user := &model.User{
		ID:      userID,
		Balance: 200.0,
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
		AccountName:  "测试账号2",
		Type:         1,
		AppKey:       "test_key2",
		AppSecret:    "test_secret2",
		Description:  "测试账号2",
		DailyLimit:   1000.0,
		MonthlyLimit: 30000.0,
		Balance:      0.0,
		Priority:     1,
		Status:       1,
	}
	if err := db.Create(account).Error; err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	// 4. 验证表是否正确创建
	var tables []string
	db.Raw("SELECT name FROM sqlite_master WHERE type='table'").Scan(&tables)
	t.Logf("数据库中的表: %v", tables)

	// 验证platform_accounts表
	var tableCount int64
	db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='platform_accounts'").Scan(&tableCount)
	if tableCount == 0 {
		t.Fatal("platform_accounts表未创建成功")
	}

	// 5. 初始化服务
	platformAccountRepo := repository.NewPlatformAccountRepository(db)
	userRepo := repository.NewUserRepository(db)
	balanceLogRepo := repository.NewBalanceLogRepository(db)
	balanceService := service.NewPlatformAccountBalanceService(db, platformAccountRepo, userRepo, balanceLogRepo)

	// 5. 先执行两次扣款，模拟用户反馈的场景
	deductAmount := 30.0
	orderID1 := int64(201)
	orderID2 := int64(202)

	// 第一次扣款
	err = balanceService.DeductBalance(ctx, accountID, deductAmount, orderID1, "订单201扣款")
	if err != nil {
		t.Fatalf("第一次扣款失败: %v", err)
	}

	// 第二次扣款
	err = balanceService.DeductBalance(ctx, accountID, deductAmount, orderID2, "订单202扣款")
	if err != nil {
		t.Fatalf("第二次扣款失败: %v", err)
	}

	// 验证扣款后余额
	var userAfterDeduct model.User
	db.First(&userAfterDeduct, userID)
	t.Logf("两次扣款后余额: %.2f (200 - 30 - 30 = 140)", userAfterDeduct.Balance)

	// 6. 模拟用户反馈的场景：获取到了2个同样的金额导致退款少退了
	// 这里我们测试如果退款逻辑有问题，是否会出现少退的情况

	// 并发退款两个不同的订单
	var wg sync.WaitGroup
	var refundResults []string
	var mu sync.Mutex

	wg.Add(2)
	go func() {
		defer wg.Done()
		err := balanceService.RefundBalance(ctx, nil, accountID, deductAmount, orderID1, "订单201退款")
		mu.Lock()
		if err != nil {
			refundResults = append(refundResults, fmt.Sprintf("订单201退款失败: %v", err))
		} else {
			refundResults = append(refundResults, "订单201退款成功")
		}
		mu.Unlock()
	}()

	go func() {
		defer wg.Done()
		// 稍微延迟一下，模拟并发但不完全同时
		time.Sleep(1 * time.Millisecond)
		err := balanceService.RefundBalance(ctx, nil, accountID, deductAmount, orderID2, "订单202退款")
		mu.Lock()
		if err != nil {
			refundResults = append(refundResults, fmt.Sprintf("订单202退款失败: %v", err))
		} else {
			refundResults = append(refundResults, "订单202退款成功")
		}
		mu.Unlock()
	}()

	wg.Wait()

	// 7. 验证结果
	for _, result := range refundResults {
		t.Log(result)
	}

	// 检查最终余额
	var finalUser model.User
	db.First(&finalUser, userID)
	expectedBalance := 200.0 // 初始200 - 扣款30 - 扣款30 + 退款30 + 退款30 = 200
	t.Logf("最终余额: %.2f, 预期余额: %.2f", finalUser.Balance, expectedBalance)

	if finalUser.Balance != expectedBalance {
		t.Errorf("❌ 余额异常！预期最终余额%.2f，实际余额%.2f，可能存在退款少退问题", expectedBalance, finalUser.Balance)
	} else {
		t.Log("✅ 余额正确，退款逻辑正常")
	}

	// 检查退款日志数量
	var refundLogCount int64
	db.Model(&model.BalanceLog{}).Where("user_id = ? AND style = ?", userID, model.BalanceStyleRefund).Count(&refundLogCount)
	t.Logf("退款日志总数: %d", refundLogCount)

	if refundLogCount != 2 {
		t.Errorf("❌ 退款日志数量异常！预期2条，实际%d条", refundLogCount)
	}

	// 详细检查所有余额日志
	var allLogs []model.BalanceLog
	db.Where("user_id = ?", userID).Order("created_at").Find(&allLogs)
	t.Log("所有余额变动日志:")
	for _, log := range allLogs {
		logType := "扣款"
		if log.Style == model.BalanceStyleRefund {
			logType = "退款"
		}
		t.Logf("  订单%d %s: 金额%.2f, 余额变化%.2f->%.2f", log.OrderID, logType, log.Amount, log.BalanceBefore, log.Balance)
	}
}