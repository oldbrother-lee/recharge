package model

import "time"

// ProductTypeCategory 产品类型分类
type ProductTypeCategory struct {
	ID        int64     `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"column:cname;not null;size:255" json:"name"`   // 分类名称
	StatusTip string    `gorm:"column:pla_mobile;size:255" json:"status_tip"` // 充值状态提示
	Icon      string    `gorm:"column:cicon;size:255" json:"icon"`            // 分类图标
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`             // 创建时间
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`             // 更新时间
}

// TableName 指定表名
func (ProductTypeCategory) TableName() string {
	return "product_type_categories"
}

// ProductType 产品类型
type ProductType struct {
	ID          int64                `gorm:"primaryKey" json:"id"`
	TypeName    string               `gorm:"column:type_name;not null;size:255" json:"type_name"`        // 类型名称
	TypecID     int64                `gorm:"column:typec_id;not null;default:4" json:"typec_id"`         // 类型分类ID
	Status      int                  `gorm:"not null;default:1" json:"status"`                           // 状态：1-上架 0-下架
	Sort        int                  `gorm:"not null;default:100" json:"sort"`                           // 排序
	AccountType int                  `gorm:"column:account_type;not null;default:1" json:"account_type"` // 充值账号类型
	TishiDoc    string               `gorm:"column:tishidoc;type:longtext" json:"tishi_doc"`             // 提示文档
	Icon        string               `gorm:"size:255" json:"icon"`                                       // 图标
	CreatedAt   time.Time            `gorm:"autoCreateTime" json:"created_at"`                           // 创建时间
	UpdatedAt   time.Time            `gorm:"autoUpdateTime" json:"updated_at"`                           // 更新时间
	Category    *ProductTypeCategory `gorm:"foreignKey:TypecID" json:"category,omitempty"`               // 关联的类型分类
}

// TableName 指定表名
func (ProductType) TableName() string {
	return "product_types"
}

// ProductTypeListRequest 产品类型列表请求
type ProductTypeListRequest struct {
	Page        int    `form:"page" binding:"required,min=1"`      // 页码
	PageSize    int    `form:"page_size" binding:"required,min=1"` // 每页数量
	TypeName    string `form:"type_name"`                          // 类型名称
	TypecID     *int64 `form:"typec_id"`                           // 类型分类ID
	Status      *int   `form:"status"`                             // 状态
	AccountType *int   `form:"account_type"`                       // 充值账号类型
}

// ProductTypeListResponse 产品类型列表响应
type ProductTypeListResponse struct {
	Total int64         `json:"total"` // 总数
	Items []ProductType `json:"items"` // 列表
}

// ProductTypeCreateRequest 创建产品类型请求
type ProductTypeCreateRequest struct {
	TypeName    string `json:"type_name" binding:"required,max=255"` // 类型名称
	TypecID     int64  `json:"typec_id" binding:"required"`          // 类型分类ID
	Status      int    `json:"status" binding:"oneof=0 1"`           // 状态
	Sort        int    `json:"sort" binding:"min=0"`                 // 排序
	AccountType int    `json:"account_type" binding:"min=1"`         // 充值账号类型
	TishiDoc    string `json:"tishi_doc"`                            // 提示文档
	Icon        string `json:"icon" binding:"omitempty,max=255"`     // 图标
}

// ProductTypeUpdateRequest 更新产品类型请求
type ProductTypeUpdateRequest struct {
	ID          int64  `json:"-"`                                    // ID从路由参数获取
	TypeName    string `json:"type_name" binding:"required,max=255"` // 类型名称
	TypecID     int64  `json:"typec_id" binding:"required"`          // 类型分类ID
	Status      int    `json:"status" binding:"oneof=0 1"`           // 状态
	Sort        int    `json:"sort" binding:"min=0"`                 // 排序
	AccountType int    `json:"account_type" binding:"min=1"`         // 充值账号类型
	TishiDoc    string `json:"tishi_doc"`                            // 提示文档
	Icon        string `json:"icon" binding:"omitempty,max=255"`     // 图标
}

// ProductTypeCategoryListRequest 产品类型分类列表请求
type ProductTypeCategoryListRequest struct {
	Page     int    `form:"page" binding:"required,min=1"`      // 页码
	PageSize int    `form:"page_size" binding:"required,min=1"` // 每页数量
	Name     string `form:"name"`                               // 分类名称
}

// ProductTypeCategoryListResponse 产品类型分类列表响应
type ProductTypeCategoryListResponse struct {
	Total int64                 `json:"total"` // 总数
	Items []ProductTypeCategory `json:"items"` // 列表
}
