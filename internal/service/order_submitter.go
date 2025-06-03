package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"recharge-go/internal/model"
	"recharge-go/internal/signature"
)

// OrderSubmitter 订单提交接口
type OrderSubmitter interface {
	// SubmitOrder 提交订单
	SubmitOrder(ctx context.Context, order *model.Order, api *model.PlatformAPI, apiParam *model.PlatformAPIParam) error
}

// BaseOrderSubmitter 基础订单提交器
type BaseOrderSubmitter struct {
	signatureHandler signature.SignatureHandler
	httpClient       *http.Client
}

// NewBaseOrderSubmitter 创建基础订单提交器
func NewBaseOrderSubmitter(handler signature.SignatureHandler) *BaseOrderSubmitter {
	return &BaseOrderSubmitter{
		signatureHandler: handler,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SubmitOrder 基础订单提交实现
func (s *BaseOrderSubmitter) SubmitOrder(ctx context.Context, order *model.Order, api *model.PlatformAPI, apiParam *model.PlatformAPIParam) error {
	// 1. 构建请求参数
	params, err := s.signatureHandler.BuildRequestParams(ctx, order, api)
	if err != nil {
		return fmt.Errorf("构建请求参数失败: %v", err)
	}

	// 2. 发送请求
	fmt.Println("发送请求", api.URL, params)
	resp, err := s.sendRequest(ctx, api.URL, params)
	if err != nil {
		return fmt.Errorf("发送请求失败: %v", err)
	}

	// 3. 处理响应
	if err := s.handleResponse(resp); err != nil {
		return fmt.Errorf("处理响应失败: %v", err)
	}

	return nil
}

// sendRequest 发送HTTP请求
func (s *BaseOrderSubmitter) sendRequest(ctx context.Context, url string, params map[string]interface{}) (map[string]interface{}, error) {
	// 将参数转换为JSON
	jsonData, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("参数序列化失败: %v", err)
	}

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 解析响应
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	return result, nil
}

// handleResponse 处理响应
func (s *BaseOrderSubmitter) handleResponse(resp map[string]interface{}) error {
	// 基础实现，具体平台需要重写
	return nil
}
