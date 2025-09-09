package signature

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// GenerateXianyinkeSign 闲赢客签名：按键名升序，拼接为 k=v&k2=v2... 再追加 appSecret，MD5 小写
func GenerateXianyinkeSign(params map[string]interface{}, appSecret string) string {
	if params == nil {
		return ""
	}
	// 1. 收集并排序键名，排除 sign
	keys := make([]string, 0, len(params))
	for k := range params {
		if k == "sign" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 2. 按 k=v& 形式拼接
	var builder strings.Builder
	for i, k := range keys {
		builder.WriteString(k)
		builder.WriteString("=")
		builder.WriteString(toString(params[k]))
		if i < len(keys)-1 {
			builder.WriteString("&")
		}
	}

	// 3. 末尾直接追加 appSecret（无任何分隔符）
	builder.WriteString(appSecret)

	// 4. 计算 MD5 小写
	h := md5.New()
	h.Write([]byte(builder.String()))
	return hex.EncodeToString(h.Sum(nil))
}

// VerifyXianyinkeSign 验证签名
func VerifyXianyinkeSign(params map[string]interface{}, sign string, appSecret string) bool {
	return GenerateXianyinkeSign(params, appSecret) == sign
}

// 将任意值转换为签名所需的字符串形式
func toString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case int:
		return strconv.Itoa(val)
	case int32:
		return strconv.FormatInt(int64(val), 10)
	case int64:
		return strconv.FormatInt(val, 10)
	case float32:
		// 去掉不必要的小数点
		return trimFloat(fmt.Sprintf("%v", val))
	case float64:
		return trimFloat(fmt.Sprintf("%v", val))
	case bool:
		return strconv.FormatBool(val)
	default:
		return fmt.Sprintf("%v", val)
	}
}

func trimFloat(s string) string {
	// 将诸如 123.000000 或 123.450000 处理为最简表示
	if strings.Contains(s, ".") {
		s = strings.TrimRight(s, "0")
		s = strings.TrimRight(s, ".")
	}
	return s
}