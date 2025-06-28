package platform

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"recharge-go/configs"
	"recharge-go/internal/model"
	"recharge-go/internal/repository"
	"recharge-go/pkg/logger"
	"recharge-go/pkg/redis"
	"recharge-go/pkg/signature"
	"strconv"
	"time"
)

// MilliTime 支持毫秒时间戳自动转 time.Time
type MilliTime struct {
	time.Time
}

func (mt *MilliTime) UnmarshalJSON(b []byte) error {
	// 去掉引号
	s := string(b)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}
	millis, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		mt.Time = time.Time{}
		return err
	}
	mt.Time = time.UnixMilli(millis)
	return nil
}

// PlatformOrder 平台返回的订单数据结构
type PlatformOrder struct {
	OrderNumber      string    `json:"orderNumber"`      // 订单号
	ChannelName      string    `json:"channelName"`      // 渠道名称
	ProductName      string    `json:"productName"`      // 产品名称
	ChannelId        int       `json:"channelId"`        // 渠道ID
	ProductId        int       `json:"productId"`        // 产品ID
	AccountNum       string    `json:"accountNum"`       // 充值账号
	AccountLocation  string    `json:"accountLocation"`  // 归属地
	SettlementAmount float64   `json:"settlementAmount"` // 结算金额
	FaceValue        float64   `json:"faceValue"`        // 面值
	OrderStatus      int       `json:"orderStatus"`      // 订单状态
	SettlementStatus int       `json:"settlementStatus"` // 结算状态
	CreateTime       MilliTime `json:"createTime"`       // 创建时间
	ExpirationTime   MilliTime `json:"expirationTime"`   // 过期时间
}

// Channel 渠道信息
type Channel struct {
	ChannelID   int       `json:"channelId"`   // 渠道编号
	ChannelName string    `json:"channelName"` // 渠道名称
	ProductList []Product `json:"productList"` // 渠道对应下的运营商信息
}

// Product 产品信息
type Product struct {
	ProductID   int    `json:"productId"`   // 运营商编号
	ProductName string `json:"productName"` // 运营商名称
}

// StockInfo 库存信息
type StockInfo struct {
	ChannelID int     `json:"channelId"` // 渠道编号
	ProductID int     `json:"productId"` // 运营商编号
	FaceValue float64 `json:"faceValue"` // 面值
	StockList []Stock `json:"stockList"` // 该面值的库存信息
}

// Stock 库存详情
type Stock struct {
	SettleAmount float64 `json:"settleAmount"` // 结算金额
	Stock        int     `json:"stock"`        // 库存数量
}

// PageResult 分页结果
type PageResult struct {
	EndRow   int64 `json:"endRow"`   // 结束行数
	PageNum  int64 `json:"pageNum"`  // 当前页码
	PageSize int64 `json:"pageSize"` // 每页多少条
	Pages    int64 `json:"pages"`    // 总页码
	StartRow int64 `json:"startRow"` // 开始行数
	Total    int64 `json:"total"`    // 总数
}

type Service struct {
	apiKey       string
	userID       string
	baseURL      string
	tokenRepo    *repository.PlatformTokenRepository
	platformRepo repository.PlatformRepository
}

func NewService(tokenRepo *repository.PlatformTokenRepository, platformRepo repository.PlatformRepository) *Service {
	cfg := configs.GetConfig()
	return &Service{
		apiKey:       cfg.API.Key,
		userID:       cfg.API.UserID,
		baseURL:      cfg.API.BaseURL,
		tokenRepo:    tokenRepo,
		platformRepo: platformRepo,
	}
}

// SubmitTask 申请做单
func (s *Service) SubmitTask(channelID int, productID string, provinces string, faceValues, minSettleAmounts string, apiKey, userID, apiURL string) (string, error) {
	params := map[string]string{
		"channelId":        strconv.Itoa(channelID),
		"productIds":       productID,
		"provinces":        "",
		"faceValues":       faceValues,
		"minSettleAmounts": minSettleAmounts,
	}
	// apiKey := "c362d30409744d7584abcbd3b58124c2"
	// userID := "558203"
	authToken, _, err := signature.GenerateXianzhuanxiaSignature(params, apiKey, userID)
	if err != nil {
		return "", fmt.Errorf("生成签名失败: %v", err)
	}
	url := fmt.Sprintf("%s/api/task/recharge/submit", apiURL)

	//添加请求头
	// 创建请求体
	logger.Info(fmt.Sprintf("申请做单url: %s", url))
	jsonData, err := json.Marshal(params)
	if err != nil {
		return "", fmt.Errorf("创建请求体失败: %v", err)
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Auth_Token", authToken)
	logger.Info(fmt.Sprintf("申请做单params: %s userid: %s", params, userID))
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logger.Error(fmt.Sprintf("申请做单请求发送失败: url=%s, error=%v", url, err))
		return "", fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error(fmt.Sprintf("申请做单响应读取失败: url=%s, error=%v", url, err))
		return "", fmt.Errorf("读取响应失败: %v", err)
	}

	logger.Info(fmt.Sprintf("申请做单响应: url=%s, status=%d, body=%s", url, resp.StatusCode, string(body)))

	if resp.StatusCode != http.StatusOK {
		logger.Error(fmt.Sprintf("申请做单HTTP状态码错误: url=%s, status=%d, body=%s", url, resp.StatusCode, string(body)))
		return "", fmt.Errorf("请求失败: %s", string(body))
	}

	var result struct {
		Code   int    `json:"code"`
		Msg    string `json:"msg"`
		Result struct {
			Token string `json:"token"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		logger.Error(fmt.Sprintf("申请做单响应解析失败: url=%s, body=%s, error=%v", url, string(body), err))
		return "", fmt.Errorf("解析响应失败: %v", err)
	}

	if result.Code != 0 {
		logger.Error(fmt.Sprintf("申请做单业务错误: url=%s, code=%d, msg=%s", url, result.Code, result.Msg))
		return "", fmt.Errorf("业务错误: %s", result.Msg)
	}

	logger.Info(fmt.Sprintf("申请做单成功: url=%s, token=%s", url, result.Result.Token))
	return result.Result.Token, nil
}

// QueryTask 查询申请做单是否匹配到订单
func (s *Service) QueryTask(token string, apiURL string, apiKey, userID string) (*PlatformOrder, error) {
	params := map[string]string{
		"token": token,
	}

	authToken, _, err := signature.GenerateXianzhuanxiaSignature(params, apiKey, userID)
	if err != nil {
		return nil, fmt.Errorf("生成签名失败: %v", err)
	}
	jsonData, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/api/task/recharge/query", apiURL)
	// url := "http://ip.jikelab.com:5000/api/orders"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		logger.Error("创建HTTP请求失败", "url", url, "error", err)
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Auth_Token", authToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("请求失败: %s", string(body))
	}
	logger.Info(fmt.Sprintf("做单查询接口返回: %v\n", string(body)))

	var result struct {
		Code   int    `json:"code"`
		Msg    string `json:"msg"`
		Result struct {
			MatchStatus int             `json:"matchStatus"`
			Orders      []PlatformOrder `json:"orders"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		logger.Error(fmt.Sprintf("查询订单响应解析失败: userid=%s url=%s, body=%s, error=%v", userID, url, string(body), err))
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	if result.Code != 0 {
		logger.Info(fmt.Sprintf("查询订单业务错误:userid=%s url=%s, code=%d, msg=%s", userID, url, result.Code, result.Msg))
		return nil, fmt.Errorf("业务错误: %s", result.Msg)
	}
	// result.Result.MatchStatus 如果等于 2 表示匹配失败，返回 nil
	if result.Result.MatchStatus == 2 {
		logger.Info(fmt.Sprintf("查询订单匹配失败:userid=%s, url=%s, code=%d, msg=%s, MatchStatus=%d", userID, url, result.Code, result.Msg, result.Result.MatchStatus))
		return nil, errors.New("匹配失败")
	}
	// result.Result.MatchStatus 如果等于 3 表示匹配成功，返回 result.Result.Orders[0]
	if result.Result.MatchStatus == 3 && len(result.Result.Orders) > 0 {
		logger.Info(fmt.Sprintf("查询订单匹配成功:userid=%s url=%s, code=%d, msg=%s, MatchStatus=%d", userID, url, result.Code, result.Msg, result.Result.MatchStatus))
		return &result.Result.Orders[0], nil
	}
	//result.Result.MatchStatus 如果等于 1 表示匹配中，返回 nil
	if result.Result.MatchStatus == 1 {
		logger.Info(fmt.Sprintf("查询订单匹配中:userid=%s url=%s, code=%d, msg=%s, MatchStatus=%d", userID, url, result.Code, result.Msg, result.Result.MatchStatus))
		return nil, nil
	}

	// if result.Result.MatchStatus != 3 || len(result.Result.Orders) == 0 {
	// 	return nil, nil
	// }

	return nil, nil
}

// ReportTask 上报做单订单结果
func (s *Service) ReportTask(orderNumber string, status int, remark, payVoucher, verifyData string) error {
	params := map[string]string{
		"orderNumber": orderNumber,
		"status":      strconv.Itoa(status),
		"remark":      remark,
		"payVoucher":  payVoucher,
		"verifyData":  verifyData,
	}

	authToken, queryTime, err := signature.GenerateXianzhuanxiaSignature(params, s.apiKey, s.userID)
	if err != nil {
		return fmt.Errorf("生成签名失败: %v", err)
	}

	url := fmt.Sprintf("%s/api/task/recharge/reported", s.baseURL)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Auth-Token", authToken)
	req.Header.Set("Query-Time", queryTime)

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

// GetOrderDetail 查询做单订单详情（单个）
func (s *Service) GetOrderDetail(orderNumber string) (*PlatformOrder, error) {
	params := map[string]string{
		"orderNumber": orderNumber,
	}

	authToken, queryTime, err := signature.GenerateXianzhuanxiaSignature(params, s.apiKey, s.userID)
	if err != nil {
		return nil, fmt.Errorf("生成签名失败: %v", err)
	}

	url := fmt.Sprintf("%s/api/task/recharge/orderDetail", s.baseURL)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		logger.Error("创建HTTP请求失败", "url", url, "error", err)
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Auth-Token", authToken)
	req.Header.Set("Query-Time", queryTime)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("请求失败: %s", string(body))
	}

	var result struct {
		Code   int           `json:"code"`
		Msg    string        `json:"msg"`
		Result PlatformOrder `json:"result"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	if result.Code != 0 {
		return nil, fmt.Errorf("业务错误: %s", result.Msg)
	}

	return &result.Result, nil
}

// GetOrderList 查询做单订单详情（分页）
func (s *Service) GetOrderList(orderNumber string, orderStatus, settlementStatus, pageNum, pageSize int, apiurl string, account *model.PlatformAccount) ([]PlatformOrder, *PageResult, error) {
	params := map[string]string{
		// "orderNumber":      orderNumber,
		"orderStatus": strconv.Itoa(orderStatus),
		// "settlementStatus": strconv.Itoa(settlementStatus),
		"pageNum":  strconv.Itoa(pageNum),
		"pageSize": strconv.Itoa(pageSize),
	}
	logger.Info(fmt.Sprintf("key %s userid %s", account.AppKey, account.AccountName))
	authToken, queryTime, err := signature.GenerateXianzhuanxiaSignature(params, account.AppKey, account.AccountName)
	if err != nil {
		return nil, nil, fmt.Errorf("生成签名失败: %v", err)
	}
	url := fmt.Sprintf("%s/api/task/recharge/page", apiurl)
	jsonData, err := json.Marshal(params)
	if err != nil {
		logger.Error("创建HTTP请求失败", "url", url, "error", err)
		return nil, nil, err
	}

	logger.Info(fmt.Sprintf("获取订单列表url: %s", url))
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, nil, fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Auth_Token", authToken)
	req.Header.Set("Query-Time", queryTime)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("读取响应失败: %v", err)
	}
	logger.Info(fmt.Sprintf("获取订单列表响应: %s", string(body)))
	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("请求失败: %s", string(body))
	}

	var result struct {
		Code   int             `json:"code"`
		Msg    string          `json:"msg"`
		Page   PageResult      `json:"page"`
		Result []PlatformOrder `json:"result"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, nil, fmt.Errorf("解析响应失败: %v", err)
	}

	if result.Code != 0 {
		return nil, nil, fmt.Errorf("业务错误: %s", result.Msg)
	}

	return result.Result, &result.Page, nil
}

// GetChannelList 查询所有渠道及对应运营商编码
func (s *Service) GetChannelList(appkey string, userid string, apiUrl string) ([]Channel, error) {
	//accoutname

	cacheKey := "xzx:channel_list"
	ctx := context.Background()
	var cached []Channel
	if val, err := redis.GetClient().Get(ctx, cacheKey).Result(); err == nil && val != "" {
		if err := json.Unmarshal([]byte(val), &cached); err == nil {
			return cached, nil
		}
	}

	params := map[string]string{}
	// apiKey := "c362d30409744d7584abcbd3b58124c2"
	// userID := "558203"
	authToken, queryTime, err := signature.GenerateXianzhuanxiaSignature(params, appkey, userid)
	if err != nil {
		return nil, fmt.Errorf("生成签名失败: %v", err)
	}

	url := fmt.Sprintf("%s/api/task/recharge/taskChannelList", apiUrl)
	fmt.Println(url, "************")
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		logger.Error("创建HTTP请求失败", "url", url, "error", err)
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Auth_Token", authToken)
	req.Header.Set("Query-Time", queryTime)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logger.Error("发送HTTP请求失败", "url", url, "error", err)
		return nil, fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	logger.Info("获取渠道列表", "url", url, "status", resp.StatusCode, "body", string(body))
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("请求失败: %s", string(body))
	}

	var result struct {
		Code   int       `json:"code"`
		Msg    string    `json:"msg"`
		Result []Channel `json:"result"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	if result.Code != 0 {
		return nil, fmt.Errorf("业务错误: %s", result.Msg)
	}

	// 写入 redis 缓存 1 小时
	if bytes, err := json.Marshal(result.Result); err == nil {
		_ = redis.GetClient().Set(ctx, cacheKey, bytes, time.Hour).Err()
	}

	return result.Result, nil
}

// GetStockInfo 查询库存信息
func (s *Service) GetStockInfo(channelID, productID int, provinces string) ([]StockInfo, error) {
	params := map[string]string{
		"channelId": strconv.Itoa(channelID),
		"productId": strconv.Itoa(productID),
		"provinces": provinces,
	}

	authToken, queryTime, err := signature.GenerateXianzhuanxiaSignature(params, s.apiKey, s.userID)
	if err != nil {
		return nil, fmt.Errorf("生成签名失败: %v", err)
	}

	url := fmt.Sprintf("%s/api/task/recharge/stock", s.baseURL)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		logger.Error("创建HTTP请求失败", "url", url, "error", err)
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Auth-Token", authToken)
	req.Header.Set("Query-Time", queryTime)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("请求失败: %s", string(body))
	}

	var result struct {
		Code   int         `json:"code"`
		Msg    string      `json:"msg"`
		Result []StockInfo `json:"result"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	if result.Code != 0 {
		return nil, fmt.Errorf("业务错误: %s", result.Msg)
	}

	return result.Result, nil
}

// GetToken 获取或申请 token，带重试机制
func (s *Service) GetToken(taskConfigID int64, channelID int, productID string, provinces string, faceValues, minSettleAmounts string, apiKey, userID, apiURL string) (string, error) {
	return s.GetTokenWithContext(context.Background(), taskConfigID, channelID, productID, provinces, faceValues, minSettleAmounts, apiKey, userID, apiURL)
}

// GetTokenWithContext 获取或申请 token，带重试机制和context支持
func (s *Service) GetTokenWithContext(ctx context.Context, taskConfigID int64, channelID int, productID string, provinces string, faceValues, minSettleAmounts string, apiKey, userID, apiURL string) (string, error) {
	// 尝试获取现有 token
	tokenData, err := s.tokenRepo.Get(taskConfigID)
	if err != nil {
		// 如果获取失败（记录不存在），申请新 token
		logger.Info(fmt.Sprintf("token 不存在，申请新 token: ChannelID=%d, ProductID=%s", channelID, productID))
		token, err := s.submitTaskWithRetryContext(ctx, channelID, productID, provinces, faceValues, minSettleAmounts, apiKey, userID, apiURL)
		if err != nil {
			return "", err
		}
		_ = s.tokenRepo.Save(taskConfigID, token)
		return token, nil
	}

	// 检查 token 是否过期（5分钟）
	if time.Since(tokenData.CreatedAt) >= 5*time.Minute {
		logger.Info(fmt.Sprintf("token 已过期，申请新 token: ChannelID=%d, ProductID=%s", channelID, productID))
		token, err := s.submitTaskWithRetryContext(ctx, channelID, productID, provinces, faceValues, minSettleAmounts, apiKey, userID, apiURL)
		if err != nil {
			return "", err
		}
		_ = s.tokenRepo.Save(taskConfigID, token)
		return token, nil
	}

	// token 有效，更新最后使用时间
	_ = s.tokenRepo.UpdateLastUsed(taskConfigID)
	logger.Info(fmt.Sprintf("使用现有 token: ChannelID=%d, ProductID=%s, token=%s", channelID, productID, tokenData.Token))
	return tokenData.Token, nil
}

// submitTaskWithRetry 带重试机制的申请token方法
func (s *Service) submitTaskWithRetry(channelID int, productID string, provinces string, faceValues, minSettleAmounts string, apiKey, userID, apiURL string) (string, error) {
	return s.submitTaskWithRetryContext(context.Background(), channelID, productID, provinces, faceValues, minSettleAmounts, apiKey, userID, apiURL)
}

// submitTaskWithRetryContext 带重试机制和context支持的申请token方法
func (s *Service) submitTaskWithRetryContext(ctx context.Context, channelID int, productID string, provinces string, faceValues, minSettleAmounts string, apiKey, userID, apiURL string) (string, error) {
	const (
		retryDelay = 1 * time.Minute // 固定1分钟重试间隔
	)

	for attempt := 0; ; attempt++ { // 无限重试
		// 检查context是否被取消
		select {
		case <-ctx.Done():
			logger.Info(fmt.Sprintf("Token申请被取消: ChannelID=%d, ProductID=%s, 原因=%v", channelID, productID, ctx.Err()))
			return "", ctx.Err()
		default:
		}

		token, err := s.SubmitTask(channelID, productID, provinces, faceValues, minSettleAmounts, apiKey, userID, apiURL)
		if err == nil {
			return token, nil
		}

		logger.Error(fmt.Sprintf("Token申请失败 (第%d次重试), 错误: %v, 60秒后重试", attempt+1, err))
		
		// 在等待期间也要检查context取消
		select {
		case <-ctx.Done():
			logger.Info(fmt.Sprintf("Token申请在等待期间被取消: ChannelID=%d, ProductID=%s, 原因=%v", channelID, productID, ctx.Err()))
			return "", ctx.Err()
		case <-time.After(retryDelay):
			// 继续下一次重试
		}
	}
}

// 匹配到订单后让 token 失效
func (s *Service) InvalidateToken(taskConfigID int64) error {
	return s.tokenRepo.Delete(taskConfigID)
}

// PushToThirdParty 推送订单到第三方平台
func (s *Service) PushToThirdParty(order *PlatformOrder, notifyUrl string) error {
	params := map[string]interface{}{
		"orderNumber":      order.OrderNumber,
		"channelName":      order.ChannelName,
		"productName":      order.ProductName,
		"accountNum":       order.AccountNum,
		"accountLocation":  order.AccountLocation,
		"settlementAmount": order.SettlementAmount,
		"orderStatus":      order.OrderStatus,
		"settlementStatus": order.SettlementStatus,
		"createTime":       order.CreateTime.UnixMilli(),
		"expirationTime":   order.ExpirationTime.UnixMilli(),
	}
	jsonData, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("参数序列化失败: %v", err)
	}

	req, err := http.NewRequest("POST", notifyUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

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

	// 可根据第三方返回内容做进一步处理
	return nil
}

// GetAPIKeyAndSecret 通过账号ID获取 appkey、appsecret、accountName
func (s *Service) GetAPIKeyAndSecret(accountID int64) (appKey string, platform *model.Platform, accountName string, err error) {
	account, err := s.platformRepo.GetPlatformAccountByID(accountID)
	if err != nil {
		return "", nil, "", err
	}
	return account.AppKey, account.Platform, account.AccountName, nil
}
