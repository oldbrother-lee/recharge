package service

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
	"time"
)

// PlatformService 平台服务
type PlatformService struct {
	platformRepo       repository.PlatformRepository
	orderRepo          repository.OrderRepository
	externalAPIKeyRepo repository.ExternalAPIKeyRepository
}

// NewPlatformService 创建平台服务
func NewPlatformService(platformRepo repository.PlatformRepository, orderRepo repository.OrderRepository, externalAPIKeyRepo repository.ExternalAPIKeyRepository) *PlatformService {
	return &PlatformService{
		platformRepo:       platformRepo,
		orderRepo:          orderRepo,
		externalAPIKeyRepo: externalAPIKeyRepo,
	}
}

// ListPlatforms 获取平台列表
func (s *PlatformService) ListPlatforms(req *model.PlatformListRequest) ([]model.Platform, int64) {
	platforms, total, _ := s.platformRepo.ListPlatforms(req)
	return platforms, total
}

// CreatePlatform 创建平台
func (s *PlatformService) CreatePlatform(req *model.PlatformCreateRequest) error {
	platform := &model.Platform{
		Name:        req.Name,
		Code:        req.Code,
		ApiURL:      req.ApiURL,
		Description: req.Description,
		Status:      1, // 默认启用
	}
	return s.platformRepo.CreatePlatform(platform)
}

// GetPlatformByID 根据ID获取平台
func (s *PlatformService) GetPlatformByID(id int64) (*model.Platform, error) {
	return s.platformRepo.GetPlatformByID(id)
}

// UpdatePlatform 更新平台
func (s *PlatformService) UpdatePlatform(id int64, req *model.PlatformUpdateRequest) error {
	platform := &model.Platform{
		ID:          id,
		Name:        req.Name,
		Code:        req.Code,
		ApiURL:      req.ApiURL,
		Description: req.Description,
	}
	if req.Status != nil {
		platform.Status = *req.Status
	}
	return s.platformRepo.UpdatePlatform(platform)
}

// DeletePlatform 删除平台
func (s *PlatformService) DeletePlatform(id int64) error {
	return s.platformRepo.Delete(id)
}

// GetPlatform 获取平台
func (s *PlatformService) GetPlatform(id int64) (*model.Platform, error) {
	return s.platformRepo.GetPlatformByID(id)
}

// ListPlatformAccounts 获取平台账号列表
func (s *PlatformService) ListPlatformAccounts(req *model.PlatformAccountListRequest) (*model.PlatformAccountListResponse, error) {
	return s.platformRepo.ListPlatformAccounts(req)
}

// CreatePlatformAccount 创建平台账号
func (s *PlatformService) CreatePlatformAccount(req *model.PlatformAccountCreateRequest) error {
	account := &model.PlatformAccount{
		PlatformID:   req.PlatformID,
		AccountName:  req.AccountName,
		Type:         req.Type,
		AppKey:       req.AppKey,
		AppSecret:    req.AppSecret,
		Description:  req.Description,
		DailyLimit:   req.DailyLimit,
		MonthlyLimit: req.MonthlyLimit,
		Priority:     req.Priority,
	}
	if req.Status != nil {
		account.Status = *req.Status
	} else {
		account.Status = 1 // 默认启用
	}
	return s.platformRepo.CreatePlatformAccount(account)
}

// GetPlatformAccount 获取平台账号
func (s *PlatformService) GetPlatformAccount(id int64) (*model.PlatformAccount, error) {
	return s.platformRepo.GetPlatformAccountByID(id)
}

// GetPlatformAccountByID 根据ID获取平台账号
func (s *PlatformService) GetPlatformAccountByID(id int64) (*model.PlatformAccount, error) {
	return s.platformRepo.GetPlatformAccountByID(id)
}

// UpdatePlatformAccount 更新平台账号
func (s *PlatformService) UpdatePlatformAccount(ctx context.Context, id int64, req *model.PlatformAccountUpdateRequest) error {
	updateMap := map[string]interface{}{}

	if req.AccountName != nil {
		updateMap["account_name"] = *req.AccountName
	}
	if req.Type != nil {
		updateMap["type"] = *req.Type
	}
	if req.AppKey != nil {
		updateMap["app_key"] = *req.AppKey
	}
	if req.AppSecret != nil {
		updateMap["app_secret"] = *req.AppSecret
	}
	if req.Description != nil {
		updateMap["description"] = *req.Description
	}
	if req.DailyLimit != nil {
		updateMap["daily_limit"] = *req.DailyLimit
	}
	if req.MonthlyLimit != nil {
		updateMap["monthly_limit"] = *req.MonthlyLimit
	}
	if req.Balance != nil {
		updateMap["balance"] = *req.Balance
	}
	if req.Priority != nil {
		updateMap["priority"] = *req.Priority
	}
	if req.Status != nil {
		updateMap["status"] = *req.Status

	}
	if req.PushStatus != nil {
		updateMap["push_status"] = *req.PushStatus
	}

	if len(updateMap) == 0 {
		return nil // 没有任何字段需要更新
	}

	return s.platformRepo.UpdatePlatformAccountFields(ctx, id, updateMap)
}

// DeletePlatformAccount 删除平台账号
func (s *PlatformService) DeletePlatformAccount(id int64) error {
	return s.platformRepo.DeleteAccount(context.Background(), id)
}

// SendNotification 发送订单状态通知
func (s *PlatformService) SendNotification(ctx context.Context, order *model.Order) error {
	// 首先检查是否为外部API通知，如果是则直接处理，无需获取平台账号
	if order.PlatformCallbackURL != "" {
		return s.sendExternalAPINotification(ctx, order)
	}

	// 1. 获取平台配置
	// platform, err := s.platformRepo.GetPlatformByID(order.PlatformId)
	// if err != nil {
	// 	return fmt.Errorf("获取平台配置失败: %w", err)
	// }
	account, err := s.platformRepo.GetPlatformAccountByID(order.PlatformAccountID)
	if err != nil {
		return fmt.Errorf("获取平台账号失败: %w", err)
	}
	platform, err := s.platformRepo.GetPlatformByID(account.PlatformID)
	if err != nil {
		return fmt.Errorf("获取平台配置失败: %w", err)
	}

	// 3. 构建通知参数
	var params map[string]interface{}
	switch platform.Code {
	case "mifeng":
		params = s.buildMf178Params(order, account)
	case "kekebang":
		params = s.buildKekebangParams(order, account)
	case "xianzhuanxia":
		// 闲赚侠一般直接调用 ReportTask 方法，不需要拼接 URL
		err := s.buildXianzhuanxiaParams(order, account, platform.ApiURL)
		if err != nil {
			return fmt.Errorf("上报订单结果失败: %w", err)
		}
		return nil
	case "external_api":
		// 外部API通知处理
		return s.sendExternalAPINotification(ctx, order)
	default:
		return fmt.Errorf("不支持的平台: %s", platform.Code)
	}
	// // s生成签名
	params["sign"] = s.generateSign(platform.Code, params, account)
	//通过platform.Code 获取对应的api_url ，并拼接参数和订单状态转换
	// fmt.Printf("platform----------: %v\n", platform)
	// fmt.Printf("account----------: %v\n", account)
	// data := map[string]interface{}{
	// 	"data": map[string]interface{}{
	// 		"user_order_id": order.OutTradeNum,
	// 		"status":        9,
	// 		"rsp_info":      "充值成功",
	// 	},
	// }
	// jsonData, err := json.Marshal(data["data"])
	// if err != nil {
	// 	fmt.Println(err)
	// }
	// params := map[string]interface{}{
	// 	"data": string(jsonData),
	// }
	// params["app_key"] = "xxxxx"
	// params["timestamp"] = strconv.FormatInt(time.Now().Unix(), 10)

	// 3. 生成签名
	// params["sign"] = signature.GenerateSign(params, account.AppSecret)
	// 4. 发送通知请求
	var url string
	switch platform.Code {
	case "mifeng":
		url = platform.ApiURL + "/userapi/sgd/updateStatus"
	case "kekebang":
		url = platform.ApiURL + "/openapi/suppler/v1/report-user"
	case "xianzhuanxia":
		url = platform.ApiURL + "/api/task/recharge/reported"
	default:
		return fmt.Errorf("不支持的平台: %s", platform.Code)
	}
	fmt.Printf("最外层params: %+v\n", params)
	resp, err := s.sendRequest(ctx, url, params)
	if err != nil {
		return fmt.Errorf("发送通知请求失败: %w", err)
	}

	// 5. 处理响应
	if platform.Code == "kekebang" {
		if resp.Code != "0" {
			return fmt.Errorf("通知发送失败kekebang:code:%s, message:%s", resp.Code, resp.Message)
		}
	} else {
		code, err := strconv.ParseInt(string(resp.Code), 10, 64)
		if err != nil {
			return fmt.Errorf("解析响应码失败: %w", err)
		}
		if code != 0 {
			return fmt.Errorf("通知发送失败1: %s", resp.Message)
		}
	}

	return nil
}

// convertOrderStatus 转换订单状态
func (s *PlatformService) convertOrderStatus(status model.OrderStatus) string {
	switch status {
	case model.OrderStatusSuccess:
		return "SUCCESS"
	case model.OrderStatusFailed:
		return "FAILED"
	case model.OrderStatusProcessing:
		return "PROCESSING"
	default:
		return "UNKNOWN"
	}
}

// sendRequest 发送HTTP请求
func (s *PlatformService) sendRequest(ctx context.Context, url string, params map[string]interface{}) (*struct {
	Code    model.StringOrNumber `json:"code"`
	Message string               `json:"message"`
}, error) {
	logger.Info(fmt.Sprintf("发送通知发送请求params: %+v", params))
	// 1. 将参数转换为JSON
	jsonData, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("参数序列化失败: %w", err)
	}

	// 2. 创建请求
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 3. 设置请求头
	req.Header.Set("Content-Type", "application/json")

	// 4. 发送请求
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	logger.Info(fmt.Sprintf("发送通知发送请求: %+v", req))
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败1: %w", err)
	}
	defer resp.Body.Close()

	// 5. 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}
	// 打印原始响应
	logger.Info(fmt.Sprintf("发送通知返回原始响应: %s\n", string(body)))
	fmt.Printf("原始响应: %s\n", string(body))
	// 6. 解析响应
	var result struct {
		Code    model.StringOrNumber `json:"code"`
		Message string               `json:"message"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &result, nil
}

// GetOrder 获取订单信息
func (s *PlatformService) GetOrder(ctx context.Context, orderID int64) (*model.Order, error) {
	return s.orderRepo.GetByID(ctx, orderID)
}

func (s *PlatformService) GetPlatformAccountByAccountName(accountName string) (*model.PlatformAccount, error) {
	return s.platformRepo.GetPlatformAccountByAccountName(accountName)
}

func (s *PlatformService) buildKekebangParams(order *model.Order, account *model.PlatformAccount) map[string]interface{} {
	data := map[string]interface{}{
		"user_order_id": order.OutTradeNum,
		"status":        s.getKekebangStatus(order.Status),
	}
	jsonStr, err := json.Marshal(data)
	if err != nil {
		fmt.Println(err)
	}
	return map[string]interface{}{
		"app_key":   account.AppKey,
		"timestamp": strconv.FormatInt(time.Now().Unix(), 10),
		"data":      string(jsonStr),
	}
	// data:= map[string]interface{}{
	// 		"user_order_id": order.OutTradeNum,
	// 		"status":        s.getKekebangStatus(order.Status),
	// 		"rsp_info":      s.getStatusText(order.Status),
	// 		"voucher":       "",
	// },

	// if err != nil {
	// 	fmt.Println(err)
	// }

	// params["data"] = data
	// params["app_key"] = account.AppKey
	// params["timestamp"] = strconv.FormatInt(time.Now().Unix(), 10)
	// fmt.Printf("kekebang params: %+v\n", params)
	// return params
}

func (s *PlatformService) buildMf178Params(order *model.Order, account *model.PlatformAccount) map[string]interface{} {

	data := map[string]interface{}{
		"data": map[string]interface{}{
			"user_order_id": order.OutTradeNum,
			"status":        s.getPlatformStatus(order.Status),
			"rsp_info":      s.getStatusText(order.Status),
		},
	}
	jsonData, err := json.Marshal(data["data"])
	if err != nil {
		fmt.Println(err)
	}
	params := map[string]interface{}{
		"data": string(jsonData),
	}
	params["app_key"] = account.AppKey
	params["timestamp"] = strconv.FormatInt(time.Now().Unix(), 10)
	return params
}

func (s *PlatformService) getPlatformStatus(orderStatus model.OrderStatus) int {
	// 根据平台代码和订单状态返回对应的平台状态码
	switch orderStatus {
	case model.OrderStatusSuccess:
		return 9 // 米蜂成功状态码
	case model.OrderStatusFailed:
		return 8 // 米蜂失败状态码
	// ... 其他状态映射
	default:
		return 0
	}
}

func (s *PlatformService) getKekebangStatus(orderStatus model.OrderStatus) int {
	// 根据平台代码和订单状态返回对应的平台状态码
	switch orderStatus {
	case model.OrderStatusSuccess:
		return 9 // 米蜂成功状态码
	case model.OrderStatusFailed:
		return 8 // 米蜂失败状态码
	// ... 其他状态映射
	default:
		return 0
	}
}

func (s *PlatformService) getStatusText(orderStatus model.OrderStatus) string {
	// 根据订单状态返回对应的文本信息
	switch orderStatus {
	case model.OrderStatusSuccess:
		return "充值成功"
	case model.OrderStatusFailed:
		return "充值失败"
	case model.OrderStatusProcessing:
		return "充值中"
	default:
		return "未知状态"
	}
}

func (s *PlatformService) generateSign(platformCode string, params map[string]interface{}, account *model.PlatformAccount) string {
	switch platformCode {
	case "mifeng":
		return signature.GenerateSign(params, account.AppSecret)
	case "kekebang":
		return signature.GenerateKekebangNotifySign(params, account.AppSecret)
	default:
		return ""
	}
}

func (s *PlatformService) buildXianzhuanxiaParams(order *model.Order, account *model.PlatformAccount, apiURL string) error {

	params := map[string]interface{}{
		"orderNumber": order.OutTradeNum,
		"status":      s.getXianzhuanxiaStatus(order.Status),
	}

	// params["app_key"] = account.AppKey
	authToken, _, err := signature.GenerateXianzhuanxiaSignature2(params, account.AppKey, account.AccountName)
	if err != nil {
		return fmt.Errorf("生成签名失败: %v", err)
	}
	jsonData, err := json.Marshal(params)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/api/task/recharge/reported", apiURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Auth_Token", authToken)
	// req.Header.Set("Query-Time", queryTime)
	fmt.Printf("req: %v\n", req)
	logger.Info(fmt.Sprintf("发送闲赚侠上报订单结果请求: %v\n", req))
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("请求失败: %s", string(body))
	}

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("解析响应失败: %v", err)
	}

	if result.Code != 0 {
		return fmt.Errorf("业务错误: %s", result.Msg)
	}

	return nil

}

func (s *PlatformService) getXianzhuanxiaStatus(orderStatus model.OrderStatus) int {
	switch orderStatus {
	case model.OrderStatusSuccess:
		return 1 // 闲赚侠平台"成功"状态码
	case model.OrderStatusFailed:
		return 2 // 闲赚侠平台"失败"状态码
	default:
		return 0 // 其他状态
	}
}

// sendExternalAPINotification 发送外部API通知
func (s *PlatformService) sendExternalAPINotification(ctx context.Context, order *model.Order) error {
	logger.Info("开始处理外部API通知",
		"order_id", order.ID,
		"order_number", order.OrderNumber,
		"customer_id", order.CustomerID,
		"order_status", order.Status,
		"callback_url", order.PlatformCallbackURL,
	)

	// 只发送成功和失败状态的通知，其他状态不发送
	if order.Status != model.OrderStatusSuccess && order.Status != model.OrderStatusFailed {
		logger.Info("订单状态不需要发送通知，跳过",
			"order_id", order.ID,
			"order_number", order.OrderNumber,
			"order_status", order.Status,
		)
		return nil // 不发送通知，直接返回
	}

	// 检查是否有回调URL
	if order.PlatformCallbackURL == "" {
		logger.Error("外部API订单缺少回调URL",
			"order_id", order.ID,
			"order_number", order.OrderNumber,
			"customer_id", order.CustomerID,
		)
		return fmt.Errorf("外部API订单缺少回调URL")
	}

	// 构建通知参数
	logger.Info("开始构建外部API通知参数",
		"order_id", order.ID,
		"order_number", order.OrderNumber,
	)
	params := s.buildExternalAPIParams(order)
	logger.Info("外部API通知参数构建完成",
		"order_id", order.ID,
		"order_number", order.OrderNumber,
		"params", params,
	)

	// 生成签名
	logger.Info("开始生成外部API签名",
		"order_id", order.ID,
		"order_number", order.OrderNumber,
		"customer_id", order.CustomerID,
	)
	sign := s.generateExternalAPISign(params, order)
	if sign == "" {
		logger.Error("外部API签名生成失败",
			"order_id", order.ID,
			"order_number", order.OrderNumber,
			"customer_id", order.CustomerID,
		)
		return fmt.Errorf("外部API签名生成失败")
	}
	params["sign"] = sign
	logger.Info("外部API签名生成成功",
		"order_id", order.ID,
		"order_number", order.OrderNumber,
		"sign_length", len(sign),
	)

	// 发送HTTP通知
	logger.Info("开始发送外部API HTTP通知",
		"order_id", order.ID,
		"order_number", order.OrderNumber,
		"callback_url", order.PlatformCallbackURL,
	)
	err := s.sendExternalAPIHTTPNotification(ctx, order.PlatformCallbackURL, params)
	if err != nil {
		logger.Error("外部API HTTP通知发送失败",
			"order_id", order.ID,
			"order_number", order.OrderNumber,
			"callback_url", order.PlatformCallbackURL,
			"error", err,
		)
		return err
	}

	logger.Info("外部API通知发送成功",
		"order_id", order.ID,
		"order_number", order.OrderNumber,
		"callback_url", order.PlatformCallbackURL,
	)
	return nil
}

// buildExternalAPIParams 构建外部API通知参数
func (s *PlatformService) buildExternalAPIParams(order *model.Order) map[string]interface{} {
	params := map[string]interface{}{
		"out_trade_num": order.OutTradeNum,
		"status":        s.getExternalAPIStatus(order.Status),
		"timestamp":     time.Now().Unix(),
		"nonce":         fmt.Sprintf("%d", time.Now().UnixNano()),
	}

	// 添加app_id参数（从订单的客户ID获取）
	apiKeys, _, err := s.externalAPIKeyRepo.GetByUserID(order.CustomerID, 0, 1)
	if err == nil && len(apiKeys) > 0 {
		params["app_id"] = apiKeys[0].AppID
	}

	return params
}

// generateExternalAPISign 生成外部API签名
func (s *PlatformService) generateExternalAPISign(params map[string]interface{}, order *model.Order) string {
	logger.Info("开始获取外部API密钥",
		"customer_id", order.CustomerID,
		"order_id", order.ID,
		"order_number", order.OrderNumber,
	)

	// 根据订单的客户ID获取外部API密钥信息
	apiKeys, total, err := s.externalAPIKeyRepo.GetByUserID(order.CustomerID, 0, 1)
	if err != nil {
		// 如果无法获取API密钥，记录日志并返回空签名
		logger.Error("获取外部API密钥失败",
			"error", err,
			"customer_id", order.CustomerID,
			"order_id", order.ID,
			"order_number", order.OrderNumber,
		)
		return ""
	}

	logger.Info("外部API密钥查询结果",
		"customer_id", order.CustomerID,
		"order_id", order.ID,
		"order_number", order.OrderNumber,
		"total_keys", total,
		"returned_keys", len(apiKeys),
	)

	// 检查是否有API密钥
	if len(apiKeys) == 0 {
		logger.Error("用户没有配置外部API密钥",
			"customer_id", order.CustomerID,
			"order_id", order.ID,
			"order_number", order.OrderNumber,
			"total_keys", total,
		)
		return ""
	}

	// 使用第一个API密钥生成签名
	apiKey := apiKeys[0]
	logger.Info("发送端签名生成参数",
		"customer_id", order.CustomerID,
		"order_id", order.ID,
		"order_number", order.OrderNumber,
		"api_key_id", apiKey.ID,
		"app_id", apiKey.AppID,
		"secret_length", len(apiKey.AppSecret),
		"params_count", len(params),
		"params", params,
	)

	// 使用外部API签名验证器生成签名
	signatureValidator := signature.NewExternalAPISignatureValidator()
	sign, err := signatureValidator.GenerateExternalAPISignature(params, apiKey.AppSecret)
	if err != nil {
		logger.Error("外部API签名生成失败",
			"error", err,
			"customer_id", order.CustomerID,
			"order_id", order.ID,
			"order_number", order.OrderNumber,
		)
		return ""
	}
	logger.Info("发送端签名生成完成",
		"customer_id", order.CustomerID,
		"order_id", order.ID,
		"order_number", order.OrderNumber,
		"api_key_id", apiKey.ID,
		"generated_sign", sign,
		"sign_length", len(sign),
	)

	return sign
}

// getExternalAPIStatus 获取外部API状态码
func (s *PlatformService) getExternalAPIStatus(orderStatus model.OrderStatus) int {
	switch orderStatus {
	case model.OrderStatusSuccess:
		return 4 // 成功
	case model.OrderStatusFailed:
		return 5 // 失败
	case model.OrderStatusRecharging:
		return 3 // 处理中
	default:
		return 0 // 未知状态
	}
}

// getExternalAPIStatusDesc 获取外部API状态描述
func (s *PlatformService) getExternalAPIStatusDesc(orderStatus model.OrderStatus) string {
	switch orderStatus {
	case model.OrderStatusSuccess:
		return "充值成功"
	case model.OrderStatusFailed:
		return "充值失败"
	case model.OrderStatusRecharging:
		return "充值中"
	default:
		return "未知状态"
	}
}

// sendExternalAPIHTTPNotification 发送外部API HTTP通知
func (s *PlatformService) sendExternalAPIHTTPNotification(ctx context.Context, callbackURL string, params map[string]interface{}) error {
	logger.Info("开始构建HTTP请求",
		"callback_url", callbackURL,
		"params_count", len(params),
	)

	// 构建请求体
	jsonData, err := json.Marshal(params)
	if err != nil {
		logger.Error("序列化参数失败",
			"error", err,
			"callback_url", callbackURL,
			"params", params,
		)
		return fmt.Errorf("序列化参数失败: %v", err)
	}

	logger.Info("HTTP请求体构建完成",
		"callback_url", callbackURL,
		"request_body_size", len(jsonData),
		"request_body", string(jsonData),
	)

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "POST", callbackURL, bytes.NewBuffer(jsonData))
	if err != nil {
		logger.Error("创建HTTP请求失败",
			"error", err,
			"callback_url", callbackURL,
		)
		return fmt.Errorf("创建请求失败: %v", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "RechargeGo-Notification/1.0")

	// 从参数中获取API密钥和签名，添加到请求头
	if appKey, ok := params["app_key"].(string); ok {
		req.Header.Set("X-API-Key", appKey)
	}
	if sign, ok := params["sign"].(string); ok {
		req.Header.Set("X-Signature", sign)
	}

	logger.Info("HTTP请求头设置完成",
		"callback_url", callbackURL,
		"content_type", req.Header.Get("Content-Type"),
		"user_agent", req.Header.Get("User-Agent"),
		"x_api_key", req.Header.Get("X-API-Key"),
		"x_signature_length", len(req.Header.Get("X-Signature")),
	)

	// 发送请求
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	logger.Info("开始发送HTTP请求",
		"callback_url", callbackURL,
		"method", "POST",
		"timeout", "30s",
	)

	startTime := time.Now()
	resp, err := client.Do(req)
	duration := time.Since(startTime)

	if err != nil {
		logger.Error("HTTP请求发送失败",
			"error", err,
			"callback_url", callbackURL,
			"duration", duration,
		)
		return fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	logger.Info("HTTP请求发送完成",
		"callback_url", callbackURL,
		"status_code", resp.StatusCode,
		"duration", duration,
		"content_length", resp.ContentLength,
	)

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("读取HTTP响应失败",
			"error", err,
			"callback_url", callbackURL,
			"status_code", resp.StatusCode,
		)
		return fmt.Errorf("读取响应失败: %v", err)
	}

	logger.Info("HTTP响应读取完成",
		"callback_url", callbackURL,
		"status_code", resp.StatusCode,
		"response_body_size", len(body),
		"response_body", string(body),
	)

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		logger.Error("HTTP请求返回错误状态码",
			"callback_url", callbackURL,
			"status_code", resp.StatusCode,
			"response_body", string(body),
		)
		return fmt.Errorf("HTTP请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	// 解析响应（可选，根据外部API的响应格式调整）
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		logger.Warn("解析HTTP响应JSON失败，但HTTP状态码正常，认为请求成功",
			"callback_url", callbackURL,
			"error", err,
			"response_body", string(body),
		)
		// 如果解析失败，只要HTTP状态码是200就认为成功
		return nil
	}

	logger.Info("HTTP响应JSON解析成功",
		"callback_url", callbackURL,
		"response_data", result,
	)

	// 检查业务状态码（根据外部API的响应格式调整）
	if code, ok := result["code"]; ok {
		if codeInt, ok := code.(float64); ok && codeInt != 0 {
			logger.Error("外部API返回业务错误",
				"callback_url", callbackURL,
				"business_code", code,
				"response_data", result,
			)
			if msg, ok := result["message"]; ok {
				return fmt.Errorf("业务错误: %v", msg)
			}
			return fmt.Errorf("业务错误，错误码: %v", code)
		}
	}

	logger.Info("外部API通知处理成功",
		"callback_url", callbackURL,
		"duration", duration,
		"response_data", result,
	)

	return nil
}
