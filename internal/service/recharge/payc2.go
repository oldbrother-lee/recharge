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
	"strconv"
	"strings"
	"time"

	"recharge-go/internal/model"
	"recharge-go/internal/repository"
	"recharge-go/internal/signature"
	"recharge-go/pkg/logger"

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
	logger.Info(fmt.Sprintf("【payc2开始提交订单】order_number: %s", order.OrderNumber))

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

	logger.Info("payc2发送请求", "url", api.URL+"/apis/wof/order/create_phone", "params", params)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %v", err)
	}

	logger.Info("payc2响应", "status", resp.StatusCode, "body", string(body))

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

	logger.Info("payc2订单提交成功", "order_number", order.OrderNumber, "api_order_id", response.Datas.UID)
	return nil
}

// QueryOrderStatus 查询订单状态
func (p *Payc2Platform) QueryOrderStatus(ctx context.Context, order *model.Order) (model.OrderStatus, error) {
	// payc2平台通常通过回调通知订单状态，这里可以实现主动查询逻辑
	// 如果平台提供查询接口，可以在这里实现
	logger.Info("payc2查询订单状态", "order_number", order.OrderNumber)

	// 暂时返回当前状态，实际应该调用平台查询接口
	return order.Status, nil
}

// verifyCallbackSignature 验证回调签名
func (p *Payc2Platform) verifyCallbackSignature(ctx context.Context, params map[string]interface{}, orderNo string) bool {
	order, err := p.orderRepo.GetByOrderNumber(ctx, orderNo)
	if err != nil {
		logger.Error("获取订单信息失败", "orderNo", orderNo, "error", err)
		return false
	}

	account, err := p.platformRepo.GetAccountByID(ctx, order.PlatformAccountID)
	if err != nil {
		logger.Error("获取平台账号信息失败", "accountID", order.PlatformAccountID, "error", err)
		return false
	}

	// 3. 获取接收到的签名
	receivedSign, ok := params["sign"].(string)
	if !ok {
		logger.Error("回调数据中缺少签名")
		return false
	}

	// 4. 移除签名字段后重新计算签名
	delete(params, "sign")

	// 5. 使用正确的AppSecret生成签名
	expectedSign := p.generateSignatureWithSecret(params, account.AppSecret)

	logger.Info("payc2回调签名验证",
		"orderNo", orderNo,
		"accountID", order.PlatformAccountID,
		"appSecret", account.AppSecret,
		"receivedSign", receivedSign,
		"expectedSign", expectedSign)

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

	logger.Info("payc2签名生成", "params", params, "signStr", signStr, "sign", sign)
	return sign
}

// ParseCallbackData 解析回调数据
func (p *Payc2Platform) ParseCallbackData(data []byte) (*model.CallbackData, error) {
	logger.Info("payc2解析回调数据", "data", string(data))

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

	// 验证签名 - 需要动态获取AppSecret
	// 临时注释签名验证
	// if !p.verifyCallbackSignature(params, orderNo) {
	// 	return nil, fmt.Errorf("回调签名验证失败")
	// }

	// 解析回调数据
	callbackData := &model.CallbackData{
		OrderNumber:   values.Get("orderNo"),
		OrderID:       values.Get("uid"),
		Amount:        values.Get("amount"), // 订单金额
		Status:        "success",            // 默认成功状态
		Message:       "",
		TransactionID: "payc2_" + values.Get("uid"),
	}

	// 根据 stateAmount 和 stateOver 判断订单状态
	stateAmount := values.Get("stateAmount")
	stateOver := values.Get("stateOver")
	amountPaid := values.Get("amountPaid")

	logger.Info("payc2回调状态信息",
		"order_number", callbackData.OrderNumber,
		"stateAmount", stateAmount,
		"stateOver", stateOver,
		"amountPaid", amountPaid)

	// 根据文档说明判断订单状态
	// stateAmount: 0零充值，1已全充，3部分充，4已超充
	// stateOver: 0未结束，1已结束
	if stateAmountInt, err := strconv.Atoi(stateAmount); err == nil {
		switch stateAmountInt {
		case 0: // 零充值
			callbackData.Status = "failed"
			callbackData.Message = "零充值"
		case 1: // 已全充
			callbackData.Status = "success"
			callbackData.Message = "充值成功"
		case 3: // 部分充
			callbackData.Status = "partial"
			callbackData.Message = "部分充值"
		case 4: // 已超充
			callbackData.Status = "success"
			callbackData.Message = "充值成功（超充）"
		default:
			callbackData.Status = "unknown"
			callbackData.Message = "未知状态"
		}
	}

	// 如果订单已结束且为零充值，则标记为失败
	if stateOver == "1" && stateAmount == "0" {
		callbackData.Status = "failed"
		callbackData.Message = "订单已结束，零充值"
	}

	logger.Info("payc2回调数据解析完成", "callback_data", callbackData)
	return callbackData, nil
}

// QueryBalance 查询账户余额
func (p *Payc2Platform) QueryBalance(ctx context.Context, accountID int64) (float64, error) {
	// 如果平台提供余额查询接口，可以在这里实现
	logger.Info("payc2查询账户余额", "account_id", accountID)

	// 暂时返回0，实际应该调用平台余额查询接口
	return 0, fmt.Errorf("payc2平台暂不支持余额查询")
}
