package model

import (
	"time"
)

// DzTaskConfig 得众平台任务配置
type DzTaskConfig struct {
    ID                int64     `json:"id" gorm:"primaryKey"`
    PlatformID        int64     `json:"platform_id" gorm:"not null;comment:平台ID"`
    PlatformName      string    `json:"platform_name" gorm:"not null;comment:平台名称"`
    PlatformAccountID int64     `json:"platform_account_id" gorm:"not null;comment:平台账号ID"`
    PlatformAccount   string    `json:"platform_account" gorm:"not null;comment:平台账号"`
    ProductID         string    `json:"product_id" gorm:"type:varchar(64);not null;comment:产品ID"`
    ProductName       string    `json:"product_name" gorm:"not null;comment:产品名称"`
    ISP               int       `json:"isp" gorm:"not null;comment:运营商 1:移动 2:电信 3:联通"`
    FaceValue         int       `json:"face_value" gorm:"not null;comment:面值"`
    PollIntervalSec   int       `json:"poll_interval_sec" gorm:"not null;default:30;comment:轮询间隔秒数"`
    Concurrency       int       `json:"concurrency" gorm:"not null;default:1;comment:并发度"`
    Enabled           int       `json:"enabled" gorm:"not null;default:1;comment:状态 1:启用 0:禁用"`
    CreatedAt         time.Time `json:"created_at" gorm:"autoCreateTime"`
    UpdatedAt         time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName 表名
func (DzTaskConfig) TableName() string {
	return "dz_task_configs"
}

// UpdateDzTaskConfigRequest 更新得众任务配置请求
type UpdateDzTaskConfigRequest struct {
    ID                *int64  `json:"id"`
    ProductID         *string `json:"product_id"`
    PlatformID        *int64  `json:"platform_id"`
    PlatformAccountID *int64  `json:"platform_account_id"`
    ISP               *int    `json:"isp"`
    FaceValue         *int    `json:"face_value"`
    PollIntervalSec   *int    `json:"poll_interval_sec"`
    Concurrency       *int    `json:"concurrency"`
    Enabled           *int    `json:"enabled"`
}