package repository

import (
	"recharge-go/internal/repository/notification"

	"gorm.io/gorm"
)

// NewNotificationRepository 创建通知记录仓库实例
func NewNotificationRepository(db *gorm.DB) notification.Repository {
	return notification.NewRepository(db)
}
