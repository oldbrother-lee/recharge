package test

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"

	"recharge-go/internal/model"
	"recharge-go/internal/repository"
	"recharge-go/internal/service"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestMultiPlatformRefund 测试同一订单通过不同平台账号退款的幂等性
func TestMultiPlatformRefund(t *testing.T) {

	// 创建测试数据库
	dbPath := "test_multi_platform.db"
	defer os.Remove(dbPath)

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("连接数据库失败: %v", err)
	}

	// 迁移表结构
	err = db.AutoMigrate(
		&model.Platform{},
		&model.User{},
		&model.PlatformAccount{},
		&model.BalanceLog{},
	)
	if err != nil {
		t.Fatalf("迁移表结构失败: %v", err)
	}

	// 创建测试数据
	// 1. 创建平台
	platform1 := model.Platform{
		Name:        "测试平台1",
		Description: "测试平台1描述",
		Status:      1,
	}
	platform2 := model.Platform{
		Name:        "测试平台2",
		Description: "测试平台2描述",
		Status:      1,
	}
	db.Create(&platform1)
	db.Create(&platform2)

	// 2. 创建用户
	user := model.User{
		ID:       1,
		Username: "testuser",
		Password: "password123",
		Balance:  200.0,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}
	t.Logf("创建用户成功，用户ID: %d", user.ID)

	// 3. 创建两个平台账号，都绑定到同一个用户
	account1 := model.PlatformAccount{
		PlatformID:  platform1.ID,
		AccountName: "account1",
		BindUserID:  &user.ID,
		Type:        1,
		AppKey:      "test_key1",
		AppSecret:   "test_secret1",
		Status:      1,
	}
	account2 := model.PlatformAccount{
		PlatformID:  platform2.ID,
		AccountName: "account2",
		BindUserID:  &user.ID,
		Type:        1,
		AppKey:      "test_key2",
		AppSecret:   "test_secret2",
		Status:      1,
	}
	if err := db.Create(&account1).Error; err != nil {
		t.Fatalf("创建平台账号1失败: %v", err)
	}
	if err := db.Create(&account2).Error; err != nil {
		t.Fatalf("创建平台账号2失败: %v", err)
	}
	t.Logf("创建平台账号成功，账号1 ID: %d，账号2 ID: %d", account1.ID, account2.ID)

	// 初始化服务
	platformAccountRepo := repository.NewPlatformAccountRepository(db)
	userRepo := repository.NewUserRepository(db)
	balanceLogRepo := repository.NewBalanceLogRepository(db)
	service := service.NewPlatformAccountBalanceService(db, platformAccountRepo, userRepo, balanceLogRepo)

	// 4. 先进行扣款操作
	ctx := context.Background()
	orderID := int64(36097) // 使用生产环境中出现问题的订单ID
	amount := 95.0

	// 通过平台账号1扣款
	err = service.DeductBalance(ctx, account1.ID, amount, orderID, "测试扣款")
	if err != nil {
		t.Fatalf("扣款失败: %v", err)
	}

	// 验证扣款后余额
	db.First(&user, user.ID)
	if user.Balance != 105.0 {
		t.Fatalf("扣款后余额错误，期望105.0，实际%.2f", user.Balance)
	}
	t.Logf("扣款成功，余额从200.00变为%.2f", user.Balance)

	// 5. 并发通过不同平台账号退款同一订单
	var wg sync.WaitGroup
	var successCount int32
	var mu sync.Mutex
	results := make([]string, 0)

	// 启动10个goroutine，5个使用平台账号1，5个使用平台账号2
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			// 交替使用两个平台账号
			accountID := account1.ID
			platformName := "平台1"
			if index%2 == 1 {
				accountID = account2.ID
				platformName = "平台2"
			}

			err := service.RefundBalance(ctx, nil, accountID, amount, orderID, fmt.Sprintf("测试退款-%s-%d", platformName, index))
			mu.Lock()
			if err != nil {
				results = append(results, fmt.Sprintf("Goroutine %d (%s) 退款失败: %v", index, platformName, err))
			} else {
				successCount++
				results = append(results, fmt.Sprintf("Goroutine %d (%s) 退款成功", index, platformName))
			}
			mu.Unlock()
		}(i)
	}

	wg.Wait()

	// 输出所有结果
	for _, result := range results {
		t.Log(result)
	}

	t.Logf("多平台退款测试完成: 成功%d次", successCount)

	// 6. 验证最终结果
	// 检查用户余额
	db.First(&user, user.ID)
	expectedBalance := 200.0 // 应该恢复到原始余额
	t.Logf("最终余额: %.2f, 预期余额: %.2f", user.Balance, expectedBalance)

	if user.Balance != expectedBalance {
		t.Errorf("❌ 余额错误！期望%.2f，实际%.2f", expectedBalance, user.Balance)
		return
	}

	// 检查退款日志数量（应该只有1条）
	var refundLogCount int64
	db.Model(&model.BalanceLog{}).Where("order_id = ? AND user_id = ? AND style = ?", orderID, user.ID, model.BalanceStyleRefund).Count(&refundLogCount)
	t.Logf("退款日志数量: %d", refundLogCount)

	if refundLogCount != 1 {
		t.Errorf("❌ 退款日志数量错误！期望1条，实际%d条", refundLogCount)
		return
	}

	// 输出所有余额变动日志
	var allLogs []model.BalanceLog
	db.Where("order_id = ? AND user_id = ?", orderID, user.ID).Order("created_at").Find(&allLogs)
	t.Logf("订单%d的所有余额日志:", orderID)
	for _, log := range allLogs {
		styleStr := "扣款"
		if log.Style == model.BalanceStyleRefund {
			styleStr = "退款"
		}
		t.Logf("  %s: 金额%.2f, 余额变化%.2f->%.2f, 时间%s", styleStr, log.Amount, log.BalanceBefore, log.Balance, log.CreatedAt.Format("15:04:05.000"))
	}

	t.Log("✅ 多平台退款幂等性测试通过！基于用户ID的幂等性校验正常工作，防止了同一订单通过不同平台账号重复退款")
}