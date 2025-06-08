package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"recharge-go/internal/router"
	"recharge-go/internal/service"
)

// ServerApp 服务器应用
type ServerApp struct {
	container      *Container
	server         *http.Server
	statisticsTask *service.StatisticsTask
}

// NewServerApp 创建新的服务器应用
func NewServerApp(container *Container) *ServerApp {
	return &ServerApp{
		container: container,
	}
}

// Start 启动服务器
func (s *ServerApp) Start(ctx context.Context) error {
	log.Println("正在启动服务器...")

	// 初始化并启动统计任务
	s.statisticsTask = s.container.GetServices().StatisticsTask
	s.statisticsTask.Start()

	// 使用优化后的路由设置
	r := router.SetupRouterV2(
		s.container.GetSecurityMiddleware(),
		s.container.GetMetricsManager(),
		s.container.GetControllers(),         // 传递控制器
		s.container.GetServices().User,       // 用户服务
		s.container.GetControllers().UserLog, // 用户日志控制器
		s.container.GetDB(),                  // 数据库
	)

	// 创建HTTP服务器
	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.container.GetConfig().Server.Port),
		Handler: r,
	}

	// 启动服务器
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("启动服务器失败: %v", err)
		}
	}()

	log.Printf("服务器已启动，监听端口: %d", s.container.GetConfig().Server.Port)
	return nil
}

// Stop 停止服务器
func (s *ServerApp) Stop(ctx context.Context) error {
	log.Println("正在停止服务器...")

	// 停止统计任务
	if s.statisticsTask != nil {
		// 注意：StatisticsTask 可能没有 Stop 方法，需要根据实际情况调整
		// s.statisticsTask.Stop()
	}

	// 停止HTTP服务器
	if s.server != nil {
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		if err := s.server.Shutdown(ctx); err != nil {
			return err
		}
	}

	// 关闭容器资源
	return s.container.Close()
}
