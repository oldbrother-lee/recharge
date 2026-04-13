package recharge

import (
	"context"
	"recharge-go/internal/model"
)

type submitTraceKey struct{}

// SubmitTraceCollector 收集一次下游提交的请求信息，供订单链路展示。
type SubmitTraceCollector struct {
	Channel string
	Op      string
	URL     string
	Request map[string]interface{}
}

// WithSubmitTraceCollector 为本次提交创建可写 collector。
func WithSubmitTraceCollector(ctx context.Context) (context.Context, *SubmitTraceCollector) {
	c := &SubmitTraceCollector{}
	return context.WithValue(ctx, submitTraceKey{}, c), c
}

// SetSubmitTraceRequest 在平台实现中写入本次真实请求参数。
func SetSubmitTraceRequest(ctx context.Context, channel, op, url string, body map[string]string) {
	v := ctx.Value(submitTraceKey{})
	c, ok := v.(*SubmitTraceCollector)
	if !ok || c == nil {
		return
	}
	req := make(map[string]interface{}, len(body))
	for k, val := range body {
		req[k] = val
	}
	c.Channel = channel
	c.Op = op
	c.URL = url
	c.Request = req
}

// BuildSubmitTracePayloadIn 生成 DOWNSTREAM_SUBMIT 的 payload_in；优先使用实际请求参数。
func BuildSubmitTracePayloadIn(order *model.Order, c *SubmitTraceCollector) map[string]interface{} {
	if c != nil && len(c.Request) > 0 {
		out := map[string]interface{}{
			"channel": c.Channel,
			"op":      c.Op,
			"url":     c.URL,
			"request": c.Request,
		}
		if order != nil {
			out["order_number"] = order.OrderNumber
			out["out_trade_num"] = order.OutTradeNum
		}
		return out
	}
	if order == nil {
		return map[string]interface{}{}
	}
	return map[string]interface{}{
		"order_number":        order.OrderNumber,
		"out_trade_num":       order.OutTradeNum,
		"mobile":              order.Mobile,
		"submit_order_number": order.OrderNumber,
	}
}

