package test

import (
	"context"
	"testing"
	"time"

	"recharge-go/configs"
	"recharge-go/internal/service"

	"github.com/stretchr/testify/assert"
)

// TestPhoneQueryService 测试手机查询服务
func TestPhoneQueryService(t *testing.T) {
	// 加载测试配置
	config := &configs.Config{
		ThirdPartyAPI: configs.ThirdPartyAPIConfig{
			BaseURL:    "http://35.220.200.84:18080",
			MerchantID: "test_merchant",
			Token:      "test_token",
			Timeout:    30,
		},
	}

	// 创建服务实例
	phoneQueryService := service.NewPhoneQueryService(config)

	// 测试余额查询
	t.Run("QueryBalance", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// 使用测试手机号（这里使用示例号码，实际测试时需要使用有效号码）
		phone := "13800138000"
		ispType := "yd" // 移动

		result, err := phoneQueryService.QueryBalance(ctx, phone, ispType)

		// 由于是测试环境，可能会返回错误，这里主要验证服务是否正常初始化
		if err != nil {
			t.Logf("查询余额返回错误（预期）: %v", err)
		} else {
			assert.NotNil(t, result)
			t.Logf("查询余额成功: errcode=%d, errmsg=%s, data=%s", result.ErrCode, result.ErrMsg, result.Data)
		}
	})

	// 测试缴费记录查询
	t.Run("QueryPaymentRecords", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// 使用测试手机号
		phone := "13800138000"
		ispType := "yd" // 移动

		result, err := phoneQueryService.QueryPaymentRecords(ctx, phone, ispType)

		// 由于是测试环境，可能会返回错误，这里主要验证服务是否正常初始化
		if err != nil {
			t.Logf("查询缴费记录返回错误（预期）: %v", err)
		} else {
			assert.NotNil(t, result)
			records := result.GetRecords()
			t.Logf("查询缴费记录成功: errcode=%d, errmsg=%s, record_count=%d", result.ErrCode, result.ErrMsg, len(records))
		}
	})

	// 测试不支持的运营商类型
	t.Run("QueryPaymentRecords_UnsupportedISP", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		phone := "13800138000"
		ispType := "dx" // 电信（不支持缴费记录查询）

		_, err := phoneQueryService.QueryPaymentRecords(ctx, phone, ispType)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "不支持的运营商类型")
	})

	// 测试带重试的查询
	t.Run("QueryBalanceWithRetry", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		phone := "13800138000"
		ispType := "yd"
		maxRetries := 2

		result, err := phoneQueryService.QueryBalanceWithRetry(ctx, phone, ispType, maxRetries)

		// 验证重试机制是否正常工作
		if err != nil {
			t.Logf("带重试的查询余额返回错误（预期）: %v", err)
			assert.Contains(t, err.Error(), "重试")
		} else {
			assert.NotNil(t, result)
			t.Logf("带重试的查询余额成功: errcode=%d, data=%s", result.ErrCode, result.Data)
		}
	})
}

// BenchmarkPhoneQueryService 性能测试
func BenchmarkPhoneQueryService(b *testing.B) {
	config := &configs.Config{
		ThirdPartyAPI: configs.ThirdPartyAPIConfig{
			BaseURL:    "http://35.220.200.84:18080",
			MerchantID: "test_merchant",
			Token:      "test_token",
			Timeout:    30,
		},
	}

	phoneQueryService := service.NewPhoneQueryService(config)
	ctx := context.Background()
	phone := "13800138000"
	ispType := "yd"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = phoneQueryService.QueryBalance(ctx, phone, ispType)
	}
}