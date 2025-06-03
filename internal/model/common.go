package model

import (
	"encoding/json"
	"fmt"
)

// StringOrNumber 兼容 string/number 的 JSON 字段
// 用于第三方响应 code 字段类型不固定的场景
// 用法：type Resp struct { Code StringOrNumber `json:"code"` }
type StringOrNumber string

func (s *StringOrNumber) UnmarshalJSON(data []byte) error {
	// 尝试解析为字符串
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		*s = StringOrNumber(str)
		return nil
	}
	// 尝试解析为数字
	var num json.Number
	if err := json.Unmarshal(data, &num); err == nil {
		*s = StringOrNumber(num.String())
		return nil
	}
	return fmt.Errorf("StringOrNumber: unsupported type: %s", string(data))
}
