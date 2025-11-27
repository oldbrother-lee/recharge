package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	logger "recharge-go/pkg/log"

	"github.com/redis/go-redis/v9"
)

// TaskConfigListener 任务配置变更监听器
type TaskConfigListener struct {
	redis       *redis.Client
	taskService *TaskService
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewTaskConfigListener 创建任务配置监听器
func NewTaskConfigListener(parentCtx context.Context, redisClient *redis.Client, taskService *TaskService) *TaskConfigListener {
	ctx, cancel := context.WithCancel(parentCtx)
	return &TaskConfigListener{
		redis:       redisClient,
		taskService: taskService,
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Start 开始监听配置变更
func (l *TaskConfigListener) Start() error {
	pubsub := l.redis.Subscribe(l.ctx, TaskConfigChangeChannel)
	defer pubsub.Close()

	logger.InfoV2("start_listen_task_config_change")

	// 使用goroutine来处理消息接收，避免阻塞
	msgChan := make(chan *redis.Message, 1)
	errorChan := make(chan error, 1)

	go func() {
		for {
			msg, err := pubsub.ReceiveMessage(l.ctx)
			if err != nil {
				select {
				case errorChan <- err:
				case <-l.ctx.Done():
				}
				return
			}
			select {
			case msgChan <- msg:
			case <-l.ctx.Done():
				return
			}
		}
	}()

	for {
		select {
		case <-l.ctx.Done():
			logger.WithContextCategory(l.ctx, "task_config").Info("任务配置监听器已停止")
			return nil
		case err := <-errorChan:
			if l.ctx.Err() != nil {
				// Context已取消，正常退出
				return nil
			}
			logger.WithContextCategory(l.ctx, "task_config").Error("接收配置变更消息失败", logger.ErrorV2(err))
			// 等待一段时间后重试
			select {
			case <-l.ctx.Done():
				return nil
			case <-time.After(5 * time.Second):
				continue
			}
		case msg := <-msgChan:
			if err := l.handleConfigChangeEvent(msg.Payload); err != nil {
				logger.WithContextCategory(l.ctx, "task_config").Error("处理配置变更事件失败", logger.ErrorV2(err))
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

	logger.WithContextCategory(l.ctx, "task_config").Info("收到任务配置变更事件",
		logger.StringV2("type", event.Type),
		logger.Int64V2("config_id", event.ConfigID),
		logger.Int64V2("timestamp", event.Timestamp),
	)

	// 触发任务配置重载
	if err := l.taskService.ReloadTaskConfig(); err != nil {
		return fmt.Errorf("重载任务配置失败: %w", err)
	}

	logger.WithContextCategory(l.ctx, "task_config").Info("任务配置重载完成")
	return nil
}
