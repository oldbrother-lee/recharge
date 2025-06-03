package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"recharge-go/pkg/redis"
	"time"

	"recharge-go/pkg/logger"

	redisV8 "github.com/go-redis/redis/v8"
)

// RedisQueue Redis队列实现
type RedisQueue struct {
	client *redisV8.Client
}

// NewRedisQueue 创建Redis队列实例
func NewRedisQueue() *RedisQueue {
	return &RedisQueue{
		client: redis.GetClient(),
	}
}

// Push 入队
func (q *RedisQueue) Push(ctx context.Context, key string, value interface{}) error {
	// 打印原始值
	logger.Info("Push 原始值",
		"value_type", fmt.Sprintf("%T", value),
		"value", value,
	)

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal value failed: %v", err)
	}

	// 打印序列化后的数据
	logger.Info("Push 序列化后的数据",
		"data_type", fmt.Sprintf("%T", data),
		"data", string(data),
	)

	return q.client.LPush(ctx, key, data).Err()
}

// Peek 查看队列头部的元素而不移除它
func (q *RedisQueue) Peek(ctx context.Context, key string) (interface{}, error) {
	result, err := q.client.LRange(ctx, key, -1, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("peek from queue failed: %v", err)
	}
	if len(result) == 0 {
		return nil, nil
	}

	// 打印从 Redis 获取的原始数据
	logger.Info("Peek 从 Redis 获取的原始数据",
		"data_type", fmt.Sprintf("%T", result[0]),
		"data", result[0],
	)

	return result[0], nil
}

// Pop 出队
func (q *RedisQueue) Pop(ctx context.Context, key string) (interface{}, error) {
	result, err := q.client.BRPop(ctx, 0, key).Result()
	if err != nil {
		return nil, fmt.Errorf("pop from queue failed: %v", err)
	}
	if len(result) < 2 {
		return nil, fmt.Errorf("invalid queue result")
	}

	// 打印从 Redis 获取的原始数据
	logger.Info("Pop 从 Redis 获取的原始数据",
		"data_type", fmt.Sprintf("%T", result[1]),
		"data", string(result[1]),
	)

	return string(result[1]), nil
}

// PushWithDelay 延迟入队
func (q *RedisQueue) PushWithDelay(ctx context.Context, key string, value interface{}, delay time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal value failed: %v", err)
	}
	return q.client.ZAdd(ctx, key, &redisV8.Z{
		Score:  float64(time.Now().Add(delay).Unix()),
		Member: data,
	}).Err()
}

// GetLength 获取队列长度
func (q *RedisQueue) GetLength(ctx context.Context, key string) (int64, error) {
	return q.client.LLen(ctx, key).Result()
}

// Remove 移除指定元素
func (q *RedisQueue) Remove(ctx context.Context, key string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal value failed: %v", err)
	}
	return q.client.LRem(ctx, key, 0, data).Err()
}

// Clear 清空队列
func (q *RedisQueue) Clear(ctx context.Context, key string) error {
	return q.client.Del(ctx, key).Err()
}
