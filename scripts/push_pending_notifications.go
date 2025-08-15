package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/go-redis/redis/v8"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"recharge-go/internal/model/notification"
)

func main() {
	// 数据库连接
	dsn := "root:qynfqepwq@tcp(localhost:3306)/recharge-new?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("连接数据库失败:", err)
	}

	// Redis连接
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	ctx := context.Background()

	// 查询所有待处理的通知记录
	var notifications []notification.NotificationRecord
	err = db.Where("status = ?", 1).Find(&notifications).Error
	if err != nil {
		log.Fatal("查询通知记录失败:", err)
	}

	fmt.Printf("找到 %d 条待处理的通知记录\n", len(notifications))

	// 批量推送到队列
	successCount := 0
	for _, notification := range notifications {
		// 将通知记录转换为JSON
		notificationJSON, err := json.Marshal(notification)
		if err != nil {
			log.Printf("序列化通知记录失败 (ID: %d): %v", notification.ID, err)
			continue
		}

		// 推送到Redis队列
		err = rdb.LPush(ctx, "notification_queue", string(notificationJSON)).Err()
		if err != nil {
			log.Printf("推送通知到队列失败 (ID: %d): %v", notification.ID, err)
			continue
		}

		successCount++
		fmt.Printf("成功推送通知记录 ID: %d, Order ID: %d, Platform: %s\n", 
			notification.ID, notification.OrderID, notification.PlatformCode)
	}

	fmt.Printf("\n批量推送完成！成功推送 %d/%d 条通知记录\n", successCount, len(notifications))

	// 检查队列长度
	queueLen, err := rdb.LLen(ctx, "notification_queue").Result()
	if err != nil {
		log.Printf("获取队列长度失败: %v", err)
	} else {
		fmt.Printf("当前队列中有 %d 条通知记录\n", queueLen)
	}
}