package pullorder

import (
	"context"
)

// ExternalOrder 简化的外部订单结构（得众）
type ExternalOrder struct {
	ID            string   // 外部ID -> out_trade_num
	Mobile        string   // 充值账号
	OperatorID    int      // 运营商ID（外部编码）
	Amount        float64  // 面值
	Discount      float64  // 用户报价支付金额
	ProvinceName  string   // 归属地（省名）
	ExternalCode  string   // 外部商品代码（若有）
}

// PullOrderPlatform 拉单平台接口
// 负责按变体（ISP+面值）拉取并返回标准化外部订单
// 注意：具体HTTP交互在实现中完成；此接口只定义拉取与标准化职责
 type PullOrderPlatform interface {
	Code() string
	Name() string
	Pull(ctx context.Context, variantID int64) ([]ExternalOrder, error)
}