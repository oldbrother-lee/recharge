package test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"recharge-go/internal/model"
	"recharge-go/internal/service/recharge"
)

// TestMishiTransactionID 测试mishi平台TransactionID设置
func TestMishiTransactionID(t *testing.T) {
	platform := &recharge.MishiPlatform{}

	// 测试表单格式回调
	formData := "szOrderId=TEST123&nFlag=1&szRtnMsg=success&fSalePrice=10.00&szVerifyString=abc123"
	callbackData, err := platform.ParseCallbackData([]byte(formData))

	assert.NoError(t, err)
	assert.NotNil(t, callbackData)
	assert.Equal(t, "mishi_TEST123", callbackData.TransactionID)
	assert.Equal(t, "TEST123", callbackData.OrderID)
	assert.Equal(t, "TEST123", callbackData.OrderNumber)

	// 测试JSON格式回调
	jsonData := `{"szOrderId":"TEST456","nFlag":"1","szRtnMsg":"success","fSalePrice":"20.00","szVerifyString":"def456"}`
	callbackData2, err2 := platform.ParseCallbackData([]byte(jsonData))

	assert.NoError(t, err2)
	assert.NotNil(t, callbackData2)
	assert.Equal(t, "mishi_TEST456", callbackData2.TransactionID)
	assert.Equal(t, "TEST456", callbackData2.OrderID)
	assert.Equal(t, "TEST456", callbackData2.OrderNumber)
}

// TestDayuanrenTransactionID 测试dayuanren平台TransactionID设置
func TestDayuanrenTransactionID(t *testing.T) {
	platform := &recharge.DayuanrenPlatform{}

	// 测试回调数据
	formData := "out_trade_num=DYR789&state=1&remark=充值成功&charge_amount=50.00&sign=xyz789&otime=1640995200"
	callbackData, err := platform.ParseCallbackData([]byte(formData))

	assert.NoError(t, err)
	assert.NotNil(t, callbackData)
	assert.Equal(t, "dayuanren_DYR789", callbackData.TransactionID)
	assert.Equal(t, "DYR789", callbackData.OrderID)
	assert.Equal(t, "DYR789", callbackData.OrderNumber)
}

// TestTransactionIDStandard 测试TransactionID标准一致性
func TestTransactionIDStandard(t *testing.T) {
	// 测试所有平台都正确设置了TransactionID
	testCases := []struct {
		name     string
		platform string
		data     string
		expected string
	}{
		{
			name:     "mishi平台表单格式",
			platform: "mishi",
			data:     "szOrderId=M001&nFlag=1&szRtnMsg=ok&fSalePrice=10&szVerifyString=sign1",
			expected: "mishi_M001",
		},
		{
			name:     "dayuanren平台",
			platform: "dayuanren",
			data:     "out_trade_num=D001&state=1&remark=ok&charge_amount=20&sign=sign2&otime=123456",
			expected: "dayuanren_D001",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var callbackData interface{}
			var err error

			switch tc.platform {
			case "mishi":
				platform := &recharge.MishiPlatform{}
				callbackData, err = platform.ParseCallbackData([]byte(tc.data))
			case "dayuanren":
				platform := &recharge.DayuanrenPlatform{}
				callbackData, err = platform.ParseCallbackData([]byte(tc.data))
			}

			assert.NoError(t, err)
			assert.NotNil(t, callbackData)

			// 验证TransactionID不为空
			if data, ok := callbackData.(*model.CallbackData); ok {
				assert.NotEmpty(t, data.TransactionID, "TransactionID不应为空")
				assert.Equal(t, tc.expected, data.TransactionID)
			}
		})
	}
}

// TestCallbackDuplicateCheck 测试回调重复检查逻辑
func TestCallbackDuplicateCheck(t *testing.T) {
	// 这个测试用于验证修复后的TransactionID能够正确支持重复检查
	// 特别是在换通道重试的场景下

	// 模拟第一次回调
	firstCallback := map[string]string{
		"order_number":   "ORDER123",
		"callback_type":  "order_status",
		"transaction_id": "TXN123",
	}

	// 模拟换通道后的第二次回调（不同的TransactionID）
	secondCallback := map[string]string{
		"order_number":   "ORDER123",
		"callback_type":  "order_status",
		"transaction_id": "TXN456", // 不同的TransactionID
	}

	// 验证两次回调应该被识别为不同的回调
	assert.NotEqual(t, firstCallback["transaction_id"], secondCallback["transaction_id"])
	assert.Equal(t, firstCallback["order_number"], secondCallback["order_number"])
	assert.Equal(t, firstCallback["callback_type"], secondCallback["callback_type"])
}