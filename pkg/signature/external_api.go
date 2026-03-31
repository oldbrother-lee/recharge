package signature

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ExternalAPISignatureConfig 外部API签名配置
type ExternalAPISignatureConfig struct {
	AppID     string `json:"app_id"`
	AppSecret string `json:"app_secret"`
	Timestamp int64  `json:"timestamp"`
	Nonce     string `json:"nonce"`
}

// ExternalAPISignatureValidator 外部API签名验证器
type ExternalAPISignatureValidator struct {
	TimeWindow int64 // 时间窗口，单位秒，默认300秒(5分钟)
}

// NewExternalAPISignatureValidator 创建外部API签名验证器
func NewExternalAPISignatureValidator() *ExternalAPISignatureValidator {
	return &ExternalAPISignatureValidator{
		TimeWindow: 300, // 默认5分钟时间窗口
	}
}

// signParamValue 将参数值转为签名字符串，与常见客户端（如 Python/JS）一致：JSON 数字按整数形式输出，避免 float64 被格式化为科学计数法导致验签失败
func signParamValue(v interface{}) string {
	switch val := v.(type) {
	case float64:
		if val == float64(int64(val)) {
			return strconv.FormatInt(int64(val), 10)
		}
		return fmt.Sprintf("%v", val)
	case float32:
		if val == float32(int64(val)) {
			return strconv.FormatInt(int64(val), 10)
		}
		return fmt.Sprintf("%v", val)
	case int:
		return strconv.FormatInt(int64(val), 10)
	case int64:
		return strconv.FormatInt(val, 10)
	case int32:
		return strconv.FormatInt(int64(val), 10)
	case uint, uint64, uint32:
		return fmt.Sprintf("%v", val)
	case string:
		return val
	default:
		return fmt.Sprintf("%v", v)
	}
}

// GenerateExternalAPISignature 生成外部API签名 - 按照API文档标准实现
func (sv *ExternalAPISignatureValidator) GenerateExternalAPISignature(params map[string]interface{}, appSecret string) (string, error) {
	// 1. 过滤掉空值参数和签名参数本身，并统一数字为整数串（与 JSON 中 number 的常见序列化一致）
	filteredParams := make(map[string]string)
	for k, v := range params {
		if k == "sign" || k == "signature" || v == nil {
			continue
		}
		if s, ok := v.(string); ok && s == "" {
			continue
		}
		filteredParams[k] = signParamValue(v)
	}

	// 2. 按参数名进行字典序排序
	keys := make([]string, 0, len(filteredParams))
	for k := range filteredParams {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 3. 按照 key=value&key=value 的格式拼接
	var paramPairs []string
	for _, k := range keys {
		paramPairs = append(paramPairs, fmt.Sprintf("%s=%s", k, filteredParams[k]))
	}
	paramString := strings.Join(paramPairs, "&")

	// 4. 在拼接字符串末尾添加 &key=app_secret
	signString := paramString + "&key=" + appSecret
	fmt.Printf("外部api signString: %s\n", signString)
	// 5. 计算MD5并转换为大写
	h := md5.New()
	h.Write([]byte(signString))
	result := strings.ToUpper(hex.EncodeToString(h.Sum(nil)))
	return result, nil
}

// ValidateExternalAPISignature 验证外部API签名
func (sv *ExternalAPISignatureValidator) ValidateExternalAPISignature(params map[string]interface{}, signature string, appSecret string) error {
	// 1. 检查时间戳
	timestampValue, exists := params["timestamp"]
	if !exists {
		return fmt.Errorf("timestamp is required")
	}

	// 处理不同类型的timestamp
	var timestamp int64
	var err error

	switch v := timestampValue.(type) {
	case string:
		// 字符串类型，直接解析
		timestamp, err = strconv.ParseInt(v, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid timestamp format: %v", err)
		}
	case int:
		// 整数类型，直接转换
		timestamp = int64(v)
	case int64:
		// int64类型，直接使用
		timestamp = v
	case float64:
		// 浮点数类型（JSON解析时数字会变成float64），转换为int64
		timestamp = int64(v)
	default:
		return fmt.Errorf("invalid timestamp type: %T", v)
	}

	// 检查时间戳是否在有效范围内
	now := time.Now().Unix()
	if abs(now-timestamp) > sv.TimeWindow {
		return fmt.Errorf("timestamp expired")
	}
	params["timestamp"] = timestamp

	// 2. 移除签名参数
	validateParams := make(map[string]interface{})
	for k, v := range params {
		if k != "sign" && k != "signature" {
			validateParams[k] = v
		}
	}

	// 3. 生成签名
	expectedSignature, err := sv.GenerateExternalAPISignature(validateParams, appSecret)
	if err != nil {
		return err
	}

	// 4. 比较签名
	if signature != expectedSignature {
		return fmt.Errorf("signature mismatch")
	}

	return nil
}

// ParseFormParams 解析表单参数
func (sv *ExternalAPISignatureValidator) ParseFormParams(formData url.Values) map[string]interface{} {
	params := make(map[string]interface{})
	for k, v := range formData {
		if len(v) > 0 {
			params[k] = v[0]
		}
	}
	return params
}

// ParseJSONParams 解析JSON参数
func (sv *ExternalAPISignatureValidator) ParseJSONParams(jsonData map[string]interface{}) map[string]interface{} {
	return jsonData
}

// GenerateExternalAPINonce 生成随机字符串
func GenerateExternalAPINonce(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}

// abs 计算绝对值
func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}
