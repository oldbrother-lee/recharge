package recharge

import (
	"bytes"
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
	"time"

	"gorm.io/gorm"
)

// MishiPlatform 秘史平台实现
type MishiPlatform struct {
	platformRepo repository.PlatformRepository
}

// NewMishiPlatform 创建秘史平台实例
func NewMishiPlatform(db *gorm.DB) *MishiPlatform {
	return &MishiPlatform{
		platformRepo: repository.NewPlatformRepository(db),
	}
}

// GetName 获取平台名称
func (p *MishiPlatform) GetName() string {
	return "mishi"
}

// getAPIKeyAndSecret 获取API密钥和密钥
func (p *MishiPlatform) getAPIKeyAndSecret(accountID int64) (string, string, string, error) {
	// accountIDStr := strconv.FormatInt(accountID, 10)
	account, err := p.platformRepo.GetPlatformAccountByID(accountID)
	if err != nil {
		return "", "", "", fmt.Errorf("获取平台账号信息失败: %v", err)
	}
	return account.AppKey, account.AppSecret, account.AccountName, nil
}

// convertOperatorCode 转换运营商编码
func convertOperatorCode(operatorCode string) string {
	switch operatorCode {
	case "1":
		return "1" // 移动
	case "3":
		return "2" // 联通
	case "2":
		return "3" // 电信
	case "虚拟":
		return "4" // 虚商
	case "国家电网":
		return "101" // 国家电网
	case "南方电网":
		return "102" // 南方电网
	case "中石化":
		return "104" // 中石化
	case "中石油":
		return "105" // 中石油
	case "腾讯":
		return "1000" // 腾讯
	case "爱奇艺":
		return "1001" // 爱奇艺
	case "优酷":
		return "1002" // 优酷
	case "抖音":
		return "1031" // 抖音
	default:
		return "1" // 默认移动
	}
}

// SubmitOrder 提交订单
func (p *MishiPlatform) SubmitOrder(ctx context.Context, order *model.Order, api *model.PlatformAPI, apiParam *model.PlatformAPIParam) error {
	logger.Info("开始提交秘史订单",
		"order_id", order.ID,
		"order_number", order.OrderNumber,
		"mobile", order.Mobile,
	)
	fmt.Printf("[mishi] api: %+v\n", api)
	fmt.Printf("[mishi] 提交秘史订单apiParam: %+v\n", apiParam)
	// 获取API密钥和密钥
	_, appSecret, accountName, err := p.getAPIKeyAndSecret(api.AccountID)
	if err != nil {
		return fmt.Errorf("meishi 获取API密钥失败!!!: %v", err)
	}

	// 构建请求参数
	szTimeStamp := time.Now().Format("2006-01-02 15:04:05")
	params := url.Values{}
	params.Add("szAgentId", accountName)                                  // 客户id
	params.Add("szOrderId", order.OrderNumber)                            // 订单号
	params.Add("szPhoneNum", order.Mobile)                                // 充值手机号
	params.Add("nMoney", strconv.FormatInt(int64(order.Denom), 10))       // 充值金额
	params.Add("nSortType", convertOperatorCode(strconv.Itoa(order.ISP))) // 运营商编码
	params.Add("nProductClass", "1")                                      // 充值产品分类
	params.Add("nProductType", "1")                                       // 充值产品类型
	params.Add("szProductId", apiParam.ProductID)
	params.Add("szTimeStamp", szTimeStamp)

	// 生成签名
	signStr := fmt.Sprintf("szAgentId=%s&szOrderId=%s&szPhoneNum=%s&nMoney=%s&nSortType=%s&nProductClass=%s&nProductType=%s&szTimeStamp=%s&szKey=%s",
		accountName, order.OrderNumber, order.Mobile, strconv.FormatInt(int64(order.Denom), 10),
		convertOperatorCode(strconv.Itoa(order.ISP)), "1", "1", szTimeStamp, appSecret)

	logger.Info("meishi 生成签名前: ", "signStr", signStr)
	sign := signature.GetMD5(signStr)
	params.Add("szVerifyString", sign)

	// 添加回调地址
	params.Add("szNotifyUrl", api.CallbackURL)

	logger.Info("meishi 发送请求: ", "params", params)
	// 发送请求
	respStr, err := p.sendRequest(ctx, api.URL+"/api/submitorder", params)
	if err != nil {
		return fmt.Errorf("发送请求失败: %v", err)
	}
	logger.Info("meishi 发送请求成功返回的参数: ", "respStr", respStr)
	// 解析响应
	var result MishiOrderResponseSubmit
	if err := json.Unmarshal([]byte(respStr), &result); err != nil {
		return fmt.Errorf("解析响应失败: %v", err)
	}

	// 处理响应
	if result.NRtn != 0 {
		logger.Error("meishi 提交订单失败: NRtn %d szRtnCode：%s", result.NRtn, result.SzRtnCode)
		return fmt.Errorf("提交订单失败: %s", result.SzRtnCode)
	}

	// 更新订单信息
	// order.APIOrderNumber = result.SzOrderId
	// order.APITradeNum = result.SzOrderId

	logger.Info("提交订单成功",
		"order_id", order.ID,
		"order_number", order.OrderNumber,
		"api_order_id", result.SzOrderId,
	)

	return nil
}

// QueryOrderStatus 查询订单状态
func (p *MishiPlatform) QueryOrderStatus(order *model.Order) (model.OrderStatus, error) {
	logger.Info("开始查询秘史订单状态",
		"order_id", order.ID,
		"order_number", order.OrderNumber,
		"api_order_id", order.APIOrderNumber,
	)

	// 获取API密钥和密钥
	_, appSecret, accountName, err := p.getAPIKeyAndSecret(order.PlatformAccountID)
	if err != nil {
		return 0, fmt.Errorf("获取API密钥失败: %v", err)
	}

	// 构建请求参数
	params := url.Values{}
	params.Add("szAgentId", accountName)
	params.Add("szOrderId", order.OrderNumber)

	// 生成签名
	signStr := fmt.Sprintf("szAgentId=%s&szOrderId=%s&szKey=%s",
		accountName, order.OrderNumber, appSecret)
	sign := signature.GetMD5(signStr)
	params.Add("szVerifyString", sign)

	// 发送请求
	respStr, err := p.sendRequest(context.Background(), order.PlatformURL+"/query", params)
	if err != nil {
		return 0, fmt.Errorf("查询订单状态失败: %v", err)
	}

	// 解析响应
	var result MishiOrderResponseQuery
	if err := json.Unmarshal([]byte(respStr), &result); err != nil {
		return 0, fmt.Errorf("解析响应失败: %v", err)
	}

	// 处理响应
	if result.SzRtnCode != "success" {
		return 0, fmt.Errorf("查询订单状态失败: %s", result.SzRtnMsg)
	}

	// 转换状态
	var status model.OrderStatus
	switch result.SzRtnMsg {
	case "1":
		status = model.OrderStatusProcessing
	case "2":
		status = model.OrderStatusSuccess
	case "3":
		status = model.OrderStatusFailed
	default:
		status = model.OrderStatusProcessing
	}

	return status, nil
}

// mapOrderState 返回本地订单状态码和字符串
func (p *MishiPlatform) mapOrderState(nFlag string, orderID, orderNumber string) (int, string) {
	var status int
	var statusStr string
	switch nFlag {
	case "2":
		status = int(model.OrderStatusSuccess)
		statusStr = strconv.Itoa(status)
		logger.Info("【秘史订单状态】充值成功", "order_id", orderID, "order_number", orderNumber)
	case "3":
		status = int(model.OrderStatusFailed)
		statusStr = strconv.Itoa(status)
		logger.Info("【秘史订单状态】充值失败", "order_id", orderID, "order_number", orderNumber)
	default:
		status = int(model.OrderStatusProcessing)
		statusStr = strconv.Itoa(status)
		logger.Info("【秘史订单状态】处理中", "order_id", orderID, "order_number", orderNumber, "nFlag", nFlag)
	}
	return status, statusStr
}

// ParseCallbackData 解析回调数据
func (p *MishiPlatform) ParseCallbackData(data []byte) (*model.CallbackData, error) {
	// 先尝试 url.ParseQuery 解析表单格式
	form, err := url.ParseQuery(string(data))
	if err == nil && len(form) > 0 {
		_, statusStr := p.mapOrderState(form["nFlag"][0], form["szOrderId"][0], form["szOrderId"][0])
		callbackData := &model.CallbackData{
			OrderID:     form["szOrderId"][0],
			Status:      statusStr,
			Message:     form["szRtnMsg"][0],
			Amount:      form["fSalePrice"][0],
			Sign:        form["szVerifyString"][0],
			OrderNumber: form["szOrderId"][0],
			Timestamp:   "",
		}
		logger.Info("mishi回调解析完成(form)", "callbackData", callbackData)
		return callbackData, nil
	}
	// 如果不是表单格式，尝试 json 解析
	var req struct {
		SzOrderId      string `json:"szOrderId"`
		NFlag          string `json:"nFlag"`
		SzRtnMsg       string `json:"szRtnMsg"`
		FSalePrice     string `json:"fSalePrice"`
		SzVerifyString string `json:"szVerifyString"`
	}
	if err := json.Unmarshal(data, &req); err != nil {
		logger.Error("mishi回调参数解析失败", "error", err, "data", string(data))
		return nil, errors.New("解析回调数据失败")
	}
	_, statusStr := p.mapOrderState(req.NFlag, req.SzOrderId, req.SzOrderId)
	callbackData := &model.CallbackData{
		OrderID:     req.SzOrderId,
		Status:      statusStr,
		Message:     req.SzRtnMsg,
		Amount:      req.FSalePrice,
		Sign:        req.SzVerifyString,
		OrderNumber: req.SzOrderId,
		Timestamp:   "",
	}
	logger.Info("mishi回调解析完成(json)", "callbackData", callbackData)
	return callbackData, nil
}

// sendRequest 发送请求
func (p *MishiPlatform) sendRequest(ctx context.Context, url string, params url.Values) (string, error) {
	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBufferString(params.Encode()))
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %v", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// 发送请求
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %v", err)
	}
	return string(body), nil

}

// QueryBalance 查询账户余额
func (p *MishiPlatform) QueryBalance(ctx context.Context, accountID int64) (float64, error) {
	logger.Info("开始查询秘史账户余额",
		"account_id", accountID,
	)

	// 获取API密钥和密钥
	_, appSecret, accountName, err := p.getAPIKeyAndSecret(accountID)
	if err != nil {
		return 0, fmt.Errorf("获取API密钥失败: %v", err)
	}

	// 获取平台API信息
	api, err := p.platformRepo.GetPlatformByCode(ctx, "mishi")
	if err != nil {
		return 0, fmt.Errorf("获取平台API信息失败: %v", err)
	}

	// 构建请求参数
	params := url.Values{}
	params.Add("szAgentId", accountName)

	// 生成签名
	signStr := fmt.Sprintf("szAgentId=%s&szKey=%s", accountName, appSecret)
	sign := signature.GetMD5(signStr)
	params.Add("szVerifyString", sign)

	// 发送请求
	respStr, err := p.sendRequest(ctx, api.URL+"/api/old/queryBalance", params)
	if err != nil {
		return 0, fmt.Errorf("查询余额失败: %v", err)
	}

	// 解析响应
	var result MishiResponse
	if err := json.Unmarshal([]byte(respStr), &result); err != nil {
		return 0, fmt.Errorf("解析响应失败: %v", err)
	}

	// 处理响应
	if result.SzRtnCode != "success" {
		return 0, fmt.Errorf("查询余额失败: %s", result.SzRtnCode)
	}

	logger.Info("查询余额成功",
		"account_id", accountID,
		"balance", result.FBalance,
	)

	return result.FBalance, nil
}

// MishiResponse 秘史平台响应
type MishiResponse struct {
	SzRtnCode string  `json:"szRtnCode"`
	SzAgentId string  `json:"szAgentId"`
	FBalance  float64 `json:"fBalance"`
	FCredit   float64 `json:"fCredit"`
	NRtn      int     `json:"nRtn"`
}

type MishiOrderResponseQuery struct {
	SzRtnCode  string  `json:"szRtnCode"`
	SzOrderId  string  `json:"szAgentId"`
	FSalePrice float64 `json:"fBalance"`
	SzRtnMsg   string  `json:"fCredit"`
}

type MishiOrderResponseSubmit struct {
	NRtn       int64   `json:"nRtn"`
	SzRtnCode  string  `json:"szRtnCode"`
	SzOrderId  string  `json:"SzOrderId"`
	FSalePrice float64 `json:"fSalePrice"`
	FNBalance  float64 `json:"fNBalance"`
}
