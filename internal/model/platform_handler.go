package model

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// PlatformHandler 平台处理器
type PlatformHandler struct {
	ID            int64         `gorm:"primaryKey;autoIncrement" json:"id"`
	PlatformName  string        `gorm:"column:platform_name;type:varchar(50);not null;comment:平台名称" json:"platform_name"`
	HandlerType   string        `gorm:"column:handler_type;type:varchar(50);not null;comment:处理器类型" json:"handler_type"`
	HandlerConfig HandlerConfig `gorm:"column:handler_config;type:json;comment:处理器配置" json:"handler_config"`
	Status        int8          `gorm:"column:status;type:tinyint;default:1;comment:状态：1-启用，0-禁用" json:"status"`
	CreateTime    time.Time     `gorm:"column:create_time;type:datetime;default:CURRENT_TIMESTAMP" json:"create_time"`
	UpdateTime    time.Time     `gorm:"column:update_time;type:datetime;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"update_time"`
}

// TableName 表名
func (PlatformHandler) TableName() string {
	return "platform_handlers"
}

// HandlerConfig 处理器配置
type HandlerConfig struct {
	// 基础配置
	AppKey    string `json:"app_key"`
	AppSecret string `json:"app_secret"`
	ApiURL    string `json:"api_url"`
	NotifyURL string `json:"notify_url"`

	// 平台特定配置
	ExtraConfig map[string]interface{} `json:"extra_config"`
}

// Value 实现 driver.Valuer 接口
func (c HandlerConfig) Value() (driver.Value, error) {
	return json.Marshal(c)
}

// Scan 实现 sql.Scanner 接口
func (c *HandlerConfig) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, c)
}
