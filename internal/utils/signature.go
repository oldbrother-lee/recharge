package utils

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

// SignatureConfig 签名配置
type SignatureConfig struct {
	AppID     string `json:"app_id"`
	AppSecret string `json:"app_secret"`
	Timestamp int64  `json:"timestamp"`
	Nonce     string `json:"nonce"`
}

// SignatureValidator 签名验证器
type SignatureValidator struct {
	TimeWindow int64 // 时间窗口，单位秒，默认300秒(5分钟)
}

// NewSignatureValidator 创建签名验证器
func NewSignatureValidator() *SignatureValidator {
	return &SignatureValidator{
		TimeWindow: 300, // 默认5分钟时间窗口
	}
}

// GenerateSignature 生成签名
func (sv *SignatureValidator) GenerateSignature(params map[string]interface{}, appSecret string) (string, error) {
	// 1. 过滤空值参数和签名参数
	filteredParams := make(map[string]string)
	for k, v := range params {
		if v != nil && v != "" && k != "sign" {
			switch val := v.(type) {
			case string:
				if val != "" {
					filteredParams[k] = val
				}
			case int, int64, float64:
				filteredParams[k] = fmt.Sprintf("%v", val)
			default:
				filteredParams[k] = fmt.Sprintf("%v", val)
			}
		}
	}

	// 2. 参数排序
	keys := make([]string, 0, len(filteredParams))
	for k := range filteredParams {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 3. 拼接参数字符串
	var paramPairs []string
	for _, k := range keys {
		paramPairs = append(paramPairs, fmt.Sprintf("%s=%s", k, filteredParams[k]))
	}
	paramString := strings.Join(paramPairs, "&")

	// 4. 添加密钥
	signString := paramString + "&key=" + appSecret
	fmt.Printf("signString##########:%+v\n", signString)
	// 5. 使用MD5计算签名
	hash := md5.Sum([]byte(signString))
	return strings.ToUpper(hex.EncodeToString(hash[:])), nil
}

// ValidateSignature 验证签名
func (sv *SignatureValidator) ValidateSignature(params map[string]interface{}, signature string, appSecret string) error {
	// 1. 检查时间戳
	fmt.Printf("params##########:%+v\n", params)

	// 获取timestamp参数
	timestampValue, exists := params["timestamp"]
	if !exists {
		return fmt.Errorf("timestamp is required!!!")
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
	expectedSignature, err := sv.GenerateSignature(validateParams, appSecret)
	if err != nil {
		return err
	}

	// 4. 比较签名
	if strings.ToUpper(signature) != strings.ToUpper(expectedSignature) {
		return fmt.Errorf("signature mismatch")
	}

	return nil
}

// ParseFormParams 解析表单参数
func (sv *SignatureValidator) ParseFormParams(formData url.Values) map[string]interface{} {
	params := make(map[string]interface{})
	for k, v := range formData {
		if len(v) > 0 {
			params[k] = v[0]
		}
	}
	return params
}

// ParseJSONParams 解析JSON参数
func (sv *SignatureValidator) ParseJSONParams(jsonData map[string]interface{}) map[string]interface{} {
	return jsonData
}

// abs 计算绝对值
func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

// GenerateNonce 生成随机字符串
func GenerateNonce(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}
