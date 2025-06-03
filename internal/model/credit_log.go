package model

import (
	"time"
)

// CreditLog 授信额度变更日志
type CreditLog struct {
	ID           int64     `json:"id" gorm:"primaryKey;type:bigint;not null"`
	UserID       int64     `json:"user_id" gorm:"type:bigint;not null;index"`
	Amount       float64   `json:"amount" gorm:"type:decimal(10,2);default:0.00;comment:变更金额"`
	Type         int       `json:"type" gorm:"type:tinyint;default:1;comment:变更类型(1:设置 2:使用 3:恢复)"`
	CreditBefore float64   `json:"credit_before" gorm:"type:decimal(10,2);default:0.00;comment:变更前额度"`
	CreditAfter  float64   `json:"credit_after" gorm:"type:decimal(10,2);default:0.00;comment:变更后额度"`
	OrderID      int64     `json:"order_id" gorm:"type:bigint;comment:关联订单ID"`
	Remark       string    `json:"remark" gorm:"size:255;comment:备注"`
	Operator     string    `json:"operator" gorm:"size:50;comment:操作人"`
	CreatedAt    time.Time `json:"created_at" gorm:"type:datetime;autoCreateTime"`
}

// TableName 指定表名
func (CreditLog) TableName() string {
	return "credit_logs"
}

// CreditLogRequest 创建授信日志请求
type CreditLogRequest struct {
	UserID   int64   `json:"user_id" binding:"required"`
	Amount   float64 `json:"creditLimit" binding:"required"`
	Type     int     `json:"type"`
	OrderID  int64   `json:"order_id"`
	Remark   string  `json:"remark"`
	Operator string  `json:"operator"`
}

// CreditLogResponse 授信日志响应
type CreditLogResponse struct {
	ID           int64     `json:"id"`
	UserID       int64     `json:"user_id"`
	Amount       float64   `json:"amount"`
	Type         int       `json:"type"`
	CreditBefore float64   `json:"credit_before"`
	CreditAfter  float64   `json:"credit_after"`
	OrderID      int64     `json:"order_id"`
	Remark       string    `json:"remark"`
	Operator     string    `json:"operator"`
	CreatedAt    time.Time `json:"created_at"`
}

// CreditLogListRequest 授信日志列表请求
type CreditLogListRequest struct {
	UserID  int64 `form:"user_id"`
	Type    int   `form:"type"`
	OrderID int64 `form:"order_id"`
	Current int   `form:"current" binding:"min=1"`
	Size    int   `form:"size" binding:"min=1,max=100"`
}

// CreditLogListResponse 授信日志列表响应
type CreditLogListResponse struct {
	List  []CreditLogResponse `json:"list"`
	Total int64               `json:"total"`
}

// CreditType 授信变更类型
const (
	CreditTypeSet     = 1 // 设置授信额度
	CreditTypeUse     = 2 // 使用授信额度
	CreditTypeRestore = 3 // 恢复授信额度
)
