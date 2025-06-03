package signature

import (
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"recharge-go/pkg/logger"
	"sort"
	"strconv"
	"strings"
	"time"
)

// GenerateXianzhuanxiaSignature 生成闲赚侠签名
func GenerateXianzhuanxiaSignature(params map[string]string, apiKey, userID string) (string, string, error) {
	// 第一步：排序并拼接非空参数
	var keys []string
	for k, v := range params {
		if v != "" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	var sb strings.Builder
	for _, k := range keys {
		v := params[k]
		// 处理数字类型，去除无效的小数点后面的0
		if strings.Contains(v, ".") {
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				v = trimTrailingZeros(f)
			}
		}
		sb.WriteString(k)
		sb.WriteString("=")
		sb.WriteString(v)
	}

	// 第二步：添加 queryTime 和 key
	queryTime := strconv.FormatInt(time.Now().UnixMilli(), 10)
	sb.WriteString("queryTime=")
	sb.WriteString(queryTime)
	sb.WriteString("key=")
	sb.WriteString(apiKey)
	// 第三步：MD5 加密
	logger.Info(fmt.Sprintf("闲赚侠签名前缀: %v\n", sb.String()))
	md5Hash := fmt.Sprintf("%x", md5.Sum([]byte(sb.String())))

	// 第四步：拼接 Auth_Token
	fmt.Printf("【闲赚侠签名】md5Hash: %s, userID: %s, queryTime: %s\n", md5Hash, userID, queryTime)
	authToken := fmt.Sprintf("%s,%s,%s", md5Hash, userID, queryTime)

	// Base64 编码
	base64Token := base64.StdEncoding.EncodeToString([]byte(authToken))

	return base64Token, queryTime, nil
}
func GenerateXianzhuanxiaSignature2(params map[string]interface{}, apiKey, userID string) (string, string, error) {
	// 第一步：排序并拼接非空参数
	var keys []string
	for k, v := range params {
		if v != "" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	var sb strings.Builder
	for _, k := range keys {
		v := params[k]
		// 处理数字类型，去除无效的小数点后面的0

		sb.WriteString(k)
		sb.WriteString("=")
		sb.WriteString(fmt.Sprintf("%v", v))
	}

	// 第二步：添加 queryTime 和 key
	queryTime := strconv.FormatInt(time.Now().UnixMilli(), 10)
	sb.WriteString("queryTime=")
	sb.WriteString(queryTime)
	sb.WriteString("key=")
	sb.WriteString(apiKey)
	// 第三步：MD5 加密
	logger.Info(fmt.Sprintf("闲赚侠签名前缀: %v\n", sb.String()))
	md5Hash := fmt.Sprintf("%x", md5.Sum([]byte(sb.String())))

	// 第四步：拼接 Auth_Token
	fmt.Printf("【闲赚侠签名】md5Hash: %s, userID: %s, queryTime: %s\n", md5Hash, userID, queryTime)
	authToken := fmt.Sprintf("%s,%s,%s", md5Hash, userID, queryTime)

	// Base64 编码
	base64Token := base64.StdEncoding.EncodeToString([]byte(authToken))

	return base64Token, queryTime, nil
}

// 去除浮点数末尾的0
func trimTrailingZeros(f float64) string {
	str := strconv.FormatFloat(f, 'f', -1, 64)
	return str
}

// VerifyXianzhuanxiaSignature 验证闲赚侠平台回调签名
func VerifyXianzhuanxiaSignature(params map[string]interface{}, sign string, apiKey string, userID string) bool {
	// 将 map[string]interface{} 转换为 map[string]string
	strParams := make(map[string]string)
	for k, v := range params {
		if v == nil {
			continue
		}
		switch val := v.(type) {
		case string:
			strParams[k] = val
		case float64:
			strParams[k] = strconv.FormatFloat(val, 'f', -1, 64)
		case int:
			strParams[k] = strconv.Itoa(val)
		case int64:
			strParams[k] = strconv.FormatInt(val, 10)
		case bool:
			strParams[k] = strconv.FormatBool(val)
		default:
			strParams[k] = fmt.Sprintf("%v", val)
		}
	}

	calculatedSign, _, err := GenerateXianzhuanxiaSignature(strParams, apiKey, userID)
	if err != nil {
		return false
	}
	return calculatedSign == sign
}
