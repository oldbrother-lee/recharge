package app

import (
	"context"
	"log"

	"recharge-go/internal/handler"
	"recharge-go/internal/task"
	"recharge-go/pkg/queue"
)

// NotificationApp 通知应用
type NotificationApp struct {
	container        *Container
	handler          *handler.NotificationHandler
	notificationTask *task.NotificationTask
}

// NewNotificationApp 创建新的通知应用
func NewNotificationApp(container *Container) *NotificationApp {
	return &NotificationApp{
		container: container,
	}
}

// Start 启动通知处理器
func (n *NotificationApp) Initialize() error {
	// 创建通知处理器
	n.handler = handler.NewNotificationHandler(
		n.container.GetServices().Notification,
	)

	// 创建通知任务处理器
	queueInstance := queue.NewRedisQueue()
	n.notificationTask = task.NewNotificationTask(
		n.container.GetServices().Notification,
		n.container.GetServices().Platform,
		queueInstance,
		3, // 最大重试次数
	)

	return nil
}

// Start 启动通知应用
func (n *NotificationApp) Start(ctx context.Context) error {
	if err := n.Initialize(); err != nil {
		return err
	}

	// 启动通知任务处理器
	log.Println("启动通知任务处理器...")
	go func() {
		if err := n.notificationTask.Start(ctx); err != nil {
			log.Printf("通知任务处理器启动失败: %v", err)
		}
	}()

	return nil
}

// Stop 停止通知应用
func (n *NotificationApp) Stop(ctx context.Context) error {
	log.Println("正在停止通知处理器...")

	// 停止通知任务处理器
	if n.notificationTask != nil {
		n.notificationTask.Stop()
	}

	// 关闭容器资源
	return n.container.Close()
}
