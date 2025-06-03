package model

import (
	"time"
)

type Permission struct {
	ID          int64        `json:"id" gorm:"primaryKey"`
	Code        string       `json:"code" gorm:"uniqueIndex;size:50"`
	Name        string       `json:"name" gorm:"size:50"`
	Type        string       `json:"type" gorm:"size:20"` // MENU or BUTTON
	ParentID    *int64       `json:"parentId" gorm:"index"`
	Path        string       `json:"path" gorm:"size:200"`
	Component   string       `json:"component" gorm:"size:200"`
	Icon        string       `json:"icon" gorm:"size:50"`
	Layout      string       `json:"layout" gorm:"size:50"`
	Method      string       `json:"method" gorm:"size:10"`
	Description string       `json:"description" gorm:"size:200"`
	Show        int          `json:"show" gorm:"default:1"`
	Enable      int          `json:"enable" gorm:"default:1"`
	Order       int          `json:"order" gorm:"default:0"`
	KeepAlive   int          `json:"keepAlive" gorm:"default:0"`
	Redirect    string       `json:"redirect" gorm:"size:200"`
	CreatedAt   time.Time    `json:"createdAt"`
	UpdatedAt   time.Time    `json:"updatedAt"`
	DeletedAt   *time.Time   `json:"deletedAt" gorm:"index"`
	Children    []Permission `json:"children" gorm:"-"`
}

// PermissionTree 用于构建权限树
type PermissionTree struct {
	ID        int64             `json:"id"`
	Code      string            `json:"code"`
	Name      string            `json:"name"`
	Type      string            `json:"type"`
	ParentID  *int64            `json:"parentId"`
	Path      string            `json:"path"`
	Component string            `json:"component"`
	Icon      string            `json:"icon"`
	Layout    string            `json:"layout"`
	Method    string            `json:"method"`
	Show      int               `json:"show"`
	Enable    int               `json:"enable"`
	Order     int               `json:"order"`
	KeepAlive int               `json:"keepAlive"`
	Redirect  string            `json:"redirect"`
	Children  []*PermissionTree `json:"children"`
}

// PermissionRequest 创建/更新权限的请求
type PermissionRequest struct {
	Code        string `json:"code" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Type        string `json:"type" binding:"required,oneof=MENU BUTTON"`
	ParentID    *int64 `json:"parentId"`
	Path        string `json:"path"`
	Component   string `json:"component"`
	Icon        string `json:"icon"`
	Layout      string `json:"layout"`
	Method      string `json:"method"`
	Description string `json:"description"`
	Show        int    `json:"show"`
	Enable      int    `json:"enable"`
	Order       int    `json:"order"`
	KeepAlive   int    `json:"keepAlive"`
	Redirect    string `json:"redirect"`
}

// PermissionResponse 权限响应
type PermissionResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type PermissionListRequest struct {
	Current  int    `json:"current" form:"current"`
	Size     int    `json:"size" form:"size"`
	Code     string `json:"code" form:"code"`
	Name     string `json:"name" form:"name"`
	Type     string `json:"type" form:"type"`
	ParentID int64  `json:"parentId" form:"parentId"`
	Status   int    `json:"status" form:"status"`
}

type PermissionListResponse struct {
	List  []Permission `json:"list"`
	Total int64        `json:"total"`
}
