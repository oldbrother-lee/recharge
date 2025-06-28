package app

import (
	"context"
	"log"

	"recharge-go/internal/service"
)

// TaskApp 任务应用
type TaskApp struct {
	container   *Container
	taskService *service.TaskService
}

// NewTaskApp 创建新的任务应用
func NewTaskApp(container *Container) *TaskApp {
	return &TaskApp{
		container: container,
	}
}

// Start 启动任务
func (t *TaskApp) Start(ctx context.Context) error {
	log.Println("正在启动任务应用...")

	// 获取启用的任务配置
	taskConfigs, err := t.container.GetRepositories().TaskConfig.GetEnabledConfigs()
	if err != nil {
		log.Printf("获取任务配置失败: %v", err)
		return err
	}

	if len(taskConfigs) == 0 {
		log.Println("没有找到启用的任务配置，但仍启动任务服务以监听配置变化")
	} else {
		log.Printf("找到 %d 个启用的任务配置", len(taskConfigs))
	}

	// 创建任务服务
	t.taskService = t.container.GetServices().Task

	// 创建并设置配置监听器
	redisClient := t.container.GetRedis()
	configListener := service.NewTaskConfigListener(redisClient, t.taskService)
	t.taskService.SetConfigListener(configListener)

	// 启动配置监听器
	t.taskService.StartConfigListener()

	// 无论是否有配置都启动任务服务
	t.taskService.StartTask()
	// 启动订单详情查询任务
	t.taskService.StartOrderDetailsTask()

	log.Println("任务应用已启动，正在处理启用的任务配置和订单详情查询")
	return nil
}

// Stop 停止任务
func (t *TaskApp) Stop(ctx context.Context) error {
	log.Println("正在停止任务应用...")

	// 停止任务服务
	if t.taskService != nil {
		t.taskService.StopTask()
	}

	// 关闭容器资源
	return t.container.Close()
}
