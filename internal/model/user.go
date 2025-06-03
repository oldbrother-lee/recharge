package model

import (
	"time"
)

// User 用户模型
type User struct {
	ID        int64     `json:"id" gorm:"primaryKey;type:bigint;not null"`
	Username  string    `json:"username" gorm:"size:50;not null;unique"`
	Password  string    `json:"-" gorm:"size:100;not null"`
	Nickname  string    `json:"nickname" gorm:"size:50"`
	Phone     string    `json:"phone" gorm:"size:20"`
	Email     string    `json:"email" gorm:"size:100"`
	Avatar    string    `json:"avatar" gorm:"size:255"`
	Type      int       `json:"type" gorm:"type:tinyint;default:1;comment:用户类型(1:普通用户 2:代理商 3:管理员)"`
	Gender    int       `json:"gender" gorm:"type:tinyint;default:0;comment:性别(0:未知 1:男 2:女)"`
	Credit    float64   `json:"credit" gorm:"type:decimal(10,2);default:0.00;comment:授信额度"`
	Balance   float64   `json:"balance" gorm:"type:decimal(10,2);default:0.00;comment:余额"`
	Status    int       `json:"status" gorm:"type:bigint;default:1"`
	LastLogin time.Time `json:"last_login" gorm:"type:datetime"`
	CreatedAt time.Time `json:"created_at" gorm:"type:datetime;autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"type:datetime;autoUpdateTime"`
}

// TableName 指定表名
func (User) TableName() string {
	return "users"
}

// UserGrade 用户等级模型
type UserGrade struct {
	ID          int64     `json:"id" gorm:"primaryKey;type:bigint;not null"`
	Name        string    `json:"name" gorm:"size:50;not null"`
	Description string    `json:"description" gorm:"size:255"`
	Icon        string    `json:"icon" gorm:"size:255"`
	GradeType   int       `json:"grade_type" gorm:"type:tinyint;default:1;comment:等级类型"`
	Status      int       `json:"status" gorm:"type:tinyint;default:1"`
	CreatedAt   time.Time `json:"created_at" gorm:"type:datetime;autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"type:datetime;autoUpdateTime"`
}

// TableName 指定表名
func (UserGrade) TableName() string {
	return "user_grades"
}

// UserTag 用户标签模型
type UserTag struct {
	ID        int64     `json:"id" gorm:"primaryKey;type:bigint;not null"`
	Name      string    `json:"name" gorm:"size:50;not null"`
	Category  string    `json:"category" gorm:"size:50"`
	Color     string    `json:"color" gorm:"size:20"`
	Status    int       `json:"status" gorm:"type:tinyint;default:1"`
	CreatedAt time.Time `json:"created_at" gorm:"type:datetime;autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"type:datetime;autoUpdateTime"`
}

// TableName 指定表名
func (UserTag) TableName() string {
	return "user_tags"
}

// UserTagRelation 用户标签关系模型
type UserTagRelation struct {
	ID        int64     `json:"id" gorm:"primaryKey;type:bigint;not null"`
	UserID    int64     `json:"user_id" gorm:"type:bigint;not null"`
	TagID     int64     `json:"tag_id" gorm:"type:bigint;not null"`
	CreatedAt time.Time `json:"created_at" gorm:"type:datetime;autoCreateTime"`
}

// TableName 指定表名
func (UserTagRelation) TableName() string {
	return "user_tag_relations"
}

// UserGradeRelation 用户等级关系模型
type UserGradeRelation struct {
	ID        int64     `json:"id" gorm:"primaryKey;type:bigint;not null"`
	UserID    int64     `json:"user_id" gorm:"type:bigint;not null"`
	GradeID   int64     `json:"grade_id" gorm:"type:bigint;not null"`
	CreatedAt time.Time `json:"created_at" gorm:"type:datetime;autoCreateTime"`
}

// TableName 指定表名
func (UserGradeRelation) TableName() string {
	return "user_grade_relations"
}

// Request and Response structures
type UserLoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type UserLoginResponse struct {
	Token        string   `json:"token"`
	RefreshToken string   `json:"refreshToken"`
	UserInfo     UserInfo `json:"userInfo"`
}

type UserInfo struct {
	UserId   string   `json:"userId"`
	Username string   `json:"userName"`
	Roles    []string `json:"roles"`
	Buttons  []string `json:"buttons"`
}

type UserRegisterRequest struct {
	Username string  `json:"username" binding:"required"`
	Password string  `json:"password" binding:"required"`
	Nickname *string `json:"nickname"`
	Email    *string `json:"email"`
	Phone    *string `json:"phone"`
}

type UserUpdateRequest struct {
	Username *string `json:"username"`
	Phone    *string `json:"phone"`
	Status   *int    `json:"status"`
}

type UserChangePasswordRequest struct {
	OldPassword string `json:"oldPassword" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required"`
}

type UserListRequest struct {
	Current  int    `json:"current" form:"current"`
	Size     int    `json:"size" form:"size"`
	Username string `json:"username" form:"username"`
	Phone    string `json:"phone" form:"phone"`
	Email    string `json:"email" form:"email"`
	Status   int    `json:"status" form:"status"`
}

// OrderUpgrade 付费升级订单模型
type OrderUpgrade struct {
	ID           int64     `json:"id" gorm:"primaryKey;type:bigint;not null"`
	OrderNumber  string    `json:"order_number" gorm:"size:50;not null;unique"`
	UserID       int64     `json:"user_id" gorm:"type:bigint;not null"`
	GradeID      int64     `json:"grade_id" gorm:"type:bigint;not null"`
	TotalPrice   float64   `json:"total_price" gorm:"type:decimal(10,2);default:0.00"`
	PayWay       int       `json:"pay_way" gorm:"type:tinyint;default:0"`
	SerialNumber string    `json:"serial_number" gorm:"size:100"`
	IsPay        int       `json:"is_pay" gorm:"type:tinyint;default:0"`
	PayTime      time.Time `json:"pay_time" gorm:"type:datetime"`
	IsRebate     int       `json:"is_rebate" gorm:"type:tinyint;default:0"`
	RebatePrice  float64   `json:"rebate_price" gorm:"type:decimal(10,2);default:0.00"`
	RebateID     int64     `json:"rebate_id" gorm:"type:bigint;default:0"`
	RewardPrice  float64   `json:"reward_price" gorm:"type:decimal(10,2);default:0.00"`
	Body         string    `json:"body" gorm:"size:255"`
	CreatedAt    time.Time `json:"created_at" gorm:"type:datetime;autoCreateTime"`
	UpdatedAt    time.Time `json:"updated_at" gorm:"type:datetime;autoUpdateTime"`
}

// TableName 指定表名
func (OrderUpgrade) TableName() string {
	return "order_upgrades"
}

// Rebate 返利模型
type Rebate struct {
	ID        int64     `json:"id" gorm:"primaryKey;type:bigint;not null"`
	UserID    int64     `json:"user_id" gorm:"type:bigint;not null"`
	OrderID   int64     `json:"order_id" gorm:"type:bigint;not null"`
	Amount    float64   `json:"amount" gorm:"type:decimal(10,2);default:0.00"`
	Status    int       `json:"status" gorm:"type:tinyint;default:0;comment:0:待发放 1:已发放 2:已取消"`
	Remark    string    `json:"remark" gorm:"size:255"`
	CreatedAt time.Time `json:"created_at" gorm:"type:datetime;autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"type:datetime;autoUpdateTime"`
}

// TableName 指定表名
func (Rebate) TableName() string {
	return "rebates"
}

// Reward 奖励模型
type Reward struct {
	ID        int64     `json:"id" gorm:"primaryKey;type:bigint;not null"`
	UserID    int64     `json:"user_id" gorm:"type:bigint;not null"`
	OrderID   int64     `json:"order_id" gorm:"type:bigint;not null"`
	Amount    float64   `json:"amount" gorm:"type:decimal(10,2);default:0.00"`
	Type      int       `json:"type" gorm:"type:tinyint;default:1;comment:1:升级奖励 2:推荐奖励"`
	Status    int       `json:"status" gorm:"type:tinyint;default:0;comment:0:待发放 1:已发放 2:已取消"`
	Remark    string    `json:"remark" gorm:"size:255"`
	CreatedAt time.Time `json:"created_at" gorm:"type:datetime;autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"type:datetime;autoUpdateTime"`
}

// TableName 指定表名
func (Reward) TableName() string {
	return "rewards"
}

// UserResponse 用户响应结构
type UserResponse struct {
	ID          int64     `json:"id"`
	Username    string    `json:"username"`
	Nickname    string    `json:"nickname"`
	Phone       string    `json:"phone"`
	Email       string    `json:"email"`
	Avatar      string    `json:"avatar"`
	Type        int       `json:"type"`
	Gender      int       `json:"gender"`
	Credit      float64   `json:"credit"`
	Status      int       `json:"status"`
	CreditLimit float64   `json:"credit_limit"`
	Balance     float64   `json:"balance"`
	LastLogin   time.Time `json:"last_login"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// UserListResponse 用户列表响应结构
type UserListResponse struct {
	List  []UserResponse `json:"list"`
	Total int64          `json:"total"`
}
