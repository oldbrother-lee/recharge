package recharge

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"recharge-go/internal/model"
	"recharge-go/internal/repository"
    "recharge-go/internal/signature"
    logger "recharge-go/pkg/log"

	"gorm.io/gorm"
)

// Payc2Platform payc2充值平台
type Payc2Platform struct {
	platformRepo repository.PlatformRepository
	orderRepo    repository.OrderRepository
	signer       *signature.Payc2Handler
}

// Payc2Response API响应结构
type Payc2Response struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
	Datas   struct {
		UID     string `json:"uid"`
		OrderNo string `json:"orderNo"`
	} `json:"datas"`
}

// NewPayc2Platform 创建payc2平台实例
func NewPayc2Platform(db *gorm.DB) *Payc2Platform {
	return &Payc2Platform{
		platformRepo: repository.NewPlatformRepository(db),
		orderRepo:    repository.NewOrderRepository(db),
		signer:       signature.NewPayc2Handler(&signature.Config{}),
	}
}

// GetName 获取平台名称
func (p *Payc2Platform) GetName() string {
	return "payc2"
}

// getAPIConfig 获取API配置信息
func (p *Payc2Platform) getAPIConfig(ctx context.Context, accountID int64) (string, string, error) {
	account, err := p.platformRepo.GetAccountByID(ctx, accountID)
	if err != nil {
		return "", "", fmt.Errorf("获取平台账号信息失败: %v", err)
	}
	return account.AccountName, account.AppKey, nil
}

// SubmitOrder 提交订单
func (p *Payc2Platform) SubmitOrder(ctx context.Context, order *model.Order, api *model.PlatformAPI, apiParam *model.PlatformAPIParam) error {
    // 注入订单号到上下文并使用 v2 recharge 类别日志
    ctx = logger.InjectOrderNumber(ctx, order.OrderNumber)
    clog := logger.WithContextCategory(ctx, "recharge")
    clog.Info("开始提交订单",
        logger.StringV2("platform", "payc2"),
        logger.StringV2("order_number", order.OrderNumber),
        logger.Int64V2("account_id", api.AccountID),
        logger.StringV2("api_code", api.Code),
    )

	// 获取API配置
	merchID, secretKey, err := p.getAPIConfig(ctx, api.AccountID)
	if err != nil {
		return fmt.Errorf("获取API配置失败: %v", err)
	}

	// 更新签名处理器配置
	p.signer = signature.NewPayc2Handler(&signature.Config{
		AppID:     merchID,
		AppSecret: secretKey,
	})

	// 构建请求参数
	params, err := p.signer.BuildRequestParams(ctx, order, api)
	if err != nil {
		return fmt.Errorf("构建请求参数失败: %v", err)
	}

	// 构造表单数据
	form := url.Values{}
	for k, v := range params {
		form.Set(k, fmt.Sprintf("%v", v))
	}

	// 发送HTTP请求
	client := &http.Client{
		Timeout: time.Duration(api.Timeout) * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, "POST", api.URL+"/apis/wof/order/create_phone", strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

    // 仅记录参数键，避免记录大量或敏感内容
    paramKeys := make([]string, 0, len(params))
    for k := range params {
        paramKeys = append(paramKeys, k)
    }
    sort.Strings(paramKeys)
    clog.Info("发送请求",
        logger.StringV2("url", api.URL+"/apis/wof/order/create_phone"),
        logger.StringV2("method", "POST"),
        logger.AnyV2("param_keys", paramKeys),
    )

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %v", err)
	}

    bodyStr := string(body)
    preview := bodyStr
    if len(bodyStr) > 512 {
        preview = bodyStr[:512] + "..."
    }
    clog.Info("收到响应",
        logger.IntV2("status", resp.StatusCode),
        logger.StringV2("body_preview", preview),
    )

	// 检查HTTP状态码
	if resp.StatusCode != 200 {
		return fmt.Errorf("API请求失败: HTTP %d - %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var response Payc2Response
	if err := json.Unmarshal(body, &response); err != nil {
		// 如果不是JSON格式，可能是错误信息
		return fmt.Errorf("API响应格式错误: %s", string(body))
	}

	// 检查响应结果
	if response.ErrCode != 0 {
		return fmt.Errorf("API返回错误: %s (错误码: %d)", response.ErrMsg, response.ErrCode)
	}

	// 更新订单信息
	order.APIOrderNumber = response.Datas.UID
	order.APITradeNum = response.Datas.OrderNo
	order.Status = model.OrderStatusRecharging

    clog.Info("订单提交成功",
        logger.StringV2("order_number", order.OrderNumber),
        logger.StringV2("api_order_id", response.Datas.UID),
        logger.StringV2("api_trade_no", response.Datas.OrderNo),
    )
    return nil
}

// QueryOrderStatus 查询订单状态
func (p *Payc2Platform) QueryOrderStatus(ctx context.Context, order *model.Order) (model.OrderStatus, error) {
    // payc2平台通常通过回调通知订单状态，这里可以实现主动查询逻辑
    // 如果平台提供查询接口，可以在这里实现
    ctx = logger.InjectOrderNumber(ctx, order.OrderNumber)
    clog := logger.WithContextCategory(ctx, "recharge")
    clog.Info("查询订单状态",
        logger.StringV2("platform", "payc2"),
    )

	// 暂时返回当前状态，实际应该调用平台查询接口
	return order.Status, nil
}

// verifyCallbackSignature 验证回调签名
func (p *Payc2Platform) verifyCallbackSignature(ctx context.Context, params map[string]interface{}, orderNo string) bool {
    // 注入订单号以便全链路日志追踪
    ctx = logger.InjectOrderNumber(ctx, orderNo)
    clog := logger.WithContextCategory(ctx, "recharge")

    order, err := p.orderRepo.GetByOrderNumber(ctx, orderNo)
    if err != nil {
        clog.Error("获取订单信息失败",
            logger.StringV2("order_number", orderNo),
            logger.ErrorV2(err),
        )
        return false
    }

    account, err := p.platformRepo.GetAccountByID(ctx, order.PlatformAccountID)
    if err != nil {
        clog.Error("获取平台账号信息失败",
            logger.Int64V2("account_id", order.PlatformAccountID),
            logger.StringV2("order_number", orderNo),
            logger.ErrorV2(err),
        )
        return false
    }

	// 3. 获取接收到的签名
    receivedSign, ok := params["sign"].(string)
    if !ok {
        clog.Error("回调数据中缺少签名",
            logger.StringV2("order_number", orderNo),
        )
        return false
    }

	// 4. 移除签名字段后重新计算签名
	delete(params, "sign")

	// 5. 使用正确的AppSecret生成签名
	expectedSign := p.generateSignatureWithSecret(params, account.AppSecret)

    // 不记录 appSecret，避免泄露敏感信息
    clog.Info("验证回调签名",
        logger.StringV2("order_number", orderNo),
        logger.Int64V2("account_id", order.PlatformAccountID),
        logger.StringV2("received_sign", receivedSign),
        logger.StringV2("expected_sign", expectedSign),
    )

	return strings.EqualFold(receivedSign, expectedSign)
}

// generateSignatureWithSecret 使用指定的AppSecret生成签名
func (p *Payc2Platform) generateSignatureWithSecret(params map[string]interface{}, appSecret string) string {
	// 1. 获取所有参数键并排序（ASCII码升序）
	var keys []string
	for k := range params {
		if k == "sign" {
			continue // 签名字段不参与签名
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
	builder.WriteString(fmt.Sprintf("&key=%s", appSecret))
	signStr := builder.String()

	// 4. MD5运算
	hash := md5.New()
	hash.Write([]byte(signStr))
	sign := hex.EncodeToString(hash.Sum(nil))

    // 使用 recharge 类别日志记录签名生成信息，避免记录完整参数内容
    clog := logger.GetCategoryLogger("recharge")
    paramKeys := make([]string, 0, len(params))
    for k := range params {
        if k == "sign" {
            continue
        }
        paramKeys = append(paramKeys, k)
    }
    sort.Strings(paramKeys)
    signStrPreview := signStr
    if len(signStrPreview) > 256 {
        signStrPreview = signStrPreview[:256] + "..."
    }
    clog.Info("生成签名",
        logger.AnyV2("param_keys", paramKeys),
        logger.StringV2("sign_string_preview", signStrPreview),
        logger.StringV2("sign", sign),
    )
    return sign
}

// ParseCallbackData 解析回调数据
func (p *Payc2Platform) ParseCallbackData(data []byte) (*model.CallbackData, error) {
    // 无上下文时使用 recharge 类别日志器
    clog := logger.GetCategoryLogger("recharge")
    dataPreview := string(data)
    if len(dataPreview) > 256 {
        dataPreview = dataPreview[:256] + "..."
    }
    clog.Info("解析回调数据",
        logger.IntV2("data_len", len(data)),
        logger.StringV2("data_preview", dataPreview),
    )

	// 解析表单数据
	values, err := url.ParseQuery(string(data))
	if err != nil {
		return nil, fmt.Errorf("解析回调数据失败: %v", err)
	}

	// 转换为map用于签名验证
	params := make(map[string]interface{})
	for k, v := range values {
		if len(v) > 0 {
			params[k] = v[0]
		}
	}

	// 获取订单号，用于查找对应的账号配置
	orderNo := values.Get("orderNo")
	if orderNo == "" {
		return nil, fmt.Errorf("回调数据中缺少订单号")
	}

	// 验证签名 - 动态获取AppSecret
	if !p.verifyCallbackSignature(context.Background(), params, orderNo) {
		return nil, fmt.Errorf("回调签名验证失败")
	}

	// 解析回调数据
	callbackData := &model.CallbackData{
		OrderNumber:   values.Get("orderNo"),
		OrderID:       values.Get("uid"),
		Amount:        values.Get("amount"), // 订单金额
		Status:        "success",            // 默认成功状态
		Message:       "",
		TransactionID: "payc2_" + values.Get("uid"),
	}

	return callbackData, nil
}

// QueryBalance 查询账户余额
func (p *Payc2Platform) QueryBalance(ctx context.Context, accountID int64) (float64, error) {
    // 如果平台提供余额查询接口，可以在这里实现
    clog := logger.WithContextCategory(ctx, "recharge")
    clog.Info("查询账户余额",
        logger.StringV2("platform", "payc2"),
        logger.Int64V2("account_id", accountID),
    )

    // 暂时返回0，实际应该调用平台余额查询接口
    return 0, fmt.Errorf("payc2平台暂不支持余额查询")
}
