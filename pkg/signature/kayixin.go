package signature

import (
	"crypto/md5"
	"encoding/hex"
	"strings"
)

const KayixinAPIVersion = "3.0"

// KayixinSign 卡易信 API 3.0 签名：MD5(X-APP-ID + appSecret + X-Version + X-Timestamp + bodyJson)
// bodyJson 为实际 POST 的原始 JSON 字符串（UTF-8），签名结果为小写十六进制。
func KayixinSign(appID, appSecret, version, timestamp, bodyJSON string) string {
	payload := appID + appSecret + version + timestamp + bodyJSON
	sum := md5.Sum([]byte(payload))
	return hex.EncodeToString(sum[:])
}

// KayixinVerify 校验卡易信回调/请求签名（大小写不敏感）。
func KayixinVerify(appID, appSecret, version, timestamp, bodyJSON, signature string) bool {
	expected := KayixinSign(appID, appSecret, version, timestamp, bodyJSON)
	return strings.EqualFold(strings.TrimSpace(signature), expected)
}
