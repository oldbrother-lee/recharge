package main

import (
	"fmt"
	"log"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type Order struct {
	ID                uint      `gorm:"primaryKey"`
	OrderNumber       string    `gorm:"column:order_number;uniqueIndex"`
	UserID            uint      `gorm:"column:user_id"`
	PlatformID        uint      `gorm:"column:platform_id"`
	PlatformAccountID uint      `gorm:"column:platform_account_id"`
	PhoneNumber       string    `gorm:"column:phone_number"`
	Amount            float64   `gorm:"column:amount"`
	Status            int       `gorm:"column:status"`
	CreatedAt         time.Time `gorm:"column:created_at"`
	UpdatedAt         time.Time `gorm:"column:updated_at"`
}

func main() {
	// 数据库连接
	dsn := "root:password@tcp(localhost:3306)/recharge-new?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("连接数据库失败:", err)
	}

	// 创建测试订单
	order := Order{
		OrderNumber:       "P20250804170530tgeCit",
		UserID:            1,
		PlatformID:        1, // 假设kekebang平台ID为1
		PlatformAccountID: 1, // 假设账号ID为1
		PhoneNumber:       "13800138000",
		Amount:            100.00,
		Status:            2, // 待充值状态
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	// 插入或更新订单
	result := db.Where("order_number = ?", order.OrderNumber).FirstOrCreate(&order)
	if result.Error != nil {
		log.Fatal("创建订单失败:", result.Error)
	}

	fmt.Printf("测试订单创建成功: %s\n", order.OrderNumber)
}