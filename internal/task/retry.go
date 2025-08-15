package task

import (
	"context"
	"encoding/json"
	"fmt"
	"recharge-go/configs"
	"recharge-go/internal/model"
	"recharge-go/internal/service"
	"recharge-go/pkg/logger"
	"recharge-go/pkg/queue"
	"time"
)

type RetryTask struct {
	retryService *service.RetryService
	stopChan     chan struct{}
	config       *configs.Config
	queue        queue.Queue
}

func NewRetryTask(retryService *service.RetryService, config *configs.Config, queue queue.Queue) *RetryTask {
	return &RetryTask{
		retryService: retryService,
		stopChan:     make(chan struct{}),
		config:       config,
		queue:        queue,
	}
}

func (t *RetryTask) Start() {
	queueName := t.config.RetryTask.QueueName
	if queueName == "" {
		queueName = "retry_queue" // 默认队列名称
	}
	pollInterval := time.Duration(t.config.RetryTask.PollInterval) * time.Second
	if pollInterval <= 0 {
		pollInterval = 1 * time.Second // 默认轮询间隔
		logger.Warn("【重试任务】配置的轮询间隔无效，使用默认值: %v", pollInterval)
	}

	logger.Info("【重试任务启动】开始从队列消费重试任务", "queue_name", queueName, "poll_interval", pollInterval)

	// 启动多个消费者
	consumerCount := t.config.RetryTask.ConsumerCount
	if consumerCount <= 0 {
		consumerCount = 1 // 默认1个消费者
	}

	for i := 0; i < consumerCount; i++ {
		go t.startConsumer(i+1, queueName, pollInterval)
	}

	// 同时保持定时扫描数据库的功能作为备用
	go t.startPeriodicRetry()
}

func (t *RetryTask) Stop() {
	logger.Info("【重试任务停止】开始停止重试任务")
	close(t.stopChan)
	logger.Info("【重试任务已停止】")
}

// processRetryTask 处理重试任务
func (t *RetryTask) processRetryTask(ctx context.Context, taskData interface{}) error {
	// 解析任务数据
	taskStr, ok := taskData.(string)
	if !ok {
		return fmt.Errorf("任务数据格式错误")
	}

	var task model.RetryTaskMessage
	if err := json.Unmarshal([]byte(taskStr), &task); err != nil {
		return fmt.Errorf("解析重试任务失败: %v", err)
	}

	logger.Info("【处理队列重试任务】", "order_id", task.OrderID, "retry_type", task.RetryType, "reason", task.Reason)

	// 获取订单信息
	order, err := t.retryService.GetOrderByID(ctx, task.OrderID)
	if err != nil {
		return fmt.Errorf("获取订单信息失败: %v", err)
	}

	// 调用重试服务处理
	if err := t.retryService.HandleRetry(ctx, order, task.RetryType); err != nil {
		return fmt.Errorf("执行重试失败: %v", err)
	}

	logger.Info("【队列重试任务处理完成】", "order_id", task.OrderID)
	return nil
}

// startConsumer 启动消费者
func (t *RetryTask) startConsumer(consumerID int, queueName string, pollInterval time.Duration) {
	logger.Info("【重试任务消费者启动】", "consumer_id", consumerID, "queue_name", queueName)
	for {
		select {
		case <-t.stopChan:
			logger.Info("【重试任务消费者停止】收到停止信号", "consumer_id", consumerID)
			return
		default:
			// 从队列中获取重试任务
			ctx := context.Background()
			taskData, err := t.queue.Pop(ctx, queueName)
			if err != nil {
				logger.Error("【从队列获取重试任务失败】", "consumer_id", consumerID, "error", err)
				time.Sleep(5 * time.Second) // 出错时等待5秒
				continue
			}

			if taskData == nil {
				// 队列为空，等待后继续
				time.Sleep(pollInterval)
				continue
			}

			// 处理重试任务
			if err := t.processRetryTask(ctx, taskData); err != nil {
				logger.Error("【处理重试任务失败】", "consumer_id", consumerID, "error", err)
			} else {
				logger.Info("【重试任务处理成功】", "consumer_id", consumerID)
			}
		}
	}
}

// startPeriodicRetry 启动定时重试（作为备用机制）
func (t *RetryTask) startPeriodicRetry() {
	interval := time.Duration(t.config.RetryTask.Interval) * time.Second
	// 如果配置的间隔为0或负数，使用默认值30秒
	if interval <= 0 {
		interval = 30 * time.Second
		logger.Warn("【定时重试任务】配置的间隔无效，使用默认值: %v", interval)
	}
	logger.Info("【定时重试任务启动】作为备用机制，执行间隔: %v", interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-t.stopChan:
			logger.Info("【定时重试任务停止】收到停止信号")
			return
		case <-ticker.C:
			logger.Info("【定时重试任务执行】开始处理待重试记录")
			if err := t.retryService.ProcessRetries(context.Background()); err != nil {
				logger.Error("【定时重试任务执行失败】error: %v", err)
			} else {
				logger.Info("【定时重试任务执行完成】")
			}
		}
	}
}
