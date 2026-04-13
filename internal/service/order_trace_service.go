package service

import (
	"context"
	"encoding/json"
	"recharge-go/internal/model"
	"recharge-go/internal/repository"
	logger "recharge-go/pkg/log"
	"strings"

	"gorm.io/datatypes"
)

// OrderTraceService 订单链路记录与查询
type OrderTraceService struct {
	repo repository.OrderTraceRepository
}

// NewOrderTraceService 创建订单链路服务
func NewOrderTraceService(repo repository.OrderTraceRepository) *OrderTraceService {
	return &OrderTraceService{repo: repo}
}

// Record 追加一条链路事件（失败只打日志，不影响主流程）
func (s *OrderTraceService) Record(ctx context.Context, in *model.OrderTraceInput) {
	if s == nil || s.repo == nil || in == nil || in.OrderID == 0 || in.Node == "" {
		return
	}
	st := in.Status
	if st == "" {
		st = model.TraceStatusInfo
	}
	pi := cloneAndMaskPayload(in.PayloadIn)
	po := cloneAndMaskPayload(in.PayloadOut)
	piBytes, _ := json.Marshal(pi)
	poBytes, _ := json.Marshal(po)
	if len(piBytes) == 0 {
		piBytes = []byte("{}")
	}
	if len(poBytes) == 0 {
		poBytes = []byte("{}")
	}
	actor := in.Actor
	if actor == "" {
		actor = "system"
	}
	ev := &model.OrderTraceEvent{
		OrderID:    in.OrderID,
		Node:       in.Node,
		Status:     st,
		DurationMs: in.DurationMs,
		PayloadIn:  datatypes.JSON(piBytes),
		PayloadOut: datatypes.JSON(poBytes),
		Actor:      actor,
	}
	if err := s.repo.Create(ctx, ev); err != nil {
		logger.WithContextCategory(ctx, "order_trace").Error("写入订单链路失败",
			logger.ErrorV2(err),
			logger.Int64V2("order_id", in.OrderID),
			logger.StringV2("node", in.Node),
		)
	}
}

// ListByOrderID 按订单 ID 升序返回链路
func (s *OrderTraceService) ListByOrderID(ctx context.Context, orderID int64) ([]model.OrderTraceEvent, error) {
	if s == nil || s.repo == nil {
		return nil, nil
	}
	return s.repo.ListByOrderID(ctx, orderID)
}

func cloneAndMaskPayload(src map[string]interface{}) map[string]interface{} {
	if len(src) == 0 {
		return nil
	}
	out := make(map[string]interface{}, len(src))
	for k, v := range src {
		lk := strings.ToLower(k)
		if lk == "mobile" {
			if s, ok := v.(string); ok {
				out[k] = maskMobile(s)
				continue
			}
		}
		out[k] = v
	}
	return out
}

func maskMobile(s string) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= 7 {
		return "****"
	}
	return string(r[:3]) + "****" + string(r[len(r)-4:])
}
