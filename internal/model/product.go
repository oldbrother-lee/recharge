package model

import (
	"time"
)

// ProductCategory 商品分类
type ProductCategory struct {
	ID        int64     `json:"id" gorm:"primaryKey;type:bigint;not null"`
	Name      string    `json:"name" gorm:"size:100;not null"`
	Sort      int       `json:"sort" gorm:"type:tinyint;default:0"`
	Type      int       `json:"type" gorm:"type:tinyint;default:1"`
	CreatedAt time.Time `json:"created_at" gorm:"type:datetime;autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"type:datetime;autoUpdateTime"`
}

// Product 商品
type Product struct {
	ID              int64            `json:"id" gorm:"primaryKey;type:bigint"`                                             // 主键ID
	Name            string           `json:"name" gorm:"size:100;not null"`                                                // 商品名称
	Description     string           `json:"description" gorm:"size:500;comment:商品描述"`                                     // 商品描述
	Price           float64          `json:"price" gorm:"type:decimal(10,2);not null;comment:价格"`                          // 商品价格
	Type            int64            `json:"type" gorm:"column:type;type:bigint;default:1;comment:'1话费 2流量'"`              // 商品类型ID
	ISP             string           `json:"isp" gorm:"size:50;default:1,2,3;comment:'支持运营商:1移动 2电信 3联通'"`                 // 运营商
	Status          int              `json:"status" gorm:"type:bigint;default:1;COMMENT:'是否上架'"`                           // 状态：1-启用，0-禁用
	Sort            int              `json:"sort" gorm:"type:bigint;default:0"`                                            // 排序权重
	APIEEnabled     bool             `json:"api_enabled" gorm:"default:false;COMMENT:'是否开启接口充值'"`                          // API是否启用
	Remark          string           `json:"remark" gorm:"size:500;COMMENT:'备注信息'"`                                        // 备注信息
	CategoryID      int64            `json:"category_id" gorm:"type:bigint;not null;COMMENT:'分类ID'"`                       // 分类ID
	OperatorTag     string           `json:"operator_tag" gorm:"size:50;COMMENT:'运营商标签'"`                                  // 运营商标签
	MaxPrice        float64          `json:"max_price" gorm:"type:decimal(10,2);COMMENT:'最高价格'"`                           // 最高价格
	VoucherPrice    string           `json:"voucher_price" gorm:"type:varchar(20);COMMENT:'代金券价格'"`                        // 代金券价格
	VoucherName     string           `json:"voucher_name" gorm:"size:100;COMMENT:'代金券名称'"`                                 // 代金券名称
	ShowStyle       int              `json:"show_style" gorm:"type:bigint;default:1;COMMENT:'显示类型：1:全部显示2:客户端3:代理端'"`      // 显示样式
	APIFailStyle    int              `json:"api_fail_style" gorm:"type:bigint;default:1;COMMENT:'api失败处理方式,1直接失败，2回到待充值'"` // API失败处理方式
	AllowProvinces  string           `json:"allow_provinces" gorm:"size:500"`                                              // 允许的省份
	AllowCities     string           `json:"allow_cities" gorm:"size:500"`                                                 // 允许的城市
	ForbidProvinces string           `json:"forbid_provinces" gorm:"size:500"`                                             // 禁止的省份
	ForbidCities    string           `json:"forbid_cities" gorm:"size:500"`                                                // 禁止的城市
	APIDelay        string           `json:"api_delay" gorm:"type:varchar(20)"`                                            // API延迟时间
	GradeIDs        string           `json:"grade_ids" gorm:"size:500"`                                                    // 等级ID列表
	APIID           int64            `json:"api_id" gorm:"type:bigint;comment:接码接口ID"`                                     // API接口ID
	APIParamID      int64            `json:"api_param_id" gorm:"type:bigint;comment:接码接口参数ID"`                             // API参数ID
	IsApi           bool             `json:"is_api" gorm:"default:false;comment:是否接码"`                                     // 是否需要解码
	CreatedAt       time.Time        `json:"created_at" gorm:"type:datetime;autoCreateTime"`                               // 创建时间
	UpdatedAt       time.Time        `json:"updated_at" gorm:"type:datetime;autoUpdateTime"`                               // 更新时间
	ProductType     *ProductType     `json:"product_type,omitempty" gorm:"foreignKey:Type;references:ID"`                  // 关联的商品类型
	Category        *ProductCategory `json:"category,omitempty" gorm:"foreignKey:CategoryID;references:ID"`                // 关联的商品分类
}

// TableName 指定表名
func (Product) TableName() string {
	return "products"
}

// ProductSpec 商品规格
type ProductSpec struct {
	ID        int64     `json:"id" gorm:"primaryKey;type:bigint"`
	ProductID int64     `json:"product_id" gorm:"type:bigint;not null"`
	Name      string    `json:"name" gorm:"size:100;not null"`
	Value     string    `json:"value" gorm:"size:100"`
	Price     float64   `json:"price" gorm:"type:decimal(10,2);default:0"`
	Stock     int       `json:"stock" gorm:"type:bigint;default:0"`
	Sort      int       `json:"sort" gorm:"type:bigint;default:0"`
	CreatedAt time.Time `json:"created_at" gorm:"type:datetime;autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"type:datetime;autoUpdateTime"`
}

// MemberGrade 会员等级
type MemberGrade struct {
	ID        int64     `json:"id" gorm:"primaryKey;type:bigint"`
	Name      string    `json:"name" gorm:"size:100;not null"`
	GradeType int       `json:"grade_type" gorm:"type:bigint;default:1"`
	Sort      int       `json:"sort" gorm:"type:bigint;default:0"`
	CreatedAt time.Time `json:"created_at" gorm:"type:datetime;autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"type:datetime;autoUpdateTime"`
}

// ProductGradePrice 商品会员价格
type ProductGradePrice struct {
	ID        int64     `json:"id" gorm:"primaryKey;type:bigint"`
	ProductID int64     `json:"product_id" gorm:"type:bigint;not null"`
	GradeID   int64     `json:"grade_id" gorm:"type:bigint;not null"`
	Price     float64   `json:"price" gorm:"type:decimal(10,2);not null"`
	CreatedAt time.Time `json:"created_at" gorm:"type:datetime;autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"type:datetime;autoUpdateTime"`
}

// ProductAPIRelation 商品API关联
type ProductAPIRelation struct {
	ID          int64     `json:"id" gorm:"primaryKey;type:bigint"`
	ProductID   int64     `json:"product_id" gorm:"type:bigint;not null;comment:商品ID"`
	APIID       int64     `json:"api_id" gorm:"type:bigint;not null;comment:API接口ID"`
	ParamID     int64     `json:"param_id" gorm:"type:bigint;not null;comment:API参数ID"`
	Sort        int       `json:"sort" gorm:"type:bigint;default:0;comment:排序"`
	Status      int       `json:"status" gorm:"type:bigint;default:1;comment:状态：1-启用，0-禁用"`
	RetryNum    int       `json:"retry_num" gorm:"type:bigint;default:0;comment:重试次数"`
	ISP         string    `json:"isp" gorm:"size:50;default:1,2,3;comment:支持运营商:1移动 2电信 3联通"`
	ProductName string    `json:"product_name"`
	APIName     string    `json:"api_name"`
	CreatedAt   time.Time `json:"created_at" gorm:"type:datetime;autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"type:datetime;autoUpdateTime"`
}

// TableName 指定表名
func (ProductAPIRelation) TableName() string {
	return "product_api_relations"
}

// ProductCategoryCreateRequest 创建商品分类请求
type ProductCategoryCreateRequest struct {
	Name string `json:"name" binding:"required"`
	Sort int    `json:"sort" binding:"required"`
	Type int    `json:"type" binding:"required"`
}
