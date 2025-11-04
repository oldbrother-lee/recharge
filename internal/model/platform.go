package model

import "time"

// Platform 平台信息
type Platform struct {
	ID          int64             `json:"id" gorm:"primaryKey"`
	Name        string            `json:"name" gorm:"size:50;not null"`
	Code        string            `json:"code" gorm:"size:20;not null;uniqueIndex"`
	ApiURL      string            `json:"api_url" gorm:"size:255;not null"`
	Description string            `json:"description" gorm:"size:255"`
	Status      int               `json:"status" gorm:"default:1"` // 1:启用 0:禁用
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	DeletedAt   *time.Time        `json:"deleted_at" gorm:"index"`
	Accounts    []PlatformAccount `gorm:"foreignKey:PlatformID" json:"accounts,omitempty"`
	APIs        []PlatformAPI     `gorm:"foreignKey:PlatformID" json:"apis,omitempty"`
}

// PlatformAccount 平台账号信息
type PlatformAccount struct {
	ID               int64      `json:"id" gorm:"primaryKey"`                                          // 主键ID
	PlatformID       int64      `json:"platform_id" gorm:"not null;index"`                             // 平台ID
	AccountName      string     `json:"account_name" gorm:"size:50;not null"`                          // 账号名称
	Type             int        `json:"type" gorm:"type:tinyint;default:1;comment:账号类型：1-测试账号，2-正式账号"` // 账号类型：1-测试账号，2-正式账号
	AppKey           string     `json:"app_key" gorm:"size:64;not null"`                               // AppKey
	AppSecret        string     `json:"app_secret" gorm:"size:64;not null"`                            // AppSecret
	AccountPassword  string     `json:"account_password" gorm:"size:255"`                              // 平台账号密码
	Description      string     `json:"description" gorm:"size:255"`                                   // 描述
	DailyLimit       float64    `json:"daily_limit" gorm:"type:decimal(10,2);default:0.00"`            // 每日限额
	MonthlyLimit     float64    `json:"monthly_limit" gorm:"type:decimal(10,2);default:0.00"`          // 每月限额
	Balance          float64    `json:"balance" gorm:"type:decimal(10,2);default:0.00"`                // 余额
	Priority         int        `json:"priority" gorm:"default:0"`                                     // 优先级
	Status           int        `json:"status" gorm:"type:tinyint;default:1;comment:状态：1-启用，0-禁用"`     // 状态：1-启用，0-禁用
	CreatedAt        time.Time  `json:"created_at" gorm:"autoCreateTime"`                              // 创建时间
	UpdatedAt        time.Time  `json:"updated_at" gorm:"autoUpdateTime"`                              // 更新时间
	DeletedAt        *time.Time `json:"deleted_at" gorm:"index"`                                       // 删除时间
	Platform         *Platform  `json:"platform,omitempty" gorm:"foreignKey:PlatformID"`               // 关联的平台信息
	BindUserID       *int64     `json:"bind_user_id" gorm:"column:bind_user_id;index"`                 // 绑定的本地用户ID
	BindUserName     string     `json:"bind_user_name" gorm:"column:bind_user_name"`                   // 绑定用户名（冗余字段）
	PushStatus       int        `gorm:"column:push_status;default:2" json:"push_status"`               // 推单状态(1:开启；2:关闭)
	EnablePullOrder  bool       `json:"enable_pull_order" gorm:"default:false;index"`                 // 是否启用拉单功能
	MaxConcurrency   int        `json:"max_concurrency" gorm:"default:1"`                             // 最大并发拉取数
	PollIntervalSec  int        `json:"poll_interval_sec" gorm:"default:10"`                          // 默认轮询间隔秒
	PullAction       string     `json:"pull_action" gorm:"size:64"`                                   // 拉单动作名
	Variants         []PlatformAccountVariant `json:"variants,omitempty" gorm:"foreignKey:PlatformAccountID"` // 关联的变体配置
}

// PlatformListRequest 平台列表请求
type PlatformListRequest struct {
	Page     int    `form:"page" binding:"required,min=1"`
	PageSize int    `form:"page_size" binding:"required,min=1,max=100"`
	Name     string `form:"name"`
	Code     string `form:"code"`
	Status   *int   `form:"status"`
}

// PlatformListResponse 平台列表响应
type PlatformListResponse struct {
	Total int64      `json:"total"`
	Items []Platform `json:"items"`
}

// PlatformAccountListRequest 平台账号列表请求
type PlatformAccountListRequest struct {
	Page       int    `form:"page" binding:"required,min=1"`
	PageSize   int    `form:"page_size" binding:"required,min=1,max=100"`
	PlatformID *int64 `form:"platform_id"`
	Status     *int   `form:"status"`
}

// PlatformAccountListResponse 平台账号列表响应
type PlatformAccountListResponse struct {
	Total int64             `json:"total"`
	Items []PlatformAccount `json:"items"`
}

// PlatformCreateRequest 创建平台请求
type PlatformCreateRequest struct {
	Name        string `json:"name" binding:"required,max=50"`
	Code        string `json:"code" binding:"required,max=20"`
	ApiURL      string `json:"api_url" binding:"required,max=255"`
	Description string `json:"description" binding:"max=255"`
}

// PlatformUpdateRequest 更新平台请求
type PlatformUpdateRequest struct {
	Name        string `json:"name" binding:"required,max=50"`
	Code        string `json:"code" binding:"required,max=20"`
	ApiURL      string `json:"api_url" binding:"required,max=255"`
	Description string `json:"description" binding:"max=255"`
	Status      *int   `json:"status" binding:"omitempty,oneof=0 1"`
}

// PlatformAccountCreateRequest 创建平台账号请求
type PlatformAccountCreateRequest struct {
	PlatformID      int64   `json:"platform_id" binding:"required"`         // 平台ID
	AccountName     string  `json:"account_name" binding:"required,max=50"` // 账号名称
	Type            int     `json:"type" binding:"required,oneof=1 2"`      // 账号类型：1-测试账号，2-正式账号
	AppKey          string  `json:"app_key" binding:"required,max=64"`      // AppKey
	AppSecret       string  `json:"app_secret" binding:"required,max=64"`   // AppSecret
	AccountPassword string  `json:"account_password" binding:"max=255"`     // 平台账号密码
	Description     string  `json:"description" binding:"max=255"`          // 描述
	DailyLimit      float64 `json:"daily_limit" binding:"min=0"`            // 每日限额
	MonthlyLimit    float64 `json:"monthly_limit" binding:"min=0"`          // 每月限额
	Priority        int     `json:"priority" binding:"min=0"`               // 优先级
	Status          *int    `json:"status" binding:"omitempty,oneof=0 1"`   // 状态：1-启用，0-禁用
	BindUserID      *int64  `json:"bind_user_id"`                           // 绑定的本地用户ID
	BindUserName    string  `json:"bind_user_name" binding:"max=50"`        // 绑定用户名
	EnablePullOrder bool    `json:"enable_pull_order"`                      // 是否启用拉单功能
	MaxConcurrency  int     `json:"max_concurrency" binding:"min=1"`        // 最大并发拉取数
	PollIntervalSec int     `json:"poll_interval_sec" binding:"min=1"`      // 默认轮询间隔秒
	PullAction      string  `json:"pull_action" binding:"max=64"`           // 拉单动作名
}

// PlatformAccountUpdateRequest 更新平台账号请求
type PlatformAccountUpdateRequest struct {
	AccountName     *string  `json:"account_name" binding:"max=50"`
	Type            *int     `json:"type" binding:"oneof=1 2"`
	AppKey          *string  `json:"app_key" binding:"max=64"`
	AppSecret       *string  `json:"app_secret" binding:"max=64"`
	AccountPassword *string  `json:"account_password" binding:"max=255"`
	Description     *string  `json:"description" binding:"max=255"`
	DailyLimit      *float64 `json:"daily_limit" binding:"min=0"`
	MonthlyLimit    *float64 `json:"monthly_limit" binding:"min=0"`
	Balance         *float64 `json:"balance" binding:"min=0"`
	Priority        *int     `json:"priority" binding:"min=0"`
	Status          *int     `json:"status" binding:"omitempty,oneof=0 1"`
	PushStatus      *int     `json:"push_status"`
	BindUserID      *int64   `json:"bind_user_id"`
	BindUserName    *string  `json:"bind_user_name" binding:"max=50"`
	EnablePullOrder *bool    `json:"enable_pull_order"`
	MaxConcurrency  *int     `json:"max_concurrency" binding:"min=1"`
	PollIntervalSec *int     `json:"poll_interval_sec" binding:"min=1"`
	PullAction      *string  `json:"pull_action" binding:"max=64"`
}

const PlatformCodeDayuanren = "dayuanren"

// TableName 返回表名
func (PlatformAccount) TableName() string {
	return "platform_accounts"
}

// PlatformAccountVariant 平台账号变体配置（替代原来的 PullSourceVariant）
type PlatformAccountVariant struct {
	ID                int64      `json:"id" gorm:"primaryKey"`                                    // 主键ID
	PlatformAccountID int64      `json:"platform_account_id" gorm:"not null;index"`              // 平台账号ID
	ISP               int        `json:"isp" gorm:"default:0;comment:运营商编码：1移动 2电信 3联通 0未知"`     // 运营商编码
	FaceValue         float64    `json:"face_value" gorm:"type:decimal(10,2);not null;default:0"` // 面值
	ProductID         *int64     `json:"product_id" gorm:"index"`                                // 关联的商品ID
	Enabled           bool       `json:"enabled" gorm:"default:true"`                            // 是否启用
	PollIntervalSec   int        `json:"poll_interval_sec" gorm:"default:10"`                    // 轮询间隔秒
	Concurrency       int        `json:"concurrency" gorm:"default:1"`                           // 并发度
	CursorToken       string     `json:"cursor_token" gorm:"size:255"`                           // 拉取游标
	LastPullAt        *time.Time `json:"last_pull_at"`                                           // 上次拉取时间
	FailCount         int        `json:"fail_count" gorm:"default:0"`                            // 连续失败计数
	NotifyURL         string     `json:"notify_url" gorm:"size:255"`                             // 变体级回调地址（可选）
	CreatedAt         time.Time  `json:"created_at" gorm:"autoCreateTime"`                       // 创建时间
	UpdatedAt         time.Time  `json:"updated_at" gorm:"autoUpdateTime"`                       // 更新时间
	PlatformAccount   *PlatformAccount `json:"platform_account,omitempty" gorm:"foreignKey:PlatformAccountID"` // 关联的平台账号
}

func (PlatformAccountVariant) TableName() string {
	return "platform_account_variants"
}

// PlatformAccountProductMap 平台账号商品映射（替代原来的 PullSourceProductMap）
type PlatformAccountProductMap struct {
	ID                int64      `json:"id" gorm:"primaryKey"`                     // 主键ID
	PlatformAccountID int64      `json:"platform_account_id" gorm:"not null;index"` // 平台账号ID
	KeyType           string     `json:"key_type" gorm:"size:32;not null"`         // 映射键类型：by_external_code | by_isp_face_value
	ExternalCode      string     `json:"external_code" gorm:"size:64"`             // 外部商品代码（可选）
	ISP               *int       `json:"isp"`                                      // 运营商编码（可选）
	FaceValue         *float64   `json:"face_value" gorm:"type:decimal(10,2)"`     // 面值（可选）
	ProductID         int64      `json:"product_id" gorm:"not null"`               // 内部商品ID
	AmountOverride    *float64   `json:"amount_override" gorm:"type:decimal(10,2)"` // 覆盖面值（可选）
	CreatedAt         time.Time  `json:"created_at" gorm:"autoCreateTime"`         // 创建时间
	UpdatedAt         time.Time  `json:"updated_at" gorm:"autoUpdateTime"`         // 更新时间
	PlatformAccount   *PlatformAccount `json:"platform_account,omitempty" gorm:"foreignKey:PlatformAccountID"` // 关联的平台账号
}

func (PlatformAccountProductMap) TableName() string {
	return "platform_account_product_map"
}
