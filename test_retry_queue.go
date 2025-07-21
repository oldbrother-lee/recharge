package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"recharge-go/internal/model"
	"recharge-go/pkg/queue"
	"recharge-go/pkg/redis"
	"time"
)

func main() {
	// 初始化 Redis
	err := redis.InitRedis("localhost", 6379, "", 0)
	if err != nil {
		log.Fatalf("初始化 Redis 失败: %v", err)
	}

	// 创建队列实例
	q := queue.NewRedisQueue()
	ctx := context.Background()

	// 测试推送重试任务
	task := model.NewRetryTaskMessage(12345, 2, "测试重试任务")
	if err := q.Push(ctx, "retry_queue", task); err != nil {
		log.Fatalf("推送任务失败: %v", err)
	}
	fmt.Println("✅ 重试任务推送成功")

	// 测试从队列获取任务
	taskData, err := q.Pop(ctx, "retry_queue")
	if err != nil {
		log.Fatalf("获取任务失败: %v", err)
	}

	if taskData == nil {
		fmt.Println("❌ 队列为空")
		return
	}

	// 解析任务数据
	taskStr, ok := taskData.(string)
	if !ok {
		fmt.Printf("❌ 任务数据格式错误: %T\n", taskData)
		return
	}

	var retrievedTask model.RetryTaskMessage
	if err := json.Unmarshal([]byte(taskStr), &retrievedTask); err != nil {
		fmt.Printf("❌ 解析任务失败: %v\n", err)
		return
	}

	fmt.Printf("✅ 成功获取重试任务:\n")
	fmt.Printf("   订单ID: %d\n", retrievedTask.OrderID)
	fmt.Printf("   重试类型: %d\n", retrievedTask.RetryType)
	fmt.Printf("   重试原因: %s\n", retrievedTask.Reason)
	fmt.Printf("   创建时间: %s\n", retrievedTask.CreatedAt.Format(time.RFC3339))

	fmt.Println("\n🎉 重试队列测试完成！")
}