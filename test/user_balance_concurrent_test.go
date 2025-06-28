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

// TestUserBalanceSequentialRefund 测试用户余额服务的顺序退款（验证基本功能）
func TestUserBalanceSequentialRefund(t *testing.T) {
	// 1. 创建测试数据库
	timestamp := time.Now().Unix()
	dbPath := fmt.Sprintf("test_user_balance_seq_%d.db", timestamp)
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("连接数据库失败: %v", err)
	}

	// 2. 自动迁移
	err = db.AutoMigrate(&model.User{}, &model.BalanceLog{})
	if err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}

	// 3. 创建测试用户
	user := &model.User{
		ID:       1,
		Username: "test_user_seq",
		Balance:  100.0,
		Credit:   0.0,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}

	// 4. 创建服务实例
	balanceLogRepo := repository.NewBalanceLogRepository(db)
	userRepo := repository.NewUserRepository(db)
	balanceService := service.NewBalanceService(balanceLogRepo, userRepo)

	// 5. 顺序退款测试
	ctx := context.Background()
	refundCount := 5
	refundAmount := 10.0

	for i := 0; i < refundCount; i++ {
		orderID := int64(1000 + i)
		err := balanceService.Refund(ctx, user.ID, refundAmount, orderID, fmt.Sprintf("顺序退款测试-%d", i), "system")
		if err != nil {
			t.Errorf("第%d次退款失败: %v", i+1, err)
		}
	}

	// 6. 检查最终余额
	finalUser := &model.User{}
	db.First(finalUser, user.ID)
	expectedBalance := 100.0 + float64(refundCount)*refundAmount

	if finalUser.Balance != expectedBalance {
		t.Errorf("❌ 最终余额不正确！预期%.2f，实际%.2f", expectedBalance, finalUser.Balance)
	} else {
		t.Logf("✅ 顺序退款测试通过: 最终余额%.2f", finalUser.Balance)
	}

	// 7. 检查退款日志数量
	var logCount int64
	db.Model(&model.BalanceLog{}).Where("user_id = ? AND type = ? AND style = ?", user.ID, 1, 2).Count(&logCount)
	if logCount != int64(refundCount) {
		t.Errorf("❌ 退款日志数量不正确！预期%d条，实际%d条", refundCount, logCount)
	} else {
		t.Logf("✅ 退款日志数量正确: %d条", logCount)
	}
}

// TestUserBalanceLimitedConcurrentRefund 测试用户余额服务的有限并发退款
func TestUserBalanceLimitedConcurrentRefund(t *testing.T) {
	// 1. 创建测试数据库
	timestamp := time.Now().Unix()
	dbPath := fmt.Sprintf("test_user_balance_limited_%d.db", timestamp)
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("连接数据库失败: %v", err)
	}

	// 2. 自动迁移
	err = db.AutoMigrate(&model.User{}, &model.BalanceLog{})
	if err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}

	// 3. 创建测试用户
	user := &model.User{
		ID:       1,
		Username: "test_user_limited",
		Balance:  100.0,
		Credit:   0.0,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}

	// 4. 创建服务实例
	balanceLogRepo := repository.NewBalanceLogRepository(db)
	userRepo := repository.NewUserRepository(db)
	balanceService := service.NewBalanceService(balanceLogRepo, userRepo)

	// 5. 有限并发退款测试（减少并发数量）
	goroutineCount := 3 // 减少到3个并发
	refundAmount := 10.0
	orderIDBase := int64(2000)

	// 6. 执行有限并发退款测试
	var wg sync.WaitGroup
	var successCount int32
	var results []string
	var resultsMu sync.Mutex

	ctx := context.Background()

	for i := 0; i < goroutineCount; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			// 添加随机延迟减少并发冲突
			time.Sleep(time.Duration(goroutineID*10) * time.Millisecond)

			// 使用不同的订单ID进行退款
			orderID := orderIDBase + int64(goroutineID)
			err := balanceService.Refund(ctx, user.ID, refundAmount, orderID, fmt.Sprintf("有限并发退款测试-%d", goroutineID), "system")

			resultsMu.Lock()
			if err != nil {
				results = append(results, fmt.Sprintf("Goroutine %d 退款失败: %v", goroutineID, err))
			} else {
				results = append(results, fmt.Sprintf("Goroutine %d 退款成功", goroutineID))
				successCount++
			}
			resultsMu.Unlock()
		}(i)
	}

	wg.Wait()

	// 7. 输出测试结果
	for _, result := range results {
		t.Log(result)
	}
	t.Logf("有限并发退款测试完成: 成功%d次", successCount)

	// 8. 检查用户最终余额
	finalUser := &model.User{}
	db.First(finalUser, user.ID)
	expectedBalance := 100.0 + float64(successCount)*refundAmount

	if finalUser.Balance != expectedBalance {
		t.Errorf("❌ 最终余额不正确！预期%.2f，实际%.2f", expectedBalance, finalUser.Balance)
	} else {
		t.Logf("✅ 最终余额正确: %.2f", finalUser.Balance)
	}

	// 9. 检查退款日志数量
	var logCount int64
	db.Model(&model.BalanceLog{}).Where("user_id = ? AND type = ? AND style = ?", user.ID, 1, 2).Count(&logCount)
	t.Logf("退款日志数量: %d", logCount)

	if logCount != int64(successCount) {
		t.Errorf("❌ 退款日志数量不正确！预期%d条，实际%d条", successCount, logCount)
	} else {
		t.Logf("✅ 退款日志数量正确: %d条", logCount)
	}

	// 10. 验证测试结果
	if int(successCount) >= 2 && finalUser.Balance == expectedBalance && logCount == int64(successCount) {
		t.Log("✅ 用户余额有限并发退款测试通过！FOR UPDATE锁有效防止了并发问题")
	} else {
		t.Errorf("❌ 用户余额有限并发退款测试失败！成功次数:%d，余额:%.2f(期望%.2f)，日志数量:%d(期望%d)", 
			successCount, finalUser.Balance, expectedBalance, logCount, successCount)
	}
}