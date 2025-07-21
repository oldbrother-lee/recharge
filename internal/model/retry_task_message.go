package model

import "time"

// RetryTaskMessage 重试任务消息
type RetryTaskMessage struct {
	OrderID   int64     `json:"order_id"`   // 订单ID
	RetryType int       `json:"retry_type"` // 重试类型：1-定时重试，2-外部回调重试
	Reason    string    `json:"reason"`     // 重试原因
	CreatedAt time.Time `json:"created_at"` // 创建时间
}

// NewRetryTaskMessage 创建重试任务消息
func NewRetryTaskMessage(orderID int64, retryType int, reason string) *RetryTaskMessage {
	return &RetryTaskMessage{
		OrderID:   orderID,
		RetryType: retryType,
		Reason:    reason,
		CreatedAt: time.Now(),
	}
}