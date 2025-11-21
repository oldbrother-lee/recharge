package main

import (
	"bytes"
	"crypto/rc4"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// 章鱼平台API对接

// ZYRequest 章鱼平台请求结构
type ZYRequest struct {
	DevID string `json:"dev_id"`
	Body  string `json:"body"` // RC4加密后的数据
}

// ZYResponse 章鱼平台响应结构
type ZYResponse struct {
	Ret  int    `json:"ret"`  // 0：成功，其他：异常
	Msg  string `json:"msg"`  // 返回结果描述
	Data string `json:"data"` // 加密后的数据，需要RC4解密
}

// ZYLoginRequest 登录请求数据
type ZYLoginRequest struct {
	Data struct {
		Username string `json:"username"`
		Password string `json:"password"`
	} `json:"data"`
}

// ZYLoginResponse 登录响应数据
type ZYLoginResponse struct {
	Data string `json:"data"` // JWT token
	Msg  string `json:"msg"`
	Ret  int    `json:"ret"`
}

// ZYGetOrderRequest 获取订单请求数据
type ZYGetOrderRequest struct {
	Token string `json:"token"`
	Flag  string `json:"flag"`
	Data  struct {
		Amount    string `json:"amount"`     // 最小面额
		MaxAmount string `json:"max_amount"` // 最大面额
		Prov      string `json:"prov"`       // 省份
	} `json:"data"`
}

// ZYGetOrderResponse 获取订单响应数据
type ZYGetOrderResponse struct {
	Data struct {
		ID         int    `json:"id"`          // 订单id
		Amount     int    `json:"amount"`      // 面额（元）
		Mobile     string `json:"mobile"`      // 被充值号码
		Operator   string `json:"operator"`    // 运营商
		OperatorID int    `json:"operator_id"` // 运营商id（1：移动，2：联通，3：电信）
		Prov       string `json:"prov"`        // 省份
		Timeout    int64  `json:"timeout"`     // 超时时间
	} `json:"data"`
	Msg string `json:"msg"`
	Ret int    `json:"ret"`
}

// ZYReportRequest 上报订单请求数据
type ZYReportRequest struct {
	Token string `json:"token"`
	Flag  string `json:"flag"`
	Data  struct {
		ID              string `json:"id"`              // 订单id
		Result          string `json:"result"`          // 支付结果，1：成功，2：失败
		Reason          string `json:"reason"`          // 支付结果信息
		PtransID        string `json:"ptransId"`        // 渠道订单号（特定渠道必填）
		Cookie          string `json:"cookie"`          // 渠道ck（特定渠道必填）
		OrderCreateTime string `json:"orderCreateTime"` // 订单创建时间（特定渠道必填）
	} `json:"data"`
}

// ZYReportResponse 上报订单响应数据
type ZYReportResponse struct {
	Msg string `json:"msg"`
	Ret int    `json:"ret"`
}

// ZYRC4Encrypt 章鱼平台RC4加密函数
func ZYRC4Encrypt(data string, key string) (string, error) {
	cipher, err := rc4.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}

	plaintext := []byte(data)
	ciphertext := make([]byte, len(plaintext))
	cipher.XORKeyStream(ciphertext, plaintext)

	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// ZYRC4Decrypt 章鱼平台RC4解密函数
func ZYRC4Decrypt(encryptedData string, key string) (string, error) {
	// Base64解码
	data, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return "", err
	}

	// RC4解密
	cipher, err := rc4.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}

	plaintext := make([]byte, len(data))
	cipher.XORKeyStream(plaintext, data)

	return string(plaintext), nil
}

// ZYLogin 章鱼平台登录
func ZYLogin(devID string, rc4Key string, username string, password string) (string, error) {
	baseURL := "https://api.zy128.cn"

	// 构造登录请求数据
	loginData := ZYLoginRequest{}
	loginData.Data.Username = username
	loginData.Data.Password = password

	// 将请求数据转换为JSON
	jsonData, err := json.Marshal(loginData)
	if err != nil {
		return "", fmt.Errorf("JSON序列化失败: %v", err)
	}

	// RC4加密
	encryptedBody, err := ZYRC4Encrypt(string(jsonData), rc4Key)
	if err != nil {
		return "", fmt.Errorf("RC4加密失败: %v", err)
	}

	// 构造最终请求
	request := ZYRequest{
		DevID: devID,
		Body:  encryptedBody,
	}

	requestJSON, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("请求序列化失败: %v", err)
	}
	fmt.Println("requestJSON: ", string(requestJSON))
	// 发送HTTP请求
	url := fmt.Sprintf("%s/api/openapi/login", baseURL)
	resp, err := http.Post(url, "application/json; charset=utf-8", bytes.NewBuffer(requestJSON))
	if err != nil {
		return "", fmt.Errorf("HTTP请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %v", err)
	}

	// 调试：打印原始响应
	fmt.Printf("[ZYLogin] HTTP %d Raw body: %s\n", resp.StatusCode, string(body))

	// 按需求：先整体解密，再解析
	decryptedBody, decErr := ZYRC4Decrypt(string(body), "QYkCe9Sv")
	if decErr != nil {
		return "", fmt.Errorf("整体解密失败: %v", decErr)
	}
	fmt.Printf("[ZYLogin] Decrypted whole body: %s\n", decryptedBody)

	// 直接按最终登录响应解析
	var loginResp ZYLoginResponse
	if err := json.Unmarshal([]byte(decryptedBody), &loginResp); err != nil {
		return "", fmt.Errorf("解密后解析响应失败: %v", err)
	}
	if loginResp.Ret != 0 {
		return "", fmt.Errorf("登录失败: %s", loginResp.Msg)
	}
	return loginResp.Data, nil // 返回JWT token
}

// ZYGetOrder 章鱼平台获取订单
func ZYGetOrder(devID string, rc4Key string, token string, flag string, amount string, maxAmount string, prov string) (*ZYGetOrderResponse, error) {
	baseURL := "https://api.zy128.cn"

	// 构造获取订单请求数据
	orderData := ZYGetOrderRequest{
		Token: token,
		Flag:  flag,
	}
	orderData.Data.Amount = amount
	orderData.Data.MaxAmount = maxAmount
	orderData.Data.Prov = ""

	// 将请求数据转换为JSON
	jsonData, err := json.Marshal(orderData)
	if err != nil {
		return nil, fmt.Errorf("JSON序列化失败: %v", err)
	}
	fmt.Println("jsonData: ", string(jsonData))
	// RC4加密
	encryptedBody, err := ZYRC4Encrypt(string(jsonData), rc4Key)
	if err != nil {
		return nil, fmt.Errorf("RC4加密失败: %v", err)
	}

	// 构造最终请求
	request := ZYRequest{
		DevID: devID,
		Body:  encryptedBody,
	}

	requestJSON, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("请求序列化失败: %v", err)
	}

	// 发送HTTP请求
	url := fmt.Sprintf("%s/api/openapi/getOrder", baseURL)
	resp, err := http.Post(url, "application/json; charset=utf-8", bytes.NewBuffer(requestJSON))
	if err != nil {
		return nil, fmt.Errorf("HTTP请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	// 调试：打印原始响应
	fmt.Printf("[ZYGetOrder] HTTP %d Raw body: %s\n", resp.StatusCode, string(body))

	// 先整体解密，再解析（与登录一致）
	decryptedBody, decErr := ZYRC4Decrypt(string(body), "QYkCe9Sv")
	if decErr != nil {
		return nil, fmt.Errorf("整体解密失败: %v", decErr)
	}
	fmt.Printf("[ZYGetOrder] Decrypted whole body: %s\n", decryptedBody)

	// 直接解析整个响应为订单响应结构（解密后的结构就是完整的订单响应）
	var orderResp ZYGetOrderResponse
	if err := json.Unmarshal([]byte(decryptedBody), &orderResp); err != nil {
		return nil, fmt.Errorf("解析订单响应失败: %v", err)
	}

	// 检查返回码
	if orderResp.Ret != 0 {
		return nil, fmt.Errorf("获取订单失败: %s", orderResp.Msg)
	}

	return &orderResp, nil
}

// ZYReportOrder 章鱼平台上报订单
func ZYReportOrder(devID string, rc4Key string, token string, flag string, orderID string, result string, reason string, ptransID string, cookie string, orderCreateTime string) (*ZYReportResponse, error) {
	baseURL := "https://api.zy128.cn"

	// 构造上报订单请求数据
	reportData := ZYReportRequest{
		Token: token,
		Flag:  flag,
	}
	reportData.Data.ID = orderID
	reportData.Data.Result = result
	reportData.Data.Reason = reason
	reportData.Data.PtransID = ptransID
	reportData.Data.Cookie = cookie
	reportData.Data.OrderCreateTime = orderCreateTime

	// 将请求数据转换为JSON
	jsonData, err := json.Marshal(reportData)
	if err != nil {
		return nil, fmt.Errorf("JSON序列化失败: %v", err)
	}

	// RC4加密
	encryptedBody, err := ZYRC4Encrypt(string(jsonData), rc4Key)
	if err != nil {
		return nil, fmt.Errorf("RC4加密失败: %v", err)
	}

	// 构造最终请求
	request := ZYRequest{
		DevID: devID,
		Body:  encryptedBody,
	}

	requestJSON, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("请求序列化失败: %v", err)
	}

	// 发送HTTP请求
	url := fmt.Sprintf("%s/api/openapi/report", baseURL)
	resp, err := http.Post(url, "application/json; charset=utf-8", bytes.NewBuffer(requestJSON))
	if err != nil {
		return nil, fmt.Errorf("HTTP请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	// 调试：打印原始响应
	fmt.Printf("[ZYReportOrder] HTTP %d Raw body: %s\n", resp.StatusCode, string(body))

	// 先整体解密，再解析（与获取订单一致）
	decryptedBody, decErr := ZYRC4Decrypt(string(body), "QYkCe9Sv")
	if decErr != nil {
		return nil, fmt.Errorf("整体解密失败: %v", decErr)
	}
	fmt.Printf("[ZYReportOrder] Decrypted whole body: %s\n", decryptedBody)

	// 直接解析整个响应为上报响应结构（解密后的结构就是完整的响应）
	var reportResp ZYReportResponse
	if err := json.Unmarshal([]byte(decryptedBody), &reportResp); err != nil {
		return nil, fmt.Errorf("解析上报响应失败: %v", err)
	}

	// 检查返回码
	if reportResp.Ret != 0 {
		return nil, fmt.Errorf("上报订单失败: %s", reportResp.Msg)
	}

	return &reportResp, nil
}
