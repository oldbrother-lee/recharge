package recharge

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"recharge-go/internal/model"
	"recharge-go/internal/repository"
	logger "recharge-go/pkg/log"

	"gorm.io/gorm"
)

// TurboPlatform Turbo 话费集成通道（文档：integration/v1/*）
type TurboPlatform struct {
	platformRepo repository.PlatformRepository
}

// NewTurboPlatform 创建 Turbo 平台实例
func NewTurboPlatform(db *gorm.DB) *TurboPlatform {
	return &TurboPlatform{platformRepo: repository.NewPlatformRepository(db)}
}

// GetName 获取平台名称
func (p *TurboPlatform) GetName() string {
	return "turbo"
}

func (p *TurboPlatform) getAccessKeySecret(accountID int64) (accessKey, secretKey string, err error) {
	acc, err := p.platformRepo.GetPlatformAccountByID(accountID)
	if err != nil {
		return "", "", fmt.Errorf("获取平台账号失败: %w", err)
	}
	if acc.AppKey == "" || acc.AppSecret == "" {
		return "", "", fmt.Errorf("Turbo 账号需配置 AppKey(accessKey) 与 AppSecret(secretKey)")
	}
	return acc.AppKey, acc.AppSecret, nil
}

func turboMD5Upper(s string) string {
	sum := md5.Sum([]byte(s))
	return strings.ToUpper(hex.EncodeToString(sum[:]))
}

// turboSign 除 sign 外非空字段按键名升序拼接 key=value&...&secretKey=xxx 后 MD5 大写（与官方接口说明一致）
func turboSign(params map[string]string, secretKey string) string {
	keys := make([]string, 0, len(params))
	for k, v := range params {
		if k == "sign" || v == "" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for i, k := range keys {
		if i > 0 {
			b.WriteByte('&')
		}
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(params[k])
	}
	b.WriteString("&secretKey=")
	b.WriteString(secretKey)
	return turboMD5Upper(b.String())
}

func turboBaseURL(apiURL string) string {
	return strings.TrimSuffix(strings.TrimSpace(apiURL), "/")
}

// turboLogDownstreamRequest 记录 Turbo 下游请求原始参数
func turboLogDownstreamRequest(ctx context.Context, op, url string, body map[string]string) {
	l := logger.WithContextCategory(ctx, "recharge")
	if l == nil {
		return
	}
	payload, _ := json.Marshal(body)
	l.Info("Turbo 下游原始请求",
		logger.StringV2("op", op),
		logger.StringV2("url", url),
		logger.IntV2("body_len", len(payload)),
		logger.StringV2("request_raw", string(payload)),
	)
}

// turboLogDownstreamRaw 记录 Turbo 下游 HTTP 原始响应体（排障用）
func turboLogDownstreamRaw(ctx context.Context, op, url string, httpStatus int, raw []byte) {
	l := logger.WithContextCategory(ctx, "recharge")
	if l == nil {
		return
	}
	l.Info("Turbo 下游原始响应",
		logger.StringV2("op", op),
		logger.StringV2("url", url),
		logger.IntV2("http_status", httpStatus),
		logger.IntV2("body_len", len(raw)),
		logger.StringV2("response_raw", string(raw)),
	)
}

func (p *TurboPlatform) httpPostJSON(ctx context.Context, url string, body map[string]string, timeoutSec int) ([]byte, int, error) {
	if timeoutSec <= 0 {
		timeoutSec = 30
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: time.Duration(timeoutSec) * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	return raw, resp.StatusCode, err
}

// SubmitOrder POST /integration/v1/checkout
func (p *TurboPlatform) SubmitOrder(ctx context.Context, order *model.Order, api *model.PlatformAPI, apiParam *model.PlatformAPIParam) error {
	l := logger.WithContextCategory(ctx, "recharge")
	accessKey, secretKey, err := p.getAccessKeySecret(api.AccountID)
	if err != nil {
		return err
	}
	notify := strings.TrimSpace(apiParam.CallbackURL)
	if notify == "" {
		notify = strings.TrimSpace(api.CallbackURL)
	}
	if notify == "" {
		return fmt.Errorf("Turbo 下单需配置回调地址（API 或套餐的 callback_url）")
	}
	productID := strings.TrimSpace(apiParam.ProductID)
	if productID == "" {
		return fmt.Errorf("Turbo 套餐 product_id 不能为空")
	}
	signParams := map[string]string{
		"accessKey":         accessKey,
		"productId":         productID,
		"phone":             order.Mobile,
		"externalOrderNo":   order.OrderNumber,
		"notificationUrl":   notify,
	}
	body := map[string]string{
		"accessKey":       signParams["accessKey"],
		"productId":       signParams["productId"],
		"phone":           signParams["phone"],
		"externalOrderNo": signParams["externalOrderNo"],
		"notificationUrl": signParams["notificationUrl"],
		"sign":            turboSign(signParams, secretKey),
	}
	u := turboBaseURL(api.URL) + "/integration/v1/checkout"
	if l != nil {
		l.Info("Turbo 提交订单",
			logger.StringV2("url", u),
			logger.StringV2("external_order_no", order.OrderNumber),
			logger.StringV2("product_id", productID),
		)
	}
	turboLogDownstreamRequest(ctx, "checkout", u, body)
	SetSubmitTraceRequest(ctx, "turbo", "checkout", u, body)
	raw, statusCode, err := p.httpPostJSON(ctx, u, body, api.Timeout)
	if err != nil {
		return fmt.Errorf("Turbo 请求失败: %w", err)
	}
	turboLogDownstreamRaw(ctx, "checkout", u, statusCode, raw)
	var wrap struct {
		Status  string          `json:"status"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
		Error   json.RawMessage `json:"error"`
	}
	if err := json.Unmarshal(raw, &wrap); err != nil {
		return fmt.Errorf("Turbo 响应解析失败: %w", err)
	}
	if wrap.Status != "success" {
		code, details := turboParseErrorPayload(wrap.Error)
		return &DownstreamError{
			Platform: "turbo",
			Code:     code,
			Message:  strings.TrimSpace(wrap.Message),
			Details:  details,
			Request:  mapStringStringToAny(body),
		}
	}
	var data struct {
		OrderNo          string `json:"orderNo"`
		OrderStatus      string `json:"orderStatus"`
		RechargeStatus   int    `json:"rechargeStatus"`
		ExternalOrderNo  string `json:"externalOrderNo"`
		TotalAmount      string `json:"totalAmount"`
		ProductID        string `json:"productId"`
	}
	if len(wrap.Data) > 0 && string(wrap.Data) != "null" {
		_ = json.Unmarshal(wrap.Data, &data)
	}
	if data.OrderNo != "" {
		order.APIOrderNumber = data.OrderNo
	}
	return nil
}

// turboParseErrorPayload 解析 Turbo 响应中的 error 字段（对象含 code/details，或少数情况下为字符串）
func turboParseErrorPayload(raw json.RawMessage) (code, details string) {
	if len(raw) == 0 || string(raw) == "null" {
		return "", ""
	}
	var obj struct {
		Code    string `json:"code"`
		Details string `json:"details"`
	}
	if err := json.Unmarshal(raw, &obj); err == nil && (obj.Code != "" || obj.Details != "") {
		return strings.TrimSpace(obj.Code), strings.TrimSpace(obj.Details)
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil && s != "" {
		return "", strings.TrimSpace(s)
	}
	return "", strings.TrimSpace(string(raw))
}

func mapStringStringToAny(src map[string]string) map[string]interface{} {
	out := make(map[string]interface{}, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

func turboMapStatus(orderStatus string, rechargeStatus int) model.OrderStatus {
	switch rechargeStatus {
	case 1:
		return model.OrderStatusSuccess
	case -1, 2:
		return model.OrderStatusFailed
	case 3:
		return model.OrderStatusPartial
	case 0, 4:
		// 充值中 / 未充值：保持处理中
		break
	}
	switch strings.TrimSpace(orderStatus) {
	case "Succeeded":
		return model.OrderStatusSuccess
	case "Failed", "Cancel":
		return model.OrderStatusFailed
	case "Created", "Processing":
		return model.OrderStatusRecharging
	default:
		return model.OrderStatusRecharging
	}
}

// QueryOrderStatus POST /integration/v1/orders
func (p *TurboPlatform) QueryOrderStatus(ctx context.Context, order *model.Order) (model.OrderStatus, error) {
	if order.PlatformAccountID == 0 {
		return order.Status, fmt.Errorf("Turbo 查单需要订单 platform_account_id")
	}
	accessKey, secretKey, err := p.getAccessKeySecret(order.PlatformAccountID)
	if err != nil {
		return order.Status, err
	}

	api, err := p.platformRepo.GetAPIByID(ctx, order.APICurID)
	if err != nil {
		return order.Status, fmt.Errorf("Turbo 查单获取 API 失败: %w", err)
	}
	orderNo := strings.TrimSpace(order.APIOrderNumber)
	if orderNo == "" {
		orderNo = order.OrderNumber
	}
	signParams := map[string]string{
		"accessKey": accessKey,
		"orderNo":   orderNo,
	}
	body := map[string]string{
		"accessKey": accessKey,
		"orderNo":   orderNo,
		"sign":      turboSign(signParams, secretKey),
	}
	u := turboBaseURL(api.URL) + "/integration/v1/orders"
	raw, statusCode, err := p.httpPostJSON(ctx, u, body, api.Timeout)
	if err != nil {
		return order.Status, err
	}
	turboLogDownstreamRaw(ctx, "query_order", u, statusCode, raw)
	var wrap struct {
		Status string          `json:"status"`
		Data   json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &wrap); err != nil {
		return order.Status, err
	}
	if wrap.Status != "success" || len(wrap.Data) == 0 || string(wrap.Data) == "null" {
		return order.Status, fmt.Errorf("Turbo 查单失败: %s", string(raw))
	}
	var data struct {
		OrderStatus    string `json:"orderStatus"`
		RechargeStatus int    `json:"rechargeStatus"`
	}
	if err := json.Unmarshal(wrap.Data, &data); err != nil {
		return order.Status, err
	}
	return turboMapStatus(data.OrderStatus, data.RechargeStatus), nil
}

// TurboCallbackBody Turbo 回调 JSON（字段以文档为准）
type TurboCallbackBody struct {
	OrderNo         string `json:"orderNo"`
	ExternalOrderNo string `json:"externalOrderNo"`
	OrderStatus     string `json:"orderStatus"`
	RechargeStatus  int    `json:"rechargeStatus"`
	Phone           string `json:"phone"`
	ProductID       string `json:"productId"`
	TotalAmount     string `json:"totalAmount"`
}

// ParseCallbackData 解析 Turbo 回调为统一结构（文档未要求回调验签字段，此处不做签名校验）
func (p *TurboPlatform) ParseCallbackData(data []byte) (*model.CallbackData, error) {
	var body TurboCallbackBody
	if err := json.Unmarshal(data, &body); err != nil {
		return nil, fmt.Errorf("Turbo 回调 JSON 解析失败: %w", err)
	}
	if body.ExternalOrderNo == "" {
		return nil, fmt.Errorf("Turbo 回调缺少 externalOrderNo")
	}
	st := turboCallbackStatusString(body.OrderStatus, body.RechargeStatus)
	return &model.CallbackData{
		OrderNumber:     body.ExternalOrderNo,
		OrderID:         body.OrderNo,
		Status:          st,
		CallbackType:    "turbo",
		TransactionID:   body.OrderNo,
		Message:         body.OrderStatus,
		Amount:          body.TotalAmount,
	}, nil
}

func turboCallbackStatusString(orderStatus string, rechargeStatus int) string {
	s := turboMapStatus(orderStatus, rechargeStatus)
	switch s {
	case model.OrderStatusSuccess:
		return "success"
	case model.OrderStatusFailed:
		return "failed"
	case model.OrderStatusPartial:
		return "partial"
	default:
		return "processing"
	}
}

// QueryBalance POST /integration/v1/accounts
func (p *TurboPlatform) QueryBalance(ctx context.Context, accountID int64) (float64, error) {
	accessKey, secretKey, err := p.getAccessKeySecret(accountID)
	if err != nil {
		return 0, err
	}
	var api model.PlatformAPI
	if err := p.platformRepo.GetDB().WithContext(ctx).
		Where("account_id = ? AND status = ?", accountID, 1).
		First(&api).Error; err != nil {
		return 0, fmt.Errorf("Turbo 余额查询需要账号下至少一条启用的 platform_api: %w", err)
	}
	signParams := map[string]string{"accessKey": accessKey}
	body := map[string]string{
		"accessKey": accessKey,
		"sign":      turboSign(signParams, secretKey),
	}
	u := turboBaseURL(api.URL) + "/integration/v1/accounts"
	raw, statusCode, err := p.httpPostJSON(ctx, u, body, api.Timeout)
	if err != nil {
		return 0, err
	}
	turboLogDownstreamRaw(ctx, "query_balance", u, statusCode, raw)
	var wrap struct {
		Status string `json:"status"`
		Data   struct {
			Balance string `json:"balance"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &wrap); err != nil {
		return 0, err
	}
	if wrap.Status != "success" {
		return 0, fmt.Errorf("Turbo 账户查询失败: %s", string(raw))
	}
	bal, err := strconv.ParseFloat(strings.TrimSpace(wrap.Data.Balance), 64)
	if err != nil {
		return 0, fmt.Errorf("Turbo 余额解析失败: %w", err)
	}
	return bal, nil
}
