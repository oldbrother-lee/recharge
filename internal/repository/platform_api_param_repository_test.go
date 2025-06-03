package repository

import (
	"context"
	"testing"
	"time"

	"recharge-go/internal/model"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	// 迁移数据库
	err = db.AutoMigrate(&model.PlatformAPIParam{})
	if err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	return db
}

func TestPlatformAPIParamRepository(t *testing.T) {
	db := setupTestDB(t)
	repo := NewPlatformAPIParamRepository(db)
	ctx := context.Background()

	// 测试数据
	param := &model.PlatformAPIParam{
		APIID:       1,
		Name:        "测试参数",
		Code:        "test_param",
		Value:       "test_value",
		Description: "测试描述",
		Cost:        10.5,
		MinCost:     10.0,
		MaxCost:     11.0,
		Status:      1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// 测试创建
	t.Run("Create", func(t *testing.T) {
		err := repo.Create(ctx, param)
		assert.NoError(t, err)
		assert.NotZero(t, param.ID)
	})

	// 测试获取
	t.Run("GetByID", func(t *testing.T) {
		got, err := repo.GetByID(ctx, param.ID)
		assert.NoError(t, err)
		assert.NotNil(t, got)
		assert.Equal(t, param.Name, got.Name)
		assert.Equal(t, param.Code, got.Code)
	})

	// 测试更新
	t.Run("Update", func(t *testing.T) {
		param.Name = "更新后的参数"
		err := repo.Update(ctx, param)
		assert.NoError(t, err)

		got, err := repo.GetByID(ctx, param.ID)
		assert.NoError(t, err)
		assert.Equal(t, "更新后的参数", got.Name)
	})

	// 测试列表
	t.Run("List", func(t *testing.T) {
		params, total, err := repo.List(ctx, 1, 1, 10)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.Len(t, params, 1)
	})

	// 测试删除
	t.Run("Delete", func(t *testing.T) {
		err := repo.Delete(ctx, param.ID)
		assert.NoError(t, err)

		got, err := repo.GetByID(ctx, param.ID)
		assert.NoError(t, err)
		assert.Nil(t, got)
	})
}
