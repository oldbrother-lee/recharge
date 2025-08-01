package model

import (
	"encoding/json"
	"time"
)

// OrderException 订单异常记录
type OrderException struct {
	ID              int64           `json:"id" gorm:"primaryKey;autoIncrement"`
	OrderID         int64           `json:"order_id" gorm:"not null;index"`
	OrderNumber     string          `json:"order_number" gorm:"size:255;not null;index"`
	ExceptionType   string          `json:"exception_type" gorm:"size:50;not null;index"`
	ExceptionReason string          `json:"exception_reason" gorm:"type:text"`
	ExceptionData   json.RawMessage `json:"exception_data" gorm:"type:json"`
	Status          string          `json:"status" gorm:"size:20;not null;default:pending;index"`
	ResolvedBy      *int64          `json:"resolved_by"`
	ResolvedAt      *time.Time      `json:"resolved_at"`
	ResolvedNote    string          `json:"resolved_note" gorm:"type:text"`
	CreatedAt       time.Time       `json:"created_at" gorm:"autoCreateTime;index"`
	UpdatedAt       time.Time       `json:"updated_at" gorm:"autoUpdateTime"`

	// 关联
	Order *Order `json:"order,omitempty" gorm:"foreignKey:OrderID"`
}

// TableName 指定表名
func (OrderException) TableName() string {
	return "order_exceptions"
}

// 异常类型常量
const (
	ExceptionTypeBalanceVerificationFailed = "balance_verification_failed"
)

// 异常状态常量
const (
	ExceptionStatusPending    = "pending"    // 待处理
	ExceptionStatusProcessing = "processing" // 处理中
	ExceptionStatusResolved   = "resolved"   // 已解决
	ExceptionStatusIgnored    = "ignored"    // 已忽略
)

// BalanceVerificationExceptionData 余额验证异常数据
type BalanceVerificationExceptionData struct {
	PreBalance    string  `json:"pre_balance"`     // 充值前余额
	PostBalance   string  `json:"post_balance"`    // 充值后余额
	ExpectedDiff  float64 `json:"expected_diff"`   // 预期差额
	ActualDiff    float64 `json:"actual_diff"`     // 实际差额
	Mobile        string  `json:"mobile"`          // 手机号
	ISPType       string  `json:"isp_type"`        // 运营商类型
	PlatformCode  string  `json:"platform_code"`   // 平台代码
	Amount        float64 `json:"amount"`          // 充值金额
	QueryDuration int64   `json:"query_duration"`  // 查询耗时(毫秒)
}

// CreateOrderExceptionRequest 创建订单异常请求
type CreateOrderExceptionRequest struct {
	OrderID         int64           `json:"order_id" binding:"required"`
	OrderNumber     string          `json:"order_number" binding:"required"`
	ExceptionType   string          `json:"exception_type" binding:"required"`
	ExceptionReason string          `json:"exception_reason"`
	ExceptionData   json.RawMessage `json:"exception_data"`
}

// UpdateOrderExceptionRequest 更新订单异常请求
type UpdateOrderExceptionRequest struct {
	Status       string `json:"status" binding:"required,oneof=pending processing resolved ignored"`
	ResolvedNote string `json:"resolved_note"`
}

// OrderExceptionListRequest 订单异常列表请求
type OrderExceptionListRequest struct {
	Page          int    `json:"page" form:"page"`
	PageSize      int    `json:"page_size" form:"page_size"`
	OrderNumber   string `json:"order_number" form:"order_number"`
	ExceptionType string `json:"exception_type" form:"exception_type"`
	Status        string `json:"status" form:"status"`
	StartDate     string `json:"start_date" form:"start_date"`
	EndDate       string `json:"end_date" form:"end_date"`
}

// OrderExceptionListResponse 订单异常列表响应
type OrderExceptionListResponse struct {
	List       []OrderException `json:"list"`
	Total      int64            `json:"total"`
	Page       int              `json:"page"`
	PageSize   int              `json:"page_size"`
	TotalPages int              `json:"total_pages"`
}