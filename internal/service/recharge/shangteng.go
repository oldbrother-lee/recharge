package recharge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"recharge-go/internal/model"
	"recharge-go/internal/repository"
	"recharge-go/pkg/logger"
	"recharge-go/pkg/signature"

	"gorm.io/gorm"
)

// ShangtengPlatform 商腾科技平台
type ShangtengPlatform struct {
	platformRepo repository.PlatformRepository
}

// NewShangtengPlatform 创建商腾科技平台实例
func NewShangtengPlatform(db *gorm.DB) *ShangtengPlatform {
	return &ShangtengPlatform{platformRepo: repository.NewPlatformRepository(db)}
}

// GetName 获取平台名称
func (p *ShangtengPlatform) GetName() string { return "shangteng" }

// getAPIKeyAndSecret 获取API密钥信息
func (p *ShangtengPlatform) getAPIKeyAndSecret(ctx context.Context, accountID int64) (string, string, string, error) {
	logger.WithContextCategory(ctx, "recharge").Info("【获取商腾科技账号信息】", logger.Int64V2("account_id", accountID))
	account, err := p.platformRepo.GetAccountByID(ctx, accountID)
	if err != nil {
		logger.WithContextCategory(ctx, "recharge").Error("【获取平台账号失败】", logger.Int64V2("account_id", accountID), logger.ErrorV2(err))
		return "", "", "", fmt.Errorf("get platform account failed: %v", err)
	}
	return account.AppKey, account.AppSecret, account.AccountName, nil
}

// SubmitOrder 提交订单到商腾科技平台
func (p *ShangtengPlatform) SubmitOrder(ctx context.Context, order *model.Order, api *model.PlatformAPI, apiParam *model.PlatformAPIParam) error {
	logger.WithContextCategory(ctx, "recharge").Info("【开始提交商腾科技订单】",
		logger.StringV2("order_number", order.OrderNumber),
		logger.Int64V2("api_id", api.ID),
		logger.Int64V2("platform_id", api.PlatformID),
		logger.Int64V2("account_id", api.AccountID),
		logger.Int64V2("param_id", apiParam.ID),
		logger.StringV2("product_id", apiParam.ProductID),
	)

	// 获取API密钥信息
	apiKey, _, userId, err := p.getAPIKeyAndSecret(ctx, api.AccountID)
	if err != nil {
		return fmt.Errorf("get api key and secret failed: %v", err)
	}

	// 确定回调地址
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

	// 时间戳（10位秒）
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	// 构建请求Body（依据文档字段）
	logger.WithContextCategory(ctx, "recharge").Info("【商腾科技构建请求参数】", logger.StringV2("url", api.URL))
	body := map[string]interface{}{
		"product_id":       mustParseInt(apiParam.ProductID),
		"recharge_account": order.Mobile,
		"external_orderno": order.OrderNumber,
		"notify_url":       callbackURL,
		// "quantity":          1,
		// "mark":            order.Remark,
	}

	// 生成签名（基于业务参数 + timestamp + apiKey）
	sign := signature.GenerateShangtengSign(body, timestamp, apiKey)
	logger.WithContextCategory(ctx, "recharge").Info("【商腾科技生成签名】", logger.StringV2("userId", userId))

	// 将公共参数放到 Header
	headers := map[string]string{
		"Content-Type": "application/json; charset=utf-8",
		// "ApiKey":       apiKey,
		"Sign":      sign,
		"Timestamp": timestamp,
		"Userid":    userId,
	}

	// 发送请求（URL 以文档为准，假定 /api/Order/create）
	logger.WithContextCategory(ctx, "recharge").Info("【商腾科技提交订单请求参数】", logger.AnyV2("body", body), logger.AnyV2("headers", headers))
	resp, err := p.sendRequest(ctx, api.URL+"/api/Order/create", body, headers)
	if err != nil {
		logger.WithContextCategory(ctx, "recharge").Error("【提交商腾科技订单失败】",
			logger.StringV2("order_number", order.OrderNumber),
			logger.ErrorV2(err),
		)
		return fmt.Errorf("submit order failed: %v", err)
	}

	// 检查响应
	if resp.Status != 200 {
		logger.WithContextCategory(ctx, "recharge").Error("【提交商腾科技订单失败】",
			logger.StringV2("order_number", order.OrderNumber),
			logger.IntV2("status", resp.Status),
			logger.StringV2("message", resp.Msg),
			logger.AnyV2("response", resp),
		)
		return fmt.Errorf("submit order failed: %s", resp.Msg)
	}

	logger.WithContextCategory(ctx, "recharge").Info("【商腾科技提交订单成功】",
		logger.StringV2("order_number", order.OrderNumber),
		logger.AnyV2("response", resp),
	)
	return nil
}

// QueryOrderStatus 查询订单状态（主要通过回调更新状态，此方法保留接口兼容性）
func (p *ShangtengPlatform) QueryOrderStatus(ctx context.Context, order *model.Order) (model.OrderStatus, error) {
	logger.WithContextCategory(ctx, "recharge").Info("【查询商腾科技订单状态】",
		logger.Int64V2("order_id", order.ID),
		logger.StringV2("order_number", order.OrderNumber),
	)
	return order.Status, nil
}

// mapOrderState 映射订单状态（示例：1处理中、2成功、3取消、4退款、5失败）
func (p *ShangtengPlatform) mapOrderState(status string, orderNumber string) (int, string) {
	var orderStatus int
	switch status {
	case "1":
		orderStatus = int(model.OrderStatusRecharging)
	case "2":
		orderStatus = int(model.OrderStatusSuccess)
	case "3":
		orderStatus = int(model.OrderStatusCancelled)
	case "4":
		orderStatus = int(model.OrderStatusRefunded)
	default:
		orderStatus = int(model.OrderStatusFailed)
	}
	return orderStatus, strconv.Itoa(orderStatus)
}

// ParseCallbackData 解析回调数据
func (p *ShangtengPlatform) ParseCallbackData(data []byte) (*model.CallbackData, error) {
	var resp ShangtengCallbackResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parse callback data failed: %v", err)
	}
	_, statusStr := p.mapOrderState(strconv.Itoa(resp.Status), resp.ExternalOrderNo)
	// 优先使用 msg 作为失败原因，其次使用 recharge_hints
	msg := resp.Msg
	if msg == "" {
		msg = resp.RechargeHints
	}
	return &model.CallbackData{
		OrderID:       resp.ExternalOrderNo,
		OrderNumber:   resp.ExternalOrderNo,
		Status:        statusStr,
		Message:       msg,
		CallbackType:  "order_status",
		Amount:        resp.TotalPrice,
		Sign:          resp.Sign,
		Timestamp:     resp.Time,
		TransactionID: "shangteng_" + resp.OrderNo,
	}, nil
}

// QueryBalance 查询余额（不支持）
func (p *ShangtengPlatform) QueryBalance(ctx context.Context, accountID int64) (float64, error) {
	return 0.0, fmt.Errorf("balance query not supported by shangteng platform")
}

// sendRequest 发送请求
func (p *ShangtengPlatform) sendRequest(ctx context.Context, url string, body map[string]interface{}, headers map[string]string) (*ShangtengResponse, error) {
	// 使用与签名一致的稳定编码，避免顺序或转义差异
	jsonStr := signature.EncodeShangtengBody(body)
	jsonData := []byte(jsonStr)
	// 打印实际发送的JSON串，确保与签名使用的一致
	logger.WithContextCategory(ctx, "recharge").Info("【商腾科技请求Body JSON】", logger.StringV2("json", string(jsonData)))
	logger.WithContextCategory(ctx, "recharge").Info("【商腾科技请求Headers】", logger.AnyV2("headers", headers))

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request failed: %v", err)
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	httpResp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request failed: %v", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response failed: %v", err)
	}
	logger.WithContextCategory(ctx, "recharge").Info("【商腾科技HTTP响应】", logger.IntV2("http_status", httpResp.StatusCode), logger.StringV2("body", string(respBody)))

	var resp ShangtengResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal response failed: %v", err)
	}
	return &resp, nil
}

// ShangtengResponse 商腾科技下单响应
type ShangtengResponse struct {
	Status int    `json:"status"`
	Msg    string `json:"msg"`
	Data   struct {
		OrderNo         string `json:"orderno"`
		ExternalOrderNo string `json:"external_orderno"`
	} `json:"data"`
}

// ShangtengCallbackResponse 商腾科技回调响应
type ShangtengCallbackResponse struct {
	ExternalOrderNo string `json:"external_orderno"`
	OrderNo         string `json:"orderno"`
	Status          int    `json:"status"`
	Msg             string `json:"msg,omitempty"`
	TotalPrice      string `json:"total_price,omitempty"`
	RechargeHints   string `json:"recharge_hints,omitempty"`
	Time            string `json:"time,omitempty"`
	Sign            string `json:"sign,omitempty"`
}
