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
	ID           int64      `json:"id" gorm:"primaryKey"`                                          // 主键ID
	PlatformID   int64      `json:"platform_id" gorm:"not null;index"`                             // 平台ID
	AccountName  string     `json:"account_name" gorm:"size:50;not null"`                          // 账号名称
	Type         int        `json:"type" gorm:"type:tinyint;default:1;comment:账号类型：1-测试账号，2-正式账号"` // 账号类型：1-测试账号，2-正式账号
	AppKey       string     `json:"app_key" gorm:"size:64;not null"`                               // AppKey
	AppSecret    string     `json:"app_secret" gorm:"size:64;not null"`                            // AppSecret
	Description  string     `json:"description" gorm:"size:255"`                                   // 描述
	DailyLimit   float64    `json:"daily_limit" gorm:"type:decimal(10,2);default:0.00"`            // 每日限额
	MonthlyLimit float64    `json:"monthly_limit" gorm:"type:decimal(10,2);default:0.00"`          // 每月限额
	Balance      float64    `json:"balance" gorm:"type:decimal(10,2);default:0.00"`                // 余额
	Priority     int        `json:"priority" gorm:"default:0"`                                     // 优先级
	Status       int        `json:"status" gorm:"type:tinyint;default:1;comment:状态：1-启用，0-禁用"`     // 状态：1-启用，0-禁用
	CreatedAt    time.Time  `json:"created_at" gorm:"autoCreateTime"`                              // 创建时间
	UpdatedAt    time.Time  `json:"updated_at" gorm:"autoUpdateTime"`                              // 更新时间
	DeletedAt    *time.Time `json:"deleted_at" gorm:"index"`                                       // 删除时间
	Platform     *Platform  `json:"platform,omitempty" gorm:"foreignKey:PlatformID"`               // 关联的平台信息
	BindUserID   *int64     `json:"bind_user_id" gorm:"column:bind_user_id"`
	BindUserName string     `json:"bind_user_name" gorm:"column:bind_user_name"`
	PushStatus   int        `gorm:"column:push_status;default:2" json:"push_status"` // 推单状态(1:开启；2:关闭)

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
	PlatformID   int64   `json:"platform_id" binding:"required"`         // 平台ID
	AccountName  string  `json:"account_name" binding:"required,max=50"` // 账号名称
	Type         int     `json:"type" binding:"required,oneof=1 2"`      // 账号类型：1-测试账号，2-正式账号
	AppKey       string  `json:"app_key" binding:"required,max=64"`      // AppKey
	AppSecret    string  `json:"app_secret" binding:"required,max=64"`   // AppSecret
	Description  string  `json:"description" binding:"max=255"`          // 描述
	DailyLimit   float64 `json:"daily_limit" binding:"min=0"`            // 每日限额
	MonthlyLimit float64 `json:"monthly_limit" binding:"min=0"`          // 每月限额
	Priority     int     `json:"priority" binding:"min=0"`               // 优先级
	Status       *int    `json:"status" binding:"omitempty,oneof=0 1"`   // 状态：1-启用，0-禁用
}

// PlatformAccountUpdateRequest 更新平台账号请求
type PlatformAccountUpdateRequest struct {
	AccountName  *string  `json:"account_name" binding:"max=50"`
	Type         *int     `json:"type" binding:"oneof=1 2"`
	AppKey       *string  `json:"app_key" binding:"max=64"`
	AppSecret    *string  `json:"app_secret" binding:"max=64"`
	Description  *string  `json:"description" binding:"max=255"`
	DailyLimit   *float64 `json:"daily_limit" binding:"min=0"`
	MonthlyLimit *float64 `json:"monthly_limit" binding:"min=0"`
	Balance      *float64 `json:"balance" binding:"min=0"`
	Priority     *int     `json:"priority" binding:"min=0"`
	Status       *int     `json:"status" binding:"omitempty,oneof=0 1"`
	PushStatus   *int     `json:"push_status"`
}

const PlatformCodeDayuanren = "dayuanren"

// TableName 返回表名
func (PlatformAccount) TableName() string {
	return "platform_accounts"
}
