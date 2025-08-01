package signature

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"recharge-go/internal/model"
	"recharge-go/pkg/logger"
)

// Payc2Handler payc2签名处理器
type Payc2Handler struct {
	*BaseSignatureHandler
}

// NewPayc2Handler 创建payc2签名处理器
func NewPayc2Handler(config *Config) *Payc2Handler {
	return &Payc2Handler{
		BaseSignatureHandler: NewBaseSignatureHandler(config),
	}
}

// GenerateSignature 生成payc2签名
// 签名步骤：
// 1. 所有请求参数按照ASCII码的升序进行排序
// 2. 按照key1=value1&key2=value2进行组合
// 3. 最后加上商户秘钥（&key=商户秘钥）
// 4. 然后进行MD5运算
func (h *Payc2Handler) GenerateSignature(ctx context.Context, params map[string]interface{}) (string, error) {
	// 1. 获取所有参数键并排序（ASCII码升序）
	var keys []string
	for k := range params {
		if k == "sign" || k == "state" {
			continue // 签名字段和state字段不参与签名
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 2. 构建签名字符串
	var builder strings.Builder
	for i, k := range keys {
		if i > 0 {
			builder.WriteString("&")
		}
		// 参数值即便为空，也必须参与签名
		builder.WriteString(fmt.Sprintf("%s=%v", k, params[k]))
	}

	// 3. 加上商户秘钥
	builder.WriteString(fmt.Sprintf("&key=%s", h.config.AppSecret))
	signStr := builder.String()

	// 4. MD5运算
	hash := md5.New()
	hash.Write([]byte(signStr))
	sign := hex.EncodeToString(hash.Sum(nil))

	logger.Info("payc2签名生成", "params", params, "signStr", signStr, "sign", sign)
	return sign, nil
}

// BuildRequestParams 构建payc2请求参数
func (h *Payc2Handler) BuildRequestParams(ctx context.Context, order *model.Order, api *model.PlatformAPI) (map[string]interface{}, error) {
	// 构建基础参数
	params := map[string]interface{}{
		"merch":         h.config.AppID,                     // 商户号
		"orderNo":       order.OrderNumber,                  // 商户订单号（可选）
		"amount":        int(order.Denom),                   // 订单金额（整数）
		"notifyUrl":     api.CallbackURL,                    // 回调通知地址
		"timeoutSecond": 360,                                // 超时时间（30分钟）
		"phone":         order.Mobile,                       // 手机号
		"telco":         h.getTelcoFromOrder(order), // 运营商
	}

	// 如果没有设置orderNo，则不传递该参数（参数如果没有传递，则不参与签名）
	if order.OrderNumber == "" {
		delete(params, "orderNo")
	}

	// 生成签名
	sign, err := h.GenerateSignature(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("生成签名失败: %v", err)
	}
	params["sign"] = sign

	return params, nil
}

// getTelcoFromOrder 根据订单中的 ISP 字段判断运营商
// 1: YD（移动）, 2: DX（电信）, 3: LT（联通）
func (h *Payc2Handler) getTelcoFromOrder(order *model.Order) string {
	switch order.ISP {
	case 1:
		return "YD" // 移动
	case 2:
		return "DX" // 电信
	case 3:
		return "LT" // 联通
	default:
		return "YD" // 默认移动
	}
}

// VerifyCallback 验证回调签名
func (h *Payc2Handler) VerifyCallback(params map[string]interface{}) bool {
	receivedSign, ok := params["sign"].(string)
	if !ok {
		return false
	}

	// 移除签名字段后重新计算签名
	delete(params, "sign")
	expectedSign, err := h.GenerateSignature(context.Background(), params)
	if err != nil {
		logger.Error("验证回调签名失败", "error", err)
		return false
	}

	return strings.EqualFold(receivedSign, expectedSign)
}
