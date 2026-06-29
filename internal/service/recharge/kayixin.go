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

// KayixinPlatform 卡易信商家客户 API 3.0
type KayixinPlatform struct {
	platformRepo repository.PlatformRepository
}

// NewKayixinPlatform 创建卡易信平台实例
func NewKayixinPlatform(db *gorm.DB) *KayixinPlatform {
	return &KayixinPlatform{
		platformRepo: repository.NewPlatformRepository(db),
	}
}

func (p *KayixinPlatform) GetName() string { return "kayixin" }

func kayixinBaseURL(apiURL string) string {
	return strings.TrimSuffix(strings.TrimSpace(apiURL), "/")
}

func (p *KayixinPlatform) getCredentials(ctx context.Context, accountID int64) (appID, secret string, err error) {
	acc, err := p.platformRepo.GetAccountByID(ctx, accountID)
	if err != nil {
		return "", "", fmt.Errorf("获取卡易信账号失败: %w", err)
	}
	appID = strings.TrimSpace(acc.AppKey)
	secret = strings.TrimSpace(acc.AppSecret)
	if appID == "" || secret == "" {
		return "", "", fmt.Errorf("卡易信账号需配置 AppKey(客户编号) 与 AppSecret(接口密钥)")
	}
	return appID, secret, nil
}

func kayixinParseAPIExtra(raw []byte) map[string]string {
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

func kayixinCallbackStatusString(status int) string {
	switch status {
	case 3:
		return "success"
	case 4, 5:
		return "failed"
	default:
		return "processing"
	}
}

func kayixinMapOrderStatus(status int) model.OrderStatus {
	switch status {
	case 3:
		return model.OrderStatusSuccess
	case 4, 5:
		return model.OrderStatusFailed
	default:
		return model.OrderStatusRecharging
	}
}

func (p *KayixinPlatform) postSigned(ctx context.Context, url string, body interface{}, accountID int64, timeoutSec int) ([]byte, int, error) {
	if timeoutSec <= 0 {
		timeoutSec = 30
	}
	appID, secret, err := p.getCredentials(ctx, accountID)
	if err != nil {
		return nil, 0, err
	}
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return nil, 0, fmt.Errorf("序列化请求体失败: %w", err)
	}
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	sign := signature.KayixinSign(appID, secret, signature.KayixinAPIVersion, timestamp, string(bodyJSON))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("X-App-Id", appID)
	req.Header.Set("X-Version", signature.KayixinAPIVersion)
	req.Header.Set("X-Timestamp", timestamp)
	req.Header.Set("X-Signature", sign)

	client := &http.Client{Timeout: time.Duration(timeoutSec) * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	return raw, resp.StatusCode, err
}

func (p *KayixinPlatform) logDownstream(ctx context.Context, op, url string, body interface{}, httpStatus int, raw []byte) {
	l := logger.WithContextCategory(ctx, "recharge")
	if l == nil {
		return
	}
	reqJSON, _ := json.Marshal(body)
	l.Info("卡易信下游交互",
		logger.StringV2("op", op),
		logger.StringV2("url", url),
		logger.StringV2("request_raw", string(reqJSON)),
		logger.IntV2("http_status", httpStatus),
		logger.StringV2("response_raw", string(raw)),
	)
}

// SubmitOrder POST /api/v3/order/create
func (p *KayixinPlatform) SubmitOrder(ctx context.Context, order *model.Order, api *model.PlatformAPI, apiParam *model.PlatformAPIParam) error {
	appID, _, err := p.getCredentials(ctx, api.AccountID)
	if err != nil {
		return err
	}

	goodsID, err := strconv.Atoi(strings.TrimSpace(apiParam.ProductID))
	if err != nil || goodsID <= 0 {
		return fmt.Errorf("卡易信下单需要有效的 goodsId(product_id): %q", apiParam.ProductID)
	}

	notify := strings.TrimSpace(apiParam.CallbackURL)
	if notify == "" {
		notify = strings.TrimSpace(api.CallbackURL)
	}
	if notify == "" {
		return fmt.Errorf("卡易信下单需配置回调地址 notifyUrl")
	}

	opts := kayixinParseAPIExtra(api.ExtraParams)
	attachName := strings.TrimSpace(opts["attach_name"])
	if attachName == "" {
		attachName = "充值账号"
	}
	sku := strings.TrimSpace(opts["sku"])

	body := kayixinCreateBody{
		GoodsID:     goodsID,
		Count:       1,
		NotifyURL:   notify,
		OuterNumber: order.OrderNumber,
		Sku:         sku,
	}
	// safePrice 为卡易信侧订单总成本校验，传套餐面值（par_value），非 cost/price
	if apiParam.ParValue > 0 {
		body.SafePrice = apiParam.ParValue
	}
	if mobile := strings.TrimSpace(order.Mobile); mobile != "" {
		body.Attach = []kayixinAttachItem{{Name: attachName, Value: mobile}}
	}

	u := kayixinBaseURL(api.URL) + "/api/v3/order/create"
	SetSubmitTraceRequest(ctx, "kayixin", "create", u, map[string]string{
		"goodsId":     strconv.Itoa(goodsID),
		"outerNumber": order.OrderNumber,
		"notifyUrl":   notify,
	})
	raw, httpStatus, err := p.postSigned(ctx, u, body, api.AccountID, api.Timeout)
	if err != nil {
		return fmt.Errorf("卡易信下单请求失败: %w", err)
	}
	p.logDownstream(ctx, "create", u, body, httpStatus, raw)

	var resp kayixinCreateResp
	if err := json.Unmarshal(raw, &resp); err != nil {
		return fmt.Errorf("卡易信下单响应解析失败: %w", err)
	}
	if resp.Code != 1000 {
		return &DownstreamError{
			Platform: "kayixin",
			Code:     strconv.Itoa(resp.Code),
			Message:  strings.TrimSpace(resp.Msg),
			Request:  map[string]interface{}{"goodsId": goodsID, "outerNumber": order.OrderNumber, "appId": appID},
		}
	}
	if resp.Data != nil && resp.Data.OrderNumber != "" {
		order.APIOrderNumber = strings.TrimSpace(resp.Data.OrderNumber)
	}
	return nil
}

// QueryOrderStatus POST /api/v3/order/getDetail
func (p *KayixinPlatform) QueryOrderStatus(ctx context.Context, order *model.Order) (model.OrderStatus, error) {
	if order.PlatformAccountID == 0 {
		return order.Status, fmt.Errorf("卡易信查单需要订单 platform_account_id")
	}
	api, err := p.platformRepo.GetAPIByID(ctx, order.APICurID)
	if err != nil {
		return order.Status, fmt.Errorf("卡易信查单获取 API 失败: %w", err)
	}

	query := kayixinDetailBody{}
	if no := strings.TrimSpace(order.OrderNumber); no != "" {
		query.OuterNumber = no
	} else if no = strings.TrimSpace(order.APIOrderNumber); no != "" {
		query.OrderNumber = no
	} else {
		return order.Status, fmt.Errorf("卡易信查单缺少订单号")
	}

	u := kayixinBaseURL(api.URL) + "/api/v3/order/getDetail"
	raw, httpStatus, err := p.postSigned(ctx, u, query, order.PlatformAccountID, api.Timeout)
	if err != nil {
		return order.Status, err
	}
	p.logDownstream(ctx, "getDetail", u, query, httpStatus, raw)

	var resp kayixinDetailResp
	if err := json.Unmarshal(raw, &resp); err != nil {
		return order.Status, fmt.Errorf("卡易信查单响应解析失败: %w", err)
	}
	if resp.Code != 1000 || resp.Data == nil {
		return order.Status, fmt.Errorf("卡易信查单失败: code=%d msg=%s", resp.Code, resp.Msg)
	}
	return kayixinMapOrderStatus(resp.Data.Status), nil
}

// ParseCallbackData 解析卡易信订单回调（验签在 controller 层完成）
func (p *KayixinPlatform) ParseCallbackData(data []byte) (*model.CallbackData, error) {
	var body KayixinCallbackBody
	if err := json.Unmarshal(data, &body); err != nil {
		return nil, fmt.Errorf("卡易信回调 JSON 解析失败: %w", err)
	}

	orderNumber := strings.TrimSpace(body.OuterNumber)
	if orderNumber == "" {
		orderNumber = strings.TrimSpace(body.OrderNumber)
	}
	if orderNumber == "" {
		return nil, fmt.Errorf("卡易信回调缺少 outerNumber/orderNumber")
	}
	platformOrderNo := strings.TrimSpace(body.OrderNumber)
	if platformOrderNo == "" {
		platformOrderNo = orderNumber
	}

	msg := strings.TrimSpace(body.Result)
	if msg == "" {
		msg = fmt.Sprintf("status=%d", body.Status)
	}

	return &model.CallbackData{
		OrderNumber:   orderNumber,
		OrderID:       platformOrderNo,
		Status:        kayixinCallbackStatusString(body.Status),
		Message:       msg,
		CallbackType:  "order_status",
		Amount:        strconv.FormatFloat(body.Money, 'f', -1, 64),
		TransactionID: "kayixin_" + platformOrderNo,
	}, nil
}

// QueryBalance POST /api/v3/user/getAccount
func (p *KayixinPlatform) QueryBalance(ctx context.Context, accountID int64) (float64, error) {
	var api model.PlatformAPI
	if err := p.platformRepo.GetDB().WithContext(ctx).
		Where("account_id = ? AND status = ?", accountID, 1).
		First(&api).Error; err != nil {
		return 0, fmt.Errorf("卡易信余额查询需要账号下至少一条启用的 platform_api: %w", err)
	}

	u := kayixinBaseURL(api.URL) + "/api/v3/user/getAccount"
	raw, httpStatus, err := p.postSigned(ctx, u, struct{}{}, accountID, api.Timeout)
	if err != nil {
		return 0, err
	}
	p.logDownstream(ctx, "getAccount", u, struct{}{}, httpStatus, raw)

	var resp kayixinBalanceResp
	if err := json.Unmarshal(raw, &resp); err != nil {
		return 0, fmt.Errorf("卡易信余额响应解析失败: %w", err)
	}
	if resp.Code != 1000 || resp.Data == nil {
		return 0, fmt.Errorf("卡易信余额查询失败: code=%d msg=%s", resp.Code, resp.Msg)
	}
	return resp.Data.Balance, nil
}
