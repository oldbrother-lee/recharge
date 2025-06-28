package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"recharge-go/pkg/logger"
)

// TaskConfigListener 任务配置变更监听器
type TaskConfigListener struct {
	redisClient *redis.Client
	taskService *TaskService
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewTaskConfigListener 创建任务配置监听器
func NewTaskConfigListener(redisClient *redis.Client, taskService *TaskService) *TaskConfigListener {
	ctx, cancel := context.WithCancel(context.Background())
	return &TaskConfigListener{
		redisClient: redisClient,
		taskService: taskService,
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Start 开始监听配置变更
func (l *TaskConfigListener) Start() error {
	pubsub := l.redisClient.Subscribe(context.Background(), TaskConfigChangeChannel)
	defer pubsub.Close()

	logger.Info("开始监听任务配置变更事件")

	for {
		select {
		case <-l.ctx.Done():
			logger.Info("任务配置监听器已停止")
			return nil
		default:
			msg, err := pubsub.ReceiveMessage(l.ctx)
			if err != nil {
				if l.ctx.Err() != nil {
					// Context已取消，正常退出
					return nil
				}
				logger.Error(fmt.Sprintf("接收配置变更消息失败: %v", err))
				// 等待一段时间后重试
				time.Sleep(5 * time.Second)
				continue
			}

			if err := l.handleConfigChangeEvent(msg.Payload); err != nil {
				logger.Error(fmt.Sprintf("处理配置变更事件失败: %v", err))
			}
		}
	}
}

// Stop 停止监听
func (l *TaskConfigListener) Stop() {
	l.cancel()
}

// handleConfigChangeEvent 处理配置变更事件
func (l *TaskConfigListener) handleConfigChangeEvent(payload string) error {
	var event TaskConfigChangeEvent
	if err := json.Unmarshal([]byte(payload), &event); err != nil {
		return fmt.Errorf("解析配置变更事件失败: %w", err)
	}

	logger.Info(fmt.Sprintf("收到任务配置变更事件: type=%s, config_id=%d, timestamp=%d", 
		event.Type, event.ConfigID, event.Timestamp))

	// 触发任务配置重载
	if err := l.taskService.ReloadTaskConfig(); err != nil {
		return fmt.Errorf("重载任务配置失败: %w", err)
	}

	logger.Info("任务配置重载完成")
	return nil
}