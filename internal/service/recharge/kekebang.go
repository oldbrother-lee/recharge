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

// KekebangPlatform 可客帮平台
type KekebangPlatform struct {
	platformRepo repository.PlatformRepository
}

// NewKekebangPlatform 创建客帮平台实例
func NewKekebangPlatform(db *gorm.DB) *KekebangPlatform {
	return &KekebangPlatform{
		platformRepo: repository.NewPlatformRepository(db),
	}
}

// GetName 获取平台名称
func (p *KekebangPlatform) GetName() string {
	return "kekebang"
}

// getAPIKeyAndSecret 获取API密钥和密钥
func (p *KekebangPlatform) getAPIKeyAndSecret(accountID int64) (string, string, error) {
	account, err := p.platformRepo.GetAccountByID(context.Background(), accountID)
	if err != nil {
		return "", "", fmt.Errorf("获取平台账号信息失败: %v", err)
	}
	return account.AppKey, account.AppSecret, nil
}

// SubmitOrder 提交订单
func (p *KekebangPlatform) SubmitOrder(ctx context.Context, order *model.Order, api *model.PlatformAPI, apiParam *model.PlatformAPIParam) error {
	logger.Info(fmt.Sprintf("【开始提交可客帮订单】order_number: %s", order.OrderNumber))
	fmt.Printf("【开始提交可客帮订单】order_number: %+v\n", apiParam)
	//通过account_id 获取到 api_key 和 api_secret
	apiKey, apiSecret, err := p.getAPIKeyAndSecret(api.AccountID)
	if err != nil {
		return fmt.Errorf("get api key and secret failed: %v", err)
	}

	// 构建请求参数
	fmt.Printf("【开始提交可客帮订单】api 信息: %+v\n", api)
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
		logger.Error(fmt.Sprintf("【提交订单失败】order_id: %s, error: %v", order.OrderNumber, err))
		return fmt.Errorf("submit order failed: %v", err)
	}

	// 确保 Code 是字符串类型
	code := fmt.Sprintf("%v", resp.Code)
	if code != "00000" {
		logger.Error(fmt.Sprintf("【提交订单失败】order_id: %s, code: %s, message: %s",
			order.OrderNumber, code, resp.Message))
		return fmt.Errorf("submit order failed: %s", resp.Message)
	}

	logger.Info(fmt.Sprintf("【kekebang提交订单成功】order_id: %s", order.OrderNumber))
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
		logger.Info("【订单状态】充值中, order_id: %d, order_number: %s", orderID, orderNumber)
	case 2:
		status = int(model.OrderStatusSuccess) // 成功
		statusStr = strconv.Itoa(status)
		logger.Info("【订单状态】充值成功, order_id: %d, order_number: %s", orderID, orderNumber)
	case 3:
		status = int(model.OrderStatusFailed) // 失败
		statusStr = strconv.Itoa(status)
		logger.Info("【订单状态】充值失败, order_id: %d, order_number: %s", orderID, orderNumber)
	case 4:
		status = int(model.OrderStatusProcessing) // 处理中（异常状态）
		statusStr = strconv.Itoa(status)
		logger.Info("【订单状态】处理中, order_id: %d, order_number: %s", orderID, orderNumber)
	default:
		status = int(model.OrderStatusFailed) // 默认失败
		statusStr = strconv.Itoa(status)
		logger.Error("【订单状态】未知状态, order_id: %d, order_number: %s, order_state: %d",
			orderID, orderNumber, orderState)
	}

	return status, statusStr
}

// QueryOrderStatus 查询订单状态
func (p *KekebangPlatform) QueryOrderStatus(order *model.Order) (model.OrderStatus, error) {
	logger.Info("【开始查询可客帮订单状态】order_id: %d, order_number: %s", order.ID, order.OrderNumber)

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
	resp, err := p.sendRequest(context.Background(), order.PlatformURL+"/query-order", params)
	if err != nil {
		logger.Error("【查询订单状态失败】order_id: %d, order_number: %s, error: %v",
			order.ID, order.OrderNumber, err)
		return 0, fmt.Errorf("query order status failed: %v", err)
	}

	// 确保 Code 是字符串类型
	code := fmt.Sprintf("%v", resp.Code)
	if code != "00000" {
		logger.Error("【查询订单状态失败】order_id: %d, order_number: %s, code: %s, message: %s",
			order.ID, order.OrderNumber, code, resp.Message)
		return 0, fmt.Errorf("query order status failed: %s", resp.Message)
	}

	// 转换状态
	status, err := strconv.Atoi(resp.Status)
	if err != nil {
		return 0, fmt.Errorf("invalid status: %s", resp.Status)
	}

	status, _ = p.mapOrderState(status, order.ID, order.OrderNumber)

	logger.Info("【查询订单状态完成】order_id: %d, order_number: %s, status: %d",
		order.ID, order.OrderNumber, status)
	return model.OrderStatus(status), nil
}

// ParseCallbackData 解析回调数据
func (p *KekebangPlatform) ParseCallbackData(data []byte) (*model.CallbackData, error) {
	// 解析平台返回的数据
	resp := &KekebangCallbackResponse{}
	if err := json.Unmarshal(data, resp); err != nil {
		return nil, fmt.Errorf("parse callback data failed: %v", err)
	}

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

	// 解析响应
	var result KekebangResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response failed: %v", err)
	}

	return &result, nil
}

// KekebangResponse 可客帮响应
type KekebangResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	OrderID string `json:"order_id"`
	Status  string `json:"status"`
	Balance string `json:"balance"`
}

// KekebangCallbackResponse 可客帮回调响应
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
	logger.Info("开始查询可客帮账户余额",
		"account_id", accountID,
	)

	// 获取API密钥和密钥
	appKey, appSecret, err := p.getAPIKeyAndSecret(accountID)
	if err != nil {
		return 0, fmt.Errorf("获取API密钥失败: %v", err)
	}

	// 获取平台API信息
	api, err := p.platformRepo.GetPlatformByCode(ctx, "kekebang")
	if err != nil {
		return 0, fmt.Errorf("获取平台API信息失败: %v", err)
	}

	// 构建请求参数
	params := map[string]interface{}{
		"app_key":   appKey,
		"timestamp": strconv.FormatInt(time.Now().Unix(), 10),
		"biz_code":  "2", // 查询余额
	}

	// 使用客帮帮平台的签名方法
	sign := signature.GenerateKekebangSign(params, appSecret)
	params["sign"] = sign

	// 发送请求
	resp, err := p.sendRequest(ctx, api.URL+"/query-balance", params)
	if err != nil {
		return 0, fmt.Errorf("查询余额失败: %v", err)
	}

	// 确保 Code 是字符串类型
	code := fmt.Sprintf("%v", resp.Code)
	if code != "00000" {
		return 0, fmt.Errorf("查询余额失败: %s", resp.Message)
	}

	// 解析余额
	balance, err := strconv.ParseFloat(resp.Balance, 64)
	if err != nil {
		return 0, fmt.Errorf("解析余额失败: %v", err)
	}

	logger.Info("查询余额成功",
		"account_id", accountID,
		"balance", balance,
	)

	return balance, nil
}
