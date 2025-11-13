package main

import (
	"context"
	"flag"
	"log"

	"recharge-go/internal/app"
	"recharge-go/internal/repository"
	"recharge-go/internal/service/pullorder"
)

func main() {
	// 命令行参数：配置路径、是否常驻、间隔秒数
	configPath := flag.String("config", "configs/config.yaml", "配置文件路径")
	watch := flag.Bool("watch", false, "是否持续运行拉单")
	interval := flag.Int("interval", 0, "拉单间隔（秒），仅在 --watch 模式下生效；默认使用配置 Task.Interval")
	flag.Parse()

	// 创建容器
	container, err := app.NewContainerWithConfigAndService(*configPath, "pullorder")
	if err != nil {
		log.Fatalf("创建容器失败: %v", err)
	}

	if *watch {
		// 常驻运行模式：使用应用框架
		// 如果提供了 --interval，覆盖配置值
		if *interval > 0 {
			container.GetConfig().Task.Interval = *interval
		}
		pullApp := app.NewPullOrderApp(container)
		framework := app.NewFramework(pullApp)
		if err := framework.Run(); err != nil {
			log.Fatalf("运行拉单常驻应用失败: %v", err)
		}
		return
	}

	// 一次性执行模式：不启用服务、不进入循环
	defer func() {
		if err := container.Close(); err != nil {
			log.Printf("关闭容器失败: %v", err)
		}
	}()

	// 实例化仓库与服务
	platformAccountRepo := repository.NewPlatformAccountRepository(container.GetDB())
	variantRepo := repository.NewPlatformAccountVariantRepository(container.GetDB())
	orderSvc := container.GetServices().Order

	// 构建拉单管理器与调度器（一次性执行）
	mgr := pullorder.NewPullOrderManager(orderSvc, platformAccountRepo, variantRepo)
	// 注册章鱼平台到管理器（统一配置拉单）
	zhangyuPlatform := pullorder.NewZhangyuPullPlatform(platformAccountRepo, variantRepo)
	mgr.RegisterPlatform(zhangyuPlatform)
	scheduler := pullorder.NewPullOrderScheduler(mgr, platformAccountRepo, variantRepo, nil)

	// 执行一次拉单处理
	if err := scheduler.ProcessOnce(context.Background()); err != nil {
		log.Fatalf("拉单执行失败: %v", err)
	}

	log.Println("拉单执行完成（一次性处理）")
}