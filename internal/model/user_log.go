package model

import (
	"time"
)

// UserLog 用户日志模型
type UserLog struct {
	ID        int64     `json:"id" gorm:"primaryKey"`
	UserID    int64     `json:"user_id" gorm:"index"`
	Action    string    `json:"action" gorm:"size:50"`
	TargetID  int64     `json:"target_id" gorm:"index"`
	Content   string    `json:"content" gorm:"type:text"`
	IP        string    `json:"ip" gorm:"size:50"`
	CreatedAt time.Time `json:"created_at"`
}

// UserLogRequest 创建用户日志请求
type UserLogRequest struct {
	UserID   int64  `json:"user_id" binding:"required"`
	Action   string `json:"action" binding:"required"`
	TargetID int64  `json:"target_id"`
	Content  string `json:"content" binding:"required"`
	IP       string `json:"ip"`
}

// UserLogResponse 用户日志响应
type UserLogResponse struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Action    string    `json:"action"`
	TargetID  int64     `json:"target_id"`
	Content   string    `json:"content"`
	IP        string    `json:"ip"`
	CreatedAt time.Time `json:"created_at"`
}

// UserLogListRequest 用户日志列表请求
type UserLogListRequest struct {
	UserID   int64  `form:"user_id"`
	Action   string `form:"action"`
	TargetID int64  `form:"target_id"`
	Current  int    `form:"current" binding:"min=1"`
	Size     int    `form:"size" binding:"min=1,max=100"`
}

// UserLogListResponse 用户日志列表响应
type UserLogListResponse struct {
	List  []UserLogResponse `json:"list"`
	Total int64             `json:"total"`
}

// TableName 指定表名
func (UserLog) TableName() string {
	return "user_logs"
}

// UserLogAction 用户操作类型
const (
	UserLogActionCreate   = "create"   // 创建用户
	UserLogActionUpdate   = "update"   // 更新用户
	UserLogActionDelete   = "delete"   // 删除用户
	UserLogActionLogin    = "login"    // 用户登录
	UserLogActionLogout   = "logout"   // 用户登出
	UserLogActionPassword = "password" // 修改密码
	UserLogActionStatus   = "status"   // 修改状态
	UserLogActionCredit   = "credit"   // 修改授信额度
	UserLogActionType     = "type"     // 修改用户类型
	UserLogActionGrade    = "grade"    // 修改用户等级
	UserLogActionTag      = "tag"      // 修改用户标签
)
