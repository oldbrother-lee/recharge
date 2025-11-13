package model

import "time"

// PullTaskConfig 统一拉单配置（支持不同平台的拉单参数）
type PullTaskConfig struct {
	ID                int64  `json:"id" gorm:"primaryKey"`
	PlatformID        int64  `json:"platform_id" gorm:"not null;index"`
	PlatformAccountID int64  `json:"platform_account_id" gorm:"not null;index"`
	PlatformCode      string `json:"platform_code" gorm:"size:32;not null;index"` // dz | zhangyu | ...
	Enabled           bool   `json:"enabled" gorm:"not null;default:true"`
	PollIntervalSec   int    `json:"poll_interval_sec" gorm:"not null;default:10"`
	Concurrency       int    `json:"concurrency" gorm:"not null;default:1"`

	// 章鱼专用
	Flag      string  `json:"flag" gorm:"size:64"`
	Province  string  `json:"province" gorm:"size:64"`
	MinAmount float64 `json:"min_amount" gorm:"type:decimal(10,2);default:0"`
	MaxAmount float64 `json:"max_amount" gorm:"type:decimal(10,2);default:0"`

	// 得众等平台可放到扩展字段（JSON）或后续专用列
	AdapterParams string `json:"adapter_params" gorm:"type:text"`

	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

func (PullTaskConfig) TableName() string { return "pull_task_configs" }