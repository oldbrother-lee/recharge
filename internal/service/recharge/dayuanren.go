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
    logger "recharge-go/pkg/log"
    "recharge-go/pkg/signature"
	"strconv"
	"strings"

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
	// 注入订单号并使用 v2 recharge 类别日志
	ctx = logger.InjectOrderNumber(ctx, order.OrderNumber)
	clog := logger.WithContextCategory(ctx, "recharge")
	clog.Info("开始提交订单",
		logger.StringV2("platform", "dayuanren"),
		logger.Int64V2("order_id", order.ID),
		logger.StringV2("order_number", order.OrderNumber),
		logger.StringV2("mobile", order.Mobile),
	)

	_, appSecret, accountName, err := p.getAPIKeyAndSecret(api.AccountID)
	if err != nil {
		clog.Error("获取API密钥失败",
			logger.Int64V2("order_id", order.ID),
			logger.StringV2("order_number", order.OrderNumber),
			logger.ErrorV2(err),
		)
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

	// 仅记录参数键和表单预览，避免记录大量或敏感内容
	paramKeys := make([]string, 0, len(params))
	for k := range params {
		paramKeys = append(paramKeys, k)
	}
	formPreview := form.Encode()
	if len(formPreview) > 512 {
		formPreview = formPreview[:512] + "..."
	}
	clog.Info("发送请求",
		logger.StringV2("platform", "dayuanren"),
		logger.Int64V2("order_id", order.ID),
		logger.StringV2("order_number", order.OrderNumber),
		logger.StringV2("out_trade_num", req.OutTradeNum),
		logger.StringV2("url", api.URL+"/index/recharge"),
		logger.AnyV2("param_keys", paramKeys),
		logger.StringV2("form_preview", formPreview),
	)

	resp, err := http.PostForm(api.URL+"/index/recharge", form)
	if err != nil {
		clog.Error("请求失败",
			logger.Int64V2("order_id", order.ID),
			logger.StringV2("order_number", order.OrderNumber),
			logger.StringV2("out_trade_num", req.OutTradeNum),
			logger.StringV2("url", api.URL+"/index/recharge"),
			logger.ErrorV2(err),
		)
		return fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		clog.Error("读取响应失败",
			logger.Int64V2("order_id", order.ID),
			logger.StringV2("order_number", order.OrderNumber),
			logger.StringV2("out_trade_num", req.OutTradeNum),
			logger.ErrorV2(err),
		)
		return fmt.Errorf("读取响应失败: %v", err)
	}
	bodyStr := string(body)
	preview := bodyStr
	if len(bodyStr) > 512 {
		preview = bodyStr[:512] + "..."
	}
	clog.Info("收到响应",
		logger.Int64V2("order_id", order.ID),
		logger.StringV2("order_number", order.OrderNumber),
		logger.StringV2("out_trade_num", req.OutTradeNum),
		logger.StringV2("body_preview", preview),
	)

	var respData Response
	if err := json.Unmarshal(body, &respData); err != nil {
		clog.Error("解析响应失败",
			logger.Int64V2("order_id", order.ID),
			logger.StringV2("order_number", order.OrderNumber),
			logger.StringV2("out_trade_num", req.OutTradeNum),
			logger.ErrorV2(err),
		)
		return fmt.Errorf("解析响应失败: %v", err)
	}
	// 记录关键字段，避免冗长日志
	clog.Info("解析响应成功",
		logger.Int64V2("order_id", order.ID),
		logger.StringV2("order_number", order.OrderNumber),
		logger.StringV2("out_trade_num", req.OutTradeNum),
		logger.IntV2("errno", respData.Errno),
		logger.StringV2("errmsg", respData.Errmsg),
		logger.IntV2("data_len", len(respData.Data)),
	)

	if respData.Errno != 0 {
		clog.Error("API错误",
			logger.Int64V2("order_id", order.ID),
			logger.StringV2("order_number", order.OrderNumber),
			logger.StringV2("out_trade_num", req.OutTradeNum),
			logger.IntV2("errno", respData.Errno),
			logger.StringV2("errmsg", respData.Errmsg),
		)
		return fmt.Errorf("API错误: %s", respData.Errmsg)
	}

	var rechargeResp RechargeResponse
	if err := json.Unmarshal(respData.Data, &rechargeResp); err != nil {
		clog.Error("解析数据失败",
			logger.Int64V2("order_id", order.ID),
			logger.StringV2("order_number", order.OrderNumber),
			logger.StringV2("out_trade_num", req.OutTradeNum),
			logger.ErrorV2(err),
		)
		return fmt.Errorf("解析数据失败: %v", err)
	}
	clog.Info("充值响应",
		logger.Int64V2("order_id", order.ID),
		logger.StringV2("order_number", order.OrderNumber),
		logger.StringV2("out_trade_num", req.OutTradeNum),
		logger.StringV2("api_order_id", rechargeResp.OrderNumber),
		logger.IntV2("product_id", rechargeResp.ProductID),
		logger.StringV2("total_price", rechargeResp.TotalPrice),
	)

	clog.Info("提交订单成功",
		logger.Int64V2("order_id", order.ID),
		logger.StringV2("order_number", order.OrderNumber),
		logger.StringV2("out_trade_num", req.OutTradeNum),
		logger.StringV2("api_order_id", rechargeResp.OrderNumber),
	)
	return nil
}

// QueryOrderStatus 查询订单状态
func (p *DayuanrenPlatform) QueryOrderStatus(ctx context.Context, order *model.Order) (model.OrderStatus, error) {
	_, appSecret, accountName, err := p.getAPIKeyAndSecret(order.PlatformAccountID)
	if err != nil {
		return 0, fmt.Errorf("获取API密钥失败: %v", err)
	}

	// 获取平台API信息
	api, err := p.platformRepo.GetPlatformByCode(ctx, "dayuanren")
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

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(api.URL, "/")+"/index/check", strings.NewReader(form.Encode()))
	if err != nil {
		return 0, fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
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
		logger.GetCategoryLogger("recharge").Info("订单状态：失败",
			logger.StringV2("platform", "dayuanren"),
			logger.StringV2("order_id", orderID),
			logger.IntV2("state", state),
		)
	case 0:
		status = int(model.OrderStatusProcessing) // 处理中
		statusStr = strconv.Itoa(status)
		logger.GetCategoryLogger("recharge").Info("订单状态：处理中",
			logger.StringV2("platform", "dayuanren"),
			logger.StringV2("order_id", orderID),
			logger.IntV2("state", state),
		)
	case 1:
		status = int(model.OrderStatusSuccess) // 成功
		statusStr = strconv.Itoa(status)
		logger.GetCategoryLogger("recharge").Info("订单状态：成功",
			logger.StringV2("platform", "dayuanren"),
			logger.StringV2("order_id", orderID),
			logger.IntV2("state", state),
		)
	case 2:
		status = int(model.OrderStatusFailed) // 失败
		statusStr = strconv.Itoa(status)
		logger.GetCategoryLogger("recharge").Info("订单状态：失败",
			logger.StringV2("platform", "dayuanren"),
			logger.StringV2("order_id", orderID),
			logger.IntV2("state", state),
		)
	case 3:
		status = int(model.OrderStatusProcessing) // 部分成功/处理中
		statusStr = strconv.Itoa(status)
		logger.GetCategoryLogger("recharge").Info("订单状态：部分成功/处理中",
			logger.StringV2("platform", "dayuanren"),
			logger.StringV2("order_id", orderID),
			logger.IntV2("state", state),
		)
	default:
		status = int(model.OrderStatusFailed) // 默认失败
		statusStr = strconv.Itoa(status)
		logger.GetCategoryLogger("recharge").Error("订单状态：未知",
			logger.StringV2("platform", "dayuanren"),
			logger.StringV2("order_id", orderID),
			logger.IntV2("state", state),
		)
	}
	return status, statusStr
}

// ParseCallbackData 解析回调数据
func (p *DayuanrenPlatform) ParseCallbackData(data []byte) (*model.CallbackData, error) {
	// 使用 recharge 类别日志器记录回调数据解析
	clog := logger.GetCategoryLogger("recharge")
	dataPreview := string(data)
	if len(dataPreview) > 256 {
		dataPreview = dataPreview[:256] + "..."
	}
	clog.Info("解析回调数据",
		logger.StringV2("platform", "dayuanren"),
		logger.IntV2("data_len", len(data)),
		logger.StringV2("data_preview", dataPreview),
	)
	// 解析 url.Values
	form, err := url.ParseQuery(string(data))
	if err != nil {
		clog.Error("回调参数解析失败",
			logger.ErrorV2(err),
		)
		return nil, errors.New("回调参数解析失败")
	}
	params := make(map[string]string)
	for k, v := range form {
		if len(v) > 0 {
			params[k] = v[0]
		}
	}
	// 仅记录参数键
	paramKeys := make([]string, 0, len(params))
	for k := range params {
		paramKeys = append(paramKeys, k)
	}
	clog.Info("回调参数",
		logger.AnyV2("param_keys", paramKeys),
	)

	state, _ := strconv.Atoi(params["state"])

	// 确定订单ID，优先使用out_trade_num（重试时的新订单号），其次使用order_number（原始订单号）
	orderID := params["out_trade_num"]
	if orderID == "" {
		orderID = params["order_number"]
	}

	_, statusStr := p.mapOrderState(state, orderID)

	callbackData := &model.CallbackData{
		OrderID:       orderID,
		Status:        statusStr,
		Message:       params["remark"],
		Amount:        params["charge_amount"],
		Sign:          params["sign"],
		Timestamp:     params["otime"],
		OrderNumber:   orderID,
		TransactionID: "dayuanren_" + orderID, // 使用平台前缀+订单号作为TransactionID
	}
	clog.Info("回调解析完成",
		logger.StringV2("order_id", callbackData.OrderID),
		logger.StringV2("order_number", callbackData.OrderNumber),
		logger.StringV2("status", callbackData.Status),
		logger.StringV2("message", callbackData.Message),
		logger.StringV2("original_order_number", params["order_number"]),
		logger.StringV2("out_trade_num", params["out_trade_num"]),
	)
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
