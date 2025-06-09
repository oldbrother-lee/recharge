package model

import (
	"encoding/json"
	"strconv"
	"time"
)

// Int64String 支持字符串和数字类型的互转
type Int64String int64

// UnmarshalJSON 实现自定义 JSON 解析
func (i *Int64String) UnmarshalJSON(data []byte) error {
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case float64:
		*i = Int64String(value)
	case string:
		// 如果是字符串，尝试转换为 int64
		if value == "" {
			*i = 0
			return nil
		}
		// 先尝试解析为 float64，再转换为 int64
		floatVal, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		*i = Int64String(floatVal)
	default:
		*i = 0
	}
	return nil
}

// MarshalJSON 实现自定义 JSON 序列化
func (i Int64String) MarshalJSON() ([]byte, error) {
	return json.Marshal(int64(i))
}

// TaskConfig 任务配置
type TaskConfig struct {
	ID                int64       `json:"id" gorm:"primaryKey"`
	PlatformID        int64       `json:"platform_id" gorm:"not null;comment:平台ID"`
	PlatformName      string      `json:"platform_name" gorm:"not null;comment:平台名称"`
	PlatformAccountID int64       `json:"platform_account_id" gorm:"not null;comment:平台账号ID"`
	PlatformAccount   string      `json:"platform_account" gorm:"not null;comment:平台账号"`
	ChannelID         int64       `json:"channel_id" gorm:"not null;comment:渠道ID"`
	ChannelName       string      `json:"channel_name" gorm:"not null;comment:渠道名称"`
	ProductID         string      `json:"product_id" gorm:"type:varchar(64);not null;comment:产品ID"`
	ProductName       string      `json:"product_name" gorm:"not null;comment:产品名称"`
	FaceValues        string      `json:"face_values" gorm:"type:text;not null;comment:面值列表"`
	MinSettleAmounts  string      `json:"min_settle_amounts" gorm:"type:text;not null;comment:最低结算价列表"`
	Provinces         string      `json:"provinces" gorm:"type:text;not null;comment:省份列表"`
	Status            int         `json:"status" gorm:"not null;default:1;comment:状态 1:启用 2:禁用"`
	OfficialPayment   Int64String `json:"official_payment" gorm:"not null;comment:官方支付金额"`
	UserQuoteType     int         `json:"user_quote_type" gorm:"not null;comment:用户报价类型"`
	CreatedAt         time.Time   `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt         time.Time   `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName 表名
func (TaskConfig) TableName() string {
	return "task_configs"
}

type UpdateTaskConfigRequest struct {
	ID                *int64       `json:"id"`
	ChannelID         *int64       `json:"channel_id"`
	ProductID         *string      `json:"product_id"`
	PlatformID        *int64       `json:"platform_id"`
	PlatformAccountID *int64       `json:"platform_account_id"`
	FaceValues        *string      `json:"face_values"`
	MinSettleAmounts  *string      `json:"min_settle_amounts"`
	Status            *int         `json:"status"`
	OfficialPayment   *Int64String `json:"official_payment"`
	UserQuoteType     *int         `json:"user_quote_type"`
}
