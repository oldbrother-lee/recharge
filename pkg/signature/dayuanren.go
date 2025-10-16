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
	md5Sum := md5.Sum([]byte(signStr))
	return strings.ToUpper(fmt.Sprintf("%x", md5Sum))
}

// VerifyDayuanrenSign 校验大猿人平台签名
func VerifyDayuanrenSign(params map[string]string, apiKey string) bool {
	sign := params["sign"]
	if sign == "" {
		return false
	}
	expectedSign := GenerateDayuanrenSign(params, apiKey)
	return sign == expectedSign
}
