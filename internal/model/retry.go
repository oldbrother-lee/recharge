package model

import (
	"time"
)

// 重试类型常量
const (
	RetryTypeNormal          = 0 // 普通重试
	RetryTypeOrderFail       = 1 // 订单失败重试
	RetryTypeExternalCallback = 2 // 外部回调触发的跨通道重试/切换通道
	// 新增：同通道重试（不切换通道，保持当前API/通道）
	RetryTypeSameChannel     = 3 // 同通道重试
)

// OrderRetryRecord 订单重试记录
type OrderRetryRecord struct {
	ID            int64     `json:"id" gorm:"primaryKey"`
	OrderID       int64     `json:"order_id" gorm:"not null"`
	APIID         int64     `json:"api_id" gorm:"not null"`
	ParamID       int64     `json:"param_id" gorm:"not null"`
	RetryType     int       `json:"retry_type" gorm:"type:tinyint;not null"`
	RetryCount    int       `json:"retry_count" gorm:"not null;default:0"`
	LastError     string    `json:"last_error" gorm:"type:text"`
	RetryParams   string    `json:"retry_params" gorm:"type:json"`
	UsedAPIs      string    `json:"used_apis" gorm:"type:json"`
	Status        int       `json:"status" gorm:"type:tinyint;not null;default:0"`
	NextRetryTime time.Time `json:"next_retry_time"`
	CreatedAt     time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt     time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	// 以下为同通道重试相关的扩展字段
	ChannelCode       string     `json:"channel_code" gorm:"size:64;comment:通道编码(平台/渠道代码)"`
	AttemptNo         int        `json:"attempt_no" gorm:"not null;default:0;comment:当前尝试序号(从1开始)"`
	LastAttemptAt     *time.Time `json:"last_attempt_at" gorm:"comment:上次尝试时间"`
	NextAttemptAt     *time.Time `json:"next_attempt_at" gorm:"comment:下次尝试时间(调度用)"`
	BackoffMs         int        `json:"backoff_ms" gorm:"not null;default:0;comment:退避毫秒数"`
	ActiveOutTradeNum string     `json:"active_out_trade_num" gorm:"size:64;index;comment:当前生效的上游下单号(同通道重试用)"`
}

// RetryCondition 重试条件
type RetryCondition struct {
	ErrorCodes []string `json:"error_codes"`
	MaxRetries int      `json:"max_retries"`
	Interval   int      `json:"interval"`
}
