package redis

import (
	"context"
	"fmt"
	"recharge-go/pkg/logger"
	"time"

	"github.com/go-redis/redis/v8"
)

var (
	client *redis.Client
)

// InitRedis 初始化Redis连接
func InitRedis(host string, port int, password string, db int) error {
	client = redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", host, port),
		Password:     password,
		DB:           db,
		PoolSize:     10,
		MinIdleConns: 5,
		MaxRetries:   3,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis connect failed: %v", err)
	}

	logger.Info("Redis连接成功: %s:%d", host, port)
	return nil
}

// GetClient 获取Redis客户端
func GetClient() *redis.Client {
	return client
}

// Close 关闭Redis连接
func Close() error {
	if client != nil {
		return client.Close()
	}
	return nil
}
