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
	"recharge-go/pkg/logger"
	"recharge-go/pkg/signature"

	"gorm.io/gorm"
)

// KekebangPlatform 可客帮平台
type KekebangPlatform struct {
	platformRepo repository.PlatformRepository
	orderRepo    repository.OrderRepository
}

// NewKekebangPlatform 创建客帮平台实例
func NewKekebangPlatform(db *gorm.DB) *KekebangPlatform {
	return &KekebangPlatform{
		platformRepo: repository.NewPlatformRepository(db),
		orderRepo:    repository.NewOrderRepository(db),
	}
}

// GetName 获取平台名称
func (p *KekebangPlatform) GetName() string {
	return "kekebang"
}

// getAPIKeyAndSecret 获取API密钥和密钥
func (p *KekebangPlatform) getAPIKeyAndSecret(ctx context.Context, accountID int64) (string, string, error) {
	account, err := p.platformRepo.GetAccountByID(ctx, accountID)
	if err != nil {
		return "", "", fmt.Errorf("获取平台账号信息失败: %v", err)
	}
	return account.AppKey, account.AppSecret, nil
}

// SubmitOrder 提交订单
func (p *KekebangPlatform) SubmitOrder(ctx context.Context, order *model.Order, api *model.PlatformAPI, apiParam *model.PlatformAPIParam) error {
	l := logger.WithContextCategory(ctx, "recharge")
	if l != nil {
		l.Info("【开始提交可客帮订单】",
			logger.StringV2("order_number", order.OrderNumber),
			logger.Int64V2("api_id", api.ID),
			logger.Int64V2("platform_id", api.PlatformID),
			logger.Int64V2("account_id", api.AccountID),
			logger.Int64V2("param_id", apiParam.ID),
			logger.StringV2("product_id", apiParam.ProductID),
		)
	}
	//通过account_id 获取到 api_key 和 api_secret
	apiKey, apiSecret, err := p.getAPIKeyAndSecret(ctx, api.AccountID)
	if err != nil {
		return fmt.Errorf("get api key and secret failed: %v", err)
	}

	// 构建请求参数
	if l != nil {
		l.Info("【kekebang 构建请求参数】",
			logger.StringV2("url", api.URL),
		)
	}
	params := map[string]interface{}{
		"app_key":    apiKey,
		"timestamp":  strconv.FormatInt(time.Now().Unix(), 10),
		"biz_code":   "1", // 充值业务
		"order_id":   order.OrderNumber,
		"sku_code":   apiParam.ProductID,
		"notify_url": order.PlatformCallbackURL,
		"data": map[string]string{
			"account": order.Mobile,
		},
	}

	// 使用客帮帮平台的签名方法
	sign := signature.GenerateKekebangSign(params, apiSecret)
	params["sign"] = sign

	// 发送请求
	resp, err := p.sendRequest(ctx, api.URL, params)
	if err != nil {
		if l != nil {
			l.Error("【提交订单失败】",
				logger.StringV2("order_number", order.OrderNumber),
				logger.ErrorV2(err),
			)
		}
		return fmt.Errorf("submit order failed: %v", err)
	}

	// 确保 Code 是字符串类型
	code := fmt.Sprintf("%v", resp.Code)
	if code != "00000" {
		if l != nil {
			l.Error("【提交订单失败】",
				logger.StringV2("order_number", order.OrderNumber),
				logger.StringV2("code", code),
				logger.StringV2("message", resp.Message),
			)
		}
		return fmt.Errorf("submit order failed: %s", resp.Message)
	}

	if l != nil {
		l.Info("【kekebang提交订单成功】",
			logger.StringV2("order_number", order.OrderNumber),
		)
	}
	return nil
}

// mapOrderState 映射订单状态
// kekebang状态码：
// 1：处理中
// 2：成功
// 3：失败
// 4：异常（需人工核实）
func (p *KekebangPlatform) mapOrderState(orderState int, orderID int64, orderNumber string) (int, string) {
	var status int
	var statusStr string

	switch orderState {
	case 1:
		status = int(model.OrderStatusRecharging) // 充值中
		statusStr = strconv.Itoa(status)
		logger.GetCategoryLogger("recharge").Info("【订单状态】充值中",
			logger.Int64V2("order_id", orderID),
			logger.StringV2("order_number", orderNumber),
		)
	case 2:
		status = int(model.OrderStatusSuccess) // 成功
		statusStr = strconv.Itoa(status)
		logger.GetCategoryLogger("recharge").Info("【订单状态】充值成功",
			logger.Int64V2("order_id", orderID),
			logger.StringV2("order_number", orderNumber),
		)
	case 3:
		status = int(model.OrderStatusFailed) // 失败
		statusStr = strconv.Itoa(status)
		logger.GetCategoryLogger("recharge").Info("【订单状态】充值失败",
			logger.Int64V2("order_id", orderID),
			logger.StringV2("order_number", orderNumber),
		)
	case 4:
		status = int(model.OrderStatusProcessing) // 处理中（异常状态）
		statusStr = strconv.Itoa(status)
		logger.GetCategoryLogger("recharge").Info("【订单状态】处理中",
			logger.Int64V2("order_id", orderID),
			logger.StringV2("order_number", orderNumber),
		)
	default:
		status = int(model.OrderStatusFailed) // 默认失败
		statusStr = strconv.Itoa(status)
		logger.GetCategoryLogger("recharge").Error("【订单状态】未知状态",
			logger.Int64V2("order_id", orderID),
			logger.StringV2("order_number", orderNumber),
			logger.IntV2("order_state", orderState),
		)
	}

	return status, statusStr
}

// QueryOrderStatus 查询订单状态
func (p *KekebangPlatform) QueryOrderStatus(ctx context.Context, order *model.Order) (model.OrderStatus, error) {
	l := logger.WithContextCategory(ctx, "recharge")
	if l != nil {
		l.Info("【开始查询可客帮订单状态】",
			logger.Int64V2("order_id", order.ID),
			logger.StringV2("order_number", order.OrderNumber),
		)
	}

	// 构建请求参数
	params := map[string]interface{}{
		"app_key":   order.PlatformAppKey,
		"timestamp": strconv.FormatInt(time.Now().Unix(), 10),
		"biz_code":  "1", // 查询订单状态
		"order_id":  order.OrderNumber,
	}

	// 使用客帮帮平台的签名方法
	sign := signature.GenerateKekebangSign(params, order.PlatformSecretKey)
	params["sign"] = sign

	// 发送请求
	resp, err := p.sendRequest(ctx, order.PlatformURL+"/query-order", params)
	if err != nil {
		if l != nil {
			l.Error("【查询订单状态失败】",
				logger.Int64V2("order_id", order.ID),
				logger.StringV2("order_number", order.OrderNumber),
				logger.ErrorV2(err),
			)
		}
		return 0, fmt.Errorf("query order status failed: %v", err)
	}

	// 确保 Code 是字符串类型
	code := fmt.Sprintf("%v", resp.Code)
	if code != "00000" {
		if l != nil {
			l.Error("【查询订单状态失败】",
				logger.Int64V2("order_id", order.ID),
				logger.StringV2("order_number", order.OrderNumber),
				logger.StringV2("code", code),
				logger.StringV2("message", resp.Message),
			)
		}
		return 0, fmt.Errorf("query order status failed: %s", resp.Message)
	}

	// 转换状态
	status, err := strconv.Atoi(resp.Status)
	if err != nil {
		return 0, fmt.Errorf("invalid status: %s", resp.Status)
	}

	status, _ = p.mapOrderState(status, order.ID, order.OrderNumber)

	if l != nil {
		l.Info("【查询订单状态完成】",
			logger.Int64V2("order_id", order.ID),
			logger.StringV2("order_number", order.OrderNumber),
			logger.IntV2("status", status),
		)
	}
	return model.OrderStatus(status), nil
}

// ParseCallbackData 解析回调数据
func (p *KekebangPlatform) ParseCallbackData(data []byte) (*model.CallbackData, error) {
	// 解析平台返回的数据到 map 用于签名校验
	var post map[string]interface{}
	if err := json.Unmarshal(data, &post); err != nil {
		return nil, fmt.Errorf("parse callback raw failed: %v", err)
	}

	// 结构化解析，便于字段映射
	resp := &KekebangCallbackResponse{}
	if err := json.Unmarshal(data, resp); err != nil {
		return nil, fmt.Errorf("parse callback data failed: %v", err)
	}

	// 根据订单号获取平台账号密钥
	order, err := p.orderRepo.GetByOrderNumber(context.Background(), resp.OrderID)
	if err != nil {
		return nil, fmt.Errorf("获取订单失败: %v", err)
	}
	account, err := p.platformRepo.GetAccountByID(context.Background(), order.PlatformAccountID)
	if err != nil {
		return nil, fmt.Errorf("获取平台账号失败: %v", err)
	}

	// 验证签名（排除 sign 和 data）
	if !signature.VerifyKekebangSign(post, resp.Sign, account.AppSecret) {
		return nil, fmt.Errorf("回调签名验证失败")
	}
	fmt.Println(account)

	// 转换订单状态
	_, statusStr := p.mapOrderState(resp.OrderState, 0, resp.OrderID)

	return &model.CallbackData{
		OrderID:       resp.TerraceID,
		OrderNumber:   resp.OrderID,
		Status:        statusStr, //订单状态
		Message:       resp.ReturnMsg,
		CallbackType:  "order_status",
		Amount:        strconv.FormatFloat(resp.Amount, 'f', 2, 64),
		Sign:          resp.Sign,
		Timestamp:     resp.Time,
		TransactionID: resp.Proof,
	}, nil
}

// sendRequest 发送请求
func (p *KekebangPlatform) sendRequest(ctx context.Context, url string, params map[string]interface{}) (*KekebangResponse, error) {
	// 将参数转换为 JSON
	jsonData, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("marshal params failed: %v", err)
	}

	stage := "submit"
	if strings.Contains(strings.ToLower(url), "query") {
		stage = "query"
	}
	logger.WithContextCategory(ctx, "recharge").Info("kekebang 请求体",
		logger.StringV2("stage", stage),
		logger.StringV2("url", url),
		logger.StringV2("body", string(jsonData)),
	)

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request failed: %v", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	client := &http.Client{}
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
	logger.WithContextCategory(ctx, "recharge").Info("kekebang 响应",
		logger.StringV2("stage", "response"),
		logger.StringV2("url", url),
		logger.IntV2("status_code", resp.StatusCode),
		logger.StringV2("body", string(body)),
	)

	// 解析响应
	var response KekebangResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("unmarshal response failed: %v", err)
	}

	return &response, nil
}

// KekebangResponse 响应结构
type KekebangResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	OrderID string `json:"order_id"`
	Status  string `json:"status"`
	Balance string `json:"balance"`
}

// KekebangCallbackResponse 回调结构
type KekebangCallbackResponse struct {
	OrderID    string  `json:"order_id"`
	TerraceID  string  `json:"terrace_id"`
	Account    string  `json:"account"`
	Time       string  `json:"time"`
	ReturnMsg  string  `json:"return_msg"`
	Amount     float64 `json:"amount"`
	Proof      string  `json:"proof"`
	CardNo     string  `json:"card_no"`
	OrderState int     `json:"order_state"`
	ErrorCode  int     `json:"error_code"`
	Sign       string  `json:"sign"`
}

// QueryBalance 查询账户余额
func (p *KekebangPlatform) QueryBalance(ctx context.Context, accountID int64) (float64, error) {
	l := logger.WithContextCategory(ctx, "recharge")
	if l != nil {
		l.Info("【开始查询可客帮账户余额】",
			logger.Int64V2("account_id", accountID),
		)
	}
	// TODO: 实现余额查询，如需
	return 0, fmt.Errorf("not implemented")
}
