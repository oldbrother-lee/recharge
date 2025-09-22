package signature

import (
	"crypto/md5"
	"fmt"
	"recharge-go/pkg/logger"
	"sort"
	"strings"
)

// GenerateDayuanrenSign 生成大猿人平台签名
func GenerateDayuanrenSign(params map[string]string, apiKey string) string {
	// 不参与签名
	delete(params, "sign")
	var keys []string
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var signParts []string
	for _, k := range keys {
		signParts = append(signParts, fmt.Sprintf("%s=%s", k, params[k]))
	}
	signStr := strings.Join(signParts, "&") + "&apikey=" + apiKey
	logger.Info(fmt.Sprintf("大猿人签名前字符串 %s", signStr))
	// 打印详细的参数信息用于调试
	logger.Info(fmt.Sprintf("大猿人签名参数详情: %+v", params))
	logger.Info(fmt.Sprintf("大猿人签名密钥: %s", apiKey))
	md5Sum := md5.Sum([]byte(signStr))
	result := strings.ToUpper(fmt.Sprintf("%x", md5Sum))
	logger.Info(fmt.Sprintf("大猿人生成的签名: %s", result))
	return result
}

// VerifyDayuanrenSign 校验大猿人平台签名
func VerifyDayuanrenSign(params map[string]string, apiKey string) bool {
	sign := params["sign"]
	if sign == "" {
		return false
	}
	// 创建参数副本，避免修改原始参数
	paramsCopy := make(map[string]string)
	for k, v := range params {
		paramsCopy[k] = v
	}
	expectedSign := GenerateDayuanrenSign(paramsCopy, apiKey)
	fmt.Printf("大猿人校验签名 sign: %s, expectedSign: %s\n", sign, expectedSign)
	return sign == expectedSign
}
