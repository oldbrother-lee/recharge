package model

import "time"

// Role represents a role in the system
type Role struct {
	ID          int64      `json:"id" gorm:"primaryKey"`
	Name        string     `json:"name" gorm:"size:50;not null;uniqueIndex" binding:"required"`
	Code        string     `json:"code" gorm:"size:50;not null;uniqueIndex" binding:"required"`
	Description string     `json:"description" gorm:"size:200"`
	Status      int        `json:"status" gorm:"default:1"` // 1: enabled, 0: disabled
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	DeletedAt   *time.Time `json:"deletedAt" gorm:"index"`
}

// RoleRequest represents the request body for creating/updating a role
type RoleRequest struct {
	Name        string `json:"name" binding:"required"`
	Code        string `json:"code" binding:"required"`
	Description string `json:"description"`
	Status      int    `json:"status"`
}

// RolePermission represents the relationship between roles and permissions
type RolePermission struct {
	ID           int64      `json:"id" gorm:"primaryKey"`
	RoleID       int64      `json:"roleId" gorm:"index"`
	PermissionID int64      `json:"permissionId" gorm:"index"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
	DeletedAt    *time.Time `json:"deletedAt" gorm:"index"`
}

// RoleWithPermissions represents a role with its permissions
type RoleWithPermissions struct {
	Role        Role         `json:"role"`
	Permissions []Permission `json:"permissions"`
}

type RoleListRequest struct {
	Current int    `json:"current" form:"current"`
	Size    int    `json:"size" form:"size"`
	Code    string `json:"code" form:"code"`
	Name    string `json:"name" form:"name"`
	Status  int    `json:"status" form:"status"`
}

type RoleListResponse struct {
	List  []Role `json:"list"`
	Total int64  `json:"total"`
}
