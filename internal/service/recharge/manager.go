package recharge

import (
	"context"
	"fmt"
    "recharge-go/internal/model"
    "recharge-go/internal/repository"
    logger "recharge-go/pkg/log"
	"reflect"
	"sync"

	"gorm.io/gorm"
)

// Manager 平台管理器
type Manager struct {
	platformRepo    repository.PlatformRepository
	platformAPIRepo repository.PlatformAPIRepository
	platforms       map[string]Platform
	platformTypes   map[string]reflect.Type
	mu              sync.RWMutex
}

// NewManager 创建平台管理器
func NewManager(db *gorm.DB) *Manager {
	return &Manager{
		platformRepo:    repository.NewPlatformRepository(db),
		platformAPIRepo: repository.NewPlatformAPIRepository(db),
		platforms:       make(map[string]Platform),
		platformTypes: map[string]reflect.Type{
			"kekebang":     reflect.TypeOf((*KekebangPlatform)(nil)).Elem(),
			"xianzhuanxia": reflect.TypeOf((*XianzhuanxiaPlatform)(nil)).Elem(),
			"mishi":        reflect.TypeOf((*MishiPlatform)(nil)).Elem(),
			"internal_api": reflect.TypeOf((*ExternalAPIPlatform)(nil)).Elem(),
			"chongzhi":     reflect.TypeOf((*ChongzhiPlatform)(nil)).Elem(),
			"payc2":        reflect.TypeOf((*Payc2Platform)(nil)).Elem(),
			"lingshi":      reflect.TypeOf((*LingshiPlatform)(nil)).Elem(),
			// 新增对 xianyinke 的支持（作为 external_api 的别名）
			"xianyinke":    reflect.TypeOf((*ExternalAPIPlatform)(nil)).Elem(),
			// 新增卡速售平台类型
			"kasushou":     reflect.TypeOf((*KasushouPlatform)(nil)).Elem(),
			// 新增商腾科技平台类型
			"shangteng":    reflect.TypeOf((*ShangtengPlatform)(nil)).Elem(),
		},
	}
}

// GetPlatform 获取平台实例
func (m *Manager) GetPlatform(platformCode string) (Platform, error) {
	// 先从缓存中查找
	m.mu.RLock()
	if platform, exists := m.platforms[platformCode]; exists {
		m.mu.RUnlock()
		return platform, nil
	}
	m.mu.RUnlock()

	// 创建平台实例
	platform, err := m.createPlatform(platformCode)
	if err != nil {
		return nil, fmt.Errorf("failed to create platform instance for %s: %v", platformCode, err)
	}

	// 缓存平台实例
	m.mu.Lock()
	m.platforms[platformCode] = platform
	m.mu.Unlock()

	return platform, nil
}

// createPlatform 创建平台实例
func (m *Manager) createPlatform(code string) (Platform, error) {
	// 创建平台实例
	var platform Platform
	switch code {
	case "kekebang":
		platform = NewKekebangPlatform(m.platformRepo.GetDB())
	case "xianzhuanxia":
		platform = NewXianzhuanxiaPlatform(m.platformRepo.GetDB())
	case "mifeng":
		platform = NewKekebangPlatform(m.platformRepo.GetDB()) // 暂时使用可客帮平台的实现
	case "mishi":
		platform = NewMishiPlatform(m.platformRepo.GetDB())
	case "dayuanren":
		platform = NewDayuanrenPlatform(m.platformRepo.GetDB())
	case "internal_api":
		platform = NewExternalAPIPlatform(m.platformRepo.GetDB())
	case "chongzhi":
		platform = NewChongzhiPlatform(m.platformRepo.GetDB())
	case "payc2":
		platform = NewPayc2Platform(m.platformRepo.GetDB())
	case "lingshi":
		platform = NewLingshiPlatform(m.platformRepo.GetDB())
	case "xianyinke":
		// 将 xianyinke 作为 external_api 的别名平台，避免启动时报不支持
		platform = NewExternalAPIPlatform(m.platformRepo.GetDB())
	case "kasushou":
		platform = NewKasushouPlatform(m.platformRepo.GetDB())
	case "shangteng":
		platform = NewShangtengPlatform(m.platformRepo.GetDB())
	default:
		return nil, fmt.Errorf("unsupported platform code: %s", code)
	}

	return platform, nil
}

// RegisterPlatform 注册新的平台类型
func (m *Manager) RegisterPlatform(code string, platformType reflect.Type) {
    m.mu.Lock()
    defer m.mu.Unlock()

    // 确保platformType实现了Platform接口
    if !reflect.TypeOf((*Platform)(nil)).Elem().Implements(platformType) {
        logger.GetCategoryLogger("recharge").Error("平台类型未实现接口",
            logger.StringV2("code", code),
        )
        return
    }

    // 注册平台类型
    m.platformTypes[code] = platformType
    logger.GetCategoryLogger("recharge").Info("注册平台成功",
        logger.StringV2("code", code),
    )
}

// LoadPlatforms 加载所有平台
func (m *Manager) LoadPlatforms() error {
	// 获取所有启用的平台
	platforms, _, err := m.platformRepo.ListPlatforms(&model.PlatformListRequest{
		Page:     1,
		PageSize: 100,
		Status:   &[]int{1}[0],
	})
	if err != nil {
		return fmt.Errorf("failed to list platforms: %v", err)
	}

	// 遍历平台列表，创建对应的平台实例
    for _, platform := range platforms {
        platformInstance, err := m.createPlatform(platform.Code)
        if err != nil {
            logger.GetCategoryLogger("recharge").Error("创建平台实例失败",
                logger.StringV2("code", platform.Code),
                logger.ErrorV2(err),
            )
            continue
        }
        m.platforms[platform.Code] = platformInstance
        logger.GetCategoryLogger("recharge").Info("加载平台成功",
            logger.StringV2("code", platform.Code),
        )
    }

	return nil
}

// SubmitOrder 提交订单到平台
func (m *Manager) SubmitOrder(ctx context.Context, order *model.Order, api *model.PlatformAPI, apiParam *model.PlatformAPIParam) error {
	// 获取平台实例
	platform, err := m.GetPlatform(api.Code)
	if err != nil {
		// 如果平台不存在，尝试动态注册
		platformType := m.getPlatformTypeByName(api.Code)
		if platformType != nil {
			m.RegisterPlatform(api.Code, platformType)
			platform, err = m.GetPlatform(api.Code)
			if err != nil {
				return fmt.Errorf("failed to get platform: %v", err)
			}
		} else {
			return fmt.Errorf("platform not found: %s", api.Code)
		}
	}
	return platform.SubmitOrder(ctx, order, api, apiParam)
}

// getPlatformTypeByName 根据平台名称获取平台类型
func (m *Manager) getPlatformTypeByName(name string) reflect.Type {
	switch name {
	case "kekebang":
		return reflect.TypeOf((*KekebangPlatform)(nil)).Elem()
	case "xianzhuanxia":
		return reflect.TypeOf((*XianzhuanxiaPlatform)(nil)).Elem()
	case "mifeng":
		return reflect.TypeOf((*KekebangPlatform)(nil)).Elem() // 暂时使用可客帮平台的实现
	case "mishi":
		return reflect.TypeOf((*MishiPlatform)(nil)).Elem()
	case "chongzhi":
		return reflect.TypeOf((*ChongzhiPlatform)(nil)).Elem()
	case "xianyinke":
		// 将 xianyinke 视为 external_api 类型
		return reflect.TypeOf((*ExternalAPIPlatform)(nil)).Elem()
	case "kasushou":
		return reflect.TypeOf((*KasushouPlatform)(nil)).Elem()
	case "shangteng":
		return reflect.TypeOf((*ShangtengPlatform)(nil)).Elem()
	default:
		return nil
	}
}

// QueryOrderStatus 查询订单状态
func (m *Manager) QueryOrderStatus(ctx context.Context, order *model.Order) error {
	// 获取平台实例
	platform, err := m.GetPlatform(order.PlatformCode)
	if err != nil {
		return fmt.Errorf("failed to get platform: %v", err)
	}

	status, err := platform.QueryOrderStatus(ctx, order)
	if err != nil {
		return fmt.Errorf("query order status failed: %v", err)
	}

	// 更新订单状态
	order.Status = status
	return nil
}

// HandleCallback 处理回调
func (m *Manager) HandleCallback(ctx context.Context, platformCode string, data []byte) error {
	platform, err := m.GetPlatform(platformCode)
	if err != nil {
		return fmt.Errorf("failed to get platform: %v", err)
	}

	callbackData, err := platform.ParseCallbackData(data)
	if err != nil {
		return fmt.Errorf("parse callback data failed: %v", err)
	}

	// TODO: 根据 callbackData 更新订单状态
	_ = callbackData
	return nil
}

// ParseCallbackData 解析回调数据
func (m *Manager) ParseCallbackData(platformCode string, data []byte) (*model.CallbackData, error) {
	platform, err := m.GetPlatform(platformCode)
	if err != nil {
		return nil, fmt.Errorf("failed to get platform: %v", err)
	}
	return platform.ParseCallbackData(data)
}
