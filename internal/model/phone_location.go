package model

import "time"

// PhoneLocation 手机归属地模型
type PhoneLocation struct {
	ID          int64     `gorm:"primaryKey" json:"id"`
	PhoneNumber string    `gorm:"unique;not null;size:20" json:"phone_number"`
	Province    string    `gorm:"not null;size:50" json:"province"`
	City        string    `gorm:"not null;size:50" json:"city"`
	ISP         string    `gorm:"not null;size:50" json:"isp"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// PhoneLocationListRequest 手机归属地列表请求
type PhoneLocationListRequest struct {
	Page     int    `form:"page" binding:"required,min=1"`      // 页码
	PageSize int    `form:"page_size" binding:"required,min=1"` // 每页数量
	Phone    string `form:"phone"`                              // 手机号
	Province string `form:"province"`                           // 省份
	City     string `form:"city"`                               // 城市
	ISP      string `form:"isp"`                                // 运营商
}

// PhoneLocationListResponse 手机归属地列表响应
type PhoneLocationListResponse struct {
	Total int64           `json:"total"` // 总数
	Items []PhoneLocation `json:"items"` // 列表
}
