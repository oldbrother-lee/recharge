package recharge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"recharge-go/internal/model"
	"recharge-go/internal/repository"
	"recharge-go/pkg/logger"
	"recharge-go/pkg/signature"
	"strconv"

	"gorm.io/gorm"
)

type DayuanrenPlatform struct {
	platformRepo repository.PlatformRepository
	orderRepo    repository.OrderRepository
}

func NewDayuanrenPlatform(db *gorm.DB) *DayuanrenPlatform {
	return &DayuanrenPlatform{
		platformRepo: repository.NewPlatformRepository(db),
		orderRepo:    repository.NewOrderRepository(db),
	}
}

func (p *DayuanrenPlatform) GetName() string {
	return "dayuanren"
}

// 定义 dayuanren 官方请求和响应结构体

type RechargeRequest struct {
	OutTradeNum string `json:"out_trade_num"`        // 商户订单号
	ProductID   string `json:"product_id"`           // 产品ID
	Mobile      string `json:"mobile"`               // 充值号码
	NotifyURL   string `json:"notify_url"`           // 回调地址
	UserID      string `json:"userid"`               // 商户ID
	Amount      string `json:"amount,omitempty"`     // 面值（可选）
	Price       string `json:"price,omitempty"`      // 最高成本（可选）
	Area        string `json:"area,omitempty"`       // 电费省份（可选）
	Ytype       string `json:"ytype,omitempty"`      // 电费验证三要素（可选）
	IDCardNo    string `json:"id_card_no,omitempty"` // 身份证后6位等（可选）
	City        string `json:"city,omitempty"`       // 地级市名（可选）
	Param1      string `json:"param1,omitempty"`     // 扩展参数1（可选）
	Param2      string `json:"param2,omitempty"`     // 扩展参数2（可选）
	Param3      string `json:"param3,omitempty"`     // 扩展参数3（可选）
}

type RechargeResponse struct {
	OrderNumber string `json:"order_number"`
	Mobile      string `json:"mobile"`
	ProductID   int    `json:"product_id"`
	TotalPrice  string `json:"total_price"`
	OutTradeNum string `json:"out_trade_num"`
	Title       string `json:"title"`
}

type Response struct {
	Errno  int             `json:"errno"`
	Errmsg string          `json:"errmsg"`
	Data   json.RawMessage `json:"data"`
}

// 工具函数：结构体转 map[string]string
func structToMap(req RechargeRequest) map[string]string {
	m := map[string]string{
		"out_trade_num": req.OutTradeNum,
		"product_id":    req.ProductID,
		"mobile":        req.Mobile,
		"notify_url":    req.NotifyURL,
		"userid":        req.UserID,
	}
	if req.Amount != "" {
		m["amount"] = req.Amount
	}
	if req.Price != "" {
		m["price"] = req.Price
	}
	if req.Area != "" {
		m["area"] = req.Area
	}
	if req.Ytype != "" {
		m["ytype"] = req.Ytype
	}
	if req.IDCardNo != "" {
		m["id_card_no"] = req.IDCardNo
	}
	if req.City != "" {
		m["city"] = req.City
	}
	if req.Param1 != "" {
		m["param1"] = req.Param1
	}
	if req.Param2 != "" {
		m["param2"] = req.Param2
	}
	if req.Param3 != "" {
		m["param3"] = req.Param3
	}
	return m
}

// SubmitOrder 提交订单
func (p *DayuanrenPlatform) SubmitOrder(ctx context.Context, order *model.Order, api *model.PlatformAPI, apiParam *model.PlatformAPIParam) error {
	logger.Info("开始提交大猿人订单", "order_id", order.ID, "order_number", order.OrderNumber, "mobile", order.Mobile)

	_, appSecret, accountName, err := p.getAPIKeyAndSecret(api.AccountID)
	if err != nil {
		logger.Error("获取API密钥失败", "order_id", order.ID, "order_number", order.OrderNumber, "error", err)
		return fmt.Errorf("获取API密钥失败: %v", err)
	}

	// 构建请求参数（如有可选参数可补充）
	req := RechargeRequest{
		OutTradeNum: order.OrderNumber,
		ProductID:   apiParam.ProductID,
		Mobile:      order.Mobile,
		NotifyURL:   api.CallbackURL,
		UserID:      accountName,
		// 可选参数可从 order 或 apiParam 取值
	}
	params := structToMap(req)
	params["sign"] = signature.GenerateDayuanrenSign(params, appSecret)

	form := url.Values{}
	for k, v := range params {
		form.Add(k, v)
	}

	logger.Info("大猿人请求参数", "order_id", order.ID, "order_number", order.OrderNumber, "url", api.URL+"/index/recharge", "form", form.Encode())

	resp, err := http.PostForm(api.URL+"/index/recharge", form)
	if err != nil {
		logger.Error("大猿人请求失败", "order_id", order.ID, "order_number", order.OrderNumber, "error", err, "url", api.URL+"/index/recharge", "form", form.Encode())
		return fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("读取大猿人响应失败", "order_id", order.ID, "order_number", order.OrderNumber, "error", err)
		return fmt.Errorf("读取响应失败: %v", err)
	}
	logger.Info("大猿人响应原文", "order_id", order.ID, "order_number", order.OrderNumber, "body", string(body))

	var respData Response
	if err := json.Unmarshal(body, &respData); err != nil {
		logger.Error("解析大猿人响应失败", "order_id", order.ID, "order_number", order.OrderNumber, "error", err, "body", string(body))
		return fmt.Errorf("解析响应失败: %v", err)
	}
	logger.Info("大猿人响应结构体", "order_id", order.ID, "order_number", order.OrderNumber, "respData", respData)

	if respData.Errno != 0 {
		logger.Error("大猿人API错误", "order_id", order.ID, "order_number", order.OrderNumber, "errno", respData.Errno, "errmsg", respData.Errmsg)
		return fmt.Errorf("API错误: %s", respData.Errmsg)
	}

	var rechargeResp RechargeResponse
	if err := json.Unmarshal(respData.Data, &rechargeResp); err != nil {
		logger.Error("解析大猿人data失败", "order_id", order.ID, "order_number", order.OrderNumber, "error", err, "data", string(respData.Data))
		return fmt.Errorf("解析数据失败: %v", err)
	}
	logger.Info("大猿人充值响应结构体", "order_id", order.ID, "order_number", order.OrderNumber, "rechargeResp", rechargeResp)

	logger.Info("提交大猿人订单成功", "order_id", order.ID, "order_number", order.OrderNumber, "api_order_id", rechargeResp.OrderNumber)
	return nil
}

// QueryOrderStatus 查询订单状态
func (p *DayuanrenPlatform) QueryOrderStatus(order *model.Order) (model.OrderStatus, error) {
	_, appSecret, accountName, err := p.getAPIKeyAndSecret(order.PlatformAccountID)
	if err != nil {
		return 0, fmt.Errorf("获取API密钥失败: %v", err)
	}

	// 获取平台API信息
	api, err := p.platformRepo.GetPlatformByCode(context.Background(), "dayuanren")
	if err != nil {
		return 0, fmt.Errorf("获取平台API信息失败: %v", err)
	}

	params := map[string]string{
		"userid":         accountName,
		"out_trade_nums": order.OrderNumber,
	}
	sign := signature.GenerateDayuanrenSign(params, appSecret)
	params["sign"] = sign

	form := url.Values{}
	for k, v := range params {
		form.Add(k, v)
	}

	resp, err := http.PostForm(api.URL+"/index/check", form)
	if err != nil {
		return 0, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	var respData struct {
		Errno  int             `json:"errno"`
		Errmsg string          `json:"errmsg"`
		Data   json.RawMessage `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
		return 0, fmt.Errorf("解析响应失败: %v", err)
	}
	if respData.Errno != 0 {
		return 0, fmt.Errorf("API错误: %s", respData.Errmsg)
	}

	var orders []struct {
		State int `json:"state"`
	}
	if err := json.Unmarshal(respData.Data, &orders); err != nil {
		return 0, fmt.Errorf("解析订单状态失败: %v", err)
	}
	if len(orders) == 0 {
		return 0, errors.New("未查询到订单")
	}

	switch orders[0].State {
	case -1:
		return model.OrderStatusFailed, nil
	case 0:
		return model.OrderStatusRecharging, nil
	case 1:
		return model.OrderStatusSuccess, nil
	case 2:
		return model.OrderStatusFailed, nil
	case 3:
		return model.OrderStatusRecharging, nil // 没有 PartialSuccess，暂用 Recharging
	default:
		return 0, nil // 没有 OrderStatusUnknown，返回 0
	}
}

// dayuanren 平台订单状态映射
func (p *DayuanrenPlatform) mapOrderState(state int, orderID string) (int, string) {
	var status int
	var statusStr string

	switch state {
	case -1:
		status = int(model.OrderStatusFailed) // 失败
		statusStr = strconv.Itoa(status)
		logger.Info("【大猿人订单状态】失败", "order_id", orderID)
	case 0:
		status = int(model.OrderStatusProcessing) // 处理中
		statusStr = strconv.Itoa(status)
		logger.Info("【大猿人订单状态】处理中", "order_id", orderID)
	case 1:
		status = int(model.OrderStatusSuccess) // 成功
		statusStr = strconv.Itoa(status)
		logger.Info("【大猿人订单状态】成功", "order_id", orderID)
	case 2:
		status = int(model.OrderStatusFailed) // 失败
		statusStr = strconv.Itoa(status)
		logger.Info("【大猿人订单状态】失败", "order_id", orderID)
	case 3:
		status = int(model.OrderStatusProcessing) // 部分成功/处理中
		statusStr = strconv.Itoa(status)
		logger.Info("【大猿人订单状态】部分成功/处理中", "order_id", orderID)
	default:
		status = int(model.OrderStatusFailed) // 默认失败
		statusStr = strconv.Itoa(status)
		logger.Error("【大猿人订单状态】未知状态", "order_id", orderID, "state", state)
	}
	return status, statusStr
}

// ParseCallbackData 解析回调数据
func (p *DayuanrenPlatform) ParseCallbackData(data []byte) (*model.CallbackData, error) {
	logger.Info("开始解析大猿人回调数据", "data", string(data))
	// 解析 url.Values
	form, err := url.ParseQuery(string(data))
	if err != nil {
		logger.Error("大猿人回调参数解析失败", "error", err, "data", string(data))
		return nil, errors.New("回调参数解析失败")
	}
	params := make(map[string]string)
	for k, v := range form {
		if len(v) > 0 {
			params[k] = v[0]
		}
	}
	logger.Info("大猿人回调参数", "params", params)

	state, _ := strconv.Atoi(params["state"])
	_, statusStr := p.mapOrderState(state, params["out_trade_num"])

	callbackData := &model.CallbackData{
		OrderID:     params["out_trade_num"],
		Status:      statusStr,
		Message:     params["remark"],
		Amount:      params["charge_amount"],
		Sign:        params["sign"],
		Timestamp:   params["otime"],
		OrderNumber: params["out_trade_num"],
	}
	logger.Info("大猿人回调解析完成", "callbackData", callbackData)
	return callbackData, nil
}

// getAPIKeyAndSecret 获取API密钥和密钥
func (p *DayuanrenPlatform) getAPIKeyAndSecret(accountID int64) (string, string, string, error) {
	account, err := p.platformRepo.GetPlatformAccountByID(accountID)
	if err != nil {
		return "", "", "", fmt.Errorf("获取平台账号信息失败: %v", err)
	}
	return account.AppKey, account.AppSecret, account.AccountName, nil
}

// QueryBalance 查询账户余额
func (p *DayuanrenPlatform) QueryBalance(ctx context.Context, accountID int64) (float64, error) {
	return 0, errors.New("大猿人平台暂不支持余额查询")
}
