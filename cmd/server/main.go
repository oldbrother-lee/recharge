package main

import (
	"log"

	"recharge-go/internal/app"
	"recharge-go/pkg/database"
)

func main() {
	// 初始化数据库
	if err := database.InitDB(); err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}

	// 创建容器
	container, err := app.NewContainer()
	if err != nil {
		log.Fatalf("创建容器失败: %v", err)
	}

	// 创建服务器应用
	serverApp := app.NewServerApp(container)

	// 创建并运行框架
	framework := app.NewFramework(serverApp)
	if err := framework.Run(); err != nil {
		log.Fatalf("运行应用失败: %v", err)
	}
}
