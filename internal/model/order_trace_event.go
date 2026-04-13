package model

import (
	"time"

	"gorm.io/datatypes"
)

// 订单链路节点（与前端展示文案对应）
const (
	TraceNodeOrderCreated       = "ORDER_CREATED"
	TraceNodeQueued             = "QUEUED"
	TraceNodeRouteSelected      = "ROUTE_SELECTED"
	TraceNodeDownstreamSubmit   = "DOWNSTREAM_SUBMIT"
	TraceNodeStatusChanged      = "STATUS_CHANGED"
	TraceNodeCallbackReceived   = "CALLBACK_RECEIVED"
)

// 链路事件状态
const (
	TraceStatusSuccess = "success"
	TraceStatusFailed  = "failed"
	TraceStatusPending = "pending"
	TraceStatusInfo    = "info"
)

// OrderTraceEvent 订单链路事件（追加写，不修改）
type OrderTraceEvent struct {
	ID         int64          `json:"id" gorm:"primaryKey;autoIncrement"`
	OrderID    int64          `json:"order_id" gorm:"not null;index:idx_otrace_oc,priority:1"`
	CreatedAt  time.Time      `json:"created_at" gorm:"index:idx_otrace_oc,priority:2;autoCreateTime"`
	Node       string         `json:"node" gorm:"size:64;not null"`
	Status     string         `json:"status" gorm:"size:16;not null"`
	DurationMs int64          `json:"duration_ms"`
	PayloadIn  datatypes.JSON `json:"payload_in" gorm:"type:json"`
	PayloadOut datatypes.JSON `json:"payload_out" gorm:"type:json"`
	Actor      string         `json:"actor" gorm:"size:64"`
}

func (OrderTraceEvent) TableName() string {
	return "order_trace_events"
}

// OrderTraceInput 写入链路时的入参（内存结构，非表字段）
type OrderTraceInput struct {
	OrderID    int64
	Node       string
	Status     string
	DurationMs int64
	PayloadIn  map[string]interface{}
	PayloadOut map[string]interface{}
	Actor      string
}
