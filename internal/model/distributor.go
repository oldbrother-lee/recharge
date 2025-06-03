package model

import "time"

// DistributorRequest 分销商请求参数
type DistributorRequest struct {
	UserID      int64   `json:"user_id" binding:"required"`    // 用户ID
	Name        string  `json:"name" binding:"required"`       // 分销商名称
	Phone       string  `json:"phone" binding:"required"`      // 联系电话
	Commission  float64 `json:"commission" binding:"required"` // 佣金比例
	Status      int     `json:"status" binding:"required"`     // 状态：0-待审核 1-正常 2-禁用
	Description string  `json:"description"`                   // 描述
}

// Distributor 分销商模型
type Distributor struct {
	ID          int64     `json:"id" gorm:"primaryKey"` // 分销商ID
	UserID      int64     `json:"user_id" gorm:"index"` // 用户ID
	Name        string    `json:"name"`                 // 分销商名称
	Phone       string    `json:"phone"`                // 联系电话
	Commission  float64   `json:"commission"`           // 佣金比例
	Status      int       `json:"status"`               // 状态：0-待审核 1-正常 2-禁用
	Description string    `json:"description"`          // 描述
	CreatedAt   time.Time `json:"created_at"`           // 创建时间
	UpdatedAt   time.Time `json:"updated_at"`           // 更新时间
}

// DistributorListRequest 分销商列表请求参数
type DistributorListRequest struct {
	Page     int    `form:"page" binding:"required,min=1"`      // 页码
	PageSize int    `form:"page_size" binding:"required,min=1"` // 每页数量
	Status   string `form:"status"`                             // 状态
}

// DistributorListResponse 分销商列表响应
type DistributorListResponse struct {
	Total int64         `json:"total"` // 总数
	List  []Distributor `json:"list"`  // 列表数据
}

// DistributorStatistics 分销商统计信息
type DistributorStatistics struct {
	TotalOrders     int64   `json:"total_orders"`     // 总订单数
	TotalAmount     float64 `json:"total_amount"`     // 总金额
	TotalCommission float64 `json:"total_commission"` // 总佣金
	MonthOrders     int64   `json:"month_orders"`     // 本月订单数
	MonthAmount     float64 `json:"month_amount"`     // 本月金额
	MonthCommission float64 `json:"month_commission"` // 本月佣金
}
