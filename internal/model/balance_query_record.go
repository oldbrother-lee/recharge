package model

import (
	"time"
	"gorm.io/gorm"
)

// BalanceQueryRecord 余额查询记录表
// 用于存储充值前后的余额查询结果，与订单表关联
type BalanceQueryRecord struct {
	ID          int64          `json:"id" gorm:"primaryKey"`
	OrderID     int64          `json:"order_id" gorm:"index;not null;comment:关联订单ID"`
	OrderNumber string         `json:"order_number" gorm:"size:32;index;comment:订单号"`
	Mobile      string         `json:"mobile" gorm:"size:20;not null;comment:手机号"`
	ISPType     string         `json:"isp_type" gorm:"size:10;not null;comment:运营商类型(yd/dx/lt)"`
	QueryType   string         `json:"query_type" gorm:"size:10;not null;default:'before';comment:查询类型(before:充值前,after:充值后)"`
	Balance     string         `json:"balance" gorm:"size:50;comment:查询到的余额"`
	QueryTime   time.Time      `json:"query_time" gorm:"not null;comment:查询时间"`
	Success     bool           `json:"success" gorm:"not null;default:false;comment:查询是否成功"`
	ErrorMsg    string         `json:"error_msg" gorm:"size:500;comment:错误信息"`
	RetryCount  int            `json:"retry_count" gorm:"default:0;comment:重试次数"`
	Duration    int64          `json:"duration" gorm:"comment:查询耗时(毫秒)"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	// 关联订单
	Order *Order `json:"order,omitempty" gorm:"foreignKey:OrderID"`
}

// TableName 指定表名
func (BalanceQueryRecord) TableName() string {
	return "balance_query_records"
}