package migrations

import (
	"recharge-go/internal/model"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func InitTestUser(db *gorm.DB) error {
	// 检查是否已存在测试用户
	var count int64
	if err := db.Model(&model.User{}).Where("username = ?", "test").Count(&count).Error; err != nil {
		return err
	}

	// 如果不存在测试用户，则创建
	if count == 0 {
		// 加密密码
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte("test123"), bcrypt.DefaultCost)
		if err != nil {
			return err
		}

		user := &model.User{
			Username:  "test",
			Password:  string(hashedPassword),
			Nickname:  "测试用户",
			Phone:     "13800138000",
			Email:     "test@example.com",
			Status:    1,
			LastLogin: time.Now(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := db.Create(user).Error; err != nil {
			return err
		}
	}

	return nil
}
