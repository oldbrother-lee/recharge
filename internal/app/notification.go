package app

import (
	"context"

	"recharge-go/internal/handler"
	"recharge-go/internal/task"
	logger "recharge-go/pkg/log"
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

	// 读取配置中的通知最大重试次数
	maxRetries := n.container.GetConfig().Notification.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3 // 兜底，避免配置缺失导致不可用
	}

	// 创建通知任务处理器
	queueInstance := n.container.GetTaskQueue()
	n.notificationTask = task.NewNotificationTask(
		n.container.GetServices().Notification,
		n.container.GetServices().Order,
		n.container.GetServices().OrderTrace,
		n.container.GetServices().Platform,
		queueInstance,
		maxRetries, // 从配置读取最大重试次数
		logger.Log, // 添加 logger 参数
	)

	return nil
}

// Start 启动通知应用
func (n *NotificationApp) Start(ctx context.Context) error {
	if err := n.Initialize(); err != nil {
		return err
	}

	// 启动通知任务处理器
	logger.Log.Info("启动通知任务处理器...")
	go func() {
		if err := n.notificationTask.Start(ctx); err != nil {
			logger.Log.Error("通知任务处理器启动失败", logger.ErrorV2(err))
		}
	}()

	return nil
}

// Stop 停止通知应用
func (n *NotificationApp) Stop(ctx context.Context) error {
	logger.Log.Info("正在停止通知处理器...")

	// 停止通知任务处理器
	if n.notificationTask != nil {
		n.notificationTask.Stop()
	}

	// 关闭容器资源
	return n.container.Close()
}
