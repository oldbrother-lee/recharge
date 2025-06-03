package signature

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"recharge-go/pkg/logger"
	"sort"
	"strings"
)

// GenerateKekebangSign 生成客帮帮平台签名
func GenerateKekebangSign(params map[string]interface{}, secretKey string) string {
	// 创建用于签名的参数集合（排除data字段）
	signParams := make(map[string]string)

	// 验证params下的time字段是否为时间戳，不是时间戳则转换成时间戳
	// timeStr, ok := params["time"].(string)
	// if !ok {
	// 	timeStr = fmt.Sprintf("%d", time.Now().Unix())
	// } else {
	// 	// 将字符串转换为时间戳
	// 	if _, err := strconv.ParseInt(timeStr, 10, 64); err != nil {
	// 		// 如果不是时间戳，则转换为时间戳
	// 		timeStr = fmt.Sprintf("%d", time.Now().Unix())
	// 	}
	// }
	// params["timestamp"] = timeStr
	for k, v := range params {
		if k == "data" {
			continue // 跳过data字段
		}
		//过滤空值
		if v == nil || v == "" || k == "sign" {
			continue
		}
		// 类型转换
		switch val := v.(type) {
		case string:
			signParams[k] = val
		default:
			signParams[k] = fmt.Sprintf("%v", val)
		}
	}

	// 对签名参数进行首字母升序排序
	keys := make([]string, 0, len(signParams))
	for k := range signParams {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 构建键值对列表
	var keyValueList []string
	for _, k := range keys {
		keyValueList = append(keyValueList, k+"="+signParams[k])
	}

	// 拼接签名明文
	plainText := strings.Join(keyValueList, "&")
	plainText += "&secret=" + secretKey
	// fmt.Println("MD5签名前串:", plainText)
	logger.Info(fmt.Sprintf("kekebang MD5签名前串: %s", plainText))
	// 计算MD5签名
	hasher := md5.New()
	hasher.Write([]byte(plainText))
	return hex.EncodeToString(hasher.Sum(nil))
}

// VerifyKekebangSign 验证客帮帮平台签名
func VerifyKekebangSign(params map[string]interface{}, sign string, secretKey string) bool {
	return GenerateKekebangSign(params, secretKey) == sign
}

// GenerateKekebangNotifySign 生成客帮通知签名
func GenerateKekebangNotifySign(params map[string]interface{}, secretKey string) string {
	// 1. 升序排序 key
	// keys := make([]string, 0, len(params))
	// for k := range params {
	// 	keys = append(keys, k)
	// }
	// sort.Strings(keys)

	// // 2. 过滤 value 为空字符串的项，拼接 key=value
	// var keyValueList []string
	// for _, k := range keys {
	// 	v := params[k]
	// 	// 只保留非空字符串
	// 	if v != nil && fmt.Sprintf("%v", v) != "" {
	// 		keyValueList = append(keyValueList, k+"="+fmt.Sprintf("%v", v))
	// 	}
	// }

	// // 3. 拼接明文
	// plainText := strings.Join(keyValueList, "&")
	// plainText += "&secret=" + secretKey

	// fmt.Println("MD5签名前串:", plainText)

	// // 4. 计算 MD5 并转小写
	// hasher := md5.New()
	// hasher.Write([]byte(plainText))
	// md5Sign := strings.ToLower(hex.EncodeToString(hasher.Sum(nil)))
	// fmt.Println("MD5签名后:", md5Sign)
	// return md5Sign
	keys := make([]string, 0, len(params))
	for k := range params {
		if k != "sign" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	// 构建签名字符串
	var sb string
	for _, k := range keys {
		sb += k + fmt.Sprintf("%v", params[k])
	}
	sb += secretKey
	logger.Info(fmt.Sprintf("kekebang Notify MD5签名前串: %s", sb))
	// 计算MD5
	return md5Sum(sb)
}

// md5Sum 函数用于计算MD5哈希值
func md5Sum(text string) string {
	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}
