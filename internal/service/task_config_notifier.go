package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

    "github.com/go-redis/redis/v8"
    logger "recharge-go/pkg/log"
)

// TaskConfigNotifier 任务配置变更通知器
type TaskConfigNotifier struct {
	redisClient *redis.Client
}

// NewTaskConfigNotifier 创建任务配置通知器
func NewTaskConfigNotifier(redisClient *redis.Client) *TaskConfigNotifier {
	return &TaskConfigNotifier{
		redisClient: redisClient,
	}
}

// TaskConfigChangeEvent 任务配置变更事件
type TaskConfigChangeEvent struct {
	Type      string `json:"type"`      // create, update, delete
	ConfigID  int64  `json:"config_id"` // 配置ID
	Timestamp int64  `json:"timestamp"` // 时间戳
}

const (
	TaskConfigChangeChannel = "task_config_change"
	EventTypeCreate         = "create"
	EventTypeUpdate         = "update"
	EventTypeDelete         = "delete"
)

// NotifyConfigChange 通知配置变更
func (n *TaskConfigNotifier) NotifyConfigChange(ctx context.Context, eventType string, configID int64) error {
	event := TaskConfigChangeEvent{
		Type:      eventType,
		ConfigID:  configID,
		Timestamp: time.Now().Unix(),
	}

	eventData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("序列化事件失败: %w", err)
	}

	err = n.redisClient.Publish(ctx, TaskConfigChangeChannel, string(eventData)).Err()
	if err != nil {
		return fmt.Errorf("发布配置变更事件失败: %w", err)
	}

	logger.WithContextCategory(ctx, "task_config").Info("已发布任务配置变更事件",
		logger.StringV2("type", eventType),
		logger.Int64V2("config_id", configID),
	)
	return nil
}

// NotifyConfigCreate 通知配置创建
func (n *TaskConfigNotifier) NotifyConfigCreate(ctx context.Context, configID int64) error {
	return n.NotifyConfigChange(ctx, EventTypeCreate, configID)
}

// NotifyConfigUpdate 通知配置更新
func (n *TaskConfigNotifier) NotifyConfigUpdate(ctx context.Context, configID int64) error {
	return n.NotifyConfigChange(ctx, EventTypeUpdate, configID)
}

// NotifyConfigDelete 通知配置删除
func (n *TaskConfigNotifier) NotifyConfigDelete(ctx context.Context, configID int64) error {
	return n.NotifyConfigChange(ctx, EventTypeDelete, configID)
}
