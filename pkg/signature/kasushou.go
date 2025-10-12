package signature

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// GenerateKasushouSign 生成卡速售平台签名: sha1(timestamp + sortedJSON(body) + apikey)
// 注意：
// - 排序规则：请求Body参数的键按ASCII码从小到大排序；
// - JSON编码：UTF-8，不转义斜杠和中文；
// - 空Body时参与签名的data为"{}"；
// - 回调地址(url)需原样参与签名（不做URL编码）。
func GenerateKasushouSign(body map[string]interface{}, timestamp string, apiKey string) string {
	// 处理空body
	var jsonStr string
	if body == nil || len(body) == 0 {
		jsonStr = "{}"
	} else {
		jsonStr = stableEncodeMap(body)
	}
	// 拼接并计算sha1
	payload := timestamp + jsonStr + apiKey
	//打印下payload
	fmt.Println(payload)
	return sha1Hex(payload)
}

// VerifyKasushouCallback 验证回调签名
// 规则：
// - 将post中的 sign、card_list、express_list 移除后，按ASCII排序并进行JSON编码；
// - 使用 body 中的 time 字段作为 timestamp；
// - 计算 sha1(time + sortedJSON(postWithoutExcluded) + apiKey) 与传入 sign 比较
func VerifyKasushouCallback(post map[string]interface{}, apiKey string) bool {
	// 读取 sign，并移除不参与签名的字段
	var receivedSign string
	if v, ok := post["sign"]; ok {
		receivedSign = strings.ToLower(strings.TrimSpace(ksToString(v)))
		delete(post, "sign")
	}
	// 移除不参与签名的字段
	delete(post, "card_list")
	delete(post, "express_list")

	// 取 time 作为 timestamp
	timestamp := ksToString(post["time"]) // 文档示例为字符串
	if timestamp == "" {
		// 若没有显式time字段，兼容部分实现以字符串化
		if v, ok := post["Timestamp"]; ok {
			timestamp = ksToString(v)
		}
	}

	// 稳定编码剩余内容
	jsonStr := stableEncodeMap(post)
	computed := sha1Hex(timestamp + jsonStr + apiKey)
	return receivedSign != "" && receivedSign == computed
}

// sha1Hex 计算SHA1并以小写十六进制返回
func sha1Hex(s string) string {
	h := sha1.New()
	h.Write([]byte(s))
	return strings.ToLower(hex.EncodeToString(h.Sum(nil)))
}

// stableEncodeMap 对 map[string]interface{} 进行稳定JSON编码（键按ASCII排序，值递归编码）
func stableEncodeMap(m map[string]interface{}) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	b.WriteByte('{')
	for i, k := range keys {
		// 编码 key
		keyJSON, _ := json.Marshal(k)
		b.WriteString(string(keyJSON))
		b.WriteByte(':')
		// 编码 value（递归）
		b.WriteString(stableEncodeValue(m[k]))
		if i < len(keys)-1 {
			b.WriteByte(',')
		}
	}
	b.WriteByte('}')
	return b.String()
}

// stableEncodeValue 对值进行稳定JSON编码，map递归排序，其他类型使用标准编码
func stableEncodeValue(v interface{}) string {
	switch vv := v.(type) {
	case map[string]interface{}:
		return stableEncodeMap(vv)
	case map[string]string:
		// 转为 map[string]interface{} 处理
		mm := make(map[string]interface{}, len(vv))
		for k, val := range vv {
			mm[k] = val
		}
		return stableEncodeMap(mm)
	case []interface{}:
		var b strings.Builder
		b.WriteByte('[')
		for i, item := range vv {
			b.WriteString(stableEncodeValue(item))
			if i < len(vv)-1 {
				b.WriteByte(',')
			}
		}
		b.WriteByte(']')
		return b.String()
	default:
		// 其它类型用标准json编码，确保字符串正确转义且UTF-8
		bs, _ := json.Marshal(v)
		return string(bs)
	}
}

// ksToString 将任意类型转换为字符串（用于timestamp、sign等处理）
func ksToString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case json.Number:
		return t.String()
	case float64:
		// 尽量避免科学计数法，使用不丢失精度的整数字符串（时间戳通常是整数）
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	case int64:
		return strconv.FormatInt(t, 10)
	case int:
		return strconv.Itoa(t)
	default:
		return strings.TrimSpace(fmtAny(v))
	}
}

// fmtAny 使用json.Marshal进行字符串化的兜底实现
func fmtAny(v interface{}) string {
	bs, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	// 去掉可能的引号（如字符串会被包成"str"），仅用于兜底字符串化
	return strings.Trim(string(bs), "\"")
}
