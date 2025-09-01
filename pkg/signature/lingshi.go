package signature

import (
	"crypto/md5"
	"encoding/hex"
	"sort"
	"strings"
)

// GenerateLingshiSign 生成灵石平台签名
// 按照灵石文档：参数按字典序排列，直接拼接键值（不含等号），最后拼接appSecret，MD5加密
func GenerateLingshiSign(params map[string]string, appSecret string) string {
	// 1. 过滤掉 sign 字段并获取所有键
	keys := make([]string, 0, len(params))
	for k := range params {
		if k != "sign" {
			keys = append(keys, k)
		}
	}
	
	// 2. 按字典序排序
	sort.Strings(keys)
	
	// 3. 拼接参数 (key+value形式，不含等号)
	var signStr strings.Builder
	for _, k := range keys {
		signStr.WriteString(k)
		signStr.WriteString(params[k])
	}
	
	// 4. 拼接 appSecret
	signStr.WriteString(appSecret)
	
	// 5. MD5加密
	h := md5.New()
	h.Write([]byte(signStr.String()))
	return strings.ToLower(hex.EncodeToString(h.Sum(nil)))
}

// VerifyLingshiSign 验证灵石平台签名
func VerifyLingshiSign(params map[string]string, sign string, appSecret string) bool {
	expectedSign := GenerateLingshiSign(params, appSecret)
	return expectedSign == sign
}