package main

import (
	"log"

	"recharge-go/internal/app"
)

func main() {
	// 创建迁移应用
	migrateApp, err := app.NewMigrateApp()
	if err != nil {
		log.Fatalf("创建迁移应用失败: %v", err)
	}

	// 创建并运行框架
	framework := app.NewFramework(migrateApp)
	if err := framework.Run(); err != nil {
		log.Fatalf("运行应用失败: %v", err)
	}
}
