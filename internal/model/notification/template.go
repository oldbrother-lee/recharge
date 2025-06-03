package notification

import (
	"time"
)

// Template 通知模板
type Template struct {
	ID               int64     `json:"id" gorm:"primaryKey"`
	PlatformCode     string    `json:"platform_code" gorm:"type:varchar(50);index:uk_platform_type,priority:1"`
	NotificationType string    `json:"notification_type" gorm:"type:varchar(50);index:uk_platform_type,priority:2"`
	Template         string    `json:"template" gorm:"type:text"`
	Status           int       `json:"status"` // 1-启用 2-禁用
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// TableName 指定表名
func (Template) TableName() string {
	return "notification_templates"
}
