package pullorder

import (
	"recharge-go/internal/repository"
	"recharge-go/internal/service"
)

// PullOrderManager 管理平台实现及上下文依赖
type PullOrderManager struct {
	platforms    map[string]PullOrderPlatform
	orderService service.OrderService
}

func NewPullOrderManager(orderSvc service.OrderService, platformAccountRepo *repository.PlatformAccountRepository, variantRepo repository.PlatformAccountVariantRepository) *PullOrderManager {
    m := &PullOrderManager{
        platforms:    make(map[string]PullOrderPlatform),
        orderService: orderSvc,
    }
    dzPlatform := NewDzPullPlatform(platformAccountRepo, variantRepo)
    m.RegisterPlatform(dzPlatform)
    // 默认注册章鱼平台，避免任何入口遗漏注册导致“未注册的平台”
    zhangyuPlatform := NewZhangyuPullPlatform(platformAccountRepo, variantRepo)
    m.RegisterPlatform(zhangyuPlatform)
    return m
}

func (m *PullOrderManager) RegisterPlatform(p PullOrderPlatform) {
	m.platforms[p.Code()] = p
}

func (m *PullOrderManager) GetPlatform(code string) PullOrderPlatform {
	return m.platforms[code]
}

func (m *PullOrderManager) ListPlatforms() []string {
	var codes []string
	for code := range m.platforms {
		codes = append(codes, code)
	}
	return codes
}