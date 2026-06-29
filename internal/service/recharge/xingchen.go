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
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"recharge-go/internal/model"
	"recharge-go/internal/repository"
	logger "recharge-go/pkg/log"
	"github.com/spf13/viper"

	"gorm.io/gorm"
)

// XingchenPlatform 兴辰网络 V1.3 平台（按商品编号接口）
type XingchenPlatform struct {
	platformRepo repository.PlatformRepository
	orderRepo    repository.OrderRepository
}

func NewXingchenPlatform(db *gorm.DB) *XingchenPlatform {
	return &XingchenPlatform{
		platformRepo: repository.NewPlatformRepository(db),
		orderRepo:    repository.NewOrderRepository(db),
	}
}

func (p *XingchenPlatform) GetName() string { return "xingchen" }

func xingchenMD5Lower(s string) string {
	sum := md5.Sum([]byte(s))
	return strings.ToLower(hex.EncodeToString(sum[:]))
}

func xingchenSign(parts ...string) string {
	var b strings.Builder
	for _, part := range parts {
		b.WriteString(strings.TrimSpace(part))
	}
	return xingchenMD5Lower(b.String())
}

func xingchenBaseURL(apiURL string) string {
	return strings.TrimSuffix(strings.TrimSpace(apiURL), "/")
}

func xingchenMapToAny(src map[string]string) map[string]interface{} {
	out := make(map[string]interface{}, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

func xingchenFormRaw(form map[string]string) string {
	if len(form) == 0 {
		return ""
	}
	keys := make([]string, 0, len(form))
	for k := range form {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	vals := url.Values{}
	for _, k := range keys {
		vals.Set(k, form[k])
	}
	return vals.Encode()
}

func xingchenLogRequest(ctx context.Context, op, reqURL string, form map[string]string) {
	l := logger.WithContextCategory(ctx, "recharge")
	if l == nil {
		return
	}
	raw := xingchenFormRaw(form)
	l.Info("兴辰网络 下游原始请求",
		logger.StringV2("op", op),
		logger.StringV2("url", reqURL),
		logger.IntV2("body_len", len(raw)),
		logger.StringV2("request_raw", raw),
	)
}

func xingchenLogResponse(ctx context.Context, op, reqURL string, httpStatus int, raw []byte) {
	l := logger.WithContextCategory(ctx, "recharge")
	if l == nil {
		return
	}
	l.Info("兴辰网络 下游原始响应",
		logger.StringV2("op", op),
		logger.StringV2("url", reqURL),
		logger.IntV2("http_status", httpStatus),
		logger.IntV2("body_len", len(raw)),
		logger.StringV2("response_raw", string(raw)),
	)
}

func (p *XingchenPlatform) postForm(ctx context.Context, reqURL string, form map[string]string, timeoutSec int) ([]byte, int, error) {
	if timeoutSec <= 0 {
		timeoutSec = 30
	}
	vals := url.Values{}
	for k, v := range form {
		vals.Set(k, v)
	}
	body := vals.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewBufferString(body))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := &http.Client{Timeout: time.Duration(timeoutSec) * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	return raw, resp.StatusCode, err
}

func (p *XingchenPlatform) getUserIDAndKey(accountID int64) (string, string, error) {
	acc, err := p.platformRepo.GetPlatformAccountByID(accountID)
	if err != nil {
		return "", "", fmt.Errorf("获取平台账号失败: %w", err)
	}
	userID := strings.TrimSpace(acc.AppKey)
	key := strings.TrimSpace(acc.AppSecret)
	if userID == "" || key == "" {
		return "", "", fmt.Errorf("xingchen 账号需配置 AppKey(userId) 与 AppSecret(key)")
	}
	return userID, key, nil
}

func parseExtraParams(raw []byte) map[string]string {
	out := map[string]string{}
	if len(raw) == 0 || string(raw) == "null" {
		return out
	}
	var mm map[string]interface{}
	if err := json.Unmarshal(raw, &mm); err != nil {
		return out
	}
	for k, v := range mm {
		out[k] = strings.TrimSpace(fmt.Sprintf("%v", v))
	}
	return out
}

func xingchenStatusToOrderStatus(s string) model.OrderStatus {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "SUCCESS":
		return model.OrderStatusSuccess
	case "REFUND":
		return model.OrderStatusFailed
	default:
		return model.OrderStatusRecharging
	}
}

func xingchenCallbackStatus(s string) string {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "SUCCESS":
		return "success"
	case "REFUND":
		return "failed"
	default:
		return "processing"
	}
}

func (p *XingchenPlatform) SubmitOrder(ctx context.Context, order *model.Order, api *model.PlatformAPI, apiParam *model.PlatformAPIParam) error {
	userID, key, err := p.getUserIDAndKey(api.AccountID)
	if err != nil {
		return err
	}
	goodsID := strings.TrimSpace(apiParam.ProductID)
	if goodsID == "" {
		return fmt.Errorf("xingchen 下单需要 goodsId(product_id)")
	}
	opts := parseExtraParams(api.ExtraParams)
	ip := strings.TrimSpace(opts["ip"])
	if ip == "" {
		// 文档要求传 ip，优先从配置读取默认值，避免硬编码
		ip = strings.TrimSpace(viper.GetString("xingchen.default_ip"))
		if ip == "" {
			ip = "61.51.110.34"
		}
	}

	faceValue := strconv.Itoa(int(order.Denom))
	if int(order.Denom) <= 0 {
		faceValue = "1"
	}
	form := map[string]string{
		"userId":     userID,
		"goodsId":    goodsID,
		"reqOrderId": order.OrderNumber,
		"account":    order.Mobile,
		"faceValue":  faceValue,
		"amount":     "1",
		"ip":         ip,
	}
	if cb := strings.TrimSpace(apiParam.CallbackURL); cb != "" {
		form["notifyUrl"] = cb
	} else if cb = strings.TrimSpace(api.CallbackURL); cb != "" {
		form["notifyUrl"] = cb
	}
	for _, k := range []string{"otherInfo", "prov", "city"} {
		if v := strings.TrimSpace(opts[k]); v != "" {
			form[k] = v
		}
	}
	form["sign"] = xingchenSign(userID, goodsID, form["account"], form["faceValue"], form["amount"], form["reqOrderId"], key)

	reqURL := xingchenBaseURL(api.URL) + "/charge/api/goodsId"
	xingchenLogRequest(ctx, "submit_goods", reqURL, form)
	SetSubmitTraceRequest(ctx, "xingchen", "submit_goods", reqURL, form)
	raw, httpStatus, err := p.postForm(ctx, reqURL, form, api.Timeout)
	if err != nil {
		return fmt.Errorf("xingchen 下单请求失败: %w", err)
	}
	xingchenLogResponse(ctx, "submit_goods", reqURL, httpStatus, raw)

	var resp struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			OrderID string `json:"orderId"`
			Status  string `json:"status"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return fmt.Errorf("xingchen 下单响应解析失败: %w", err)
	}
	if strings.EqualFold(strings.TrimSpace(resp.Code), "SUCCESS") {
		if strings.TrimSpace(resp.Data.OrderID) != "" {
			order.APIOrderNumber = strings.TrimSpace(resp.Data.OrderID)
		}
		return nil
	}
	manualReviewCodes := map[string]struct{}{
		"OUT_ORDER_NO_USED": {},
		"SYSTEM_ERROR":      {},
	}
	code := strings.TrimSpace(resp.Code)
	if _, ok := manualReviewCodes[code]; ok {
		logger.WithContextCategory(ctx, "recharge").Warn("兴辰网络下单返回人工核实状态码，保持处理中",
			logger.StringV2("code", code),
			logger.StringV2("msg", strings.TrimSpace(resp.Msg)),
			logger.StringV2("order_number", order.OrderNumber),
		)
		return nil
	}
	msg := strings.TrimSpace(resp.Msg)
	return &DownstreamError{
		Platform: "xingchen",
		Code:     code,
		Message:  msg,
		Details:  msg,
		Request:  xingchenMapToAny(form),
	}
}

func (p *XingchenPlatform) QueryOrderStatus(ctx context.Context, order *model.Order) (model.OrderStatus, error) {
	if order.PlatformAccountID == 0 {
		return order.Status, fmt.Errorf("xingchen 查单需要订单 platform_account_id")
	}
	userID, key, err := p.getUserIDAndKey(order.PlatformAccountID)
	if err != nil {
		return order.Status, err
	}
	api, err := p.platformRepo.GetAPIByID(ctx, order.APICurID)
	if err != nil {
		return order.Status, fmt.Errorf("xingchen 查单获取 API 失败: %w", err)
	}
	form := map[string]string{
		"userId":     userID,
		"reqOrderId": order.OrderNumber,
	}
	form["sign"] = xingchenSign(form["userId"], form["reqOrderId"], key)
	reqURL := xingchenBaseURL(api.URL) + "/charge/api/getOrderStatus"
	xingchenLogRequest(ctx, "query_order", reqURL, form)
	raw, httpStatus, err := p.postForm(ctx, reqURL, form, api.Timeout)
	if err != nil {
		return order.Status, err
	}
	xingchenLogResponse(ctx, "query_order", reqURL, httpStatus, raw)

	var resp struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			Status  string `json:"status"`
			Voucher string `json:"voucher"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return order.Status, err
	}
	if !strings.EqualFold(strings.TrimSpace(resp.Code), "SUCCESS") {
		return order.Status, fmt.Errorf("xingchen 查单未返回有效订单信息，需人工核实: code=%s msg=%s", strings.TrimSpace(resp.Code), strings.TrimSpace(resp.Msg))
	}
	if v := strings.TrimSpace(resp.Data.Voucher); v != "" {
		order.APITradeNum = v
	}
	return xingchenStatusToOrderStatus(resp.Data.Status), nil
}

func parseXingchenCallbackBody(data []byte) map[string]string {
	out := map[string]string{}
	raw := strings.TrimSpace(string(data))
	if raw == "" {
		return out
	}
	if strings.HasPrefix(raw, "{") {
		_ = json.Unmarshal(data, &out)
		return out
	}
	vals, err := url.ParseQuery(raw)
	if err != nil {
		return out
	}
	for k, vs := range vals {
		if len(vs) > 0 {
			out[k] = vs[0]
		}
	}
	return out
}

func (p *XingchenPlatform) ParseCallbackData(data []byte) (*model.CallbackData, error) {
	body := parseXingchenCallbackBody(data)
	userID := strings.TrimSpace(body["userId"])
	reqOrderID := strings.TrimSpace(body["reqOrderId"])
	status := strings.TrimSpace(body["status"])
	sign := strings.TrimSpace(body["sign"])
	if reqOrderID == "" {
		return nil, fmt.Errorf("xingchen 回调缺少 reqOrderId")
	}
	if userID == "" || status == "" || sign == "" {
		return nil, fmt.Errorf("xingchen 回调参数不完整")
	}
	order, err := p.orderRepo.GetByOrderNumber(context.Background(), reqOrderID)
	if err != nil {
		return nil, fmt.Errorf("xingchen 回调获取订单失败: %w", err)
	}
	acc, err := p.platformRepo.GetPlatformAccountByAccountName(userID)
	source := "account_name"
	if err != nil || acc == nil || strings.TrimSpace(acc.AppSecret) == "" {
		var byAppKey model.PlatformAccount
		dbErr := p.platformRepo.GetDB().WithContext(context.Background()).
			Where("app_key = ? AND status = ?", userID, 1).
			Order("id DESC").
			First(&byAppKey).Error
		if dbErr == nil && strings.TrimSpace(byAppKey.AppSecret) != "" {
			acc = &byAppKey
			source = "app_key"
		} else {
			acc, err = p.platformRepo.GetPlatformAccountByID(order.PlatformAccountID)
			if err != nil {
				return nil, fmt.Errorf("xingchen 回调获取账号失败: %w", err)
			}
			source = "order_platform_account_id"
		}
	}
	signRaw := userID + reqOrderID + status + acc.AppSecret
	expectSign := xingchenMD5Lower(signRaw)
	logger.WithContextCategory(context.Background(), "recharge").Info("xingchen 回调签名对比",
		logger.StringV2("user_id", userID),
		logger.StringV2("req_order_id", reqOrderID),
		logger.StringV2("status", status),
		logger.StringV2("app_secret", acc.AppSecret),
		logger.StringV2("sign_raw", signRaw),
		logger.StringV2("expect_sign", expectSign),
		logger.StringV2("callback_sign", sign),
		logger.Int64V2("platform_account_id", order.PlatformAccountID),
		logger.Int64V2("sign_account_id", acc.ID),
		logger.StringV2("sign_secret_source", source),
	)
	if !strings.EqualFold(expectSign, sign) {
		return nil, fmt.Errorf("xingchen 回调签名验证失败")
	}
	return &model.CallbackData{
		OrderNumber:   reqOrderID,
		OrderID:       reqOrderID,
		Status:        xingchenCallbackStatus(status),
		CallbackType:  "xingchen",
		TransactionID: strings.TrimSpace(body["voucher"]),
		Message:       strings.TrimSpace(body["remark"]),
		Sign:          sign,
	}, nil
}

func (p *XingchenPlatform) QueryBalance(ctx context.Context, accountID int64) (float64, error) {
	userID, key, err := p.getUserIDAndKey(accountID)
	if err != nil {
		return 0, err
	}
	var api model.PlatformAPI
	if err := p.platformRepo.GetDB().WithContext(ctx).
		Where("account_id = ? AND status = ?", accountID, 1).
		First(&api).Error; err != nil {
		return 0, fmt.Errorf("xingchen 余额查询需要账号下至少一条启用的 platform_api: %w", err)
	}
	form := map[string]string{
		"userId": userID,
		"sign":   xingchenSign(userID, key),
	}
	reqURL := xingchenBaseURL(api.URL) + "/charge/api/getUser"
	xingchenLogRequest(ctx, "query_balance", reqURL, form)
	raw, httpStatus, err := p.postForm(ctx, reqURL, form, api.Timeout)
	if err != nil {
		return 0, err
	}
	xingchenLogResponse(ctx, "query_balance", reqURL, httpStatus, raw)

	var resp struct {
		Msg    string `json:"msg"`
		Status bool   `json:"status"`
		Data   struct {
			Balance float64 `json:"balance"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return 0, err
	}
	if !resp.Status {
		return 0, fmt.Errorf("xingchen 余额查询失败: %s", strings.TrimSpace(resp.Msg))
	}
	return resp.Data.Balance, nil
}
