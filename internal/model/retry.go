package model

import (
	"time"
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
}

// RetryCondition 重试条件
type RetryCondition struct {
	ErrorCodes []string `json:"error_codes"`
	MaxRetries int      `json:"max_retries"`
	Interval   int      `json:"interval"`
}
