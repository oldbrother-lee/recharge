package model

import "time"

// PhoneBalanceRequest 余额查询请求
type PhoneBalanceRequest struct {
	Phone   string `json:"phone" form:"phone" binding:"required" validate:"required,len=11"`
	ISPType string `json:"isp_type" form:"isp_type" binding:"required" validate:"required,oneof=dx yd lt"`
}

// PhoneBalanceResponse 余额查询响应
type PhoneBalanceResponse struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
	Data    string `json:"datas"` // 余额字符串
}

// PaymentRecordRequest 缴费记录查询请求
type PaymentRecordRequest struct {
	Phone   string `json:"phone" form:"phone" binding:"required" validate:"required,len=11"`
	ISPType string `json:"isp_type" form:"isp_type" binding:"required" validate:"required,oneof=yd lt"` // 仅支持移动和联通
}

// PaymentRecord 缴费记录
type PaymentRecord struct {
	PayTimeStamp int64  `json:"payTimeStamp"`
	PayAmount    string `json:"payAmount"`
	PayTime      string `json:"payTime"`
	Channel      string `json:"channel,omitempty"` // 联通特有字段
}

// PaymentRecordResponse 缴费记录查询响应
type PaymentRecordResponse struct {
	ErrCode int             `json:"errcode"`
	ErrMsg  string          `json:"errmsg"`
	Data    []PaymentRecord `json:"datas,omitempty"` // 移动使用datas
	Records []PaymentRecord `json:"data,omitempty"`  // 联通使用data
}

// GetRecords 统一获取缴费记录
func (p *PaymentRecordResponse) GetRecords() []PaymentRecord {
	if len(p.Data) > 0 {
		return p.Data
	}
	return p.Records
}

// ThirdPartyAPIRequest 第三方API通用请求
type ThirdPartyAPIRequest struct {
	Merch string `form:"merch"`
	Token string `form:"token"`
	Phone string `form:"phone"`
	Type  string `form:"type"`
}

// ThirdPartyAPIResponse 第三方API通用响应
type ThirdPartyAPIResponse struct {
	ErrCode int         `json:"errcode"`
	ErrMsg  string      `json:"errmsg"`
	Data    interface{} `json:"datas,omitempty"`
	Records interface{} `json:"data,omitempty"`
}

// PhoneQueryLog 手机查询日志
type PhoneQueryLog struct {
	ID          int64     `json:"id" gorm:"primaryKey"`
	Phone       string    `json:"phone" gorm:"size:11;not null;index"`
	ISPType     string    `json:"isp_type" gorm:"size:10;not null"`
	QueryType   string    `json:"query_type" gorm:"size:20;not null"` // balance, payment_record
	RequestData string    `json:"request_data" gorm:"type:text"`
	Response    string    `json:"response" gorm:"type:text"`
	Success     bool      `json:"success" gorm:"not null;default:false"`
	ErrorMsg    string    `json:"error_msg" gorm:"size:500"`
	Duration    int64     `json:"duration"` // 请求耗时(毫秒)
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TableName 指定表名
func (PhoneQueryLog) TableName() string {
	return "phone_query_logs"
}