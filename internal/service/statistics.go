package service

import (
	"context"
	"recharge-go/internal/model"
	"recharge-go/internal/repository"
	"time"

	"gorm.io/gorm"
)

// 定义专用类型
type OperatorOrderCount struct {
	Operator int   `json:"operator"`
	Total    int64 `json:"total"`
}

type StatisticsService interface {
	GetOrderOverview(ctx context.Context) (*model.OrderStatisticsOverview, error)
	GetOperatorStatistics(ctx context.Context, startDate, endDate time.Time) ([]model.OrderStatisticsOperator, error)
	GetOperatorStatisticsByUser(ctx context.Context, startDate, endDate time.Time, userId int64) ([]model.OrderStatisticsOperator, error)
	GetDailyStatistics(ctx context.Context, startDate, endDate time.Time) ([]model.OrderStatisticsDaily, error)
	GetTrendStatistics(ctx context.Context, startDate, endDate time.Time, operator string) ([]model.OrderStatisticsTrend, error)
	UpdateStatistics(ctx context.Context) error
	GetOrderRealtimeStatistics(ctx context.Context, userId int64) (interface{}, error)
	GetOperatorOrderCount(ctx context.Context, start, end time.Time) ([]model.OperatorOrderCount, error)
}

type statisticsService struct {
	orderStatsRepo repository.OrderStatisticsRepository
	orderRepo      repository.OrderRepository
}

func NewStatisticsService(
	orderStatsRepo repository.OrderStatisticsRepository,
	orderRepo repository.OrderRepository,
) StatisticsService {
	return &statisticsService{
		orderStatsRepo: orderStatsRepo,
		orderRepo:      orderRepo,
	}
}

func (s *statisticsService) GetOrderOverview(ctx context.Context) (*model.OrderStatisticsOverview, error) {
	return s.orderStatsRepo.GetOverview(ctx)
}

func (s *statisticsService) GetOperatorStatistics(ctx context.Context, startDate, endDate time.Time) ([]model.OrderStatisticsOperator, error) {
	return s.orderRepo.GetOperatorRealtimeStatistics(ctx, startDate, endDate)
}

func (s *statisticsService) GetOperatorStatisticsByUser(ctx context.Context, startDate, endDate time.Time, userId int64) ([]model.OrderStatisticsOperator, error) {
	return s.orderRepo.GetOperatorRealtimeStatisticsByUser(ctx, startDate, endDate, userId)
}

func (s *statisticsService) GetDailyStatistics(ctx context.Context, startDate, endDate time.Time) ([]model.OrderStatisticsDaily, error) {
	return s.orderStatsRepo.GetDailyStats(ctx, startDate, endDate)
}

func (s *statisticsService) GetTrendStatistics(ctx context.Context, startDate, endDate time.Time, operator string) ([]model.OrderStatisticsTrend, error) {
	return s.orderStatsRepo.GetTrend(ctx, startDate, endDate, operator)
}

func (s *statisticsService) UpdateStatistics(ctx context.Context) error {
	// 获取今天的日期
	today := time.Now()
	todayStr := today.Format("2006-01-02")

	// 获取所有运营商
	operators := []struct {
		ID   int64
		Name string
	}{
		{1, "移动"},
		{2, "联通"},
		{3, "电信"},
	}

	for _, operator := range operators {
		// 获取该运营商今天的订单统计
		stats, err := s.orderStatsRepo.GetByDateAndOperator(ctx, today, operator.Name)
		if err != nil && err != gorm.ErrRecordNotFound {
			return err
		}

		if stats == nil {
			stats = &model.OrderStatistics{
				Date:     today,
				Operator: operator.Name,
			}
		}

		// 从订单表中获取实际数据
		var totalOrders, successOrders, failedOrders int64
		var costAmount, profitAmount float64

		// 获取总订单数
		if err := s.orderRepo.DB().Model(&model.Order{}).
			Joins("JOIN products ON orders.product_id = products.id").
			Where("products.isp = ? AND DATE(orders.create_time) = ?", operator.ID, todayStr).
			Count(&totalOrders).Error; err != nil {
			return err
		}

		// 获取成功订单数
		if err := s.orderRepo.DB().Model(&model.Order{}).
			Joins("JOIN products ON orders.product_id = products.id").
			Where("products.isp = ? AND DATE(orders.create_time) = ? AND orders.status = ?", operator.ID, todayStr, model.OrderStatusSuccess).
			Count(&successOrders).Error; err != nil {
			return err
		}

		// 获取失败订单数
		if err := s.orderRepo.DB().Model(&model.Order{}).
			Joins("JOIN products ON orders.product_id = products.id").
			Where("products.isp = ? AND DATE(orders.create_time) = ? AND orders.status IN (?)", operator.ID, todayStr, []model.OrderStatus{
				model.OrderStatusFailed,
				model.OrderStatusCancelled,
				model.OrderStatusRefunded,
			}).Count(&failedOrders).Error; err != nil {
			return err
		}

		// 获取成本和利润
		if err := s.orderRepo.DB().Model(&model.Order{}).
			Joins("JOIN products ON orders.product_id = products.id").
			Where("products.isp = ? AND DATE(orders.create_time) = ?", operator.ID, todayStr).
			Select("COALESCE(SUM(orders.total_price), 0) as cost_amount, COALESCE(SUM(orders.total_price - orders.price), 0) as profit_amount").
			Row().Scan(&costAmount, &profitAmount); err != nil {
			return err
		}

		// 更新统计数据
		stats.TotalOrders = totalOrders
		stats.SuccessOrders = successOrders
		stats.FailedOrders = failedOrders
		stats.CostAmount = costAmount
		stats.ProfitAmount = profitAmount

		// 保存或更新统计记录
		if stats.ID == 0 {
			if err := s.orderStatsRepo.Create(ctx, stats); err != nil {
				return err
			}
		} else {
			if err := s.orderStatsRepo.Update(ctx, stats); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *statisticsService) GetOrderRealtimeStatistics(ctx context.Context, userId int64) (interface{}, error) {
	// 直接传递 userId 给仓库层
	return s.orderRepo.GetOrderRealtimeStatistics(ctx, userId)
}

func (s *statisticsService) GetOperatorOrderCount(ctx context.Context, start, end time.Time) ([]model.OperatorOrderCount, error) {
	return s.orderRepo.GetOperatorOrderCount(ctx, start, end)
}
