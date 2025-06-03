package migrations

import (
	"recharge-go/internal/model"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func InitAdmin(db *gorm.DB) error {
	// 检查是否已存在管理员账号
	var count int64
	if err := db.Model(&model.User{}).Where("username = ?", "admin").Count(&count).Error; err != nil {
		return err
	}

	// 如果不存在管理员账号，则创建
	if count == 0 {
		// 加密密码
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
		if err != nil {
			return err
		}

		// 创建管理员用户
		admin := &model.User{
			Username:  "admin",
			Password:  string(hashedPassword),
			Nickname:  "超级管理员",
			Phone:     "13800138001",
			Email:     "admin@example.com",
			Status:    1,
			LastLogin: time.Now(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := db.Create(admin).Error; err != nil {
			return err
		}

		// 查找超级管理员角色
		var role model.Role
		if err := db.Where("code = ?", "SUPER_ADMIN").First(&role).Error; err != nil {
			return err
		}

		// 创建用户角色关联
		userRole := &model.UserRole{
			UserID:    admin.ID,
			RoleID:    role.ID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := db.Create(userRole).Error; err != nil {
			return err
		}
	}

	return nil
}
