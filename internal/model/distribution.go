package model

import (
	"time"
)

// DistributionGrade 分销等级模型
type DistributionGrade struct {
	ID          int64     `json:"id" gorm:"primaryKey;type:bigint;not null"`
	Name        string    `json:"name" gorm:"size:50;not null"`
	Description string    `json:"description" gorm:"size:255"`
	Icon        string    `json:"icon" gorm:"size:255"`
	MinPoints   int64     `json:"min_points" gorm:"type:bigint;default:0"`
	Commission  float64   `json:"commission" gorm:"type:decimal(10,2);default:0.00;comment:佣金比例"`
	Status      int       `json:"status" gorm:"type:tinyint;default:1"`
	CreatedAt   time.Time `json:"created_at" gorm:"type:datetime;autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"type:datetime;autoUpdateTime"`
}

// TableName 指定表名
func (DistributionGrade) TableName() string {
	return "distribution_grades"
}

// DistributionRule 分销规则模型
type DistributionRule struct {
	ID          int64     `json:"id" gorm:"primaryKey;type:bigint;not null"`
	GradeID     int64     `json:"grade_id" gorm:"type:bigint;not null;comment:分销等级ID"`
	ProductType int       `json:"product_type" gorm:"type:tinyint;not null;comment:产品类型"`
	Commission  float64   `json:"commission" gorm:"type:decimal(10,2);default:0.00;comment:佣金比例"`
	MinAmount   float64   `json:"min_amount" gorm:"type:decimal(10,2);default:0.00;comment:最低金额"`
	MaxAmount   float64   `json:"max_amount" gorm:"type:decimal(10,2);default:0.00;comment:最高金额"`
	Status      int       `json:"status" gorm:"type:tinyint;default:1"`
	CreatedAt   time.Time `json:"created_at" gorm:"type:datetime;autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"type:datetime;autoUpdateTime"`
}

// TableName 指定表名
func (DistributionRule) TableName() string {
	return "distribution_rules"
}

// DistributionCommission 分销佣金记录模型
type DistributionCommission struct {
	ID        int64     `json:"id" gorm:"primaryKey;type:bigint;not null"`
	UserID    int64     `json:"user_id" gorm:"type:bigint;not null;comment:用户ID"`
	OrderID   int64     `json:"order_id" gorm:"type:bigint;not null;comment:订单ID"`
	Amount    float64   `json:"amount" gorm:"type:decimal(10,2);default:0.00;comment:佣金金额"`
	Status    int       `json:"status" gorm:"type:tinyint;default:0;comment:状态(0:待结算 1:已结算 2:已取消)"`
	CreatedAt time.Time `json:"created_at" gorm:"type:datetime;autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"type:datetime;autoUpdateTime"`
}

// TableName 指定表名
func (DistributionCommission) TableName() string {
	return "distribution_commissions"
}

// DistributionWithdrawal 分销提现记录模型
type DistributionWithdrawal struct {
	ID        int64     `json:"id" gorm:"primaryKey;type:bigint;not null"`
	UserID    int64     `json:"user_id" gorm:"type:bigint;not null;comment:用户ID"`
	Amount    float64   `json:"amount" gorm:"type:decimal(10,2);default:0.00;comment:提现金额"`
	Status    int       `json:"status" gorm:"type:tinyint;default:0;comment:状态(0:待审核 1:已通过 2:已拒绝)"`
	Remark    string    `json:"remark" gorm:"size:255;comment:备注"`
	CreatedAt time.Time `json:"created_at" gorm:"type:datetime;autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"type:datetime;autoUpdateTime"`
}

// TableName 指定表名
func (DistributionWithdrawal) TableName() string {
	return "distribution_withdrawals"
}

// DistributionStatistics 分销统计模型
type DistributionStatistics struct {
	ID              int64     `json:"id" gorm:"primaryKey;type:bigint;not null"`
	UserID          int64     `json:"user_id" gorm:"type:bigint;not null;comment:用户ID"`
	TotalOrders     int64     `json:"total_orders" gorm:"type:bigint;default:0;comment:总订单数"`
	TotalAmount     float64   `json:"total_amount" gorm:"type:decimal(10,2);default:0.00;comment:总金额"`
	TotalCommission float64   `json:"total_commission" gorm:"type:decimal(10,2);default:0.00;comment:总佣金"`
	CreatedAt       time.Time `json:"created_at" gorm:"type:datetime;autoCreateTime"`
	UpdatedAt       time.Time `json:"updated_at" gorm:"type:datetime;autoUpdateTime"`
}

// TableName 指定表名
func (DistributionStatistics) TableName() string {
	return "distribution_statistics"
}
