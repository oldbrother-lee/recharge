package repository

import (
	"context"
	"recharge-go/internal/model"
	"time"

	"gorm.io/gorm"
)

type OrderStatisticsRepository interface {
	Create(ctx context.Context, stats *model.OrderStatistics) error
	Update(ctx context.Context, stats *model.OrderStatistics) error
	GetByDateAndOperator(ctx context.Context, date time.Time, operator string) (*model.OrderStatistics, error)
	GetOverview(ctx context.Context) (*model.OrderStatisticsOverview, error)
	GetOperatorStats(ctx context.Context, startDate, endDate time.Time) ([]model.OrderStatisticsOperator, error)
	GetDailyStats(ctx context.Context, startDate, endDate time.Time) ([]model.OrderStatisticsDaily, error)
	GetTrend(ctx context.Context, startDate, endDate time.Time, operator string) ([]model.OrderStatisticsTrend, error)
}

type orderStatisticsRepo struct {
	db *gorm.DB
}

func NewOrderStatisticsRepository(db *gorm.DB) OrderStatisticsRepository {
	return &orderStatisticsRepo{db: db}
}

func (r *orderStatisticsRepo) Create(ctx context.Context, stats *model.OrderStatistics) error {
	return r.db.WithContext(ctx).Create(stats).Error
}

func (r *orderStatisticsRepo) Update(ctx context.Context, stats *model.OrderStatistics) error {
	return r.db.WithContext(ctx).Save(stats).Error
}

func (r *orderStatisticsRepo) GetByDateAndOperator(ctx context.Context, date time.Time, operator string) (*model.OrderStatistics, error) {
	var stats model.OrderStatistics
	err := r.db.WithContext(ctx).
		Where("date = ? AND operator = ?", date.Format("2006-01-02"), operator).
		First(&stats).Error
	if err != nil {
		return nil, err
	}
	return &stats, nil
}

func (r *orderStatisticsRepo) GetOverview(ctx context.Context) (*model.OrderStatisticsOverview, error) {
	var overview model.OrderStatisticsOverview

	// 获取总订单数
	if err := r.db.WithContext(ctx).Model(&model.OrderStatistics{}).
		Select("COALESCE(SUM(total_orders), 0) as total").
		Scan(&overview.Total.Total).Error; err != nil {
		return nil, err
	}

	// 获取昨日订单数
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	if err := r.db.WithContext(ctx).Model(&model.OrderStatistics{}).
		Where("date = ?", yesterday).
		Select("COALESCE(SUM(total_orders), 0) as yesterday").
		Scan(&overview.Total.Yesterday).Error; err != nil {
		return nil, err
	}

	// 获取今日订单数
	today := time.Now().Format("2006-01-02")
	if err := r.db.WithContext(ctx).Model(&model.OrderStatistics{}).
		Where("date = ?", today).
		Select("COALESCE(SUM(total_orders), 0) as today").
		Scan(&overview.Total.Today).Error; err != nil {
		return nil, err
	}

	// 获取订单状态统计
	if err := r.db.WithContext(ctx).Model(&model.OrderStatistics{}).
		Where("date = ?", today).
		Select(
			"COALESCE(SUM(total_orders - success_orders - failed_orders), 0) as processing",
			"COALESCE(SUM(success_orders), 0) as success",
			"COALESCE(SUM(failed_orders), 0) as failed",
		).
		Scan(&overview.Status).Error; err != nil {
		return nil, err
	}

	// 获取盈利统计
	if err := r.db.WithContext(ctx).Model(&model.OrderStatistics{}).
		Where("date = ?", today).
		Select(
			"COALESCE(SUM(cost_amount), 0) as cost_amount",
			"COALESCE(SUM(profit_amount), 0) as profit_amount",
		).
		Scan(&overview.Profit).Error; err != nil {
		return nil, err
	}

	return &overview, nil
}

func (r *orderStatisticsRepo) GetOperatorStats(ctx context.Context, startDate, endDate time.Time) ([]model.OrderStatisticsOperator, error) {
	var stats []model.OrderStatisticsOperator

	err := r.db.WithContext(ctx).Model(&model.OrderStatistics{}).
		Where("date BETWEEN ? AND ?", startDate.Format("2006-01-02"), endDate.Format("2006-01-02")).
		Select(
			"operator",
			"SUM(total_orders) as total_orders",
			"SUM(success_orders) as success_orders",
			"SUM(failed_orders) as failed_orders",
			"SUM(cost_amount) as cost_amount",
			"SUM(profit_amount) as profit_amount",
		).
		Group("operator").
		Scan(&stats).Error

	if err != nil {
		return nil, err
	}

	// // 计算成功率
	// for i := range stats {
	// 	if stats[i].TotalOrders > 0 {
	// 		stats[i].SuccessRate = float64(stats[i].SuccessOrders) / float64(stats[i].TotalOrders) * 100
	// 	}
	// }

	return stats, nil
}

func (r *orderStatisticsRepo) GetDailyStats(ctx context.Context, startDate, endDate time.Time) ([]model.OrderStatisticsDaily, error) {
	var stats []model.OrderStatisticsDaily

	err := r.db.WithContext(ctx).Model(&model.OrderStatistics{}).
		Where("date BETWEEN ? AND ?", startDate.Format("2006-01-02"), endDate.Format("2006-01-02")).
		Select(
			"date",
			"SUM(total_orders) as total_orders",
			"SUM(success_orders) as success_orders",
			"SUM(failed_orders) as failed_orders",
			"SUM(cost_amount) as cost_amount",
			"SUM(profit_amount) as profit_amount",
		).
		Group("date").
		Order("date ASC").
		Scan(&stats).Error

	if err != nil {
		return nil, err
	}

	// 计算成功率
	for i := range stats {
		if stats[i].TotalOrders > 0 {
			stats[i].SuccessRate = float64(stats[i].SuccessOrders) / float64(stats[i].TotalOrders) * 100
		}
	}

	return stats, nil
}

func (r *orderStatisticsRepo) GetTrend(ctx context.Context, startDate, endDate time.Time, operator string) ([]model.OrderStatisticsTrend, error) {
	var trends []model.OrderStatisticsTrend

	query := r.db.WithContext(ctx).Model(&model.OrderStatistics{}).
		Where("date BETWEEN ? AND ?", startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))

	if operator != "" {
		query = query.Where("operator = ?", operator)
	}

	err := query.Select(
		"date",
		"SUM(total_orders) as total_orders",
		"SUM(success_orders) as success_orders",
		"SUM(failed_orders) as failed_orders",
		"SUM(cost_amount) as cost_amount",
		"SUM(profit_amount) as profit_amount",
	).
		Group("date").
		Order("date ASC").
		Scan(&trends).Error

	if err != nil {
		return nil, err
	}

	// 计算成功率
	for i := range trends {
		if trends[i].TotalOrders > 0 {
			trends[i].SuccessRate = float64(trends[i].SuccessOrders) / float64(trends[i].TotalOrders) * 100
		}
	}

	return trends, nil
}
