package signature

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strings"
)

// ChongzhiSignature chongzhi平台签名工具
type ChongzhiSignature struct{}

// NewChongzhiSignature 创建chongzhi签名实例
func NewChongzhiSignature() *ChongzhiSignature {
	return &ChongzhiSignature{}
}

// GenerateSign 生成MD5签名
// 参数顺序: userid, productid, price, num, mobile, spordertime, sporderid, key
func (s *ChongzhiSignature) GenerateSign(userid, productid, price, num, mobile, spordertime, sporderid, key string) string {
	// 按顺序拼接参数
	str := fmt.Sprintf("userid=%s&productid=%s&price=%s&num=%s&mobile=%s&spordertime=%s&sporderid=%s&key=%s",
		userid, productid, price, num, mobile, spordertime, sporderid, key)
	
	// 计算MD5哈希
	h := md5.New()
	h.Write([]byte(str))
	
	// 返回大写的十六进制字符串
	return strings.ToUpper(hex.EncodeToString(h.Sum(nil)))
}

// VerifySign 验证签名
func (s *ChongzhiSignature) VerifySign(userid, productid, price, num, mobile, spordertime, sporderid, key, sign string) bool {
	expectedSign := s.GenerateSign(userid, productid, price, num, mobile, spordertime, sporderid, key)
	return strings.EqualFold(expectedSign, sign)
}