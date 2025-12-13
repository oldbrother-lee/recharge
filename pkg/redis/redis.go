package redis

import (
	"context"
	"fmt"
	logger "recharge-go/pkg/log"
	"time"

	"github.com/redis/go-redis/v9"
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
		PoolSize:     200, // 增加连接池大小以支持高并发阻塞操作
		MinIdleConns: 50,  // 保持更多空闲连接
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

	logger.InfoV2("Redis连接成功",
		logger.StringV2("host", host),
		logger.Int64V2("port", int64(port)))
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
