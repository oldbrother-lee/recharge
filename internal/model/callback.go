package model

// CallbackData 回调数据
type CallbackData struct {
	OrderID       string `json:"order_id"`       // 订单号
	OrderNumber   string `json:"order_number"`   // 订单编号
	Status        string `json:"status"`         // 订单状态
	Message       string `json:"message"`        // 消息
	CallbackType  string `json:"callback_type"`  // 回调类型
	Amount        string `json:"amount"`         // 金额
	Sign          string `json:"sign"`           // 签名
	Timestamp     string `json:"timestamp"`      // 时间戳
	TransactionID string `json:"transaction_id"` // 交易ID
}
