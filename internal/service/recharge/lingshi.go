package recharge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"recharge-go/internal/model"
	"recharge-go/internal/repository"
	"recharge-go/pkg/logger"
	"recharge-go/pkg/signature"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

// LingshiPlatform 灵石平台实现
// 文档要点：
// - 提交订单、余额查询：POST JSON，含 appId、sign
// - 回调：POST JSON，返回纯文本 "success"，字段包含 appId、code、msg、orderId、systemOrderId、mobile、faceValue、price、productId、sign
// - 签名：参数按字典序拼接后+appSecret做MD5

type LingshiPlatform struct {
	platformRepo repository.PlatformRepository
	orderRepo    repository.OrderRepository
}

func NewLingshiPlatform(db *gorm.DB) *LingshiPlatform {
	return &LingshiPlatform{
		platformRepo: repository.NewPlatformRepository(db),
		orderRepo:    repository.NewOrderRepository(db),
	}
}

func (p *LingshiPlatform) GetName() string { return "lingshi" }

// helper 获取账号 key/secret
func (p *LingshiPlatform) getKeySecretAndAPI(ctx context.Context, code string, accountID int64) (appId, appSecret, baseURL string, err error) {
	// account 的 AppKey 作为 appId，AppSecret 作为 appSecret
	account, err := p.platformRepo.GetAccountByID(ctx, accountID)
	if err != nil {
		return "", "", "", fmt.Errorf("get account failed: %v", err)
	}
	api, err := p.platformRepo.GetPlatformByCode(ctx, code)
	if err != nil {
		return "", "", "", fmt.Errorf("get platform by code failed: %v", err)
	}
	return account.AppKey, account.AppSecret, api.URL, nil
}

// SubmitOrder 提交订单
func (p *LingshiPlatform) SubmitOrder(ctx context.Context, order *model.Order, api *model.PlatformAPI, apiParam *model.PlatformAPIParam) error {
	appId, appSecret, baseURL, err := p.getKeySecretAndAPI(ctx, "lingshi", api.AccountID)
	if err != nil { return err }

	// 按文档构造参数
	params := map[string]string{
		"appId":       appId,
		"orderId":     order.OrderNumber,
		"mobile":      order.Mobile,
		"productId":   apiParam.ProductID,
		"faceValue":   strconv.FormatFloat(apiParam.ParValue, 'f', -1, 64),
		"price":       strconv.FormatFloat(apiParam.Price, 'f', -1, 64),
		"notifyUrl":   order.PlatformCallbackURL,
		"timestamp":   strconv.FormatInt(time.Now().Unix(), 10),
	}
	params["sign"] = signature.GenerateLingshiSign(params, appSecret)

	body, _ := json.Marshal(params)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/order/submit", bytes.NewReader(body))
	if err != nil { return err }
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil { return fmt.Errorf("lingshi submit request failed: %v", err) }
	defer resp.Body.Close()
    respBody, _ := io.ReadAll(resp.Body)
    // 注入订单号并使用 v2 recharge 类别日志
    ctx = logger.InjectOrderNumber(ctx, order.OrderNumber)
    clog := logger.WithContextCategory(ctx, "recharge")
    bodyStr := string(respBody)
    preview := bodyStr
    if len(bodyStr) > 512 {
        preview = bodyStr[:512] + "..."
    }
    clog.Info("提交响应",
        logger.StringV2("platform", "lingshi"),
        logger.IntV2("status", resp.StatusCode),
        logger.StringV2("body_preview", preview),
    )

	// 期望成功 code=="0" 或文档自定义，这里宽松校验
	var r struct { Code string `json:"code"`; Msg string `json:"msg"` }
	_ = json.Unmarshal(respBody, &r)
	if r.Code != "0" && r.Code != "0000" && r.Code != "SUCCESS" && r.Code != "200" {
		return fmt.Errorf("lingshi submit failed: code=%s msg=%s", r.Code, r.Msg)
	}
	return nil
}

// QueryOrderStatus 查询订单状态（若文档提供）
func (p *LingshiPlatform) QueryOrderStatus(ctx context.Context, order *model.Order) (model.OrderStatus, error) {
    ctx = logger.InjectOrderNumber(ctx, order.OrderNumber)
    clog := logger.WithContextCategory(ctx, "recharge")
    clog.Info("查询订单状态",
        logger.StringV2("platform", "lingshi"),
        logger.Int64V2("order_id", order.ID),
        logger.StringV2("order_number", order.OrderNumber),
    )

	appId, appSecret, api, err := p.getKeySecretAndAPI(ctx, "lingshi", order.PlatformAccountID)
	if err != nil {
		return 0, fmt.Errorf("获取API配置失败: %v", err)
	}

	body := map[string]interface{}{
		"orderNo": order.OrderNumber,
	}

	payloadBytes, err := json.Marshal(body)
	if err != nil {
		return 0, fmt.Errorf("序列化请求体失败: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(api, "/")+"/api/order/queryOrder", bytes.NewReader(payloadBytes))
	if err != nil {
		return 0, fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("appId", appId)
	req.Header.Set("sign", signature.GenerateLingshiSign(map[string]string{
		"appId":   appId,
		"orderNo": order.OrderNumber,
		"ts":     strconv.FormatInt(time.Now().Unix(), 10),
	}, appSecret))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("读取响应失败: %v", err)
	}

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			Status int `json:"status"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBytes, &result); err != nil {
		return 0, fmt.Errorf("解析响应失败: %v", err)
	}
	if result.Code != 0 {
		return 0, fmt.Errorf("平台返回错误: %s", result.Msg)
	}

	return model.OrderStatus(result.Data.Status), nil
}

// ParseCallbackData 解析回调数据
func (p *LingshiPlatform) ParseCallbackData(data []byte) (*model.CallbackData, error) {
	// 回调为 JSON
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("lingshi callback invalid json: %v", err)
	}
	// 提取 sign 并转成 map[string]string 以复用签名
	signStr, _ := m["sign"].(string)
	strParams := make(map[string]string)
	for k, v := range m {
		if k == "sign" { continue }
		strParams[k] = fmt.Sprintf("%v", v)
	}

	// 通过 appId 找账号，进行验签
	appId := strParams["appId"]
	if appId == "" {
		return nil, fmt.Errorf("lingshi callback missing appId")
	}
	account, err := p.platformRepo.GetPlatformAccountByAccountName(appId)
	if err != nil {
		return nil, fmt.Errorf("get account by appId failed: %v", err)
	}
	if !signature.VerifyLingshiSign(strParams, signStr, account.AppSecret) {
		return nil, fmt.Errorf("lingshi callback sign verify failed")
	}

	// 状态映射
	code := strParams["code"]
	status := "processing"
	if code == "0" || strings.EqualFold(code, "SUCCESS") || code == "0000" || code == "200" {
		status = "success"
	} else if code == "2" || strings.EqualFold(code, "FAIL") || strings.EqualFold(strParams["msg"], "failed") {
		status = "failed"
	}

	return &model.CallbackData{
		OrderID:       strParams["systemOrderId"],      // 平台订单号
		OrderNumber:   strParams["orderId"],            // 商户订单号
		Status:        status,
		Message:       strParams["msg"],
		CallbackType:  "order_status",
		Amount:        strParams["faceValue"],
		Sign:          signStr,
		Timestamp:     strParams["timestamp"],
		TransactionID: strParams["systemOrderId"],
	}, nil
}

// QueryBalance 查询账户余额
func (p *LingshiPlatform) QueryBalance(ctx context.Context, accountID int64) (float64, error) {
	appId, appSecret, baseURL, err := p.getKeySecretAndAPI(ctx, "lingshi", accountID)
	if err != nil { return 0, err }
	params := map[string]string{
		"appId": appId,
		"timestamp": strconv.FormatInt(time.Now().Unix(), 10),
	}
	params["sign"] = signature.GenerateLingshiSign(params, appSecret)

	b, _ := json.Marshal(params)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/balance", bytes.NewReader(b))
	if err != nil { return 0, err }
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil { return 0, err }
	defer resp.Body.Close()
    respBody, _ := io.ReadAll(resp.Body)
    clog := logger.WithContextCategory(ctx, "recharge")
    bodyStr := string(respBody)
    preview := bodyStr
    if len(bodyStr) > 512 {
        preview = bodyStr[:512] + "..."
    }
    clog.Info("余额查询响应",
        logger.StringV2("platform", "lingshi"),
        logger.IntV2("status", resp.StatusCode),
        logger.StringV2("body_preview", preview),
    )

	var r struct { Code string `json:"code"`; Msg string `json:"msg"`; Balance string `json:"balance"`; Credit string `json:"credit"` }
	_ = json.Unmarshal(respBody, &r)
	if r.Code != "0" && r.Code != "0000" && r.Code != "SUCCESS" && r.Code != "200" {
		return 0, fmt.Errorf("lingshi balance failed: %s", r.Msg)
	}
	bal, err := strconv.ParseFloat(r.Balance, 64)
	if err != nil { return 0, fmt.Errorf("parse balance failed: %v", err) }
	return bal, nil
}