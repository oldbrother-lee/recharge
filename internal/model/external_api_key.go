package model

import (
	"time"

	"gorm.io/gorm"
)

// ExternalAPIKey 外部API密钥模型
type ExternalAPIKey struct {
	ID                int64          `json:"id" gorm:"primaryKey"`
	UserID            int64          `json:"user_id" gorm:"index;comment:用户ID"`
	PlatformAccountID int64          `json:"platform_account_id" gorm:"index;comment:平台账号ID"`
	AppID             string         `json:"app_id" gorm:"size:64;uniqueIndex;comment:应用ID"`
	AppKey            string         `json:"app_key" gorm:"size:128;comment:应用密钥"`
	AppSecret         string         `json:"app_secret" gorm:"size:256;comment:应用秘钥"`
	AppName           string         `json:"app_name" gorm:"size:128;comment:应用名称"`
	Description       string         `json:"description" gorm:"size:255;comment:应用描述"`
	Status            int            `json:"status" gorm:"default:1;comment:状态 1:启用 0:禁用"`
	IPWhitelist       string         `json:"ip_whitelist" gorm:"type:text;comment:IP白名单,逗号分隔"`
	NotifyURL         string         `json:"notify_url" gorm:"size:512;comment:回调通知URL"`
	RateLimit         int            `json:"rate_limit" gorm:"default:1000;comment:每分钟请求限制"`
	ExpireTime        *time.Time     `json:"expire_time" gorm:"comment:过期时间"`
	CreatedAt         time.Time      `json:"created_at" gorm:"comment:创建时间"`
	UpdatedAt         time.Time      `json:"updated_at" gorm:"comment:更新时间"`
	DeletedAt         gorm.DeletedAt `json:"deleted_at" gorm:"index;comment:删除时间"`
}

// TableName 表名
func (ExternalAPIKey) TableName() string {
	return "external_api_keys"
}

// IsActive 检查API密钥是否有效
func (e *ExternalAPIKey) IsActive() bool {
	if e.Status != 1 {
		return false
	}
	if e.ExpireTime != nil && e.ExpireTime.Before(time.Now()) {
		return false
	}
	return true
}

// IsIPAllowed 检查IP是否在白名单中
func (e *ExternalAPIKey) IsIPAllowed(ip string) bool {
	if e.IPWhitelist == "" {
		return true // 没有设置白名单，允许所有IP
	}
	// TODO: 实现IP白名单检查逻辑
	return true
}
