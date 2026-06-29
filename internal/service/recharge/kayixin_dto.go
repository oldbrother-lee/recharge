package recharge

// kayixinAttachItem 下单充值信息项
type kayixinAttachItem struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// kayixinCreateBody 商品下单请求体
type kayixinCreateBody struct {
	GoodsID     int                 `json:"goodsId"`
	Count       int                 `json:"count"`
	NotifyURL   string              `json:"notifyUrl"`
	OuterNumber string              `json:"outerNumber"`
	SafePrice   float64             `json:"safePrice,omitempty"`
	Sku         string              `json:"sku,omitempty"`
	Attach      []kayixinAttachItem `json:"attach,omitempty"`
}

// kayixinDetailBody 订单详情查询请求体
type kayixinDetailBody struct {
	OrderNumber string `json:"orderNumber,omitempty"`
	OuterNumber string `json:"outerNumber,omitempty"`
}

// kayixinAPIResponse 卡易信通用响应
type kayixinAPIResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// kayixinCreateData 下单成功 data
type kayixinCreateData struct {
	OrderNumber string `json:"orderNumber"`
}

// kayixinCreateResp 下单响应
type kayixinCreateResp struct {
	Code int                `json:"code"`
	Msg  string             `json:"msg"`
	Data *kayixinCreateData `json:"data"`
}

// kayixinDetailData 订单详情 data
type kayixinDetailData struct {
	OrderNumber string  `json:"orderNumber"`
	OuterNumber string  `json:"outerNumber"`
	Status      int     `json:"status"`
	Count       int     `json:"count"`
	Money       float64 `json:"money"`
	RetMoney    float64 `json:"retMoney"`
	Result      string  `json:"result"`
}

// kayixinDetailResp 订单详情响应
type kayixinDetailResp struct {
	Code int                `json:"code"`
	Msg  string             `json:"msg"`
	Data *kayixinDetailData `json:"data"`
}

// kayixinBalanceData 账户余额 data
type kayixinBalanceData struct {
	Balance float64 `json:"balance"`
}

// kayixinBalanceResp 账户查询响应
type kayixinBalanceResp struct {
	Code int                 `json:"code"`
	Msg  string              `json:"msg"`
	Data *kayixinBalanceData `json:"data"`
}

// KayixinCallbackBody 订单回调通知 body
type KayixinCallbackBody struct {
	OrderNumber string  `json:"orderNumber"`
	OuterNumber string  `json:"outerNumber"`
	Status      int     `json:"status"`
	Cards       string  `json:"cards"`
	Money       float64 `json:"money"`
	RetMoney    float64 `json:"retMoney"`
	StartCount  string  `json:"startCount"`
	NowCount    string  `json:"nowCount"`
	EndCount    string  `json:"endCount"`
	Result      string  `json:"result"`
}
