package task

import (
	"context"
	"recharge-go/internal/service"
	"recharge-go/pkg/logger"
	"time"
)

type RetryTask struct {
	retryService *service.RetryService
	stopChan     chan struct{}
}

func NewRetryTask(retryService *service.RetryService) *RetryTask {
	return &RetryTask{
		retryService: retryService,
		stopChan:     make(chan struct{}),
	}
}

func (t *RetryTask) Start() {
	logger.Info("【重试任务启动】开始执行重试任务")
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-t.stopChan:
				logger.Info("【重试任务停止】收到停止信号")
				return
			case <-ticker.C:
				logger.Info("【重试任务执行】开始处理待重试记录")
				if err := t.retryService.ProcessRetries(context.Background()); err != nil {
					logger.Error("【重试任务执行失败】error: %v", err)
				} else {
					logger.Info("【重试任务执行完成】")
				}
			}
		}
	}()
}

func (t *RetryTask) Stop() {
	logger.Info("【重试任务停止】开始停止重试任务")
	close(t.stopChan)
	logger.Info("【重试任务已停止】")
}
