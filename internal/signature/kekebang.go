package signature

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"recharge-go/internal/model"
	logger "recharge-go/pkg/log"
)

// KekebangHandler 客客帮签名处理器
type KekebangHandler struct {
	*BaseSignatureHandler
}

// NewKekebangHandler 创建客客帮签名处理器
func NewKekebangHandler(config *Config) *KekebangHandler {
	return &KekebangHandler{
		BaseSignatureHandler: NewBaseSignatureHandler(config),
	}
}

// GenerateSignature 生成客客帮签名
func (h *KekebangHandler) GenerateSignature(ctx context.Context, params map[string]interface{}) (string, error) {
	// 1. 参数排序
	var keys []string
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 2. 构建签名字符串
	var builder strings.Builder
	for _, k := range keys {
		if k == "sign" {
			continue
		}
		builder.WriteString(fmt.Sprintf("%s=%v&", k, params[k]))
	}
	builder.WriteString(fmt.Sprintf("key=%s", h.config.AppSecret))
	signStr := builder.String()

	// 3. MD5加密
	hash := md5.New()
	hash.Write([]byte(signStr))
	sign := hex.EncodeToString(hash.Sum(nil))

	logger.InfoV2("kekebang_generate_signature",
		logger.AnyV2("params", params),
		logger.StringV2("signStr", signStr),
		logger.StringV2("sign", sign),
	)
	return sign, nil
}

// BuildRequestParams 构建客客帮请求参数
func (h *KekebangHandler) BuildRequestParams(ctx context.Context, order *model.Order, api *model.PlatformAPI) (map[string]interface{}, error) {
	// 优先使用 OutTradeNum 作为上游商户订单号；若为空则回退到 OrderNumber
	orderID := order.OutTradeNum
	if orderID == "" {
		orderID = order.OrderNumber
	}

	params := map[string]interface{}{
		"appid":      h.config.AppID,
		"timestamp":  time.Now().Unix(),
		"order_id":   orderID,
		"mobile":     order.Mobile,
		"amount":     order.TotalPrice,
		"product_id": order.ProductID,
	}

	// 生成签名
	sign, err := h.GenerateSignature(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("生成签名失败: %v", err)
	}
	params["sign"] = sign

	return params, nil
}
