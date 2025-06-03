package model

import (
	"time"
)

const (
	RechargeTaskStatusPending    = 0 // 待处理
	RechargeTaskStatusProcessing = 1 // 处理中
	RechargeTaskStatusSuccess    = 2 // 成功
	RechargeTaskStatusFailed     = 3 // 失败
)

// RechargeTask 充值任务
type RechargeTask struct {
	ID          int64     `gorm:"primaryKey"`
	OrderID     int64     `gorm:"index"`     // 关联订单ID
	PlatformID  int64     `gorm:"index"`     // 充值平台ID
	Status      int       `gorm:"default:0"` // 状态：0待处理 1处理中 2成功 3失败
	RetryTimes  int       `gorm:"default:0"` // 重试次数
	MaxRetries  int       `gorm:"default:3"` // 最大重试次数
	NextRetryAt time.Time `gorm:"index"`     // 下次重试时间
	Result      string    `gorm:"type:text"` // 充值结果
	ErrorMsg    string    `gorm:"type:text"` // 错误信息
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
