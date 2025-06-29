package model

import (
	"time"

	"gorm.io/gorm"
)

// OrderStatus 订单状态
type OrderStatus int

const (
	OrderStatusPendingPayment  OrderStatus = iota + 1 // 待支付 (1)
	OrderStatusPendingRecharge                        // 待充值 (2)
	OrderStatusRecharging                             // 充值中 (3)
	OrderStatusSuccess                                // 充值成功 (4)
	OrderStatusFailed                                 // 充值失败 (5)
	OrderStatusRefunded                               // 已退款 (6)
	OrderStatusCancelled                              // 已取消 (7)
	OrderStatusPartial                                // 部分充值 (8)
	OrderStatusSplit                                  // 已拆单 (9)
	OrderStatusProcessing                             // 处理中 (10)
)

// String 返回订单状态的字符串表示
func (s OrderStatus) String() string {
	switch s {
	case OrderStatusPendingPayment:
		return "pending_payment"
	case OrderStatusPendingRecharge:
		return "pending_recharge"
	case OrderStatusRecharging:
		return "recharging"
	case OrderStatusSuccess:
		return "success"
	case OrderStatusFailed:
		return "failed"
	case OrderStatusRefunded:
		return "refunded"
	case OrderStatusCancelled:
		return "cancelled"
	case OrderStatusPartial:
		return "partial"
	case OrderStatusSplit:
		return "split"
	case OrderStatusProcessing:
		return "processing"
	default:
		return "unknown"
	}
}

// Order 订单模型
type Order struct {
	ID                int64       `json:"id" gorm:"primaryKey"`
	OrderNumber       string      `json:"order_number" gorm:"size:32;uniqueIndex;comment:订单号"`
	CustomerID        int64       `json:"customer_id" gorm:"index;comment:客户ID"`
	Mobile            string      `json:"mobile" gorm:"size:20;index;comment:手机号"`
	ProductID         int64       `json:"product_id" gorm:"index;comment:产品ID"`
	Status            OrderStatus `json:"status" gorm:"comment:订单状态"`
	Denom             float64     `json:"denom" gorm:"type:decimal(10,2);comment:面值"`
	TotalPrice        float64     `json:"total_price" gorm:"type:decimal(10,2);comment:总价"`
	Price             float64     `json:"price" gorm:"type:decimal(10,2);comment:单价"`
	OfficialPayment   float64     `json:"official_payment" gorm:"type:decimal(10,2);comment:官方支付金额"`
	UserQuotePayment  float64     `json:"user_quote_payment" gorm:"type:decimal(10,2);comment:用户报价支付金额"`
	UserPayment       float64     `json:"user_payment" gorm:"type:decimal(10,2);comment:用户支付金额"`
	PayWay            int         `json:"pay_way" gorm:"comment:支付方式"`
	SerialNumber      string      `json:"serial_number" gorm:"size:64;comment:支付流水号"`
	IsPay             int         `json:"is_pay" gorm:"comment:是否支付"`
	PayTime           *time.Time  `json:"pay_time" gorm:"comment:支付时间"`
	CreateTime        time.Time   `json:"create_time" gorm:"comment:创建时间"`
	FinishTime        *time.Time  `json:"finish_time" gorm:"comment:完成时间"`
	Remark            string      `json:"remark" gorm:"size:255;comment:备注"`
	ISP               int         `json:"isp" gorm:"comment:运营商"`
	AccountLocation   string      `json:"account_location" gorm:"size:255;comment:归属地"`
	Param1            string      `json:"param1" gorm:"size:255;comment:参数1"`
	Param2            string      `json:"param2" gorm:"size:255;comment:参数2"`
	Param3            string      `json:"param3" gorm:"size:255;comment:参数3"`
	OutTradeNum       string      `json:"out_trade_num" gorm:"size:64;uniqueIndex;comment:外部交易号"`
	APICurID          int64       `json:"api_cur_id" gorm:"comment:当前API ID"`
	APIOrderNumber    string      `json:"api_order_number" gorm:"size:64;comment:API订单号"`
	APITradeNum       string      `json:"api_trade_num" gorm:"size:64;comment:API交易号"`
	APICurParamID     int64       `json:"api_cur_param_id" gorm:"comment:API当前参数ID"`
	APICurCount       int         `json:"api_cur_count" gorm:"comment:API当前计数"`
	APIOpen           int         `json:"api_open" gorm:"comment:API是否开启"`
	APICurIndex       int         `json:"api_cur_index" gorm:"comment:API当前索引"`
	IsApart           int         `json:"is_apart" gorm:"comment:是否拆单"`
	ApartOrderNumber  string      `json:"apart_order_number" gorm:"size:32;comment:拆单订单号"`
	DelayTime         int         `json:"delay_time" gorm:"comment:延迟时间"`
	ApplyRefund       int         `json:"apply_refund" gorm:"comment:申请退款"`
	IsRebate          int         `json:"is_rebate" gorm:"comment:是否返利"`
	IsDel             int         `json:"is_del" gorm:"comment:是否删除"`
	Guishu            string      `json:"guishu" gorm:"size:64;comment:归属"`
	Client            int         `json:"client" gorm:"comment:客户端"`
	WeixinAppID       string      `json:"weixin_appid" gorm:"size:64;comment:微信APPID"`
	UpdatedAt         time.Time   `json:"updated_at" gorm:"comment:更新时间"`
	PlatformId        int64       `json:"platform_id" gorm:"comment:平台ID"`
	PlatformAccountID int64       `json:"platform_account_id" gorm:"comment:平台账号ID"`
	UserOrderId       string      `json:"user_order_id" gorm:"size:64;comment:用户订单ID"`
	PlatformName      string      `json:"platform_name" gorm:"size:255;comment:平台名称"`
	PlatformCode      string      `json:"platform_code" gorm:"size:50;comment:平台代码"`
	ConstPrice        float64     `json:"const_price" gorm:"type:decimal(10,2);comment:成本价格"`
	// 平台配置信息
	PlatformAppKey      string         `json:"platform_app_key" gorm:"size:255;comment:平台AppKey"`
	PlatformSecretKey   string         `json:"platform_secret_key" gorm:"size:255;comment:平台SecretKey"`
	PlatformURL         string         `json:"platform_url" gorm:"size:255;comment:平台URL"`
	PlatformCallbackURL string         `json:"platform_callback_url" gorm:"size:255;comment:平台回调URL"`
	APIID               int64          `json:"api_id" gorm:"index"`                          // 当前订单使用的 API ID
	UsedAPIs            string         `json:"used_apis" gorm:"type:text;comment:已使用的API列表"` // 已使用的API列表，JSON格式
	CreatedAt           time.Time      `json:"created_at" gorm:"index"`
	DeletedAt           gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}

type StatisticsQuery struct {
	UserID int64  // 用户ID
	Start  string // 开始时间
	End    string // 结束时间
	// 可根据需要添加更多字段
}

// OrderWithNotification 订单与通知信息的响应结构体
type OrderWithNotification struct {
	*Order
	NotificationTime   *time.Time `json:"notification_time"`   // 通知时间
	NotificationStatus *int       `json:"notification_status"` // 通知状态 1:待处理 2:处理中 3:成功 4:失败
}

// TableName 表名
func (Order) TableName() string {
	return "orders"
}
