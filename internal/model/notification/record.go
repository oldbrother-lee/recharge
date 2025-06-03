package notification

import (
	"time"
)

// NotificationRecord 通知记录
type NotificationRecord struct {
	ID               int64     `json:"id" gorm:"primaryKey"`
	OrderID          int64     `json:"order_id" gorm:"index"`
	PlatformCode     string    `json:"platform_code" gorm:"type:varchar(50)"`
	NotificationType string    `json:"notification_type" gorm:"type:varchar(50)"`
	Content          string    `json:"content" gorm:"type:text"`
	Status           int       `json:"status" gorm:"type:tinyint;default:1"` // 1:待处理 2:处理中 3:成功 4:失败
	RetryCount       int       `json:"retry_count" gorm:"type:int;default:0"`
	NextRetryTime    time.Time `json:"next_retry_time"`
	SuccessAt        time.Time `json:"success_at" gorm:"type:datetime"` // 通知成功时间
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// TableName 指定表名
func (NotificationRecord) TableName() string {
	return "notification_records"
}
