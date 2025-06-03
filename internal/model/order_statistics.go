package model

import (
	"time"

	"gorm.io/gorm"
)

// OrderStatistics 订单统计模型
type OrderStatistics struct {
	ID            uint64         `gorm:"primarykey" json:"id"`
	Date          time.Time      `gorm:"type:date;not null;index:idx_date_operator" json:"date"`            // 统计日期
	Operator      string         `gorm:"type:varchar(20);not null;index:idx_date_operator" json:"operator"` // 运营商
	TotalOrders   int64          `gorm:"not null;default:0" json:"totalOrders"`                             // 总订单数
	SuccessOrders int64          `gorm:"not null;default:0" json:"successOrders"`                           // 成功订单数
	FailedOrders  int64          `gorm:"not null;default:0" json:"failedOrders"`                            // 失败订单数
	CostAmount    float64        `gorm:"type:decimal(10,2);not null" json:"costAmount"`                     // 成本金额
	ProfitAmount  float64        `gorm:"type:decimal(10,2);not null" json:"profitAmount"`                   // 盈利金额
	CreatedAt     time.Time      `json:"createdAt"`                                                         // 创建时间
	UpdatedAt     time.Time      `json:"updatedAt"`                                                         // 更新时间
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`                                                    // 删除时间
}

// TableName 指定表名
func (OrderStatistics) TableName() string {
	return "order_statistics"
}

// OrderStatusStat 订单状态统计
type OrderStatusStat struct {
	Processing int64 `json:"processing"` // 充值中订单数
	Success    int64 `json:"success"`    // 成功订单数
	Failed     int64 `json:"failed"`     // 失败订单数

}

// OrderStatisticsOverview 订单统计概览
type OrderStatisticsOverview struct {
	Total struct {
		Total     int64 `json:"total"`     // 总订单数
		Yesterday int64 `json:"yesterday"` // 昨日订单数
		Today     int64 `json:"today"`     // 今日订单数
	} `json:"total"`

	Status          OrderStatusStat `json:"status"`
	YesterdayStatus struct {
		YesterdayProcessing int64 `json:"yesterday_processing"` // 昨日充值中订单数
		YesterdaySuccess    int64 `json:"yesterday_success"`    // 昨日成功订单数
		YesterdayFailed     int64 `json:"yesterday_failed"`     // 昨日失败订单数
	} `json:"yesterday_status"`

	Profit struct {
		CostAmount   float64 `json:"costAmount"`   // 成本价格
		ProfitAmount float64 `json:"profitAmount"` // 盈利价格
	} `json:"profit"`
}

// OrderStatisticsOperator 运营商订单统计
type OrderStatisticsOperator struct {
	Isp         int   `json:"isp"`
	TotalOrders int64 `json:"totalOrders"`
}

// OrderStatisticsDaily 每日订单统计
type OrderStatisticsDaily struct {
	Date          time.Time `json:"date"`          // 日期
	TotalOrders   int64     `json:"totalOrders"`   // 总订单数
	SuccessOrders int64     `json:"successOrders"` // 成功订单数
	FailedOrders  int64     `json:"failedOrders"`  // 失败订单数
	SuccessRate   float64   `json:"successRate"`   // 成功率
	CostAmount    float64   `json:"costAmount"`    // 成本金额
	ProfitAmount  float64   `json:"profitAmount"`  // 盈利金额
}

// OrderStatisticsTrend 订单趋势
type OrderStatisticsTrend struct {
	Date          time.Time `json:"date"`          // 日期
	TotalOrders   int64     `json:"totalOrders"`   // 总订单数
	SuccessOrders int64     `json:"successOrders"` // 成功订单数
	FailedOrders  int64     `json:"failedOrders"`  // 失败订单数
	SuccessRate   float64   `json:"successRate"`   // 成功率
	CostAmount    float64   `json:"costAmount"`    // 成本金额
	ProfitAmount  float64   `json:"profitAmount"`  // 盈利金额
}

// 按运营商分组统计订单总数结构体
type OperatorOrderCount struct {
	Operator int   `json:"operator"`
	Total    int64 `json:"total"`
}
