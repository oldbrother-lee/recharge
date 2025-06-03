package model

import "time"

// CallbackLog 回调日志
type CallbackLog struct {
	ID           int64     `gorm:"primaryKey"`
	OrderID      string    `gorm:"index"`     // 订单号
	PlatformID   string    `gorm:"index"`     // 平台ID
	CallbackType string    `gorm:"index"`     // 回调类型
	Status       int       `gorm:"index"`     // 处理状态
	RequestData  string    `gorm:"type:text"` // 请求数据
	ResponseData string    `gorm:"type:text"` // 响应数据
	ErrorMessage string    `gorm:"type:text"` // 错误信息
	CreateTime   time.Time // 创建时间
	UpdateTime   time.Time // 更新时间
}

// TableName 指定表名
func (CallbackLog) TableName() string {
	return "callback_logs"
}
