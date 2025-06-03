package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"recharge-go/internal/model/notification"
	notificationModel "recharge-go/internal/model/notification"
	"recharge-go/pkg/logger"
	"time"
)

// Sender 通知发送器接口
type Sender interface {
	// Send 发送通知
	Send(ctx context.Context, record *notification.NotificationRecord, template *notificationModel.Template) error
}

// HTTPSender HTTP通知发送器
type HTTPSender struct {
	client *http.Client
}

// NewHTTPSender 创建HTTP通知发送器
func NewHTTPSender() *HTTPSender {
	return &HTTPSender{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Send 发送HTTP通知
func (s *HTTPSender) Send(ctx context.Context, record *notification.NotificationRecord, template *notificationModel.Template) error {
	// 解析通知内容
	var content map[string]interface{}
	if err := json.Unmarshal([]byte(record.Content), &content); err != nil {
		return fmt.Errorf("unmarshal content failed: %v", err)
	}

	// 解析模板
	var templateData map[string]interface{}
	if err := json.Unmarshal([]byte(template.Template), &templateData); err != nil {
		return fmt.Errorf("unmarshal template failed: %v", err)
	}

	// 获取请求URL和方法
	url, ok := templateData["url"].(string)
	if !ok {
		return fmt.Errorf("invalid template: missing url")
	}
	method, ok := templateData["method"].(string)
	if !ok {
		method = "POST" // 默认使用POST方法
	}

	// 构建请求体
	requestBody := make(map[string]interface{})

	// 合并模板参数和通知内容
	for k, v := range templateData {
		if k != "url" && k != "method" {
			requestBody[k] = v
		}
	}
	for k, v := range content {
		requestBody[k] = v
	}

	// 序列化请求体
	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("marshal request body failed: %v", err)
	}

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return fmt.Errorf("create request failed: %v", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	if headers, ok := templateData["headers"].(map[string]interface{}); ok {
		for k, v := range headers {
			if str, ok := v.(string); ok {
				req.Header.Set(k, str)
			}
		}
	}

	// 发送请求
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("send request failed: %v", err)
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("request failed with status code: %d", resp.StatusCode)
	}

	// 记录成功日志
	logger.Info("notification sent successfully",
		"record_id", record.ID,
		"platform", record.PlatformCode,
		"type", record.NotificationType,
		"status_code", resp.StatusCode,
	)

	return nil
}

// WebhookSender Webhook通知发送器
type WebhookSender struct {
	client *http.Client
}

// NewWebhookSender 创建Webhook通知发送器
func NewWebhookSender() *WebhookSender {
	return &WebhookSender{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Send 发送Webhook通知
func (s *WebhookSender) Send(ctx context.Context, record *notification.NotificationRecord, template *notificationModel.Template) error {
	// 解析模板获取webhook URL
	var templateData map[string]interface{}
	if err := json.Unmarshal([]byte(template.Template), &templateData); err != nil {
		return fmt.Errorf("unmarshal template failed: %v", err)
	}

	webhookURL, ok := templateData["webhook_url"].(string)
	if !ok {
		return fmt.Errorf("invalid template: missing webhook_url")
	}

	// 构建webhook请求体
	webhookBody := map[string]interface{}{
		"order_id":          record.OrderID,
		"platform_code":     record.PlatformCode,
		"notification_type": record.NotificationType,
		"content":           record.Content,
		"timestamp":         time.Now().Unix(),
	}

	// 序列化请求体
	bodyBytes, err := json.Marshal(webhookBody)
	if err != nil {
		return fmt.Errorf("marshal webhook body failed: %v", err)
	}

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return fmt.Errorf("create webhook request failed: %v", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	if secret, ok := templateData["secret"].(string); ok {
		// TODO: 实现签名逻辑
		req.Header.Set("X-Webhook-Signature", secret)
	}

	// 发送请求
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("send webhook request failed: %v", err)
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook request failed with status code: %d", resp.StatusCode)
	}

	// 记录成功日志
	logger.Info("webhook notification sent successfully",
		"record_id", record.ID,
		"platform", record.PlatformCode,
		"type", record.NotificationType,
		"status_code", resp.StatusCode,
	)

	return nil
}
