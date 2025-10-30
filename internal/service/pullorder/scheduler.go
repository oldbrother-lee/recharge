package pullorder

import (
	"context"
	"fmt"
	"time"
	"recharge-go/internal/model"
	"recharge-go/internal/repository"
	"recharge-go/pkg/logger"
)

// PullOrderScheduler 拉单任务调度器（骨架）
type PullOrderScheduler struct {
	mgr  *PullOrderManager
	repo *repository.PullSourceRepositoryImpl
}

func NewPullOrderScheduler(mgr *PullOrderManager, repo *repository.PullSourceRepositoryImpl) *PullOrderScheduler {
	return &PullOrderScheduler{mgr: mgr, repo: repo}
}

// ProcessOnce 执行一次拉单处理（按所有已启用源与变体）
// 注意：此方法不会启动服务或循环；符合“不要自己运行服务”的要求
func (s *PullOrderScheduler) ProcessOnce(ctx context.Context) error {
	logger.InfoV2("开始执行拉单处理")

	sources, err := s.repo.GetEnabledSources(ctx)
	if err != nil { return err }
	logger.InfoV2("获取到启用的拉单源", logger.IntV2("count", len(sources)))

	for _, src := range sources {
		logger.InfoV2("处理拉单源", logger.Int64V2("source_id", src.ID), logger.StringV2("code", src.Code), logger.StringV2("name", src.Name))
		fmt.Printf("[DEBUG] 尝试获取平台: code=%s\n", src.Code)

		plat := s.mgr.GetPlatform(src.Code)
		if plat == nil {
			fmt.Printf("[DEBUG] 平台未找到: code=%s, 已注册的平台: %v\n", src.Code, s.mgr.ListPlatforms())
			logger.InfoV2("未注册的平台，跳过", logger.StringV2("code", src.Code), logger.StringV2("name", src.Name))
			continue
		}
		fmt.Printf("[DEBUG] 找到平台: code=%s, platform=%T\n", src.Code, plat)

		variants, err := s.repo.GetVariantsBySource(ctx, src.ID)
		if err != nil { return err }
		logger.InfoV2("变体数量", logger.IntV2("variant_count", len(variants)))

		for _, v := range variants {
			logger.InfoV2("处理变体", logger.Int64V2("variant_id", v.ID), logger.IntV2("isp", v.ISP), logger.Float64V2("face_value", v.FaceValue), logger.StringV2("cursor", v.Cursor))

			start := time.Now()
			orders, err := plat.Pull(ctx, v.ID)
			elapsed := time.Since(start)
			if err != nil {
				logger.ErrorLogV2("拉取外部订单失败", logger.ErrorV2(err), logger.Int64V2("variant_id", v.ID), logger.DurationV2("elapsed", elapsed))
				continue
			}

			if len(orders) == 0 {
				logger.InfoV2("未拉取到外部订单", logger.Int64V2("variant_id", v.ID), logger.DurationV2("elapsed", elapsed))
				continue
			}
			logger.InfoV2("拉取到外部订单", logger.IntV2("count", len(orders)), logger.Int64V2("variant_id", v.ID), logger.DurationV2("elapsed", elapsed))

			for _, ext := range orders {
				logger.InfoV2("开始映射外部订单", logger.StringV2("out_trade_num", ext.ID))

				// 先尝试 external_code 映射
				var mapped *model.PullProductMap
				if ext.ExternalCode != "" {
					mapped, err = s.repo.GetMapByExternalCode(ctx, src.ID, ext.ExternalCode)
					if err != nil { return err }
				}
				// 回退到 isp+面值 映射
				if mapped == nil {
					isp := v.ISP
					if isp == 0 { isp = ext.OperatorID }
					mapped, err = s.repo.GetMapByIspDenom(ctx, src.ID, isp, ext.Amount)
					if err != nil { return err }
				}
				if mapped == nil {
					logger.ErrorLogV2("未找到商品映射，跳过订单", logger.StringV2("out_trade_num", ext.ID))
					continue
				}

				// 构造订单
				order, err := s.mgr.GetPlatform(src.Code).(*DzPullPlatform).MapToOrder(ctx, ext, mapped.ProductID)
				if err != nil { return fmt.Errorf("映射订单失败: %w", err) }
				// 记录来源 SourceID，便于后续通知上报使用
				order.Param1 = fmt.Sprintf("%d", src.ID)

				logger.InfoV2("准备创建订单", logger.StringV2("out_trade_num", order.OutTradeNum), logger.Int64V2("product_id", mapped.ProductID))
				// 创建订单（由系统服务处理状态与充值队列）
				if err := s.mgr.orderService.CreateOrder(ctx, order); err != nil {
					logger.ErrorLogV2("创建订单失败", logger.ErrorV2(err), logger.StringV2("out_trade_num", order.OutTradeNum))
					continue
				}

				logger.InfoV2("创建订单成功", logger.StringV2("out_trade_num", order.OutTradeNum))
			}
		}
	}
	logger.InfoV2("拉单处理完成")
	return nil
}