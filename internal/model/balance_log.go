package model

import (
	"time"
)

// BalanceLog 余额变动日志
type BalanceLog struct {
	ID                int64     `json:"id" gorm:"primaryKey"`
	UserID            int64     `json:"user_id" gorm:"not null;index"`                     // 用户ID
	OrderID           int64     `json:"order_id" gorm:"index"`                             // 关联订单ID
	PlatformAccountID int64     `json:"platform_account_id" gorm:"index"`                  // 平台账号ID
	PlatformID        int64     `json:"platform_id" gorm:"index"`                          // 平台ID
	PlatformCode      string    `json:"platform_code" gorm:"size:20;index"`                // 平台代码
	PlatformName      string    `json:"platform_name" gorm:"size:50"`                      // 平台名称
	Amount            float64   `json:"amount" gorm:"type:decimal(10,2);not null"`         // 变动金额
	Type              int       `json:"type" gorm:"type:tinyint;not null"`                 // 变动类型：1-收入，2-支出
	Style             int       `json:"style" gorm:"type:tinyint;not null"`                // 变动方式：1-订单扣款，2-退款，3-手动调整等
	Balance           float64   `json:"balance" gorm:"type:decimal(10,2);not null"`        // 变动后余额
	BalanceBefore     float64   `json:"balance_before" gorm:"type:decimal(10,2);not null"` // 变动前余额
	Remark            string    `json:"remark" gorm:"size:255"`                            // 备注
	Operator          string    `json:"operator" gorm:"size:100"`                          // 操作人
	CreatedAt         time.Time `json:"created_at" gorm:"index"`                           // 创建时间
}

// TableName 返回表名
func (BalanceLog) TableName() string {
	return "balance_logs"
}

// 余额变动类型
const (
	BalanceTypeIncome  = 1 // 收入
	BalanceTypeExpense = 2 // 支出
)

// 余额变动方式
const (
	BalanceStyleOrderDeduct = 1 // 订单扣款
	BalanceStyleRefund      = 2 // 退款
	BalanceStyleManual      = 3 // 手动调整
	BalanceStyleRecharge    = 4 // 充值
)
