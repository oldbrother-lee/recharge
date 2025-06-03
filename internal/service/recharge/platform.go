package recharge

import (
	"context"
	"recharge-go/internal/model"
)

// Platform 平台接口
type Platform interface {
	// GetName 获取平台名称
	GetName() string
	// SubmitOrder 提交订单
	SubmitOrder(ctx context.Context, order *model.Order, api *model.PlatformAPI, apiParam *model.PlatformAPIParam) error
	// QueryOrderStatus 查询订单状态
	QueryOrderStatus(order *model.Order) (model.OrderStatus, error)
	// ParseCallbackData 解析回调数据
	ParseCallbackData(data []byte) (*model.CallbackData, error)
	// QueryBalance 查询账户余额
	QueryBalance(ctx context.Context, accountID int64) (float64, error)
}

// PlatformConfig 平台配置
type PlatformConfig struct {
	AppKey    string
	AppSecret string
	ApiURL    string
	NotifyURL string
}

// BasePlatform 基础平台实现
type BasePlatform struct {
	platformName string
}

// NewBasePlatform 创建基础平台
func NewBasePlatform(platformName string) *BasePlatform {
	return &BasePlatform{
		platformName: platformName,
	}
}

// GetPlatformName 获取平台名称
func (p *BasePlatform) GetPlatformName() string {
	return p.platformName
}
