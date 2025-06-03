package service

import (
	"context"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

type StatisticsTask struct {
	statisticsSvc StatisticsService
	logger        *zap.Logger
}

func NewStatisticsTask(statisticsSvc StatisticsService, logger *zap.Logger) *StatisticsTask {
	return &StatisticsTask{
		statisticsSvc: statisticsSvc,
		logger:        logger,
	}
}

// Start 启动统计任务
func (t *StatisticsTask) Start() {
	c := cron.New()

	// 每天凌晨1点执行统计任务
	_, err := c.AddFunc("51 1 * * *", func() {
		t.logger.Info("开始执行统计任务")
		ctx := context.Background()

		if err := t.statisticsSvc.UpdateStatistics(ctx); err != nil {
			t.logger.Error("统计任务执行失败", zap.Error(err))
			return
		}

		t.logger.Info("统计任务执行完成")
	})

	if err != nil {
		t.logger.Error("添加统计任务失败", zap.Error(err))
		return
	}

	c.Start()
	t.logger.Info("统计任务已启动")
}
