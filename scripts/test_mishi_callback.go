package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type Order struct {
	ID           int64     `gorm:"column:id"`
	CustomerID   int64     `gorm:"column:customer_id"`
	OrderNumber  string    `gorm:"column:order_number"`
	ProductID    int64     `gorm:"column:product_id"`
	Denom        string    `gorm:"column:denom"`
	TotalPrice   float64   `gorm:"column:total_price"`
	Price        float64   `gorm:"column:price"`
	Status       int       `gorm:"column:status"`
	CreateTime   time.Time `gorm:"column:create_time"`
}

func main() {
	// 数据库连接
	dsn := "root:123456@tcp(localhost:3306)/recharge?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("数据库连接测试失败:", err)
	}
	fmt.Println("数据库连接成功")

	// 生成订单号
	orderNumber := fmt.Sprintf("M%s%s", time.Now().Format("20060102150405"), "testMS")
	fmt.Printf("生成订单号: %s\n", orderNumber)

	// 检查订单是否已存在
	var existingID int64
	err = db.QueryRow("SELECT id FROM orders WHERE order_number = ?", orderNumber).Scan(&existingID)
	if err == nil {
		fmt.Printf("订单已存在，ID: %d\n", existingID)
		return
	}

	// 创建测试订单
	query := `INSERT INTO orders (customer_id, order_number, product_id, denom, total_price, price, status, create_time, update_time) 
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	now := time.Now()
	result, err := db.Exec(query, 1, orderNumber, 1, "100", 10.0, 10.0, 1, now, now)
	if err != nil {
		log.Fatal("创建订单失败:", err)
	}

	orderID, err := result.LastInsertId()
	if err != nil {
		log.Fatal("获取订单ID失败:", err)
	}

	fmt.Printf("测试订单创建成功，ID: %d, 订单号: %s\n", orderID, orderNumber)
}