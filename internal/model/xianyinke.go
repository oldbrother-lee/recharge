package model

// XianyinkeOrderRequest 闲赢客推送订单请求体
// 对应文档《推送订单（需要我们提供接口）》字段
// 说明：
// - id: 平台订单号（使用该值作为我们系统的外部交易号以保证幂等）
// - chan_pro_code: 商户自定义产品编码（建议传我方商品ID，便于直接映射）
// 其他字段按文档原样保留，便于签名校验
type XianyinkeOrderRequest struct {
	ID           int64  `json:"id"`            // 平台订单号
	ProductID    int64  `json:"product_id"`    // 文档含义为运营商编码，保留
	ProvinceID   string `json:"province_id"`   // 归属地编码
	ChanProCode  string `json:"chan_pro_code"` // 商户自定义产品编码（建议传内部商品ID字符串）
	MarketPrice  int64  `json:"market_price"`  // 充值金额(元)，按文档为整数
	Account      string `json:"account"`       // 充值账号
	SettleMoney  string `json:"settle_money"`  // 结算金额(元)
	SettleStatus int    `json:"settle_status"` // 结算状态码
	Timeout      int64  `json:"timeout"`       // 超时时间戳
	Status       int    `json:"status"`        // 订单状态码（2处理中/3异常/4失败/5成功）
	AppKey       string `json:"app_key"`       // 商户号
	Sign         string `json:"sign"`          // 签名
}
