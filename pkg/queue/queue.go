package queue

import (
	"context"
	"time"
)

// Queue 队列接口
type Queue interface {
	// Push 入队
	Push(ctx context.Context, key string, value interface{}) error
	// Pop 出队
	Pop(ctx context.Context, key string) (interface{}, error)
	// Peek 查看队列头部的元素而不移除它
	Peek(ctx context.Context, key string) (interface{}, error)
	// PushWithDelay 延迟入队
	PushWithDelay(ctx context.Context, key string, value interface{}, delay time.Duration) error
	// GetLength 获取队列长度
	GetLength(ctx context.Context, key string) (int64, error)
	// Remove 移除指定元素
	Remove(ctx context.Context, key string, value interface{}) error
	// Clear 清空队列
	Clear(ctx context.Context, key string) error
}
