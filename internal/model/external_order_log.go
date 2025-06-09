package model

import "time"

type ExternalOrderLog struct {
	ID               int64     `json:"id" gorm:"primaryKey"`
	Platform         string    `json:"platform" gorm:"size:32;comment:平台"`
	OrderID          string    `json:"order_id" gorm:"size:64;comment:外部订单号"`
	Mobile           string    `json:"mobile" gorm:"size:20;comment:手机号"`
	OuterGoodsCode   string    `json:"outer_goods_code" gorm:"size:64;comment:外部产品编码"`
	BizType          string    `json:"biz_type" gorm:"size:32;comment:业务类型"`
	Amount           float64   `json:"amount" gorm:"type:decimal(10,2);comment:金额"`
	RawData          string    `json:"raw_data" gorm:"type:text;comment:原始请求数据"`
	Status           int       `json:"status" gorm:"default:0;comment:处理状态：0-待处理，1-处理成功，2-处理失败"`
	ErrorMsg         string    `json:"error_msg" gorm:"size:255;comment:错误信息"`
	AppKey           string    `json:"app_key" gorm:"size:64;comment:应用密钥"`
	VenderID         int       `json:"vender_id" gorm:"comment:供应商ID"`
	GoodsID          int64     `json:"goods_id" gorm:"comment:商品ID"`
	GoodsName        string    `json:"goods_name" gorm:"size:100;comment:商品名称"`
	OfficialPayment  float64   `json:"official_payment" gorm:"type:decimal(10,2);comment:官方支付金额"`
	UserQuoteType    int       `json:"user_quote_type" gorm:"comment:用户报价类型"`
	UserQuotePayment int       `json:"user_quote_payment" gorm:"comment:用户报价支付金额"`
	UserPayment      float64   `json:"user_payment" gorm:"type:decimal(10,2);comment:用户支付金额"`
	ProcessTime      int       `json:"process_time" gorm:"comment:处理时间(毫秒)"`
	Timestamp        int64     `json:"timestamp" gorm:"comment:时间戳"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}
