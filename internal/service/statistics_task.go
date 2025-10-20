package service

import (
	"context"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"recharge-go/pkg/logger"
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
func (t *StatisticsTask) Start(ctx context.Context) {
	c := cron.New()

	// 每天凌晨1点执行统计任务
	_, err := c.AddFunc("51 1 * * *", func() {
		logger.WithContextCategory(ctx, "statistics").Info("开始执行统计任务")
		if err := t.statisticsSvc.UpdateStatistics(ctx); err != nil {
			logger.WithContextCategory(ctx, "statistics").Error("统计任务执行失败", logger.ErrorV2(err))
			return
		}

		logger.WithContextCategory(ctx, "statistics").Info("统计任务执行完成")
	})

	if err != nil {
		logger.WithContextCategory(ctx, "statistics").Error("添加统计任务失败", logger.ErrorV2(err))
		return
	}

	c.Start()
	logger.WithContextCategory(ctx, "statistics").Info("统计任务已启动")

	// 在ctx取消时停止cron
	go func() {
		<-ctx.Done()
		logger.WithContextCategory(ctx, "statistics").Info("统计任务停止中...")
		c.Stop()
		logger.WithContextCategory(ctx, "statistics").Info("统计任务已停止")
	}()
}
