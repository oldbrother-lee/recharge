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

// GenerateShangtengSign 生成商腾科技平台签名: sha1(timestamp + sortedJSON(body) + apiKey)
// 规则与示例提供一致：
// - 排序：Body 键按 ASCII 升序；值递归稳定 JSON 编码（不转义斜杠与中文）。
// - 空 Body：使用 "{}"。
// - 时间戳：字符串形式，建议 13 位毫秒。
func GenerateShangtengSign(body map[string]interface{}, timestamp string, apiKey string) string {
    var jsonStr string
    if body == nil || len(body) == 0 {
        jsonStr = "{}"
    } else {
        jsonStr = stStableEncodeMap(body)
    }
    payload := timestamp + jsonStr + apiKey
    // 调试输出：打印签名相关的全部参数
    fmt.Printf("【商腾签名】timestamp=%s\n", timestamp)
    fmt.Printf("【商腾签名】sortedJSON=%s\n", jsonStr)
    fmt.Printf("【商腾签名】apiKey=%s\n", apiKey)
    fmt.Printf("【商腾签名】payload=%s\n", payload)
    sign := stSha1Hex(payload)
    fmt.Printf("【商腾签名】sign=%s\n", sign)
    return sign
}

// VerifyShangtengCallback 验证回调签名
// 约定：
// - 从 post 中提取 sign（小写十六进制），并移除；
// - 使用 time 或 timestamp 字段作为时间戳；
// - 计算 sha1(time + sortedJSON(postWithoutSign) + apiKey) 与传入 sign 比较。
func VerifyShangtengCallback(post map[string]interface{}, apiKey string) bool {
    var receivedSign string
    if v, ok := post["sign"]; ok {
        receivedSign = strings.ToLower(strings.TrimSpace(stToString(v)))
        delete(post, "sign")
    }

    // 取时间戳
    timestamp := stToString(post["time"]) // 优先使用 time
    if timestamp == "" {
        timestamp = stToString(post["timestamp"]) // 兼容 timestamp
    }

    jsonStr := stStableEncodeMap(post)
    computed := stSha1Hex(timestamp + jsonStr + apiKey)
    return receivedSign != "" && receivedSign == computed
}

// stSha1Hex 计算SHA1并返回小写十六进制（商腾专用，不依赖其他平台方法）
func stSha1Hex(s string) string {
    h := sha1.New()
    h.Write([]byte(s))
    return strings.ToLower(hex.EncodeToString(h.Sum(nil)))
}

// stStableEncodeMap 对 map[string]interface{} 进行稳定JSON编码（键按ASCII排序，值递归编码）
func stStableEncodeMap(m map[string]interface{}) string {
    keys := make([]string, 0, len(m))
    for k := range m {
        keys = append(keys, k)
    }
    sort.Strings(keys)

    var b strings.Builder
    b.WriteByte('{')
    for i, k := range keys {
        // 编码 key（总是字符串）
        keyJSON, _ := json.Marshal(k)
        b.WriteString(string(keyJSON))
        b.WriteByte(':')
        // 编码 value（递归，确保UTF-8、斜杠与中文不转义）
        b.WriteString(stStableEncodeValue(m[k]))
        if i < len(keys)-1 {
            b.WriteByte(',')
        }
    }
    b.WriteByte('}')
    return b.String()
}

// stStableEncodeValue 对值进行稳定JSON编码
func stStableEncodeValue(v interface{}) string {
    switch vv := v.(type) {
    case map[string]interface{}:
        return stStableEncodeMap(vv)
    case map[string]string:
        mm := make(map[string]interface{}, len(vv))
        for k, val := range vv {
            mm[k] = val
        }
        return stStableEncodeMap(mm)
    case []interface{}:
        var b strings.Builder
        b.WriteByte('[')
        for i, item := range vv {
            b.WriteString(stStableEncodeValue(item))
            if i < len(vv)-1 {
                b.WriteByte(',')
            }
        }
        b.WriteByte(']')
        return b.String()
    default:
        // 使用标准json编码，Go默认UTF-8且不转义斜杠和中文，满足要求
        bs, _ := json.Marshal(v)
        return string(bs)
    }
}

// EncodeShangtengBody 将业务参数编码为稳定的JSON字符串（键按ASCII升序，斜杠与中文不转义）
// 与签名使用的编码完全一致，避免签名JSON与实际请求JSON不一致导致验签失败。
func EncodeShangtengBody(body map[string]interface{}) string {
    if body == nil || len(body) == 0 {
        return "{}"
    }
    return stStableEncodeMap(body)
}

// stToString 将任意类型转换为字符串（用于timestamp、sign等处理）
func stToString(v interface{}) string {
    if v == nil {
        return ""
    }
    switch t := v.(type) {
    case string:
        return t
    case json.Number:
        return t.String()
    case float64:
        if t == float64(int64(t)) {
            return strconv.FormatInt(int64(t), 10)
        }
        return strconv.FormatFloat(t, 'f', -1, 64)
    case int64:
        return strconv.FormatInt(t, 10)
    case int:
        return strconv.Itoa(t)
    default:
        bs, _ := json.Marshal(t)
        return strings.Trim(string(bs), "\"")
    }
}