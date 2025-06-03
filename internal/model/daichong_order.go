package model

// DaichongOrder 代充订单模型
// 对应表：dyr_daichong_orders

type DaichongOrder struct {
	ID           int64       `gorm:"primaryKey;autoIncrement" json:"id"`                    // 主键，自增ID
	User         string      `gorm:"type:varchar(255)" json:"user"`                         // 用户ID，发起代充的用户标识
	OrderID      string      `gorm:"type:varchar(35);not null" json:"order_id"`             // 代付订单号，第三方平台订单号
	Account      string      `gorm:"type:varchar(35);not null" json:"account"`              // 充值账号，手机号或账号
	Denom        float64     `gorm:"type:decimal(10,2);not null" json:"denom"`              // 面值
	SettlePrice  float64     `gorm:"type:decimal(10,2);not null" json:"settlePrice"`        // 结算价
	CreateTime   int64       `gorm:"not null" json:"createTime"`                            // 订单创建时间
	ChargeTime   int64       `json:"chargeTime"`                                            // 接单时间
	UploadTime   int64       `json:"uploadTime"`                                            // 上报充值时间
	Status       OrderStatus `gorm:"not null" json:"status"`                                // 订单状态：5->充值中; 6->核实中; 7->申请人工介入; 8->待确认;9->弃单;
	SettleStatus int         `gorm:"not null" json:"settleStatus"`                          // 结算状态：0->未结算; 1->已结算;3->结算中;
	YrOrderID    string      `gorm:"type:varchar(120)" json:"yr_order_id"`                  // 系统订单号，内部唯一标识
	IsPost       int         `gorm:"not null;default:0" json:"is_post"`                     // 是否已推送到第三方，0-未推送，1-已推送
	Base         string      `gorm:"type:longtext" json:"base"`                             // 凭证图，充值凭证图片（base64或URL）
	Type         int         `gorm:"not null;default:0" json:"type"`                        // 类型，业务自定义字段
	SbType       int         `gorm:"not null;default:0" json:"sb_type"`                     // sb类型，业务自定义字段
	Yunying      string      `gorm:"type:varchar(255);not null;default:'1'" json:"yunying"` // 运营商，1-移动 2-联通 3-电信
	Beizhu       string      `gorm:"type:varchar(225)" json:"beizhu"`                       // 备注，订单备注信息
	Prov         string      `gorm:"type:varchar(255)" json:"prov"`                         // 省份，充值账号归属地
	Way          int         `gorm:"default:1" json:"way"`                                  // 订单来源模式：1-取单 2-推单
	PlatformID   int64       `gorm:"not null" json:"platform_id"`                           // 平台ID
	PlatformName string      `gorm:"type:varchar(255)" json:"platform_name"`                // 平台名称
	PlatformCode string      `gorm:"type:varchar(255)" json:"platform_code"`                // 平台代码
}

func (DaichongOrder) TableName() string {
	return "daichong_orders"
}
