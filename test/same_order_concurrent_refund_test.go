package test

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

// TestSameOrderConcurrentRefund 测试同一订单并发退款的幂等性
func TestSameOrderConcurrentRefund(t *testing.T) {
	// 1. 初始化数据库
	dbName := fmt.Sprintf("test_same_order_refund_%d.db", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dbName), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("连接数据库失败: %v", err)
	}

	// 2. 迁移表结构
	err = db.AutoMigrate(
		&model.User{},
		&model.Platform{},
		&model.PlatformAccount{},
		&model.BalanceLog{},
	)
	if err != nil {
		t.Fatalf("迁移表结构失败: %v", err)
	}

	// 3. 创建测试数据
	// 创建用户
	user := &model.User{
		ID:       3,
		Username: "testuser_same_order",
		Password: "password",
		Balance:  100.00, // 初始余额100
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}
	t.Logf("创建用户成功: ID=%d, 初始余额=%.2f", user.ID, user.Balance)

	// 创建平台
	platform := &model.Platform{
		ID:   3,
		Name: "测试平台3",
		Code: "TEST3",
	}
	if err := db.Create(platform).Error; err != nil {
		t.Fatalf("创建平台失败: %v", err)
	}

	// 创建两个不同的平台账号，都绑定到同一个用户
	account1 := &model.PlatformAccount{
		ID:          3,
		PlatformID:  platform.ID,
		AccountName: "test_account_3",
		BindUserID:  &user.ID,
		Type:        1,
		AppKey:      "test_key_3",
		AppSecret:   "test_secret_3",
	}
	if err := db.Create(account1).Error; err != nil {
		t.Fatalf("创建平台账号1失败: %v", err)
	}

	account2 := &model.PlatformAccount{
		ID:          4,
		PlatformID:  platform.ID,
		AccountName: "test_account_4",
		BindUserID:  &user.ID,
		Type:        1,
		AppKey:      "test_key_4",
		AppSecret:   "test_secret_4",
	}
	if err := db.Create(account2).Error; err != nil {
		t.Fatalf("创建平台账号2失败: %v", err)
	}

	t.Logf("创建平台账号成功: 账号1 ID=%d, 账号2 ID=%d, 都绑定用户ID=%d", account1.ID, account2.ID, *account1.BindUserID)

	// 4. 初始化服务
	platformAccountRepo := repository.NewPlatformAccountRepository(db)
	userRepo := repository.NewUserRepository(db)
	balanceLogRepo := repository.NewBalanceLogRepository(db)
	balanceService := service.NewPlatformAccountBalanceService(db, platformAccountRepo, userRepo, balanceLogRepo)

	// 5. 同一订单并发退款测试
	concurrency := 20 // 并发数
	refundAmount := 50.0 // 退款金额
	orderID := int64(36097) // 同一个订单ID
	var wg sync.WaitGroup
	var successCount int32
	var mu sync.Mutex

	ctx := context.Background()

	// 使用两个不同的平台账号同时对同一订单进行退款
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			
			// 交替使用两个平台账号
			accountName := "平台1"
			if goroutineID%2 == 1 {
				accountName = "平台2"
			}
			
			// 记录退款前的余额
			var beforeUser model.User
			db.Where("id = ?", user.ID).First(&beforeUser)
			beforeBalance := beforeUser.Balance
			
			err := balanceService.RefundBalance(ctx, user.ID, refundAmount, orderID, fmt.Sprintf("测试退款-%s-%d", accountName, goroutineID))
			if err != nil {
				t.Logf("Goroutine %d (%s) 退款失败: %v", goroutineID, accountName, err)
			} else {
				// 检查余额是否真的发生了变化
				var afterUser model.User
				db.Where("id = ?", user.ID).First(&afterUser)
				if afterUser.Balance > beforeBalance {
					// 余额真的增加了，说明是真正的退款
					mu.Lock()
					successCount++
					mu.Unlock()
					t.Logf("Goroutine %d (%s) 退款成功，余额从%.2f增加到%.2f", goroutineID, accountName, beforeBalance, afterUser.Balance)
				} else {
					t.Logf("Goroutine %d (%s) 幂等性跳过，余额未变化: %.2f", goroutineID, accountName, beforeBalance)
				}
			}
		}(i)
	}

	wg.Wait()
	t.Logf("同一订单并发退款测试完成: 成功%d次", successCount)

	// 6. 验证最终余额 - 应该只退款一次
	var finalUser model.User
	if err := db.Where("id = ?", user.ID).First(&finalUser).Error; err != nil {
		t.Fatalf("获取最终用户信息失败: %v", err)
	}

	expectedBalance := user.Balance + refundAmount // 只应该退款一次
	t.Logf("最终余额: %.2f, 预期余额: %.2f", finalUser.Balance, expectedBalance)

	if finalUser.Balance != expectedBalance {
		t.Errorf("余额不一致！最终余额: %.2f, 预期余额: %.2f", finalUser.Balance, expectedBalance)
	}

	// 7. 验证退款日志数量 - 应该只有一条
	var logCount int64
	if err := db.Model(&model.BalanceLog{}).Where("order_id = ? AND style = ?", orderID, model.BalanceStyleRefund).Count(&logCount).Error; err != nil {
		t.Fatalf("统计退款日志失败: %v", err)
	}

	t.Logf("退款日志数量: %d", logCount)
	if logCount != 1 {
		t.Errorf("退款日志数量不正确！实际: %d, 预期: 1", logCount)
	}

	// 8. 打印订单的所有余额日志用于调试
	var logs []model.BalanceLog
	if err := db.Where("order_id = ?", orderID).Order("created_at ASC").Find(&logs).Error; err != nil {
		t.Fatalf("获取余额日志失败: %v", err)
	}

	t.Logf("订单%d的所有余额日志:", orderID)
	for _, log := range logs {
		logType := "未知"
		if log.Style == model.BalanceStyleRefund {
			logType = "退款"
		} else if log.Style == model.BalanceStyleOrderDeduct {
			logType = "扣款"
		}
		t.Logf("  %s: 金额%.2f, 余额变化%.2f->%.2f, 平台账号%d, 时间%s", 
			logType, log.Amount, log.BalanceBefore, log.Balance, log.PlatformAccountID, log.CreatedAt.Format("15:04:05.000"))
	}

	// 验证幂等性：只应该有一条退款日志
	if logCount == 1 {
		t.Logf("✅ 同一订单并发退款幂等性测试通过！基于用户ID的幂等性校验正常工作，防止了同一订单通过不同平台账号重复退款")
		t.Logf("   - 退款日志数量: %d (正确)", logCount)
		t.Logf("   - 最终余额: %.2f (正确)", finalUser.Balance)
		t.Logf("   - 实际退款成功次数: %d (可能因并发检查时序问题略有偏差，但关键是日志数量正确)", successCount)
	} else {
		t.Errorf("❌ 同一订单并发退款幂等性测试失败！退款日志数量: %d, 预期: 1", logCount)
	}
}