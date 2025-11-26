package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"recharge-go/configs"
	"recharge-go/internal/model"
	"recharge-go/pkg/log"

	"go.uber.org/zap"
)

// PhoneQueryService 手机查询服务接口
type PhoneQueryService interface {
	// QueryBalance 查询余额
	QueryBalance(ctx context.Context, phone, ispType string) (*model.PhoneBalanceResponse, error)
	// QueryPaymentRecords 查询缴费记录
	QueryPaymentRecords(ctx context.Context, phone, ispType string) (*model.PaymentRecordResponse, error)
	// QueryBalanceWithRetry 带重试的余额查询
	QueryBalanceWithRetry(ctx context.Context, phone, ispType string, maxRetries int) (*model.PhoneBalanceResponse, error)
	// QueryPaymentRecordsWithRetry 带重试的缴费记录查询
	QueryPaymentRecordsWithRetry(ctx context.Context, phone, ispType string, maxRetries int) (*model.PaymentRecordResponse, error)
}

type phoneQueryService struct {
	config *configs.ThirdPartyAPIConfig
	client *http.Client
}

// NewPhoneQueryService 创建手机查询服务
func NewPhoneQueryService(cfg *configs.Config) PhoneQueryService {
	return &phoneQueryService{
		config: &cfg.ThirdPartyAPI,
		client: &http.Client{
			Timeout: time.Duration(cfg.ThirdPartyAPI.Timeout) * time.Second,
		},
	}
}

// QueryBalance 查询余额
func (s *phoneQueryService) QueryBalance(ctx context.Context, phone, ispType string) (*model.PhoneBalanceResponse, error) {
	log.Log.Info("开始查询手机余额",
		zap.String("phone", phone),
		zap.String("isp_type", ispType),
	)

	params := map[string]string{
		"merch": s.config.MerchantID,
		"token": s.config.Token,
		"type":  ispType,
		"phone": phone,
	}

	start := time.Now()
	resp, err := s.sendFormRequest(ctx, "/apis/balance", params)
	duration := time.Since(start)

	if err != nil {
		log.Log.Error("查询手机余额失败",
			zap.String("phone", phone),
			zap.String("isp_type", ispType),
			zap.Error(err),
			zap.Duration("duration", duration),
		)
		return nil, err
	}

	var balanceResp model.PhoneBalanceResponse
	if err := json.Unmarshal(resp, &balanceResp); err != nil {
		log.Log.Error("解析余额查询响应失败",
			zap.String("phone", phone),
			zap.String("response", string(resp)),
			zap.Error(err),
		)
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	log.Log.Info("查询手机余额成功",
		zap.String("phone", phone),
		zap.String("isp_type", ispType),
		zap.Int("errcode", balanceResp.ErrCode),
		zap.String("balance", balanceResp.Data),
		zap.Duration("duration", duration),
	)

	return &balanceResp, nil
}

// QueryPaymentRecords 查询缴费记录
func (s *phoneQueryService) QueryPaymentRecords(ctx context.Context, phone, ispType string) (*model.PaymentRecordResponse, error) {
	log.Log.Info("开始查询缴费记录",
		zap.String("phone", phone),
		zap.String("isp_type", ispType),
	)

	// 验证运营商类型（仅支持移动和联通）
	if ispType != "yd" && ispType != "lt" && ispType != "dx" {
		return nil, fmt.Errorf("不支持的运营商类型: %s，仅支持 yd(移动)、lt(联通) 和 dx(电信)", ispType)
	}

	params := map[string]string{
		"merch": s.config.MerchantID,
		"token": s.config.Token,
		"type":  ispType,
		"phone": phone,
	}

	start := time.Now()
	resp, err := s.sendFormRequest(ctx, "/apis/phone/payRecord", params)
	duration := time.Since(start)

	if err != nil {
		log.Log.Error("查询缴费记录失败",
			zap.String("phone", phone),
			zap.String("isp_type", ispType),
			zap.Error(err),
			zap.Duration("duration", duration),
		)
		return nil, err
	}

	var recordResp model.PaymentRecordResponse
	if err := json.Unmarshal(resp, &recordResp); err != nil {
		log.Log.Error("解析缴费记录响应失败",
			zap.String("phone", phone),
			zap.String("response", string(resp)),
			zap.Error(err),
		)
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	records := recordResp.GetRecords()
	log.Log.Info("查询缴费记录成功",
		zap.String("phone", phone),
		zap.String("isp_type", ispType),
		zap.Int("errcode", recordResp.ErrCode),
		zap.Int("record_count", len(records)),
		zap.Duration("duration", duration),
	)

	return &recordResp, nil
}

// QueryBalanceWithRetry 带重试的余额查询
func (s *phoneQueryService) QueryBalanceWithRetry(ctx context.Context, phone, ispType string, maxRetries int) (*model.PhoneBalanceResponse, error) {
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		resp, err := s.QueryBalance(ctx, phone, ispType)
		if err == nil && resp.ErrCode == 0 {
			return resp, nil
		}

		lastErr = err
		if err == nil {
			lastErr = fmt.Errorf("API返回错误: errcode=%d, errmsg=%s", resp.ErrCode, resp.ErrMsg)
		}

		if i < maxRetries-1 {
			retryDelay := time.Duration(i+1) * time.Second
			log.Log.Warn("余额查询失败，准备重试",
				zap.String("phone", phone),
				zap.Int("retry_count", i+1),
				zap.Int("max_retries", maxRetries),
				zap.Duration("retry_delay", retryDelay),
				zap.Error(lastErr),
			)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(retryDelay):
			}
		}
	}

	return nil, fmt.Errorf("重试%d次后仍失败: %v", maxRetries, lastErr)
}

// QueryPaymentRecordsWithRetry 带重试的缴费记录查询
func (s *phoneQueryService) QueryPaymentRecordsWithRetry(ctx context.Context, phone, ispType string, maxRetries int) (*model.PaymentRecordResponse, error) {
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		resp, err := s.QueryPaymentRecords(ctx, phone, ispType)
		if err == nil && resp.ErrCode == 0 {
			return resp, nil
		}

		lastErr = err
		if err == nil {
			lastErr = fmt.Errorf("API返回错误: errcode=%d, errmsg=%s", resp.ErrCode, resp.ErrMsg)
		}

		if i < maxRetries-1 {
			retryDelay := time.Duration(i+1) * time.Second
			log.Log.Warn("缴费记录查询失败，准备重试",
				zap.String("phone", phone),
				zap.Int("retry_count", i+1),
				zap.Int("max_retries", maxRetries),
				zap.Duration("retry_delay", retryDelay),
				zap.Error(lastErr),
			)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(retryDelay):
			}
		}
	}

	return nil, fmt.Errorf("重试%d次后仍失败: %v", maxRetries, lastErr)
}

// sendFormRequest 发送form-data请求
func (s *phoneQueryService) sendFormRequest(ctx context.Context, endpoint string, params map[string]string) ([]byte, error) {
	// 构建form-data请求体
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	for key, value := range params {
		if err := writer.WriteField(key, value); err != nil {
			return nil, fmt.Errorf("写入表单字段失败: %v", err)
		}
	}

	contentType := writer.FormDataContentType()
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("关闭表单写入器失败: %v", err)
	}

	// 构建完整URL
	url := s.config.BaseURL + endpoint

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "POST", url, &buf)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("User-Agent", "RechargeGo-PhoneQuery/1.0")

	log.Log.Info("发送第三方API请求",
		zap.String("url", url),
		zap.String("method", "POST"),
		zap.String("content_type", contentType),
		zap.Any("params", params),
	)

	// 发送请求
	start := time.Now()
	resp, err := s.client.Do(req)
	duration := time.Since(start)

	if err != nil {
		log.Log.Error("第三方API请求失败",
			zap.String("url", url),
			zap.Error(err),
			zap.Duration("duration", duration),
		)
		return nil, fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Log.Error("读取第三方API响应失败",
			zap.String("url", url),
			zap.Int("status_code", resp.StatusCode),
			zap.Error(err),
		)
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	log.Log.Info("第三方API请求完成",
		zap.String("url", url),
		zap.Int("status_code", resp.StatusCode),
		zap.Int("response_size", len(body)),
		zap.Duration("duration", duration),
	)

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		log.Log.Error("第三方API返回错误状态码",
			zap.String("url", url),
			zap.Int("status_code", resp.StatusCode),
			zap.String("response_body", string(body)),
		)
		return nil, fmt.Errorf("HTTP请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	return body, nil
}
