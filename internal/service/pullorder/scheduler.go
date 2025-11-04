package pullorder

import (
	"context"
	"fmt"
	"time"

	"recharge-go/internal/model"
	"recharge-go/internal/repository"
	"recharge-go/internal/utils"
	"recharge-go/pkg/logger"
)

// PullOrderScheduler 拉单任务调度器（骨架）
type PullOrderScheduler struct {
	mgr                 *PullOrderManager
	platformAccountRepo *repository.PlatformAccountRepository
	variantRepo         repository.PlatformAccountVariantRepository
}

func NewPullOrderScheduler(mgr *PullOrderManager, platformAccountRepo *repository.PlatformAccountRepository, variantRepo repository.PlatformAccountVariantRepository) *PullOrderScheduler {
	return &PullOrderScheduler{
		mgr:                 mgr,
		platformAccountRepo: platformAccountRepo,
		variantRepo:         variantRepo,
	}
}

// ProcessOnce 执行一次拉单处理（按所有已启用的平台账号与变体）
// 注意：此方法不会启动服务或循环；符合"不要自己运行服务"的要求
func (s *PullOrderScheduler) ProcessOnce(ctx context.Context) error {
	logger.InfoV2("开始执行拉单处理")

	// 获取启用拉单的平台账号
	accounts, err := s.platformAccountRepo.GetPullOrderAccounts(ctx)
	if err != nil {
		return err
	}
	logger.InfoV2("获取到启用拉单的平台账号", logger.IntV2("count", len(accounts)))

	for _, account := range accounts {
		logger.InfoV2("处理平台账号", logger.Int64V2("account_id", account.ID), logger.StringV2("platform_code", account.Platform.Code), logger.StringV2("account_name", account.AccountName))
		fmt.Printf("[DEBUG] 尝试获取平台: code=%s\n", account.Platform.Code)

		plat := s.mgr.GetPlatform(account.Platform.Code)
		if plat == nil {
			fmt.Printf("[DEBUG] 平台未找到: code=%s, 已注册的平台: %v\n", account.Platform.Code, s.mgr.ListPlatforms())
			logger.InfoV2("未注册的平台，跳过", logger.StringV2("code", account.Platform.Code), logger.StringV2("platform_name", account.Platform.Name))
			continue
		}
		fmt.Printf("[DEBUG] 找到平台: code=%s, platform=%T\n", account.Platform.Code, plat)

		// 获取该平台账号的启用变体
		variants, err := s.variantRepo.GetEnabledVariants(ctx, account.ID)
		if err != nil {
			return err
		}
		logger.InfoV2("变体数量", logger.IntV2("variant_count", len(variants)))

		for _, v := range variants {
			logger.InfoV2("处理变体", logger.Int64V2("variant_id", v.ID), logger.IntV2("isp", v.ISP), logger.Float64V2("face_value", v.FaceValue), logger.StringV2("cursor", v.CursorToken))

			start := time.Now()
			orders, err := plat.Pull(ctx, v.ID)
			if err != nil {
				logger.WithContext(ctx).Error("拉取订单失败", logger.ErrorV2(err), logger.Int64V2("variant_id", v.ID))
				// 增加失败计数
				if updateErr := s.variantRepo.IncrementFailCount(ctx, v.ID); updateErr != nil {
					logger.WithContext(ctx).Error("更新失败计数失败", logger.ErrorV2(updateErr))
				}
				continue
			}
			elapsed := time.Since(start)
			logger.InfoV2("拉取订单完成", logger.IntV2("order_count", len(orders)), logger.DurationV2("elapsed", elapsed))

			// 重置失败计数
			if resetErr := s.variantRepo.ResetFailCount(ctx, v.ID); resetErr != nil {
				logger.WithContext(ctx).Error("重置失败计数失败", logger.ErrorV2(resetErr))
			}

			// 更新最后拉取时间
			if updateErr := s.variantRepo.UpdateLastPullAt(ctx, v.ID); updateErr != nil {
				logger.WithContext(ctx).Error("更新最后拉取时间失败", logger.ErrorV2(updateErr))
			}

			// 处理拉取到的订单
			for _, order := range orders {
				logger.InfoV2("处理外部订单", logger.StringV2("external_id", order.ID), logger.StringV2("mobile", order.Mobile), logger.Float64V2("amount", order.Amount))

				// 1) 直接使用变种配置中的商品ID
				if v.ProductID == nil {
					logger.WithContext(ctx).Error("变种配置未设置商品ID，跳过落库", logger.Int64V2("variant_id", v.ID))
					continue
				}
				productID := *v.ProductID

				// 2) 取绑定用户作为下单用户
				if account.BindUserID == nil {
					logger.WithContext(ctx).Error("平台账号未绑定本地用户，无法落库", logger.Int64V2("platform_account_id", account.ID))
					continue
				}
				customerID := *account.BindUserID

				// 3) 构造订单模型（复用平台映射逻辑）
				var localOrder *model.Order
				if dzPlat, ok := plat.(*DzPullPlatform); ok {
					mapped, merr := dzPlat.MapToOrder(ctx, order, productID, customerID)
					if merr != nil {
						logger.WithContext(ctx).Error("外部订单映射失败", logger.ErrorV2(merr))
						continue
					}
					localOrder = mapped
					// 兜底：确保基础字段正确设置
					if localOrder.PlatformName == "" {
						localOrder.PlatformName = "得众"
					}
					if localOrder.PlatformCode == "" {
						localOrder.PlatformCode = "dz"
					}
					// 设置平台账号ID，用于预通知
					localOrder.PlatformAccountID = account.ID
				} else {
					// 兜底：直接构造必要字段
					ispCode := utils.DzOperatorIDToCode(order.OperatorID)
					localOrder = &model.Order{
						CustomerID:       customerID,
						Mobile:           order.Mobile,
						ProductID:        productID,
						Denom:            order.Amount,
						UserQuotePayment: order.Discount,
						ISP:              ispCode,
						AccountLocation:  order.ProvinceName,
						OutTradeNum:      order.ID,
						Client:           3,
						PlatformName:     account.Platform.Name,
						PlatformCode:     account.Platform.Code,
						Remark:           "得众拉单",
					}
				}

				// 4) 创建外部订单（仅落库，不扣款）
				if cerr := s.mgr.orderService.CreateExternalOrderWithoutDeduction(ctx, localOrder, customerID); cerr != nil {
					logger.WithContext(ctx).Error("创建外部订单失败", logger.ErrorV2(cerr), logger.StringV2("out_trade_num", order.ID))
					continue
				}
				logger.InfoV2("创建外部订单成功", logger.StringV2("out_trade_num", order.ID), logger.Int64V2("product_id", productID))
			}

			// 如果有订单，更新游标（假设最后一个订单的ID作为新游标）
			if len(orders) > 0 {
				newCursor := orders[len(orders)-1].ID
				if updateErr := s.variantRepo.UpdateCursor(ctx, v.ID, newCursor); updateErr != nil {
					logger.WithContext(ctx).Error("更新游标失败", logger.ErrorV2(updateErr))
				}
			}
		}
	}
	logger.InfoV2("拉单处理完成")
	return nil
}