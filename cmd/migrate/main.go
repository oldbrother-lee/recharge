package main

import (
	"flag"
	"fmt"
	"log"

	"recharge-go/migrations"

	"github.com/spf13/viper"
)

func loadConfig() error {
	// 设置配置文件路径
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("configs")

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config file: %v", err)
	}

	return nil
}

func main() {
	// 解析命令行参数
	check := flag.Bool("check", false, "只检查源数据库表结构")
	migrateData := flag.Bool("data", false, "只迁移数据")
	flag.Parse()

	// 加载配置
	if err := loadConfig(); err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 构建数据库连接字符串
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		viper.GetString("database.user"),
		viper.GetString("database.password"),
		viper.GetString("database.host"),
		viper.GetInt("database.port"),
		viper.GetString("database.dbname"),
	)

	log.Printf("数据库连接信息: %s", dsn)

	// 创建迁移器
	migrator, err := migrations.NewMigrator(dsn)
	if err != nil {
		log.Fatalf("创建迁移器失败: %v", err)
	}
	defer migrator.Close()

	if *check {
		log.Println("开始检查源数据库表结构")
		if err := migrator.CheckSourceTables(); err != nil {
			log.Fatalf("检查源数据库失败: %v", err)
		}
		log.Println("检查完成")
		return
	}

	if *migrateData {
		log.Println("开始迁移数据")
		if err := migrator.MigrateData(); err != nil {
			log.Fatalf("迁移数据失败: %v", err)
		}
		log.Println("数据迁移完成")
		return
	}

	// 否则执行所有未应用的迁移
	log.Println("开始执行所有未应用的迁移")
	if err := migrator.RunMigrations(); err != nil {
		log.Fatalf("执行迁移失败: %v", err)
	}
	log.Println("迁移完成")
}
