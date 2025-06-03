package model

// KekebangOrderRequest 可客帮订单请求
type KekebangOrderRequest struct {
	AppKey string `json:"app_key"` // 应用密钥
	Datas  struct {
		Amount     float64 `json:"amount"`      // 面值
		OperatorID string  `json:"operator_id"` // 运营商
		ProvCode   string  `json:"prov_code"`   // 省份代码
	} `json:"datas"`
	GoodsID          string `json:"goods_id"`           // 商品ID
	GoodsName        string `json:"goods_name"`         // 商品名称
	OfficialPayment  string `json:"official_payment"`   // 官方价格
	OuterGoodsCode   string `json:"outer_goods_code"`   // 外部商品编码
	Sign             string `json:"sign"`               // 签名
	Target           string `json:"target"`             // 充值手机号
	Timestamp        int64  `json:"timestamp"`          // 时间戳
	UserOrderID      int64  `json:"user_order_id"`      // 用户订单ID
	UserPayment      string `json:"user_payment"`       // 用户支付金额
	UserQuotePayment string `json:"user_quote_payment"` // 用户报价金额
	UserQuoteType    int    `json:"user_quote_type"`    // 用户报价类型
	VenderID         int64  `json:"vender_id"`          // 供应商ID
}
