package test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"recharge-go/internal/model"
	notificationModel "recharge-go/internal/model/notification"
	"recharge-go/internal/repository"
	"recharge-go/internal/service"
	"recharge-go/pkg/lock"
)

// TestConcurrentOrderFailRefund 测试并发订单失败退款的修复
func TestConcurrentOrderFailRefund(t *testing.T) {
	// 1. 创建测试数据库
	timestamp := time.Now().Unix()
	dbPath := fmt.Sprintf("test_concurrent_order_fail_%d.db", timestamp)
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("连接数据库失败: %v", err)
	}

	// 2. 自动迁移
	err = db.AutoMigrate(&model.User{}, &model.Order{}, &model.BalanceLog{}, &model.Platform{}, &model.PlatformAccount{})
	if err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}

	// 3. 创建测试数据
	ctx := context.Background()

	// 创建测试用户
	user := &model.User{
		ID:       1,
		Username: "testuser",
		Password: "password123",
		Balance:  9813628.00, // 初始余额
		Status:   1,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}

	// 创建测试平台
	platform := &model.Platform{
		Name: "测试平台",
		Code: "test",
	}
	if err := db.Create(platform).Error; err != nil {
		t.Fatalf("创建测试平台失败: %v", err)
	}

	// 创建测试平台账户
	account := &model.PlatformAccount{
		PlatformID:  platform.ID,
		AccountName: "测试账户",
		Balance:     10000.00,
	}
	if err := db.Create(account).Error; err != nil {
		t.Fatalf("创建测试平台账户失败: %v", err)
	}
	t.Logf("创建的平台账户ID: %d", account.ID)

	// 创建两个测试订单
	order1 := &model.Order{
		OrderNumber:       "ORDER001",
		OutTradeNum:       "OUT001",
		CustomerID:        user.ID,
		PlatformAccountID: account.ID,
		Price:             96.50,
		Status:            model.OrderStatusRecharging, // 充值中状态
		Client:            1,                           // 平台订单
	}
	if err := db.Create(order1).Error; err != nil {
		t.Fatalf("创建测试订单1失败: %v", err)
	}
	t.Logf("创建的订单1 ID: %d, PlatformAccountID: %d", order1.ID, order1.PlatformAccountID)

	order2 := &model.Order{
		OrderNumber:       "ORDER002",
		OutTradeNum:       "OUT002",
		CustomerID:        user.ID,
		PlatformAccountID: account.ID,
		Price:             96.50,
		Status:            model.OrderStatusRecharging, // 充值中状态
		Client:            1,                           // 平台订单
	}
	if err := db.Create(order2).Error; err != nil {
		t.Fatalf("创建测试订单2失败: %v", err)
	}
	t.Logf("创建的订单2 ID: %d, PlatformAccountID: %d", order2.ID, order2.PlatformAccountID)

	// 4. 创建服务实例
	userRepo := repository.NewUserRepository(db)
	balanceLogRepo := repository.NewBalanceLogRepository(db)
	orderRepo := repository.NewOrderRepository(db)
	platformAccountRepo := repository.NewPlatformAccountRepository(db)

	balanceService := service.NewBalanceService(balanceLogRepo, userRepo)
	platformAccountBalanceService := service.NewPlatformAccountBalanceService(db, platformAccountRepo, userRepo, balanceLogRepo)

	// 创建模拟的分布式锁管理器
	mockDistributedLock := &MockDistributedLock{}
	refundLockManager := lock.NewRefundLockManager(mockDistributedLock)



	// 创建统一退款服务
	unifiedRefundService := service.NewUnifiedRefundService(
		db,
		userRepo,
		orderRepo,
		balanceLogRepo,
		refundLockManager,
		balanceService,
		platformAccountBalanceService,
	)

	// 创建模拟的充值服务
	rechargeService := &MockRechargeService{
		balanceService: platformAccountBalanceService,
		userBalanceService: balanceService,
	}

	// 创建模拟的通知仓库
	notificationRepo := &MockNotificationRepo{}

	// 创建模拟的队列
	queueInstance := &MockQueue{}

	// 创建授信服务
	creditLogRepo := repository.NewCreditLogRepository(db)
	creditService := service.NewCreditService(userRepo, creditLogRepo)

	orderService := service.NewOrderService(orderRepo, balanceLogRepo, userRepo, rechargeService, unifiedRefundService, refundLockManager, notificationRepo, queueInstance, db, nil, creditService)

	// 5. 并发执行订单失败处理
	var wg sync.WaitGroup
	errorChan := make(chan error, 2)

	// 并发处理两个订单失败
	wg.Add(2)
	go func() {
		defer wg.Done()
		if err := orderService.ProcessOrderFail(ctx, order1.ID, "测试并发失败1"); err != nil {
			errorChan <- fmt.Errorf("订单1失败处理错误: %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		if err := orderService.ProcessOrderFail(ctx, order2.ID, "测试并发失败2"); err != nil {
			errorChan <- fmt.Errorf("订单2失败处理错误: %v", err)
		}
	}()

	wg.Wait()
	close(errorChan)

	// 检查是否有错误
	for err := range errorChan {
		if err != nil {
			t.Errorf("并发处理错误: %v", err)
		}
	}

	// 6. 验证结果
	// 检查用户最终余额
	var finalUser model.User
	if err := db.First(&finalUser, user.ID).Error; err != nil {
		t.Fatalf("获取最终用户信息失败: %v", err)
	}

	// 预期余额：初始余额 + 两笔退款
	expectedBalance := 9813628.00 + 96.50 + 96.50
	if finalUser.Balance != expectedBalance {
		t.Errorf("用户最终余额不正确，期望: %.2f，实际: %.2f", expectedBalance, finalUser.Balance)
	} else {
		t.Logf("✅ 用户最终余额正确: %.2f", finalUser.Balance)
	}

	// 检查余额日志数量
	var logCount int64
	if err := db.Model(&model.BalanceLog{}).Where("user_id = ? AND style = ?", user.ID, model.BalanceStyleRefund).Count(&logCount).Error; err != nil {
		t.Fatalf("统计余额日志失败: %v", err)
	}

	if logCount != 2 {
		t.Errorf("退款日志数量不正确，期望: 2，实际: %d", logCount)
	} else {
		t.Logf("✅ 退款日志数量正确: %d", logCount)
	}

	// 检查订单状态
	var finalOrder1, finalOrder2 model.Order
	if err := db.First(&finalOrder1, order1.ID).Error; err != nil {
		t.Fatalf("获取最终订单1信息失败: %v", err)
	}
	if err := db.First(&finalOrder2, order2.ID).Error; err != nil {
		t.Fatalf("获取最终订单2信息失败: %v", err)
	}

	if finalOrder1.Status != model.OrderStatusFailed {
		t.Errorf("订单1状态不正确，期望: %d，实际: %d", model.OrderStatusFailed, finalOrder1.Status)
	}
	if finalOrder2.Status != model.OrderStatusFailed {
		t.Errorf("订单2状态不正确，期望: %d，实际: %d", model.OrderStatusFailed, finalOrder2.Status)
	}

	// 检查余额日志的详细信息
	var logs []model.BalanceLog
	if err := db.Where("user_id = ? AND style = ?", user.ID, model.BalanceStyleRefund).Order("created_at").Find(&logs).Error; err != nil {
		t.Fatalf("获取余额日志失败: %v", err)
	}

	if len(logs) == 2 {
		// 验证第一笔退款
		if logs[0].BalanceBefore != 9813628.00 {
			t.Errorf("第一笔退款前余额不正确，期望: 9813628.00，实际: %.2f", logs[0].BalanceBefore)
		}
		if logs[0].Balance != 9813724.50 {
			t.Errorf("第一笔退款后余额不正确，期望: 9813724.50，实际: %.2f", logs[0].Balance)
		}

		// 验证第二笔退款
		if logs[1].BalanceBefore != 9813724.50 {
			t.Errorf("第二笔退款前余额不正确，期望: 9813724.50，实际: %.2f", logs[1].BalanceBefore)
		}
		if logs[1].Balance != 9813821.00 {
			t.Errorf("第二笔退款后余额不正确，期望: 9813821.00，实际: %.2f", logs[1].Balance)
		}

		t.Logf("✅ 余额变化记录正确:")
		t.Logf("   第一笔: %.2f -> %.2f (+%.2f)", logs[0].BalanceBefore, logs[0].Balance, logs[0].Amount)
		t.Logf("   第二笔: %.2f -> %.2f (+%.2f)", logs[1].BalanceBefore, logs[1].Balance, logs[1].Amount)
	}

	t.Logf("✅ 并发订单失败退款测试通过！")
}

// MockRechargeService 模拟充值服务
type MockRechargeService struct {
	balanceService     *service.PlatformAccountBalanceService
	userBalanceService *service.BalanceService
}

func (m *MockRechargeService) GetBalanceService() *service.PlatformAccountBalanceService {
	return m.balanceService
}

func (m *MockRechargeService) GetUserBalanceService() *service.BalanceService {
	return m.userBalanceService
}

// 实现RechargeService接口的其他方法（空实现）
func (m *MockRechargeService) Recharge(ctx context.Context, orderID int64) error { return nil }
func (m *MockRechargeService) HandleCallback(ctx context.Context, platformName string, data []byte) error { return nil }
func (m *MockRechargeService) GetPendingTasks(ctx context.Context, limit int) ([]*model.Order, error) { return nil, nil }
func (m *MockRechargeService) ProcessRechargeTask(ctx context.Context, order *model.Order) error { return nil }
func (m *MockRechargeService) CreateRechargeTask(ctx context.Context, orderID int64) error { return nil }
func (m *MockRechargeService) GetPlatformAPIByOrderID(ctx context.Context, orderID string) (*model.PlatformAPI, *model.PlatformAPIParam, error) { return nil, nil, nil }
func (m *MockRechargeService) PushToRechargeQueue(ctx context.Context, orderID int64) error { return nil }
func (m *MockRechargeService) PopFromRechargeQueue(ctx context.Context) (int64, error) { return 0, nil }
func (m *MockRechargeService) GetOrderByID(ctx context.Context, orderID int64) (*model.Order, error) { return nil, nil }
func (m *MockRechargeService) RemoveFromProcessingQueue(ctx context.Context, orderID int64) error { return nil }
func (m *MockRechargeService) CheckRechargingOrders(ctx context.Context) error { return nil }
func (m *MockRechargeService) SubmitOrder(ctx context.Context, order *model.Order, api *model.PlatformAPI, apiParam *model.PlatformAPIParam) error { return nil }

// MockNotificationRepo 模拟通知仓库
type MockNotificationRepo struct{}

func (m *MockNotificationRepo) Create(ctx context.Context, notification *notificationModel.NotificationRecord) error { return nil }
func (m *MockNotificationRepo) GetByID(ctx context.Context, id int64) (*notificationModel.NotificationRecord, error) { return nil, nil }
func (m *MockNotificationRepo) Update(ctx context.Context, notification *notificationModel.NotificationRecord) error { return nil }
func (m *MockNotificationRepo) Delete(ctx context.Context, id int64) error { return nil }
func (m *MockNotificationRepo) List(ctx context.Context, params map[string]interface{}, page, pageSize int) ([]*notificationModel.NotificationRecord, int64, error) { return nil, 0, nil }
func (m *MockNotificationRepo) GetPendingRecords(ctx context.Context, limit int) ([]*notificationModel.NotificationRecord, error) { return nil, nil }
func (m *MockNotificationRepo) UpdateStatus(ctx context.Context, id int64, status int) error { return nil }
func (m *MockNotificationRepo) IncrementRetryCount(ctx context.Context, id int64) error { return nil }

// MockQueue 模拟队列
type MockQueue struct{}

func (m *MockQueue) Push(ctx context.Context, key string, value interface{}) error { return nil }
func (m *MockQueue) Pop(ctx context.Context, key string) (interface{}, error) { return nil, nil }
func (m *MockQueue) Peek(ctx context.Context, key string) (interface{}, error) { return nil, nil }
func (m *MockQueue) PushWithDelay(ctx context.Context, key string, value interface{}, delay time.Duration) error { return nil }
func (m *MockQueue) GetLength(ctx context.Context, key string) (int64, error) { return 0, nil }
func (m *MockQueue) Remove(ctx context.Context, key string, value interface{}) error { return nil }
func (m *MockQueue) Clear(ctx context.Context, key string) error { return nil }

// MockDistributedLock 模拟分布式锁
type MockDistributedLock struct{}

func (m *MockDistributedLock) Lock(ctx context.Context, key string, expiration time.Duration) (bool, error) {
	return true, nil
}

func (m *MockDistributedLock) LockWithRetry(ctx context.Context, key string, expiration time.Duration, maxRetries int, retryInterval time.Duration) (string, error) {
	return "mock-lock-value", nil
}

func (m *MockDistributedLock) Unlock(ctx context.Context, key string, value string) error {
	return nil
}



func (m *MockRechargeService) ProcessRetryTask(ctx context.Context, retryRecord *model.OrderRetryRecord) error { return nil }
func (m *MockRechargeService) SetOrderService(orderService service.OrderService) {}