package model

import "time"

// UserRole represents the relationship between users and roles
type UserRole struct {
	ID        int64      `json:"id" gorm:"primaryKey"`
	UserID    int64      `json:"user_id" gorm:"index"`
	RoleID    int64      `json:"role_id" gorm:"index"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" gorm:"index"`
}

// UserWithRoles represents a user with their roles
type UserWithRoles struct {
	User  User   `json:"user"`
	Roles []Role `json:"roles"`
}
