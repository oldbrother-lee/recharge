package model

import "time"

// PlatformToken 平台 token 模型
// 用于存储第三方平台 token 及其生成时间
// 只保留一条记录即可

type PlatformToken struct {
	ID           int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	TaskConfigID int64     `gorm:"type:bigint;not null;index" json:"task_config_id"`        // 任务配置ID
	Token        string    `gorm:"type:varchar(255);not null" json:"token"`                 // token 字符串
	CreatedAt    time.Time `gorm:"type:datetime;not null;autoCreateTime" json:"created_at"` // token 获取时间
	LastUsedAt   time.Time `gorm:"type:datetime;not null" json:"last_used_at"`              // 最后使用时间
}

func (PlatformToken) TableName() string {
	return "platform_token"
}
