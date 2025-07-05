package lock

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"recharge-go/pkg/logger"
)

// DistributedLock 分布式锁接口
type DistributedLock interface {
	// Lock 获取锁
	Lock(ctx context.Context, key string, expiration time.Duration) (bool, error)
	// Unlock 释放锁
	Unlock(ctx context.Context, key string, value string) error
	// LockWithRetry 带重试的获取锁
	LockWithRetry(ctx context.Context, key string, expiration time.Duration, maxRetries int, retryInterval time.Duration) (string, error)
}

// RedisDistributedLock Redis分布式锁实现
type RedisDistributedLock struct {
	client *redis.Client
}

// NewRedisDistributedLock 创建Redis分布式锁实例
func NewRedisDistributedLock(client *redis.Client) *RedisDistributedLock {
	return &RedisDistributedLock{
		client: client,
	}
}

// Lock 获取锁
func (r *RedisDistributedLock) Lock(ctx context.Context, key string, expiration time.Duration) (bool, error) {
	value := fmt.Sprintf("%d", time.Now().UnixNano())
	result, err := r.client.SetNX(ctx, key, value, expiration).Result()
	if err != nil {
		logger.Error("获取分布式锁失败", "key", key, "error", err)
		return false, err
	}
	return result, nil
}

// Unlock 释放锁
func (r *RedisDistributedLock) Unlock(ctx context.Context, key string, value string) error {
	// 使用Lua脚本确保原子性：只有持有锁的客户端才能释放锁
	luaScript := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`
	result, err := r.client.Eval(ctx, luaScript, []string{key}, value).Result()
	if err != nil {
		logger.Error("释放分布式锁失败", "key", key, "error", err)
		return err
	}
	
	if result.(int64) == 0 {
		logger.Warn("锁已被其他客户端释放或已过期", "key", key)
		return errors.New("锁已被其他客户端释放或已过期")
	}
	
	logger.Info("成功释放分布式锁", "key", key)
	return nil
}

// LockWithRetry 带重试的获取锁
func (r *RedisDistributedLock) LockWithRetry(ctx context.Context, key string, expiration time.Duration, maxRetries int, retryInterval time.Duration) (string, error) {
	value := fmt.Sprintf("%d", time.Now().UnixNano())
	
	for i := 0; i <= maxRetries; i++ {
		result, err := r.client.SetNX(ctx, key, value, expiration).Result()
		if err != nil {
			logger.Error("获取分布式锁失败", "key", key, "retry", i, "error", err)
			if i == maxRetries {
				return "", err
			}
			continue
		}
		
		if result {
			logger.Info("成功获取分布式锁", "key", key, "retry", i, "value", value)
			return value, nil
		}
		
		if i < maxRetries {
			logger.Info("获取锁失败，等待重试", "key", key, "retry", i+1, "interval", retryInterval)
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(retryInterval):
				// 继续重试
			}
		}
	}
	
	return "", errors.New("获取分布式锁超时")
}

// RefundLockManager 退款锁管理器
type RefundLockManager struct {
	lock DistributedLock
}

// NewRefundLockManager 创建退款锁管理器
func NewRefundLockManager(lock DistributedLock) *RefundLockManager {
	return &RefundLockManager{
		lock: lock,
	}
}

// LockUserRefund 锁定用户退款操作
func (r *RefundLockManager) LockUserRefund(ctx context.Context, userID int64) (string, error) {
	lockKey := fmt.Sprintf("refund:user:%d", userID)
	return r.lock.LockWithRetry(ctx, lockKey, 30*time.Second, 3, 100*time.Millisecond)
}

// UnlockUserRefund 解锁用户退款操作
func (r *RefundLockManager) UnlockUserRefund(ctx context.Context, userID int64, lockValue string) error {
	lockKey := fmt.Sprintf("refund:user:%d", userID)
	return r.lock.Unlock(ctx, lockKey, lockValue)
}

// LockOrderRefund 锁定订单退款操作
func (r *RefundLockManager) LockOrderRefund(ctx context.Context, orderID int64) (string, error) {
	lockKey := fmt.Sprintf("refund:order:%d", orderID)
	return r.lock.LockWithRetry(ctx, lockKey, 30*time.Second, 3, 100*time.Millisecond)
}

// UnlockOrderRefund 解锁订单退款操作
func (r *RefundLockManager) UnlockOrderRefund(ctx context.Context, orderID int64, lockValue string) error {
	lockKey := fmt.Sprintf("refund:order:%d", orderID)
	return r.lock.Unlock(ctx, lockKey, lockValue)
}