package model

import "time"

// TaskOrder 取单订单记录表
// 用于记录自动取单获取到的订单信息
type TaskOrder struct {
	ID                     int64     `gorm:"primaryKey"`                            // 主键ID
	OrderNumber            string    `gorm:"type:varchar(32);not null;uniqueIndex"` // 订单号
	ChannelID              int       `gorm:"not null;index"`                        // 渠道ID
	ChannelName            string    `gorm:"type:varchar(50)"`                      // 渠道名称
	ProductID              string    `gorm:"not null;index"`                        // 运营商ID
	ProductName            string    `gorm:"type:varchar(50)"`                      // 运营商名称
	FaceValue              float64   `gorm:"type:decimal(10,2);not null"`           // 面值
	AccountNum             string    `gorm:"type:varchar(20);not null"`             // 充值账号
	AccountLocation        string    `gorm:"type:varchar(50)"`                      // 归属地
	SettlementAmount       float64   `gorm:"type:decimal(10,2);not null"`           // 结算金额
	OrderStatus            int       `gorm:"not null;index"`                        // 订单状态
	SettlementStatus       int       `gorm:"not null;index"`                        // 结算状态
	CreateTime             int64     `gorm:"not null"`                              // 创建时间(毫秒时间戳)
	ExpirationTime         int64     `gorm:"not null"`                              // 过期时间(毫秒时间戳)
	SettlementTime         int64     `gorm:""`                                      // 结算时间(毫秒时间戳)
	ExpectedSettlementTime int64     `gorm:""`                                      // 预计结算时间(毫秒时间戳)
	CreatedAt              time.Time // 创建时间
	UpdatedAt              time.Time // 更新时间
}
