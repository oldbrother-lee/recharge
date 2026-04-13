package recharge

import (
	"errors"
	"fmt"
)

// DownstreamError 下游通道业务/提交失败时的结构化错误，供订单链路 payload_out 统一展示；各通道可逐步返回此类型。
type DownstreamError struct {
	Platform string // 平台 code，如 turbo
	Code     string // 下游错误码，如 FORBIDDEN
	Message  string // 下游顶层文案（如 JSON 的 message）
	Details  string // 用户可读详细原因（如 Turbo error.details）
	Request  map[string]interface{} // 实际提交给下游的请求参数
	Cause    error  // 包装的内层错误
}

func (e *DownstreamError) Error() string {
	if e == nil {
		return ""
	}
	if e.Details != "" {
		return e.Details
	}
	if e.Message != "" {
		if e.Code != "" {
			return fmt.Sprintf("%s: %s", e.Code, e.Message)
		}
		return e.Message
	}
	if e.Cause != nil {
		return e.Cause.Error()
	}
	if e.Platform != "" {
		return e.Platform + ": downstream error"
	}
	return "downstream error"
}

// Unwrap 支持 errors.Is / errors.As 链
func (e *DownstreamError) Unwrap() error { return e.Cause }

// SubmitFailurePayload 生成 DOWNSTREAM_SUBMIT 失败时的 payload_out：固定字段 + 各通道扩展（如 details）
func SubmitFailurePayload(err error) map[string]interface{} {
	out := map[string]interface{}{}
	var de *DownstreamError
	if errors.As(err, &de) && de != nil {
		if de.Platform != "" {
			out["channel"] = de.Platform
		}
		if de.Code != "" {
			out["channel_code"] = de.Code
		}
		if de.Message != "" {
			out["channel_message"] = de.Message
		}
		if de.Details != "" {
			out["details"] = de.Details
		}
		if len(de.Request) > 0 {
			out["request"] = de.Request
		}
	}
	errMsg := err.Error()
	if details, ok := out["details"].(string); !ok || details == "" || details != errMsg {
		out["error"] = errMsg
	}
	return out
}
