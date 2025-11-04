package model

import "time"

// PlatformTaskConfig 平台任务配置接口
type PlatformTaskConfig interface {
	GetID() int64
	GetPlatformID() int64
	GetPlatformAccountID() int64
	GetProductID() string
	GetStatus() int
	GetCreatedAt() time.Time
	GetUpdatedAt() time.Time
	GetTableName() string
}

// TaskConfigType 任务配置类型
type TaskConfigType string

const (
	TaskConfigTypeXianzhuanxia TaskConfigType = "xianzhuanxia" // 闲赚侠
	TaskConfigTypeDz           TaskConfigType = "dz"           // 得众
)

// UnifiedTaskConfigRequest 统一的任务配置请求
type UnifiedTaskConfigRequest struct {
	ID                *int64         `json:"id"`
	PlatformID        *int64         `json:"platform_id"`
	PlatformAccountID *int64         `json:"platform_account_id"`
	ProductID         *string        `json:"product_id"`
	ConfigType        TaskConfigType `json:"config_type"`
	
	// 闲赚侠平台字段
	ChannelID         *int64  `json:"channel_id,omitempty"`
	FaceValues        *string `json:"face_values,omitempty"`
	MinSettleAmounts  *string `json:"min_settle_amounts,omitempty"`
	Status            *int    `json:"status,omitempty"`
	
	// 得众平台字段
	ISP             *int `json:"isp,omitempty"`
	FaceValue       *int `json:"face_value,omitempty"`
	PollIntervalSec *int `json:"poll_interval_sec,omitempty"`
	Concurrency     *int `json:"concurrency,omitempty"`
	Enabled         *int `json:"enabled,omitempty"`
}

// ToXianzhuanxiaConfig 转换为闲赚侠配置
func (r *UnifiedTaskConfigRequest) ToXianzhuanxiaConfig() *UpdateTaskConfigRequest {
	return &UpdateTaskConfigRequest{
		ID:                r.ID,
		PlatformID:        r.PlatformID,
		PlatformAccountID: r.PlatformAccountID,
		ProductID:         r.ProductID,
		ChannelID:         r.ChannelID,
		FaceValues:        r.FaceValues,
		MinSettleAmounts:  r.MinSettleAmounts,
		Status:            r.Status,
	}
}

// ToDzConfig 转换为得众配置
func (r *UnifiedTaskConfigRequest) ToDzConfig() *UpdateDzTaskConfigRequest {
	return &UpdateDzTaskConfigRequest{
		ID:                r.ID,
		PlatformID:        r.PlatformID,
		PlatformAccountID: r.PlatformAccountID,
		ProductID:         r.ProductID,
		ISP:               r.ISP,
		FaceValue:         r.FaceValue,
		PollIntervalSec:   r.PollIntervalSec,
		Concurrency:       r.Concurrency,
		Enabled:           r.Enabled,
	}
}