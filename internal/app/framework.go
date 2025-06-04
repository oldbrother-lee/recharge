package app

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
)

// Application 定义应用接口
type Application interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// Framework 应用运行框架
type Framework struct {
	app Application
}

// NewFramework 创建新的应用框架
func NewFramework(app Application) *Framework {
	return &Framework{
		app: app,
	}
}

// Run 运行应用
func (f *Framework) Run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动应用
	if err := f.app.Start(ctx); err != nil {
		return err
	}

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("正在优雅关闭应用...")

	// 停止应用
	if err := f.app.Stop(ctx); err != nil {
		log.Printf("停止应用时出错: %v", err)
		return err
	}

	log.Println("应用已成功关闭")
	return nil
}