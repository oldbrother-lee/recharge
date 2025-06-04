package main

import (
	"log"

	"recharge-go/internal/app"
)

func main() {
	// 创建容器
	container, err := app.NewContainer()
	if err != nil {
		log.Fatalf("创建容器失败: %v", err)
	}

	// 创建充值应用
	rechargeApp := app.NewRechargeApp(container)

	// 创建并运行框架
	framework := app.NewFramework(rechargeApp)
	if err := framework.Run(); err != nil {
		log.Fatalf("运行应用失败: %v", err)
	}
}