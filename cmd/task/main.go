package main

import (
	"flag"
	"log"

	"recharge-go/internal/app"
)

func main() {
	// 添加命令行参数
	configPath := flag.String("config", "configs/config.yaml", "配置文件路径")
	flag.Parse()

	// 创建容器时传入配置文件路径和服务名
	container, err := app.NewContainerWithConfigAndService(*configPath, "task")
	if err != nil {
		log.Fatalf("创建容器失败: %v", err)
	}

	// 创建任务应用
	taskApp := app.NewTaskApp(container)

	// 创建并运行框架
	framework := app.NewFramework(taskApp)
	if err := framework.Run(); err != nil {
		log.Fatalf("运行应用失败: %v", err)
	}
}
