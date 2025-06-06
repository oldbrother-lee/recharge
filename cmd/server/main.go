package main

import (
	"flag"
	"log"
	"time"

	"recharge-go/internal/app"
	"recharge-go/internal/utils"
	"recharge-go/pkg/database"
)

func main() {
	// 添加命令行参数
	configPath := flag.String("config", "configs/config.yaml", "配置文件路径")
	flag.Parse()

	// 记录应用启动时间
	uptimeManager := utils.GetUptimeManager()
	uptimeManager.SetStartTime(time.Now())
	log.Println("应用启动时间已记录")

	// 初始化数据库
	if err := database.InitDB(); err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}

	// 创建容器时传入配置文件路径和服务名
	container, err := app.NewContainerWithConfigAndService(*configPath, "server")
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
