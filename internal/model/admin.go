package model

import (
	"time"
)

// Admin 管理员
type Admin struct {
	ID        int64     `json:"id" gorm:"primaryKey;type:bigint;not null"`
	Username  string    `json:"username" gorm:"size:50;not null;unique"`
	Password  string    `json:"-" gorm:"size:100;not null"`
	Nickname  string    `json:"nickname" gorm:"size:50"`
	Status    int       `json:"status" gorm:"type:bigint;default:1"`
	CreatedAt time.Time `json:"created_at" gorm:"type:datetime;autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"type:datetime;autoUpdateTime"`
}

// TableName 指定表名
func (Admin) TableName() string {
	return "admins"
}
