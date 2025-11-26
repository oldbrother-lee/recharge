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
    logger "recharge-go/pkg/log"
    "recharge-go/pkg/signature"

	"gorm.io/gorm"
)

// KasushouPlatform 卡速售平台
type KasushouPlatform struct {
	platformRepo repository.PlatformRepository
	orderRepo    repository.OrderRepository
}

// NewKasushouPlatform 创建卡速售平台实例
func NewKasushouPlatform(db *gorm.DB) *KasushouPlatform {
	return &KasushouPlatform{
		platformRepo: repository.NewPlatformRepository(db),
		orderRepo:    repository.NewOrderRepository(db),
	}
}

// GetName 获取平台名称
func (p *KasushouPlatform) GetName() string {
	return "kasushou"
}

// getAPIKeyAndSecret 获取API密钥和密钥
func (p *KasushouPlatform) getAPIKeyAndSecret(ctx context.Context, accountID int64) (string, string, string, error) {
    logger.WithContextCategory(ctx, "recharge").Info("【获取卡速售账号信息】", logger.Int64V2("account_id", accountID))
    account, err := p.platformRepo.GetAccountByID(ctx, accountID)
    if err != nil {
        logger.WithContextCategory(ctx, "recharge").Error("【获取平台账号失败】", logger.Int64V2("account_id", accountID), logger.ErrorV2(err))
        return "", "", "", fmt.Errorf("get platform account failed: %v", err)
    }
    return account.AppKey, account.AppSecret, account.AccountName, nil
}

// SubmitOrder 提交订单到卡速售平台
func (p *KasushouPlatform) SubmitOrder(ctx context.Context, order *model.Order, api *model.PlatformAPI, apiParam *model.PlatformAPIParam) error {
    logger.WithContextCategory(ctx, "recharge").Info("【开始提交卡速售订单】",
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

	// 生成时间戳（13位毫秒）
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)

	// 构建请求Body参数
    logger.WithContextCategory(ctx, "recharge").Info("【卡速售构建请求参数】", logger.StringV2("url", api.URL))
	body := map[string]interface{}{
		"id":               mustParseInt(apiParam.ProductID),
		"url":              callbackURL,
		"external_orderno": order.OrderNumber,
		// "safe_price":       strconv.FormatFloat(apiParam.Price, 'f', 2, 64),
		// "mark":             order.Remark,
		"quantity": 1, // 默认数量为1，可根据需要调整
	}

	// 构建attach参数（手工订单模板参数）
	// attach := map[string]interface{}{
	// 	"recharge_account": order.Mobile,
	// }
	// if order.Param1 != "" {
	// 	attach["lblName1"] = order.Param1
	// }
	// if order.Param2 != "" {
	// 	attach["lblName2"] = order.Param2
	// }
	// if order.Param3 != "" {
	// 	attach["lblName3"] = order.Param3
	// }
	// if len(attach) > 1 { // 如果有除 recharge_account 外的其他参数
	// 	body["attach"] = attach
	// }

	// 生成签名
    sign := signature.GenerateKasushouSign(body, timestamp, apiKey)
    logger.WithContextCategory(ctx, "recharge").Info("【卡速售生成签名】", logger.StringV2("userId", userId))
	// 构建请求头
	headers := map[string]string{
		"Content-Type": "application/json; charset=utf-8",
		"Sign":         sign,
		"Timestamp":    timestamp,
		"UserId":       "ibSloEwegXjK2M73IN5xy4UnOdsBp1qC",
	}

	// 发送请求
	//打印下body
    logger.WithContextCategory(ctx, "recharge").Info("【卡速售提交订单请求参数】", logger.AnyV2("body", body))
    resp, err := p.sendRequest(ctx, api.URL+"/api/v1/order/buy", body, headers)
    if err != nil {
        logger.WithContextCategory(ctx, "recharge").Error("【提交卡速售订单失败】",
            logger.StringV2("order_number", order.OrderNumber),
            logger.ErrorV2(err),
        )
        return fmt.Errorf("submit order failed: %v", err)
    }

	// 检查响应
    if resp.Code != 200 {
        logger.WithContextCategory(ctx, "recharge").Error("【提交卡速售订单失败】",
            logger.StringV2("order_number", order.OrderNumber),
            logger.IntV2("code", resp.Code),
            logger.StringV2("message", resp.Msg),
        )
        return fmt.Errorf("submit order failed: %s", resp.Msg)
    }

    logger.WithContextCategory(ctx, "recharge").Info("【卡速售提交订单成功】",
        logger.StringV2("order_number", order.OrderNumber),
        logger.StringV2("platform_order_id", resp.Data.OrderSN),
    )
    return nil
}

// QueryOrderStatus 查询订单状态（当前卡速售平台主要通过回调更新状态，此方法保留接口兼容性）
func (p *KasushouPlatform) QueryOrderStatus(ctx context.Context, order *model.Order) (model.OrderStatus, error) {
    logger.WithContextCategory(ctx, "recharge").Info("【查询卡速售订单状态】",
        logger.Int64V2("order_id", order.ID),
        logger.StringV2("order_number", order.OrderNumber),
    )

	// 卡速售主要通过异步回调来通知订单状态变更
	// 此处返回当前状态，不做实际查询
	return order.Status, nil
}

// mapOrderState 映射订单状态
// 卡速售状态码：
// 2：正在处理
// 3：已完成
// 4：取消交易
// 5：已退款
func (p *KasushouPlatform) mapOrderState(status string, orderNumber string) (int, string) {
	var orderStatus int
	var statusStr string

	switch status {
	case "2":
		orderStatus = int(model.OrderStatusRecharging) // 充值中
		statusStr = strconv.Itoa(orderStatus)
        logger.GetCategoryLogger("recharge").Info("【卡速售订单状态】正在处理", logger.StringV2("order_number", orderNumber))
	case "3":
		orderStatus = int(model.OrderStatusSuccess) // 成功
		statusStr = strconv.Itoa(orderStatus)
        logger.GetCategoryLogger("recharge").Info("【卡速售订单状态】充值成功", logger.StringV2("order_number", orderNumber))
	case "4":
		orderStatus = int(model.OrderStatusCancelled) // 取消交易
		statusStr = strconv.Itoa(orderStatus)
        logger.GetCategoryLogger("recharge").Info("【卡速售订单状态】取消交易", logger.StringV2("order_number", orderNumber))
	case "5":
		orderStatus = int(model.OrderStatusRefunded) // 已退款
		statusStr = strconv.Itoa(orderStatus)
        logger.GetCategoryLogger("recharge").Info("【卡速售订单状态】已退款", logger.StringV2("order_number", orderNumber))
	default:
		orderStatus = int(model.OrderStatusFailed) // 默认失败
		statusStr = strconv.Itoa(orderStatus)
        logger.GetCategoryLogger("recharge").Info("【卡速售订单状态】未知状态，设为失败", logger.StringV2("order_number", orderNumber), logger.StringV2("status", status))
	}

	return orderStatus, statusStr
}

// ParseCallbackData 解析回调数据
func (p *KasushouPlatform) ParseCallbackData(data []byte) (*model.CallbackData, error) {
	// 先解析为通用 map 用于签名校验
	var post map[string]interface{}
	if err := json.Unmarshal(data, &post); err != nil {
		return nil, fmt.Errorf("parse callback raw failed: %v", err)
	}

	// 提取外部订单号，用于定位订单和账号信息
	externalNo := fmt.Sprintf("%v", post["external_orderno"])
	if externalNo == "" {
		return nil, fmt.Errorf("缺少外部订单号external_orderno")
	}

	// 获取订单以拿到平台账号密钥
	order, err := p.orderRepo.GetByOrderNumber(context.Background(), externalNo)
	if err != nil {
		return nil, fmt.Errorf("获取订单失败: %v", err)
	}
	account, err := p.platformRepo.GetAccountByID(context.Background(), order.PlatformAccountID)
	if err != nil {
		return nil, fmt.Errorf("获取平台账号失败: %v", err)
	}

	// 验证签名
	if !signature.VerifyKasushouCallback(post, account.AppKey) {
		return nil, fmt.Errorf("回调签名验证失败")
	}

	// 解析结构体便于后续字段映射
	var resp KasushouCallbackResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parse callback data failed: %v", err)
	}

	// 转换订单状态（以外部订单号记录日志）
	_, statusStr := p.mapOrderState(resp.Status, resp.ExternalOrderNo)

	return &model.CallbackData{
		OrderID:       resp.OrderSN,
		OrderNumber:   resp.ExternalOrderNo,
		Status:        statusStr,
		Message:       resp.RechargeHints,
		CallbackType:  "order_status",
		Amount:        resp.TotalPrice,
		Sign:          resp.Sign,
		Timestamp:     resp.Time,
		TransactionID: "kasushou_" + resp.OrderSN,
	}, nil
}

// QueryBalance 查询余额
func (p *KasushouPlatform) QueryBalance(ctx context.Context, accountID int64) (float64, error) {
	// 卡速售平台暂不支持余额查询接口
	return 0.0, fmt.Errorf("balance query not supported by kasushou platform")
}

// sendRequest 发送请求
func (p *KasushouPlatform) sendRequest(ctx context.Context, url string, body map[string]interface{}, headers map[string]string) (*KasushouResponse, error) {
	// 将参数转换为JSON
	jsonData, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal body failed: %v", err)
	}

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request failed: %v", err)
	}

	// 设置请求头
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// 发送请求
	client := &http.Client{Timeout: 30 * time.Second}
	httpResp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request failed: %v", err)
	}
	defer httpResp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response failed: %v", err)
	}

	// 解析响应
	var resp KasushouResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal response failed: %v", err)
	}

	return &resp, nil
}

// mustParseInt 解析字符串为整数，失败返回0
func mustParseInt(s string) int {
	if i, err := strconv.Atoi(s); err == nil {
		return i
	}
	return 0
}

// KasushouResponse 卡速售下单响应
type KasushouResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		OrderSN         string `json:"ordersn"`
		ExternalOrderNo string `json:"external_orderno"`
	} `json:"data"`
}

// KasushouCallbackResponse 卡速售回调响应
type KasushouCallbackResponse struct {
	ExternalOrderNo string `json:"external_orderno"` // 外部订单号
	OrderSN         string `json:"ordersn"`          // 本地订单号
	Status          string `json:"status"`           // 订单状态
	HasBackMoney    string `json:"has_back_money"`   // 退款金额
	TotalPrice      string `json:"total_price"`      // 下单金额
	RechargeHints   string `json:"recharge_hints"`   // 订单处理返回信息
	Time            string `json:"time"`             // 13位毫秒时间戳
	Sign            string `json:"sign"`             // 签名
	CardList        []any  `json:"card_list"`        // 卡密信息（不参与签名）
	ExpressList     []any  `json:"express_list"`     // 物流信息（不参与签名）
}
