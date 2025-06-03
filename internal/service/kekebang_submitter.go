package service

import (
	"context"
	"fmt"

	"recharge-go/internal/model"
	"recharge-go/internal/signature"
	"recharge-go/pkg/logger"
)

// KekebangSubmitter 客客帮订单提交器
type KekebangSubmitter struct {
	*BaseOrderSubmitter
}

// NewKekebangSubmitter 创建客客帮订单提交器
func NewKekebangSubmitter(handler signature.SignatureHandler) *KekebangSubmitter {
	return &KekebangSubmitter{
		BaseOrderSubmitter: NewBaseOrderSubmitter(handler),
	}
}

// SubmitOrder 提交订单
func (s *KekebangSubmitter) SubmitOrder(ctx context.Context, order *model.Order, api *model.PlatformAPI, apiParam *model.PlatformAPIParam) error {
	// 调用基础实现
	return s.BaseOrderSubmitter.SubmitOrder(ctx, order, api, apiParam)
}

// handleResponse 处理客客帮响应
func (s *KekebangSubmitter) handleResponse(resp map[string]interface{}) error {
	// 检查响应状态码
	code, ok := resp["code"].(float64)
	if !ok {
		return fmt.Errorf("响应状态码格式错误")
	}

	if code != 0 {
		msg, _ := resp["msg"].(string)
		return fmt.Errorf("请求失败: %s", msg)
	}

	// 检查订单状态
	status, ok := resp["status"].(float64)
	if !ok {
		return fmt.Errorf("订单状态格式错误")
	}

	// 根据状态码处理
	switch int(status) {
	case 1: // 成功
		logger.Info("客客帮订单提交成功", "response", resp)
		return nil
	case 2: // 处理中
		logger.Info("客客帮订单处理中", "response", resp)
		return nil
	case 3: // 失败
		msg, _ := resp["msg"].(string)
		return fmt.Errorf("订单提交失败: %s", msg)
	default:
		return fmt.Errorf("未知的订单状态: %d", int(status))
	}
}
