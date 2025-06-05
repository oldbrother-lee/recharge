package model

import (
	"time"
)

// SystemConfig 系统配置
type SystemConfig struct {
	ID          int64     `json:"id" gorm:"primaryKey;type:bigint;not null"`
	ConfigKey   string    `json:"config_key" gorm:"size:100;not null;unique;comment:配置键"`
	ConfigValue string    `json:"config_value" gorm:"type:text;comment:配置值"`
	ConfigDesc  string    `json:"config_desc" gorm:"size:255;comment:配置描述"`
	ConfigType  string    `json:"config_type" gorm:"size:50;default:string;comment:配置类型(string,number,boolean,json)"`
	IsSystem    int       `json:"is_system" gorm:"type:tinyint;default:0;comment:是否系统配置(0:否,1:是)"`
	Status      int       `json:"status" gorm:"type:tinyint;default:1;comment:状态(0:禁用,1:启用)"`
	CreatedAt   time.Time `json:"created_at" gorm:"type:datetime;autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"type:datetime;autoUpdateTime"`
}

// TableName 指定表名
func (SystemConfig) TableName() string {
	return "system_configs"
}

// SystemConfigRequest 系统配置请求结构
type SystemConfigRequest struct {
	ConfigKey   string `json:"config_key" binding:"required" validate:"required,max=100"`
	ConfigValue string `json:"config_value" binding:"required"`
	ConfigDesc  string `json:"config_desc" validate:"max=255"`
	ConfigType  string `json:"config_type" validate:"oneof=string number boolean json"`
}

// SystemConfigResponse 系统配置响应结构
type SystemConfigResponse struct {
	ID          int64     `json:"id"`
	ConfigKey   string    `json:"config_key"`
	ConfigValue string    `json:"config_value"`
	ConfigDesc  string    `json:"config_desc"`
	ConfigType  string    `json:"config_type"`
	IsSystem    int       `json:"is_system"`
	Status      int       `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
