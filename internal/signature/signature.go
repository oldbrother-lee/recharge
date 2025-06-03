package signature

import (
	"context"
	"recharge-go/internal/model"
)

// SignatureHandler 签名处理接口
type SignatureHandler interface {
	// GenerateSignature 生成签名
	GenerateSignature(ctx context.Context, params map[string]interface{}) (string, error)
	// BuildRequestParams 构建请求参数
	BuildRequestParams(ctx context.Context, order *model.Order, api *model.PlatformAPI) (map[string]interface{}, error)
}

// BaseSignatureHandler 基础签名处理器
type BaseSignatureHandler struct {
	config *Config
}

// Config 签名配置
type Config struct {
	AppID     string
	AppSecret string
	// 其他配置项...
}

// NewBaseSignatureHandler 创建基础签名处理器
func NewBaseSignatureHandler(config *Config) *BaseSignatureHandler {
	return &BaseSignatureHandler{
		config: config,
	}
}

// GenerateSignature 基础签名生成方法
func (h *BaseSignatureHandler) GenerateSignature(ctx context.Context, params map[string]interface{}) (string, error) {
	// 基础实现，具体平台需要重写
	return "", nil
}

// BuildRequestParams 基础参数构建方法
func (h *BaseSignatureHandler) BuildRequestParams(ctx context.Context, order *model.Order, api *model.PlatformAPI) (map[string]interface{}, error) {
	// 基础实现，具体平台需要重写
	return nil, nil
}
