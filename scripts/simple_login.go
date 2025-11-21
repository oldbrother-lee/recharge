package main

import (
	"crypto/rc4"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// RequestData 定义请求数据结构
type RequestData struct {
	Action string `json:"action"`
	Flag   string `json:"flag"`
	Ver    string `json:"ver"`
	Token  string `json:"token"`
	Data   struct {
		Username string `json:"username"`
		Password string `json:"password"`
	} `json:"data"`
}

// OrderRequestData 定义获取订单请求数据结构
type OrderRequestData struct {
	Action string `json:"action"`
	Flag   string `json:"flag"`
	Ver    string `json:"ver"`
	Token  string `json:"token"`
	Data   struct {
		Amount    string `json:"amount"`
		MaxAmount string `json:"max_amount"`
		Operator  int    `json:"operator"`
		Discount  int    `json:"discount"`
		Prov      string `json:"prov"`
	} `json:"data"`
}

// LoginResponse 定义登录响应结构
type LoginResponse struct {
	Ret  int    `json:"ret"`
	Msg  string `json:"msg"`
	Data string `json:"data"`
}

// OrderResponse 定义订单响应结构
type OrderResponse struct {
	Ret  int    `json:"ret"`
	Msg  string `json:"msg"`
	Data struct {
		Amount     int    `json:"amount"`
		OperatorID int    `json:"operator_id"`
		Mobile     string `json:"mobile"`
		Discount   string `json:"discount"`
		ID         int64  `json:"id"`
		Prov       string `json:"prov"`
		Timeout    int64  `json:"timeout"`
	} `json:"data"`
}

// RC4Encrypt RC4加密函数
func RC4Encrypt(data string, key string) (string, error) {
	cipher, err := rc4.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}

	plaintext := []byte(data)
	ciphertext := make([]byte, len(plaintext))
	cipher.XORKeyStream(ciphertext, plaintext)

	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// RC4Decrypt RC4解密函数
func RC4Decrypt(encryptedData string, key string) (string, error) {
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

// SimpleLogin 执行登录操作，返回token
func SimpleLogin() (string, error) {
	baseURL := "http://api.hdb666.com"
	rc4Key := "IT8MYBDY"
	username := "1863563749302"
	password := "JpE9F7heOl1fP%S0"

	// 构造请求数据
	requestData := RequestData{
		Action: "login",
		Flag:   "invite_dxfs",
		Ver:    "1.0.0.0",
		Token:  "",
	}
	requestData.Data.Username = username
	requestData.Data.Password = password

	// 将请求数据转换为JSON
	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return "", fmt.Errorf("JSON序列化失败: %v", err)
	}

	// RC4加密
	encryptedData, err := RC4Encrypt(string(jsonData), rc4Key)
	if err != nil {
		return "", fmt.Errorf("RC4加密失败: %v", err)
	}

	// 发送HTTP请求
	url := fmt.Sprintf("%s/api/phrecharge?devKey=10028", baseURL)
	resp, err := http.Post(url, "application/x-www-form-urlencoded", strings.NewReader(encryptedData))
	if err != nil {
		return "", fmt.Errorf("HTTP请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %v", err)
	}

	// 检查响应是否为空
	if len(body) == 0 {
		return "", fmt.Errorf("服务器返回空响应")
	}

	// 尝试解析JSON响应（服务器可能返回JSON格式的字符串）
	var responseData string
	err = json.Unmarshal(body, &responseData)
	if err != nil {
		// 如果不是JSON格式，直接使用原始数据
		responseData = string(body)
	}

	// RC4解密响应
	decryptedData, err := RC4Decrypt(responseData, rc4Key)
	if err != nil {
		return "", fmt.Errorf("RC4解密失败: %v", err)
	}

	// 解析登录响应
	var loginResp LoginResponse
	err = json.Unmarshal([]byte(decryptedData), &loginResp)
	if err != nil {
		return "", fmt.Errorf("解析登录响应失败: %v", err)
	}

	// 检查登录是否成功
	if loginResp.Ret != 0 {
		return "", fmt.Errorf("登录失败: %s", loginResp.Msg)
	}

	return loginResp.Data, nil
}

// GetOrder 获取订单
func GetOrder(token string, amount string, maxAmount string, operator int, discount int, prov string) (*OrderResponse, error) {
	baseURL := "http://localhost:8081" // 使用mock服务器
	rc4Key := "IT8MYBDY"

	// 构造请求数据
	requestData := OrderRequestData{
		Action: "get",
		Flag:   "invite_dxfs",
		Ver:    "1.0.0.0",
		Token:  token,
	}
	requestData.Data.Amount = amount
	requestData.Data.MaxAmount = maxAmount
	requestData.Data.Operator = operator
	requestData.Data.Discount = discount
	requestData.Data.Prov = prov

	// 将请求数据转换为JSON
	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return nil, fmt.Errorf("JSON序列化失败: %v", err)
	}
	fmt.Println("jsonData: ", string(jsonData))
	// RC4加密
	encryptedData, err := RC4Encrypt(string(jsonData), rc4Key)
	if err != nil {
		return nil, fmt.Errorf("RC4加密失败: %v", err)
	}

	// 发送HTTP请求
	url := fmt.Sprintf("%s/api/phrecharge?devKey=10028", baseURL)
	resp, err := http.Post(url, "application/x-www-form-urlencoded", strings.NewReader(encryptedData))
	if err != nil {
		return nil, fmt.Errorf("HTTP请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	// 检查响应是否为空
	if len(body) == 0 {
		return nil, fmt.Errorf("服务器返回空响应")
	}

	// 尝试解析JSON响应（服务器可能返回JSON格式的字符串）
	var responseData string
	err = json.Unmarshal(body, &responseData)
	if err != nil {
		// 如果不是JSON格式，直接使用原始数据
		responseData = string(body)
	}

	// RC4解密响应
	decryptedData, err := RC4Decrypt(responseData, rc4Key)
	if err != nil {
		return nil, fmt.Errorf("RC4解密失败: %v", err)
	}

	// 解析订单响应
	var orderResp OrderResponse
	err = json.Unmarshal([]byte(decryptedData), &orderResp)
	if err != nil {
		return nil, fmt.Errorf("解析订单响应失败: %v", err)
	}
	fmt.Println("orderResp: ", orderResp)
	return &orderResp, nil
}

// StatusResponse 定义状态上报响应结构
type StatusResponse struct {
	Ret  int    `json:"ret"`
	Msg  string `json:"msg"`
	Data struct {
		OrderID   int64  `json:"order_id"`
		Status    int    `json:"status"`
		Message   string `json:"message"`
		Timestamp int64  `json:"timestamp"`
	} `json:"data"`
}

// StatusRequestData 定义状态上报请求数据结构
type StatusRequestData struct {
	Action string `json:"action"`
	Flag   string `json:"flag"`
	Ver    string `json:"ver"`
	Token  string `json:"token"`
	Data   struct {
		OrderID   int64  `json:"order_id"`
		Status    int    `json:"status"` // 1=预上报成功, 2=支付成功, 3=支付失败
		Message   string `json:"message"`
		Timestamp int64  `json:"timestamp"`
	} `json:"data"`
}

// ReportPreOrder 预上报订单状态
func ReportPreOrder(token string, orderID int64, status int, message string) (*StatusResponse, error) {
	baseURL := "http://localhost:8081" // 使用mock服务器
	rc4Key := "IT8MYBDY"

	// 构造请求数据
	requestData := StatusRequestData{
		Action: "status",
		Flag:   "invite_dxfs",
		Ver:    "1.0.0.0",
		Token:  token,
	}
	requestData.Data.OrderID = orderID
	requestData.Data.Status = status
	requestData.Data.Message = message
	requestData.Data.Timestamp = time.Now().Unix()

	// 将请求数据转换为JSON
	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return nil, fmt.Errorf("JSON序列化失败: %v", err)
	}

	// RC4加密
	encryptedData, err := RC4Encrypt(string(jsonData), rc4Key)
	if err != nil {
		return nil, fmt.Errorf("RC4加密失败: %v", err)
	}

	// 发送HTTP请求
	url := fmt.Sprintf("%s/api/phrecharge?devKey=10028", baseURL)
	resp, err := http.Post(url, "application/x-www-form-urlencoded", strings.NewReader(encryptedData))
	if err != nil {
		return nil, fmt.Errorf("HTTP请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	// 尝试解析JSON响应
	var responseData string
	err = json.Unmarshal(body, &responseData)
	if err != nil {
		responseData = string(body)
	}

	// RC4解密响应
	decryptedData, err := RC4Decrypt(responseData, rc4Key)
	if err != nil {
		return nil, fmt.Errorf("RC4解密失败: %v", err)
	}

	// 解析响应
	var statusResp StatusResponse
	err = json.Unmarshal([]byte(decryptedData), &statusResp)
	if err != nil {
		return nil, fmt.Errorf("解析状态上报响应失败: %v", err)
	}

	return &statusResp, nil
}

func main() {
	// 1. 登录获取token
	fmt.Println("=== 开始登录 ===")
	token, err := SimpleLogin()
	if err != nil {
		fmt.Printf("登录失败: %v\n", err)
		return
	}
	fmt.Printf("登录成功，获取到token: %s\n", token)

	// 2. 无限循环获取订单，每秒一次
	// token = "08921744c33435876376e38a14926440"
	fmt.Println("\n=== 无限循环获取订单（每秒一次）===")
	fmt.Println("按 Ctrl+C 停止程序")

	counter := 1
	for {
		fmt.Printf("\n--- 第 %d 次获取订单 ---\n", counter)
		fmt.Printf("时间: %s\n", time.Now().Format("2006-01-02 15:04:05"))

		orderResp, err := GetOrder(token, "10", "500", 2, 0, "")
		if err != nil {
			fmt.Printf("获取订单失败: %v\n", err)
		} else {
			// 显示订单信息
			if orderResp.Ret == 0 {
				fmt.Printf("获取订单成功!\n")
				fmt.Printf("订单ID: %d\n", orderResp.Data.ID)
				fmt.Printf("充值面值: %d元\n", orderResp.Data.Amount)
				fmt.Printf("充值号码: %s\n", orderResp.Data.Mobile)
				fmt.Printf("运营商: %d\n", orderResp.Data.OperatorID)
				fmt.Printf("折扣: %s\n", orderResp.Data.Discount)
				fmt.Printf("省份: %s\n", orderResp.Data.Prov)
				fmt.Printf("超时时间: %d\n", orderResp.Data.Timeout)

				// 预上报订单状态
				fmt.Println("\n[预上报] 开始上报订单状态...")
				statusResp, err := ReportPreOrder(token, orderResp.Data.ID, 1, "预上报成功")
				if err != nil {
					fmt.Printf("[预上报] 失败: %v\n", err)
				} else {
					fmt.Printf("[预上报] 成功!\n")
					fmt.Printf("[预上报] 订单ID: %d\n", statusResp.Data.OrderID)
					fmt.Printf("[预上报] 状态: %d\n", statusResp.Data.Status)
					fmt.Printf("[预上报] 消息: %s\n", statusResp.Data.Message)
				}
			} else {
				fmt.Printf("获取订单失败: %s\n", orderResp.Msg)
			}
		}

		counter++

		// 等待1秒
		time.Sleep(1 * time.Second)
	}
}
