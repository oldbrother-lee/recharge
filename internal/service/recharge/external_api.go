package recharge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

    "recharge-go/internal/model"
    "recharge-go/internal/repository"
    logger "recharge-go/pkg/log"
    "recharge-go/pkg/signature"

	"gorm.io/gorm"
)

// ExternalAPIPlatform 外部API平台（使用本系统的外部API创建订单）
type ExternalAPIPlatform struct {
	platformRepo       repository.PlatformRepository
	signatureValidator *signature.ExternalAPISignatureValidator
}

// NewExternalAPIPlatform 创建外部API平台实例
func NewExternalAPIPlatform(db *gorm.DB) *ExternalAPIPlatform {
	return &ExternalAPIPlatform{
		platformRepo:       repository.NewPlatformRepository(db),
		signatureValidator: signature.NewExternalAPISignatureValidator(),
	}
}

// GetName 获取平台名称
func (p *ExternalAPIPlatform) GetName() string {
	return "internal_api"
}

// getAPIKeyAndSecret 获取API密钥和密钥
func (p *ExternalAPIPlatform) getAPIKeyAndSecret(ctx context.Context, accountID int64) (string, string, string, string, error) {
	logger.WithContextCategory(ctx, "recharge").Info("【获取外部API账号与平台信息】", logger.Int64V2("account_id", accountID))
	account, err := p.platformRepo.GetAccountByID(ctx, accountID)
	if err != nil {
		logger.WithContextCategory(ctx, "recharge").Error("【获取平台账号失败】", logger.Int64V2("account_id", accountID), logger.ErrorV2(err))
		return "", "", "", "", fmt.Errorf("get platform account failed: %v", err)
	}
	// 不再在此处查询平台记录，URL 由调用方根据 api/url 字段传入或回退
	return account.AccountName, account.AppKey, account.AppSecret, "", nil
}

// SubmitOrder 提交订单到外部API
func (p *ExternalAPIPlatform) SubmitOrder(ctx context.Context, order *model.Order, api *model.PlatformAPI, apiParam *model.PlatformAPIParam) error {
	logger.WithContextCategory(ctx, "recharge").Info("【开始提交外部API订单】",
		logger.Int64V2("order_id", order.ID),
		logger.StringV2("order_number", order.OrderNumber),
		logger.Int64V2("api_id", api.ID),
		logger.Int64V2("api_account_id", api.AccountID),
		logger.Int64V2("platform_id", api.PlatformID),
		logger.StringV2("api_code", api.Code),
		logger.Int64V2("param_id", apiParam.ID),
	)

	// 获取API密钥和URL
	appID, appKey, appSecret, apiURL, err := p.getAPIKeyAndSecret(ctx, api.AccountID)
	if err != nil {
		return fmt.Errorf("get api key and secret failed: %v", err)
	}
	// 若未取到 URL，回退为 api.URL
	apiBaseURL := apiURL
	if apiBaseURL == "" {
		apiBaseURL = api.URL
	}

	// 确定回调地址：优先使用apiParam中的CallbackURL，如果为空则使用api中的CallbackURL
	callbackURL := apiParam.CallbackURL
	callbackFrom := "api_param"
	if callbackURL == "" {
		callbackURL = api.CallbackURL
		callbackFrom = "api"
	}
	logger.WithContextCategory(ctx, "recharge").Info("【选择回调地址】",
		logger.Int64V2("order_id", order.ID),
		logger.StringV2("callback_url", callbackURL),
		logger.StringV2("from", callbackFrom),
	)

	// 构建请求参数
	params := map[string]interface{}{
		"app_id": appID,
		// "app_key":       appKey,
		"timestamp": time.Now().Unix(),
		"nonce":     p.generateNonce(),
		// 传递给下游使用 OrderNumber 字段
		"out_trade_num": order.OrderNumber,
		"product_id": func() int64 {
			if id, err := strconv.ParseInt(apiParam.ProductID, 10, 64); err == nil {
				return id
			}
			return 0
		}(),
		"mobile":      order.Mobile,
		"amount":      order.Price,
		"notify_url":  callbackURL,
		"customer_id": order.CustomerID,
		"isp":         order.ISP,
		// "param1":        order.Param1,
		// "param2":        order.Param2,
		// "param3":        order.Param3,
		// "remark":        order.Remark,
	}

	// 生成签名
	sign := p.generateSign(ctx, params, appSecret)
	params["sign"] = sign

	logger.WithContextCategory(ctx, "recharge").Info("【准备发送外部API请求】",
		logger.StringV2("url", apiBaseURL+"/external/order"),
		logger.StringV2("order_number", order.OrderNumber),
		logger.StringV2("out_trade_num", order.OrderNumber),
		logger.AnyV2("param_keys", func() []string {
			keys := make([]string, 0, len(params))
			for k := range params {
				if k != "sign" {
					keys = append(keys, k)
				}
			}
			return keys
		}()),
	)

	// 发送请求到外部API
	resp, err := p.sendRequest(ctx, appKey, apiBaseURL+"/external/order", params)
	if err != nil {
		logger.WithContextCategory(ctx, "recharge").Error("【提交到外部系统订单失败】",
			logger.StringV2("url", apiBaseURL+"/external/order"),
			logger.StringV2("order_number", order.OrderNumber),
			logger.StringV2("out_trade_num", order.OrderNumber),
			logger.ErrorV2(err),
		)
		return fmt.Errorf("submit order failed: %v", err)
	}

	// 检查响应
	if resp.Code != 200 {
		logger.WithContextCategory(ctx, "recharge").Error("【提交订单失败】",
			logger.StringV2("order_number", order.OrderNumber),
			logger.StringV2("out_trade_num", order.OrderNumber),
			logger.IntV2("code", resp.Code),
			logger.StringV2("message", resp.Message),
		)
		return fmt.Errorf("submit order failed: %s", resp.Message)
	}

	logger.WithContextCategory(ctx, "recharge").Info("【外部API提交订单成功】",
		logger.StringV2("order_number", order.OrderNumber),
		logger.StringV2("out_trade_num", order.OrderNumber),
	)
	return nil
}

// QueryOrderStatus 查询订单状态
func (p *ExternalAPIPlatform) QueryOrderStatus(ctx context.Context, order *model.Order) (model.OrderStatus, error) {
	logger.WithContextCategory(ctx, "recharge").Info("internal_api 查询订单状态",
		logger.Int64V2("order_id", order.ID),
		logger.StringV2("order_number", order.OrderNumber),
		logger.StringV2("out_trade_num", order.OrderNumber),
	)

	// 获取API密钥和URL
	appID, appKey, appSecret, apiURL, err := p.getAPIKeyAndSecret(ctx, order.PlatformAccountID)
	if err != nil {
		return model.OrderStatusFailed, fmt.Errorf("get api key and secret failed: %v", err)
	}
	// 若未取到 URL，回退为订单记录中的 PlatformURL
	apiBaseURL := apiURL
	if apiBaseURL == "" {
		apiBaseURL = order.PlatformURL
	}

	// 构建查询参数
	params := map[string]interface{}{
		"app_id":    appID,
		"timestamp": time.Now().Unix(),
		"nonce":     p.generateNonce(),
		// 传递给下游使用 OrderNumber 字段
		"out_trade_num": order.OrderNumber,
	}

	// 生成签名
	sign := p.generateSign(ctx, params, appSecret)
	params["sign"] = sign

	// 发送查询请求
	resp, err := p.sendQueryRequest(ctx, appKey, apiBaseURL+"/external/order/query", params)
	if err != nil {
		return model.OrderStatusFailed, fmt.Errorf("query order status failed: %v", err)
	}

	// 检查响应
	if resp.Code != 200 {
		return model.OrderStatusFailed, fmt.Errorf("query order status failed: %s", resp.Message)
	}

	// 解析订单状态
	if resp.Data != nil {
		if data, ok := resp.Data.(map[string]interface{}); ok {
			if status, exists := data["status"]; exists {
				if statusInt, ok := status.(float64); ok {
					return p.mapOrderStatus(int(statusInt)), nil
				}
			}
		}
	}

	return model.OrderStatusFailed, fmt.Errorf("invalid response data")
}

// ParseCallbackData 解析回调数据
func (p *ExternalAPIPlatform) ParseCallbackData(data []byte) (*model.CallbackData, error) {
	var callbackData struct {
		AppID       string  `json:"app_id"`
		OrderNumber string  `json:"order_number"`
		OutTradeNum string  `json:"out_trade_num"`
		Status      int     `json:"status"`
		Amount      float64 `json:"amount"`
		Message     string  `json:"message"`
		Timestamp   int64   `json:"timestamp"`
		Nonce       string  `json:"nonce"`
		Sign        string  `json:"sign"`
	}

	if err := json.Unmarshal(data, &callbackData); err != nil {
		return nil, fmt.Errorf("parse callback data failed: %v", err)
	}

	// 确定订单ID，优先使用out_trade_num，其次使用order_number
	orderID := callbackData.OutTradeNum
	if orderID == "" {
		orderID = callbackData.OrderNumber
	}

	if orderID == "" {
		return nil, fmt.Errorf("both out_trade_num and order_number are empty")
	}

	// 转换状态为字符串
	statusStr := fmt.Sprintf("%d", callbackData.Status)

	// 格式化金额
	amountStr := fmt.Sprintf("%.2f", callbackData.Amount)

	// 格式化时间戳
	timestampStr := fmt.Sprintf("%d", callbackData.Timestamp)

	return &model.CallbackData{
		OrderID:       orderID,
		OrderNumber:   callbackData.OrderNumber,
		Status:        statusStr,
		Amount:        amountStr,
		Message:       callbackData.Message,
		CallbackType:  "order_status",
		Sign:          callbackData.Sign,
		Timestamp:     timestampStr,
		TransactionID: "internal_api_" + orderID,
	}, nil
}

// QueryBalance 查询账户余额
func (p *ExternalAPIPlatform) QueryBalance(ctx context.Context, accountID int64) (float64, error) {
	// 获取API密钥和URL
	appID, appKey, appSecret, apiURL, err := p.getAPIKeyAndSecret(ctx, accountID)
	if err != nil {
		return 0, fmt.Errorf("get api key and secret failed: %v", err)
	}

	// 构建查询参数
	params := map[string]interface{}{
		"app_id": appID,
		// "app_key":   appKey,
		"timestamp": time.Now().Unix(),
		"nonce":     p.generateNonce(),
	}

	// 生成签名
	sign := p.generateSign(ctx, params, appSecret)
	params["sign"] = sign

	// 发送查询请求
	resp, err := p.sendQueryRequest(ctx, appKey, apiURL+"/external/balance", params)
	if err != nil {
		return 0, fmt.Errorf("query balance failed: %v", err)
	}

	// 检查响应
	if resp.Code != 200 {
		return 0, fmt.Errorf("query balance failed: %s", resp.Message)
	}

	// 解析余额
	if resp.Data != nil {
		if data, ok := resp.Data.(map[string]interface{}); ok {
			if balance, exists := data["balance"]; exists {
				if balanceFloat, ok := balance.(float64); ok {
					return balanceFloat, nil
				}
			}
		}
	}

	return 0, fmt.Errorf("invalid balance response")
}

// generateNonce 生成随机字符串
func (p *ExternalAPIPlatform) generateNonce() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// generateSign 生成签名
func (p *ExternalAPIPlatform) generateSign(ctx context.Context, params map[string]interface{}, appSecret string) string {
	sign, err := p.signatureValidator.GenerateExternalAPISignature(params, appSecret)
	if err != nil {
		logger.WithContextCategory(ctx, "recharge").Error("Failed to generate external API signature", logger.ErrorV2(err))
		return ""
	}
	return sign
}

// mapOrderStatus 映射订单状态
func (p *ExternalAPIPlatform) mapOrderStatus(status int) model.OrderStatus {
	switch status {
	case 1: // 待处理
		return model.OrderStatusPendingRecharge
	case 2: // 处理中
		return model.OrderStatusRecharging
	case 3: // 成功
		return model.OrderStatusSuccess
	case 4: // 失败
		return model.OrderStatusFailed
	default:
		return model.OrderStatusFailed
	}
}

// APIResponse 外部API响应结构
type APIResponse struct {
	Code      int         `json:"code"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data"`
	Timestamp int64       `json:"timestamp"`
}

// sendRequest 发送HTTP请求
func (p *ExternalAPIPlatform) sendRequest(ctx context.Context, app_key, url string, params map[string]interface{}) (*APIResponse, error) {
	// 序列化请求参数（不打印敏感字段）
	jsonData, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("marshal request params failed: %v", err)
	}

	stage := "submit"
	ul := strings.ToLower(url)
	if strings.Contains(ul, "/query") || strings.Contains(ul, "balance") {
		stage = "query"
	}
	logger.WithContextCategory(ctx, "recharge").Info("外部API请求体",
		logger.StringV2("stage", stage),
		logger.StringV2("url", url),
		logger.StringV2("body", string(jsonData)),
	)

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request failed: %v", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "recharge-go/1.0")

	// 从参数中获取API密钥和签名
	// if appKey, ok := params["app_key"].(string); ok {
	req.Header.Set("X-API-Key", app_key)
	// }
	if sign, ok := params["sign"].(string); ok {
		req.Header.Set("X-Signature", sign)
	}

	// 发送请求
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request failed: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response failed: %v", err)
	}

	logger.WithContextCategory(ctx, "recharge").Info("【外部API响应】",
		logger.StringV2("stage", "response"),
		logger.StringV2("url", url),
		logger.IntV2("status_code", resp.StatusCode),
		logger.StringV2("body", string(body)),
	)

	// 解析响应
	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("unmarshal response failed: %v, response body: %s", err, string(body))
	}

	return &apiResp, nil
}

// sendQueryRequest 发送查询请求
func (p *ExternalAPIPlatform) sendQueryRequest(ctx context.Context, appKey, url string, params map[string]interface{}) (*APIResponse, error) {
	return p.sendRequest(ctx, appKey, url, params)
}
