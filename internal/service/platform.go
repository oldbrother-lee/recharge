package service

import (
	"bytes"
	"context"
	"crypto/rc4"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"recharge-go/internal/model"
	"recharge-go/internal/repository"
	zclient "recharge-go/internal/service/zhangyu"
	logger "recharge-go/pkg/log"
	"recharge-go/pkg/redis"
	"recharge-go/pkg/signature"
	"strconv"
	"strings"
	"sync"
	"time"
)

// PlatformService 平台服务
type PlatformService struct {
	platformRepo       repository.PlatformRepository
	orderRepo          repository.OrderRepository
	externalAPIKeyRepo repository.ExternalAPIKeyRepository
	// 新增：用于得众预上报的平台账号配置与token缓存
	platformAccountRepo *repository.PlatformAccountRepository
	dzTokenCache        map[string]string
	dzTokenMu           sync.RWMutex
}

var ErrSkipNonTerminal = errors.New("skip non-terminal notification")

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
		OrderMode:   1,
	}
	// 若请求中携带了平台模式，则使用请求值
	if req.OrderMode != 0 {
		platform.OrderMode = req.OrderMode
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
	if req.OrderMode != nil {
		platform.OrderMode = *req.OrderMode
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
	// 计算账号模式：未传或非法值时继承平台模式，否则使用传入值
	orderMode := req.OrderMode
	if orderMode != 1 && orderMode != 2 {
		plat, err := s.GetPlatformByID(req.PlatformID)
		if err == nil && plat != nil && (plat.OrderMode == 1 || plat.OrderMode == 2) {
			orderMode = plat.OrderMode
		} else {
			orderMode = 1 // 默认推单
		}
	}

	// 拉单开关与模式保持一致：推单模式强制关闭拉单；拉单模式默认开启
	enablePull := req.EnablePullOrder
	if orderMode == 1 {
		enablePull = false
	} else {
		enablePull = true
	}

	// 推单状态默认值：推单模式默认开启；拉单模式默认关闭
	pushStatus := 2
	if orderMode == 1 {
		pushStatus = 1
	}

	account := &model.PlatformAccount{
		PlatformID:      req.PlatformID,
		AccountName:     req.AccountName,
		Type:            req.Type,
		AppKey:          req.AppKey,
		AppSecret:       req.AppSecret,
		AccountPassword: req.AccountPassword,
		Description:     req.Description,
		DailyLimit:      req.DailyLimit,
		MonthlyLimit:    req.MonthlyLimit,
		Priority:        req.Priority,
		OrderMode:       orderMode,
		PushStatus:      pushStatus,
		EnablePullOrder: enablePull,
		MaxConcurrency:  req.MaxConcurrency,
		PollIntervalSec: req.PollIntervalSec,
		PullAction:      req.PullAction,
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
	if req.AccountPassword != nil {
		// 允许置空或更新密码（传入空字符串将清空密码）
		updateMap["account_password"] = *req.AccountPassword
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
	if req.OrderMode != nil {
		updateMap["order_mode"] = *req.OrderMode
	}
	// 补充：账号更新接口同时支持拉单配置相关字段
	if req.EnablePullOrder != nil {
		updateMap["enable_pull_order"] = *req.EnablePullOrder
	}
	if req.MaxConcurrency != nil {
		updateMap["max_concurrency"] = *req.MaxConcurrency
	}
	if req.PollIntervalSec != nil {
		updateMap["poll_interval_sec"] = *req.PollIntervalSec
	}
	if req.PullAction != nil {
		updateMap["pull_action"] = *req.PullAction
	}
	if req.BindUserID != nil {
		updateMap["bind_user_id"] = *req.BindUserID
	}
	if req.BindUserName != nil {
		updateMap["bind_user_name"] = *req.BindUserName
	}

	if len(updateMap) == 0 {
		return nil // 没有任何字段需要更新
	}

	return s.platformRepo.UpdatePlatformAccountFields(ctx, id, updateMap)
}

// DeletePlatformAccount 删除平台账号
func (s *PlatformService) DeletePlatformAccount(ctx context.Context, id int64) error {
	return s.platformRepo.DeleteAccount(ctx, id)
}

// SendNotification 发送订单状态通知
func (s *PlatformService) SendNotification(ctx context.Context, order *model.Order) error {
	// 首先检查是否为外部API通知，如果是则直接处理，无需获取平台账号
	if order.PlatformCallbackURL != "" {
		return s.sendExternalAPINotification(ctx, order)
	}

	// 1. 平台分支与配置
	// 得众平台走 token 认证，不依赖平台账号
	if strings.EqualFold(order.PlatformCode, "dz") {
		return s.sendDzReport(ctx, order)
	}
	// 其他平台需加载账号与平台配置
	account, err := s.platformRepo.GetPlatformAccountByID(order.PlatformAccountID)
	if err != nil {
		return fmt.Errorf("获取平台账号失败: %w", err)
	}
	platform, err := s.platformRepo.GetPlatformByID(account.PlatformID)
	if err != nil {
		return fmt.Errorf("获取平台配置失败: %w", err)
	}

	if order.Status != model.OrderStatusSuccess && order.Status != model.OrderStatusFailed {
		if !strings.EqualFold(platform.Code, "dz") {
			logger.WithContextCategory(ctx, "platform").Info("跳过非终态上报",
				logger.Int64V2("order_id", order.ID),
				logger.StringV2("order_number", order.OrderNumber),
				logger.StringV2("platform_code", platform.Code),
				logger.IntV2("order_status", int(order.Status)),
			)
			return ErrSkipNonTerminal
		}
	}

	// 3. 构建通知参数或进行平台特定处理
	var params map[string]interface{}
	switch platform.Code {
	case "mifeng":
		params = s.buildMf178Params(order, account)
	case "kekebang":
		params = s.buildKekebangParams(order, account)
	case "xianzhuanxia":
		// 闲赚侠一般直接调用 ReportTask 方法，不需要拼接 URL
		err := s.buildXianzhuanxiaParams(ctx, order, account, platform.ApiURL)
		if err != nil {
			return fmt.Errorf("上报订单结果失败: %w", err)
		}
		return nil
	case "internal_api":
		// 外部API通知处理
		return s.sendExternalAPINotification(ctx, order)
	case "xianyinke":
		params = s.buildXianyinkeParams(order, account)
	case "dz":
		// 得众平台预上报
		if err := s.sendDzPreReport(ctx, order); err != nil {
			return err
		}
		return nil
	case "zhangyu":
		// 章鱼平台：按状态上报结果（仅成功/失败）
		if order.Status == model.OrderStatusSuccess || order.Status == model.OrderStatusFailed {
			return s.sendZhangyuReport(ctx, account, order)
		}
		// 处理中或其它状态暂不需要上报
		logger.WithContextCategory(ctx, "platform").Info("章鱼平台当前状态无需上报",
			logger.Int64V2("order_id", order.ID),
			logger.IntV2("status", int(order.Status)))
		return nil
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
	case "xianyinke":
		url = platform.ApiURL + "/api/v1/helpOrder/report"
	default:
		return fmt.Errorf("不支持的平台: %s", platform.Code)
	}
	logger.WithContextCategory(ctx, "platform").Info("通知请求参数", logger.AnyV2("params", params))
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
	l := logger.WithContextCategory(ctx, "platform")
	l.Info("准备发送平台通知请求",
		logger.StringV2("url", url),
		logger.IntV2("params_count", len(params)),
	)
	// 1. 将参数转换为JSON
	jsonData, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("参数序列化失败: %w", err)
	}

	l.Info("平台通知请求体",
		logger.StringV2("stage", "notify_send"),
		logger.StringV2("body", string(jsonData)),
	)

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
	l.Info("请求已构建，开始发送",
		logger.StringV2("stage", "notify_send"),
		logger.StringV2("method", "POST"),
		logger.StringV2("url", url),
		logger.IntV2("timeout_seconds", 10),
	)
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
	// 响应预览
	previewLen := 256
	if len(body) < previewLen {
		previewLen = len(body)
	}
	l.Info("收到平台通知响应",
		logger.StringV2("stage", "notify_result"),
		logger.IntV2("status_code", resp.StatusCode),
		logger.StringV2("body", string(body)),
	)
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

func (s *PlatformService) buildXianyinkeParams(order *model.Order, account *model.PlatformAccount) map[string]interface{} {
	params := map[string]interface{}{
		"app_key":        account.AppKey,
		"id":             order.OutTradeNum,
		"status":         s.getXianyinkeStatus(order.Status),
		"error_msg":      s.getXianyinkeErrorMsg(order),
		"voucher_text":   "",
		"voucher_base64": "",
		"timestamp":      time.Now().Format("2006-01-02 15:04:05"),
	}
	return params
}

func (s *PlatformService) buildKekebangParams(order *model.Order, account *model.PlatformAccount) map[string]interface{} {
	data := map[string]interface{}{
		"user_order_id": order.OutTradeNum,
		"status":        s.getKekebangStatus(order.Status),
	}
	jsonStr, err := json.Marshal(data)
	if err != nil {
		logger.GetCategoryLogger("platform").Error("序列化通知数据失败", logger.ErrorV2(err), logger.StringV2("order_number", order.OutTradeNum))
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
		logger.GetCategoryLogger("platform").Error("序列化通知数据失败", logger.ErrorV2(err), logger.StringV2("order_number", order.OutTradeNum))
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

func (s *PlatformService) getXianyinkeStatus(orderStatus model.OrderStatus) int {
	switch orderStatus {
	case model.OrderStatusSuccess:
		return 5 // 闲赢客成功状态码
	case model.OrderStatusFailed:
		return 4 // 闲赢客失败状态码
	case model.OrderStatusProcessing:
		return 2 // 闲赢客处理中
	default:
		return 0
	}
}

// 新增：闲赢客 error_msg 生成逻辑
func (s *PlatformService) getXianyinkeErrorMsg(order *model.Order) string {
	switch order.Status {
	case model.OrderStatusFailed:
		if order.Remark != "" {
			return order.Remark
		}
		return "充值失败"
	case model.OrderStatusSuccess:
		return "充值成功"
	case model.OrderStatusProcessing:
		return ""
	default:
		return ""
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
	case "xianyinke":
		return signature.GenerateXianyinkeSign(params, account.AppSecret)
	default:
		return ""
	}
}

func (s *PlatformService) buildXianzhuanxiaParams(ctx context.Context, order *model.Order, account *model.PlatformAccount, apiURL string) error {

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

	logger.WithContextCategory(ctx, "platform").Info("闲赚侠通知参数",
		logger.StringV2("body", string(jsonData)),
	)

	url := fmt.Sprintf("%s/api/task/recharge/reported", apiURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Auth_Token", authToken)
	// req.Header.Set("Query-Time", queryTime)
	logger.WithContextCategory(ctx, "platform").Info("闲赚侠请求对象预览", logger.StringV2("method", req.Method), logger.IntV2("header_count", len(req.Header)))
	logger.WithContextCategory(ctx, "platform").Info("发送闲赚侠上报订单结果请求",
		logger.StringV2("url", url),
		logger.StringV2("content_type", req.Header.Get("Content-Type")),
	)
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
	// 将订单号注入上下文，便于日志链路追踪
	ctx = logger.InjectOrderNumber(ctx, order.OrderNumber)
	l := logger.WithContextCategory(ctx, "platform")
	l.Info("开始处理外部API通知",
		logger.Int64V2("order_id", order.ID),
		logger.StringV2("order_number", order.OrderNumber),
		logger.Int64V2("customer_id", order.CustomerID),
		logger.IntV2("order_status", int(order.Status)),
		logger.StringV2("callback_url", order.PlatformCallbackURL),
	)

	// 只发送成功和失败状态的通知，其他状态不发送
	if order.Status != model.OrderStatusSuccess && order.Status != model.OrderStatusFailed {
		l.Info("订单状态不需要发送通知，跳过",
			logger.Int64V2("order_id", order.ID),
			logger.StringV2("order_number", order.OrderNumber),
			logger.IntV2("order_status", int(order.Status)),
		)
		return nil // 不发送通知，直接返回
	}

	// 检查是否有回调URL
	if order.PlatformCallbackURL == "" {
		l.Error("外部API订单缺少回调URL",
			logger.Int64V2("order_id", order.ID),
			logger.StringV2("order_number", order.OrderNumber),
			logger.Int64V2("customer_id", order.CustomerID),
		)
		return fmt.Errorf("外部API订单缺少回调URL")
	}

	// 构建通知参数
	l.Info("开始构建外部API通知参数",
		logger.Int64V2("order_id", order.ID),
		logger.StringV2("order_number", order.OrderNumber),
	)
	params := s.buildExternalAPIParams(order)
	l.Info("外部API通知参数构建完成",
		logger.Int64V2("order_id", order.ID),
		logger.StringV2("order_number", order.OrderNumber),
		logger.IntV2("params_count", len(params)),
	)

	// 生成签名
	l.Info("开始生成外部API签名",
		logger.Int64V2("order_id", order.ID),
		logger.StringV2("order_number", order.OrderNumber),
		logger.Int64V2("customer_id", order.CustomerID),
	)
	sign := s.generateExternalAPISign(params, order)
	if sign == "" {
		l.Error("外部API签名生成失败",
			logger.Int64V2("order_id", order.ID),
			logger.StringV2("order_number", order.OrderNumber),
			logger.Int64V2("customer_id", order.CustomerID),
		)
		return fmt.Errorf("外部API签名生成失败")
	}
	params["sign"] = sign
	l.Info("外部API签名生成成功",
		logger.Int64V2("order_id", order.ID),
		logger.StringV2("order_number", order.OrderNumber),
		logger.IntV2("sign_length", len(sign)),
	)

	// 发送HTTP通知
	l.Info("开始发送外部API HTTP通知",
		logger.Int64V2("order_id", order.ID),
		logger.StringV2("order_number", order.OrderNumber),
		logger.StringV2("callback_url", order.PlatformCallbackURL),
	)
	err := s.sendExternalAPIHTTPNotification(ctx, order.PlatformCallbackURL, params)
	if err != nil {
		l.Error("外部API HTTP通知发送失败",
			logger.Int64V2("order_id", order.ID),
			logger.StringV2("order_number", order.OrderNumber),
			logger.StringV2("callback_url", order.PlatformCallbackURL),
			logger.ErrorV2(err),
		)
		return err
	}

	l.Info("外部API通知发送成功",
		logger.Int64V2("order_id", order.ID),
		logger.StringV2("order_number", order.OrderNumber),
		logger.StringV2("callback_url", order.PlatformCallbackURL),
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
	l := logger.GetCategoryLogger("platform")
	l.Info("开始获取外部API密钥",
		logger.Int64V2("customer_id", order.CustomerID),
		logger.Int64V2("order_id", order.ID),
		logger.StringV2("order_number", order.OrderNumber),
	)

	// 根据订单的客户ID获取外部API密钥信息
	apiKeys, total, err := s.externalAPIKeyRepo.GetByUserID(order.CustomerID, 0, 1)
	if err != nil {
		// 如果无法获取API密钥，记录日志并返回空签名
		l.Error("获取外部API密钥失败",
			logger.ErrorV2(err),
			logger.Int64V2("customer_id", order.CustomerID),
			logger.Int64V2("order_id", order.ID),
			logger.StringV2("order_number", order.OrderNumber),
		)
		return ""
	}

	l.Info("外部API密钥查询结果",
		logger.Int64V2("customer_id", order.CustomerID),
		logger.Int64V2("order_id", order.ID),
		logger.StringV2("order_number", order.OrderNumber),
		logger.IntV2("total_keys", int(total)),
		logger.IntV2("returned_keys", len(apiKeys)),
	)

	// 检查是否有API密钥
	if len(apiKeys) == 0 {
		l.Error("用户没有配置外部API密钥",
			logger.Int64V2("customer_id", order.CustomerID),
			logger.Int64V2("order_id", order.ID),
			logger.StringV2("order_number", order.OrderNumber),
			logger.IntV2("total_keys", int(total)),
		)
		return ""
	}

	// 使用第一个API密钥生成签名
	apiKey := apiKeys[0]
	l.Info("发送端签名生成参数",
		logger.Int64V2("customer_id", order.CustomerID),
		logger.Int64V2("order_id", order.ID),
		logger.StringV2("order_number", order.OrderNumber),
		logger.Int64V2("api_key_id", apiKey.ID),
		logger.StringV2("app_id", apiKey.AppID),
		logger.IntV2("secret_length", len(apiKey.AppSecret)),
		logger.IntV2("params_count", len(params)),
	)

	// 使用外部API签名验证器生成签名
	signatureValidator := signature.NewExternalAPISignatureValidator()
	sign, err := signatureValidator.GenerateExternalAPISignature(params, apiKey.AppSecret)
	if err != nil {
		l.Error("外部API签名生成失败",
			logger.ErrorV2(err),
			logger.Int64V2("customer_id", order.CustomerID),
			logger.Int64V2("order_id", order.ID),
			logger.StringV2("order_number", order.OrderNumber),
		)
		return ""
	}
	l.Info("发送端签名生成完成",
		logger.Int64V2("customer_id", order.CustomerID),
		logger.Int64V2("order_id", order.ID),
		logger.StringV2("order_number", order.OrderNumber),
		logger.Int64V2("api_key_id", apiKey.ID),
		logger.IntV2("sign_length", len(sign)),
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
	l := logger.WithContextCategory(ctx, "platform")
	l.Info("开始构建HTTP请求",
		logger.StringV2("callback_url", callbackURL),
		logger.IntV2("params_count", len(params)),
	)

	// 构建请求体
	jsonData, err := json.Marshal(params)
	if err != nil {
		l.Error("序列化参数失败",
			logger.ErrorV2(err),
			logger.StringV2("callback_url", callbackURL),
			logger.IntV2("params_count", len(params)),
		)
		return fmt.Errorf("序列化参数失败: %v", err)
	}

	l.Info("HTTP请求体",
		logger.StringV2("callback_url", callbackURL),
		logger.StringV2("request_body", string(jsonData)),
	)

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "POST", callbackURL, bytes.NewBuffer(jsonData))
	if err != nil {
		l.Error("创建HTTP请求失败",
			logger.ErrorV2(err),
			logger.StringV2("callback_url", callbackURL),
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

	l.Info("HTTP请求头设置完成",
		logger.StringV2("callback_url", callbackURL),
		logger.StringV2("content_type", req.Header.Get("Content-Type")),
		logger.StringV2("user_agent", req.Header.Get("User-Agent")),
		logger.StringV2("x_api_key", req.Header.Get("X-API-Key")),
		logger.IntV2("x_signature_length", len(req.Header.Get("X-Signature"))),
	)

	// 发送请求
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	l.Info("开始发送HTTP请求",
		logger.StringV2("callback_url", callbackURL),
		logger.StringV2("method", "POST"),
		logger.StringV2("timeout", "30s"),
	)

	startTime := time.Now()
	resp, err := client.Do(req)
	duration := time.Since(startTime)

	if err != nil {
		l.Error("HTTP请求发送失败",
			logger.ErrorV2(err),
			logger.StringV2("callback_url", callbackURL),
			logger.DurationV2("duration", duration),
		)
		return fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	l.Info("HTTP请求发送完成",
		logger.StringV2("callback_url", callbackURL),
		logger.IntV2("status_code", resp.StatusCode),
		logger.DurationV2("duration", duration),
		logger.Int64V2("content_length", resp.ContentLength),
	)

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		l.Error("读取HTTP响应失败",
			logger.ErrorV2(err),
			logger.StringV2("callback_url", callbackURL),
			logger.IntV2("status_code", resp.StatusCode),
		)
		return fmt.Errorf("读取响应失败: %v", err)
	}

	respPreviewLen := 256
	if len(body) < respPreviewLen {
		respPreviewLen = len(body)
	}
	l.Info("HTTP响应读取完成",
		logger.StringV2("callback_url", callbackURL),
		logger.IntV2("status_code", resp.StatusCode),
		logger.IntV2("response_body_size", len(body)),
		logger.StringV2("response_body_preview", string(body[:respPreviewLen])),
	)

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		l.Error("HTTP请求返回错误状态码",
			logger.StringV2("callback_url", callbackURL),
			logger.IntV2("status_code", resp.StatusCode),
			logger.StringV2("response_body_preview", string(body[:respPreviewLen])),
		)
		return fmt.Errorf("HTTP请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	// 解析响应（可选，根据外部API的响应格式调整）
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		l.Warn("解析HTTP响应JSON失败，但HTTP状态码正常，认为请求成功",
			logger.StringV2("callback_url", callbackURL),
			logger.ErrorV2(err),
			logger.StringV2("response_body_preview", string(body[:respPreviewLen])),
		)
		// 如果解析失败，只要HTTP状态码是200就认为成功
		return nil
	}

	l.Info("HTTP响应JSON解析成功",
		logger.StringV2("callback_url", callbackURL),
		logger.AnyV2("response_data_keys", func() []string {
			keys := make([]string, 0, len(result))
			for k := range result {
				keys = append(keys, k)
			}
			return keys
		}()),
	)

	// 检查业务状态码（根据外部API的响应格式调整）
	if code, ok := result["code"]; ok {
		if codeInt, ok := code.(float64); ok && codeInt != 200 {
			l.Error("外部API返回业务错误",
				logger.StringV2("callback_url", callbackURL),
				logger.Float64V2("business_code", codeInt),
				logger.AnyV2("response_data_keys", func() []string {
					keys := make([]string, 0, len(result))
					for k := range result {
						keys = append(keys, k)
					}
					return keys
				}()),
			)
			if msg, ok := result["message"]; ok {
				return fmt.Errorf("业务错误: %v", msg)
			}
			return fmt.Errorf("业务错误，错误码: %v", code)
		}
	}

	l.Info("外部API通知处理成功",
		logger.StringV2("callback_url", callbackURL),
		logger.DurationV2("duration", duration),
		logger.AnyV2("response_data_keys", func() []string {
			keys := make([]string, 0, len(result))
			for k := range result {
				keys = append(keys, k)
			}
			return keys
		}()),
	)

	return nil
}

// 新增：得众平台预上报实现
func (s *PlatformService) sendDzPreReport(ctx context.Context, order *model.Order) error {
	l := logger.WithContextCategory(ctx, "platform")
	// 使用订单的 PlatformAccountID 字段
	if order.PlatformAccountID == 0 {
		l.Error("得众预上报缺少平台账号ID", logger.Int64V2("order_id", order.ID))
		return fmt.Errorf("缺少平台账号ID用于得众预上报")
	}
	if s.platformAccountRepo == nil {
		// 惰性初始化：在首次使用时创建仓库实例（避免改动过多构造处）
		s.platformAccountRepo = repository.NewPlatformAccountRepository(s.platformRepo.GetDB())
	}
	platformAccount, err := s.platformAccountRepo.GetByIDWithContext(ctx, order.PlatformAccountID)
	if err != nil || platformAccount == nil {
		l.Error("获取平台账号失败", logger.Int64V2("platform_account_id", order.PlatformAccountID), logger.ErrorV2(err))
		return fmt.Errorf("获取平台账号失败: %w", err)
	}
	baseURL := strings.TrimSpace(platformAccount.Platform.ApiURL)
	rc4Key := strings.TrimSpace(platformAccount.AppSecret) // 使用AppSecret作为RC4密钥
	username := strings.TrimSpace(platformAccount.AccountName)
	password := strings.TrimSpace(platformAccount.AccountPassword)
	if baseURL == "" || rc4Key == "" || username == "" {
		l.Error("得众平台账号配置不完整",
			logger.Int64V2("platform_account_id", order.PlatformAccountID),
			logger.StringV2("base_url", baseURL),
			logger.StringV2("rc4_key", rc4Key),
			logger.StringV2("username", username),
		)
		return fmt.Errorf("得众平台账号配置不完整")
	}
	if password == "" {
		// 允许从环境变量兜底
		if envPwd := strings.TrimSpace(os.Getenv("RECHARGE_DZ_PASSWORD")); envPwd != "" {
			password = envPwd
		}
	}
	// 仅从 Redis 读取 token（通知不负责登录/写入）
	var token string
	redisKey := fmt.Sprintf("dz:token:%d:%s", order.PlatformAccountID, username)
	if rc := redis.GetClient(); rc != nil {
		if v, err := rc.Get(ctx, redisKey).Result(); err == nil && strings.TrimSpace(v) != "" {
			token = v
			l.Info("得众复用Redis缓存token", logger.StringV2("mask", maskToken(v)))
		}
	}
	if strings.TrimSpace(token) == "" {
		l.Error("得众Redis未找到token", logger.StringV2("redis_key", redisKey))
		return fmt.Errorf("缺少得众token，请确保拉单模块已登录并写入Redis(key=%s)", redisKey)
	}
	// 构造预上报请求（使用正确的格式）
	dzResult := s.getDzResult(order.Status)
	reason := s.getDzReason(order.Status)

	// 构造remark字段（包含订单相关信息）
	remark := fmt.Sprintf("运营商:%s;订单号:%s;手机号:%s;面额:%.2f;状态:%s;版本号:1.0.0.0",
		s.getISPName(order.ISP), order.OrderNumber, order.Mobile, order.Denom, s.getStatusText(order.Status))

	// 构造context字段（使用固定值）
	context := "ptransId=INVITE_2023666888;cookie=invite_dxfs"

	req := map[string]interface{}{
		"action": "report",
		"flag":   "invite_dxfs",
		"ver":    "1.0.0.0",
		"token":  token,
		"data": map[string]interface{}{
			"id":      order.OutTradeNum, // 使用字符串格式的订单ID
			"mobile":  order.Mobile,
			"target":  order.Mobile, // target通常和mobile相同
			"reason":  reason,
			"remark":  remark,
			"context": context,
			"result":  dzResult,
		},
	}

	// 记录上报参数和地址
	l.Info("得众预上报请求详情",
		logger.StringV2("base_url", baseURL),
		logger.Int64V2("platform_account_id", order.PlatformAccountID),
		logger.Int64V2("order_id", order.ID),
		logger.StringV2("out_trade_num", order.OutTradeNum),
		logger.IntV2("dz_result", dzResult),
		logger.StringV2("dz_reason", reason),
		logger.StringV2("username", username),
		logger.StringV2("rc4_key_length", fmt.Sprintf("%d", len(rc4Key))),
	)
	enc, encErr := s.rc4EncryptJSON(req, rc4Key)
	if encErr != nil {
		l.Error("得众请求加密失败", logger.ErrorV2(encErr))
		return encErr
	}

	// 记录加密后的请求数据（截取前100字符避免日志过长）
	encPreview := enc
	if len(enc) > 100 {
		encPreview = enc[:100] + "..."
	}
	l.Info("得众加密请求数据", logger.StringV2("encrypted_data", encPreview))

	// 构建包含devKey参数的完整URL（与登录保持一致）
	apiEndpoint := s.buildDzEndpoint(baseURL, platformAccount.AppKey)
	l.Info("得众预上报完整地址", logger.StringV2("api_endpoint", apiEndpoint))

	dec, postErr := s.dzPostAndDecrypt(apiEndpoint, enc, rc4Key)
	if postErr != nil {
		l.Error("得众HTTP请求失败", logger.ErrorV2(postErr))
		return postErr
	}

	// 记录原始响应数据
	l.Info("得众原始响应", logger.StringV2("response", dec))

	var statusResp map[string]interface{}
	if err := json.Unmarshal([]byte(dec), &statusResp); err != nil {
		l.Error("解析得众响应失败", logger.ErrorV2(err), logger.StringV2("resp", dec))
		return nil
	}

	// 记录解析后的响应结构
	respJSON, _ := json.Marshal(statusResp)
	l.Info("得众解析后响应", logger.StringV2("parsed_response", string(respJSON)))

	if ret, ok := statusResp["ret"].(float64); ok && int(ret) != 0 {
		msg := ""
		if m, ok := statusResp["msg"].(string); ok {
			msg = m
		}
		// 不做重登重试：通知只读Redis token
		l.Error("得众预上报返回失败",
			logger.IntV2("ret_code", int(ret)),
			logger.StringV2("msg", msg),
			logger.StringV2("full_response", string(respJSON)),
		)
		return fmt.Errorf("得众预上报失败: ret=%d, msg=%s", int(ret), msg)
	}
	l.Info("得众预上报完成", logger.Int64V2("order_id", order.ID), logger.IntV2("dz_result", dzResult))
	return nil
}

// RC4(JSON)->Base64 加密
func (s *PlatformService) rc4EncryptJSON(data interface{}, key string) (string, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	c, err := rc4.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}
	out := make([]byte, len(b))
	c.XORKeyStream(out, b)
	return base64.StdEncoding.EncodeToString(out), nil
}

// 得众HTTP请求并解密
func (s *PlatformService) dzPostAndDecrypt(url string, enc string, rc4Key string) (string, error) {
	// 记录HTTP请求详情
	logger.Log.Info("得众HTTP请求详情",
		logger.StringV2("url", url),
		logger.StringV2("method", "POST"),
		logger.IntV2("data_length", len(enc)),
	)
	resp, err := http.Post(url, "application/x-www-form-urlencoded", strings.NewReader(enc))
	if err != nil {
		return "", fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 记录HTTP响应状态
	logger.Log.Info("得众HTTP响应状态",
		logger.IntV2("status_code", resp.StatusCode),
		logger.StringV2("status", resp.Status),
		logger.StringV2("content_type", resp.Header.Get("Content-Type")),
	)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}

	// 记录原始响应内容（前200字符）
	bodyPreview := string(body)
	if len(bodyPreview) > 200 {
		bodyPreview = bodyPreview[:200] + "..."
	}
	logger.Log.Info("得众HTTP原始响应",
		logger.IntV2("body_length", len(body)),
		logger.StringV2("body_preview", bodyPreview),
	)

	var base64Data string
	// 响应可能是JSON字符串或直接Base64
	if len(body) > 0 && body[0] == '"' && body[len(body)-1] == '"' {
		if err := json.Unmarshal(body, &base64Data); err != nil {
			logger.Log.Error("解析JSON响应失败", logger.ErrorV2(err), logger.StringV2("raw_body", string(body)))
			return "", err
		}
		previewLen := 100
		if len(base64Data) < previewLen {
			previewLen = len(base64Data)
		}
		logger.Log.Info("得众响应为JSON格式", logger.StringV2("base64_data_preview", base64Data[:previewLen]))
	} else {
		base64Data = string(body)
		previewLen := 100
		if len(base64Data) < previewLen {
			previewLen = len(base64Data)
		}
		logger.Log.Info("得众响应为直接base64格式", logger.StringV2("base64_data_preview", base64Data[:previewLen]))
	}

	// 解密
	data, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		logger.Log.Error("base64解码失败",
			logger.ErrorV2(err),
			logger.StringV2("base64_data", base64Data),
			logger.IntV2("data_length", len(base64Data)),
		)
		return "", err
	}
	c, err := rc4.NewCipher([]byte(rc4Key))
	if err != nil {
		return "", err
	}
	out := make([]byte, len(data))
	c.XORKeyStream(out, data)
	return string(out), nil
}

// 得众状态码映射：1 预上报，2 成功，3 失败
func (s *PlatformService) getDzStatus(orderStatus model.OrderStatus) int {
	switch orderStatus {
	case model.OrderStatusProcessing:
		return 1
	case model.OrderStatusSuccess:
		return 2
	case model.OrderStatusFailed:
		return 3
	default:
		return 1
	}
}

func (s *PlatformService) getDzMessage(status int) string {
	switch status {
	case 1:
		return "预上报成功"
	case 2:
		return "充值成功"
	case 3:
		return "充值失败"
	default:
		return "状态上报"
	}
}

// getDzResult 得众结果码映射：1 成功
func (s *PlatformService) getDzResult(orderStatus model.OrderStatus) int {
	switch orderStatus {
	case model.OrderStatusProcessing:
		return 1 // 预上报成功
	case model.OrderStatusSuccess:
		return 1 // 充值成功
	case model.OrderStatusFailed:
		return 2 // 充值失败
	default:
		return 1
	}
}

// getDzReason 得众原因描述
func (s *PlatformService) getDzReason(orderStatus model.OrderStatus) string {
	switch orderStatus {
	case model.OrderStatusProcessing:
		return "下单成功，付款成功"
	case model.OrderStatusSuccess:
		return "充值成功"
	case model.OrderStatusFailed:
		return "充值失败"
	default:
		return "状态上报"
	}
}

// getISPName 获取运营商名称
func (s *PlatformService) getISPName(isp int) string {
	switch isp {
	case 1:
		return "移动"
	case 2:
		return "联通"
	case 3:
		return "电信"
	default:
		return "未知"
	}
}

// 辅助：掩码token日志展示
func maskToken(s string) string {
	if len(s) <= 6 {
		return s
	}
	return s[:3] + "***" + s[len(s)-3:]
}

// buildDzEndpoint 构建得众平台请求地址：优先使用配置中的完整URL；否则拼接 /api/phrecharge?devKey=<AppKey>
func (s *PlatformService) buildDzEndpoint(baseURL, appKey string) string {
	u := strings.TrimSpace(baseURL)
	if u == "" {
		return ""
	}
	// 如果已经包含 devKey 参数，直接使用
	if strings.Contains(u, "devKey=") {
		return u
	}
	// 如果未包含标准路径，则补上
	endpoint := strings.TrimRight(u, "/")
	if !strings.Contains(endpoint, "/api/phrecharge") {
		endpoint += "/api/phrecharge"
	}
	// 处理已有查询参数
	sep := "?"
	if strings.Contains(endpoint, "?") {
		sep = "&"
	}
	return endpoint + sep + "devKey=" + url.QueryEscape(appKey)
}

// sendDzReport 根据订单状态发送DZ平台上报
func (s *PlatformService) sendDzReport(ctx context.Context, order *model.Order) error {
	l := logger.WithContextCategory(ctx, "platform")

	switch order.Status {
	case model.OrderStatusProcessing:
		// 处理中状态：发送预上报
		l.Info("订单处理中，发送DZ预上报", logger.Int64V2("order_id", order.ID))
		return s.sendDzPreReport(ctx, order)
	case model.OrderStatusSuccess, model.OrderStatusFailed:
		// 成功或失败状态：发送真实状态上报
		l.Info("订单已完成，发送DZ真实状态上报",
			logger.Int64V2("order_id", order.ID),
			logger.IntV2("status", int(order.Status)))
		return s.sendDzRealStatusReport(ctx, order)
	default:
		// 其他状态暂不处理
		l.Info("订单状态无需上报",
			logger.Int64V2("order_id", order.ID),
			logger.IntV2("status", int(order.Status)))
		return nil
	}
}

// sendDzRealStatusReport 发送DZ平台真实状态上报
func (s *PlatformService) sendDzRealStatusReport(ctx context.Context, order *model.Order) error {
	l := logger.WithContextCategory(ctx, "platform")

	// 从订单 PlatformAccountID 获取平台账号ID
	if order.PlatformAccountID == 0 {
		l.Error("得众真实状态上报缺少平台账号ID", logger.Int64V2("order_id", order.ID))
		return fmt.Errorf("缺少平台账号ID用于得众真实状态上报")
	}

	if s.platformAccountRepo == nil {
		// 惰性初始化：在首次使用时创建仓库实例
		s.platformAccountRepo = repository.NewPlatformAccountRepository(s.platformRepo.GetDB())
	}
	platformAccount, err := s.platformAccountRepo.GetByIDWithContext(ctx, order.PlatformAccountID)
	if err != nil || platformAccount == nil {
		l.Error("获取平台账号失败", logger.Int64V2("platform_account_id", order.PlatformAccountID), logger.ErrorV2(err))
		return fmt.Errorf("获取平台账号失败: %w", err)
	}

	baseURL := strings.TrimSpace(platformAccount.Platform.ApiURL)
	rc4Key := strings.TrimSpace(platformAccount.AppSecret)
	username := strings.TrimSpace(platformAccount.AccountName)
	password := strings.TrimSpace(platformAccount.AccountPassword)

	if baseURL == "" || rc4Key == "" || username == "" {
		return fmt.Errorf("源配置不完整: base_url/app_key/account_name 不能为空")
	}
	if password == "" {
		// 允许从环境变量兜底
		if envPwd := strings.TrimSpace(os.Getenv("RECHARGE_DZ_PASSWORD")); envPwd != "" {
			password = envPwd
		}
	}

	// 仅从 Redis 读取 token（通知不负责登录/写入）
	var token string
	redisKey := fmt.Sprintf("dz:token:%d:%s", order.PlatformAccountID, username)
	if rc := redis.GetClient(); rc != nil {
		if v, err := rc.Get(ctx, redisKey).Result(); err == nil && strings.TrimSpace(v) != "" {
			token = v
			l.Info("得众复用Redis缓存token", logger.StringV2("mask", maskToken(v)))
		}
	}
	if strings.TrimSpace(token) == "" {
		l.Error("得众Redis未找到token", logger.StringV2("redis_key", redisKey))
		return fmt.Errorf("缺少得众token，请确保拉单模块已登录并写入Redis(key=%s)", redisKey)
	}

	// 构建真实状态上报请求
	dzResult := s.getDzResult(order.Status)
	dzReason := s.getDzReason(order.Status)

	// 根据订单状态确定action参数
	// 成功状态：action=status, result=1
	// 失败状态：action=report, result=2
	var action string
	if order.Status == model.OrderStatusSuccess {
		action = "status"
	} else {
		action = "report"
	}

	// 构造remark字段（包含订单相关信息）
	remark := fmt.Sprintf("运营商:%s;订单号:%s;手机号:%s;面额:%.2f;状态:%s;版本号:1.0.0.0",
		s.getISPName(order.ISP), order.OrderNumber, order.Mobile, order.Denom, s.getStatusText(order.Status))

	// 构造context字段（使用固定值）
	context := "ptransId=INVITE_2023666888;cookie=invite_dxfs"

	req := map[string]interface{}{
		"action": action,
		"flag":   "invite_dxfs",
		"ver":    "1.0.0.0",
		"token":  token,
		"data": map[string]interface{}{
			"id":      order.OutTradeNum, // 使用字符串格式的订单ID
			"mobile":  order.Mobile,
			"target":  order.Mobile, // target通常和mobile相同
			"reason":  dzReason,
			"remark":  remark,
			"context": context,
			"result":  dzResult,
		},
	}

	// 记录完整的请求参数以便调试
	reqJSON, _ := json.Marshal(req)
	l.Info("得众真实状态上报请求详情",
		logger.StringV2("base_url", baseURL),
		logger.Int64V2("platform_account_id", order.PlatformAccountID),
		logger.Int64V2("order_id", order.ID),
		logger.StringV2("out_trade_num", order.OutTradeNum),
		logger.StringV2("action", action),
		logger.IntV2("dz_result", dzResult),
		logger.StringV2("dz_reason", dzReason),
		logger.StringV2("mobile", order.Mobile),
		logger.StringV2("order_status", s.getStatusText(order.Status)),
		logger.StringV2("username", username),
		logger.StringV2("rc4_key_length", fmt.Sprintf("%d", len(rc4Key))),
		logger.StringV2("full_request", string(reqJSON)),
	)
	enc, encErr := s.rc4EncryptJSON(req, rc4Key)
	if encErr != nil {
		l.Error("得众请求加密失败", logger.ErrorV2(encErr))
		return encErr
	}

	// 记录加密后的请求数据（截取前100字符避免日志过长）
	encPreview := enc
	if len(enc) > 100 {
		encPreview = enc[:100] + "..."
	}
	l.Info("得众加密请求数据", logger.StringV2("encrypted_data", encPreview))

	// 构建包含devKey参数的完整URL
	fullURL := s.buildDzEndpoint(baseURL, platformAccount.AppKey)
	l.Info("得众实时状态上报URL", logger.StringV2("url", fullURL))

	dec, postErr := s.dzPostAndDecrypt(fullURL, enc, rc4Key)
	if postErr != nil {
		l.Error("得众HTTP请求失败", logger.ErrorV2(postErr))
		return postErr
	}

	// 记录原始响应数据
	l.Info("得众原始响应", logger.StringV2("response", dec))

	var statusResp map[string]interface{}
	if err := json.Unmarshal([]byte(dec), &statusResp); err != nil {
		l.Error("解析得众响应失败", logger.ErrorV2(err), logger.StringV2("resp", dec))
		return nil
	}

	// 记录解析后的响应结构
	respJSON, _ := json.Marshal(statusResp)
	l.Info("得众解析后响应", logger.StringV2("parsed_response", string(respJSON)))

	if ret, ok := statusResp["ret"].(float64); ok && int(ret) != 0 {
		msg := ""
		if m, ok := statusResp["msg"].(string); ok {
			msg = m
		}
		// 不做重登重试：通知只读Redis token
		l.Error("得众真实状态上报返回失败",
			logger.IntV2("ret_code", int(ret)),
			logger.StringV2("msg", msg),
			logger.StringV2("full_response", string(respJSON)),
		)
		return fmt.Errorf("得众真实状态上报失败: ret=%d, msg=%s", int(ret), msg)
	}
	l.Info("得众真实状态上报完成", logger.Int64V2("order_id", order.ID), logger.IntV2("dz_result", dzResult))
	return nil
}

// sendZhangyuReport 基于章鱼客户端进行订单结果上报（仅成功/失败）
func (s *PlatformService) sendZhangyuReport(ctx context.Context, account *model.PlatformAccount, order *model.Order) error {
	l := logger.WithContextCategory(ctx, "zhangyu")

	if account == nil || account.Platform.ID == 0 {
		return fmt.Errorf("章鱼平台账号信息缺失")
	}

	baseURL := strings.TrimSpace(account.Platform.ApiURL)
	if baseURL == "" {
		return fmt.Errorf("章鱼平台ApiURL未配置")
	}

	// 仅从Redis读取token（通知不负责登录与写入）
	client := zclient.NewClient(baseURL)
	token, _ := client.LoadToken(ctx, account)
	if strings.TrimSpace(token) == "" {
		// 与得众保持一致：只读Redis，不做登录
		username := strings.TrimSpace(account.AccountName)
		redisKey := fmt.Sprintf("zhangyu:token:%d:%s", account.ID, username)
		l.Error("章鱼Redis未找到token", logger.StringV2("redis_key", redisKey))
		return fmt.Errorf("缺少章鱼token，请确保拉单模块已登录并写入Redis(key=%s)", redisKey)
	}

	// 计算渠道 flag：优先读取订单 param1；为空则按变种配置；最后回退 dxfs
	flag := strings.TrimSpace(order.Param1)
	if flag == "" {
		flag = "dxfs"
		variantRepo := repository.NewPlatformAccountVariantRepository(s.platformRepo.GetDB())
		if variantRepo != nil {
			if v, err := variantRepo.GetByISPAndFaceValue(ctx, order.PlatformAccountID, order.ISP, order.Denom); err == nil && v != nil {
				if fv := strings.TrimSpace(v.Flag); fv != "" {
					flag = fv
				}
			}
		}
	}

	// 仅成功/失败需要上报
	var result string
	var reason string
	switch order.Status {
	case model.OrderStatusSuccess:
		result = "1"
		reason = "充值成功"
	case model.OrderStatusFailed:
		result = "2"
		if r := strings.TrimSpace(order.Remark); r != "" {
			reason = r
		} else {
			reason = "充值失败"
		}
	default:
		l.Info("章鱼平台跳过非完成状态上报", logger.Int64V2("order_id", order.ID), logger.IntV2("status", int(order.Status)))
		return nil
	}

	// 订单创建时间格式化（无值则用当前时间）
	orderCreate := time.Now().Format("2006-01-02 15:04:05")
	if !order.CreateTime.IsZero() {
		orderCreate = order.CreateTime.Format("2006-01-02 15:04:05")
	}

	payload := zclient.ReportPayload{
		ID:              order.OutTradeNum,
		Result:          result,
		Reason:          reason,
		PtransID:        "",
		Cookie:          "",
		OrderCreateTime: orderCreate,
	}

	l.Info("章鱼上报请求构造完成",
		logger.Int64V2("order_id", order.ID),
		logger.StringV2("out_trade_num", order.OutTradeNum),
		logger.StringV2("flag", flag),
		logger.StringV2("result", result),
		logger.StringV2("reason", reason),
		logger.StringV2("api_url", baseURL),
	)

	if err := client.ReportOrder(ctx, account, token, flag, payload); err != nil {
		l.Error("章鱼上报失败", logger.ErrorV2(err))
		return fmt.Errorf("章鱼上报失败: %w", err)
	}
	l.Info("章鱼上报成功", logger.Int64V2("order_id", order.ID))
	return nil
}
