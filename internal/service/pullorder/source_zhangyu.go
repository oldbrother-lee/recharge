package pullorder

import (
	"context"
	"fmt"
	"strings"

	"recharge-go/internal/repository"
    zclient "recharge-go/internal/service/zhangyu"
    logger "recharge-go/pkg/log"
)

// ZhangyuPullPlatform 章鱼拉单平台实现
type ZhangyuPullPlatform struct {
	platformAccountRepo *repository.PlatformAccountRepository
	variantRepo         repository.PlatformAccountVariantRepository
}

func NewZhangyuPullPlatform(platformAccountRepo *repository.PlatformAccountRepository, variantRepo repository.PlatformAccountVariantRepository) *ZhangyuPullPlatform {
	return &ZhangyuPullPlatform{platformAccountRepo: platformAccountRepo, variantRepo: variantRepo}
}

func (p *ZhangyuPullPlatform) Code() string { return "zhangyu" }
func (p *ZhangyuPullPlatform) Name() string { return "章鱼" }

// Pull 使用统一配置ID作为输入
func (p *ZhangyuPullPlatform) Pull(ctx context.Context, variantID int64) ([]ExternalOrder, error) {
	// 读取变体
	variant, err := p.variantRepo.GetByID(ctx, variantID)
	if err != nil {
		return nil, fmt.Errorf("读取变体失败: %w", err)
	}
	if variant == nil || !variant.Enabled {
		return nil, nil
	}
	// 读取账号
	account, err := p.platformAccountRepo.GetByIDWithContext(ctx, variant.PlatformAccountID)
	if err != nil || account == nil {
		return nil, fmt.Errorf("读取平台账号失败")
	}

	client := zclient.NewClient(account.Platform.ApiURL)
	// 读取或登录token
	token, _ := client.LoadToken(ctx, account)
	if strings.TrimSpace(token) == "" {
		t, lerr := client.Login(ctx, account)
		if lerr != nil {
			return nil, fmt.Errorf("登录失败: %w", lerr)
		}
		token = t
	}

	// 拉单
	// 临时按示例渠道编码使用 dxfs（可根据配置覆盖）
	flag := "dxfs"
	if v := strings.TrimSpace(variant.Flag); v != "" {
		flag = v
	}
	amountStr := fmt.Sprintf("%g", variant.FaceValue)
	eo, gerr := client.GetOrder(ctx, account, token, flag, amountStr, amountStr, "")
	if gerr != nil {
		// 无订单或错误，返回空
		logger.WithContextCategory(ctx, "zhangyu").Info("拉单失败或暂无订单", logger.ErrorV2(gerr))
		return nil, nil
	}

	// 转换为统一结构
	order := ExternalOrder{
		ID:           eo.ID,
		Mobile:       eo.Mobile,
		OperatorID:   eo.OperatorID,
		Amount:       eo.Amount,
		Discount:     0,
		ProvinceName: eo.ProvinceName,
		ExternalCode: flag,
	}
	return []ExternalOrder{order}, nil
}
