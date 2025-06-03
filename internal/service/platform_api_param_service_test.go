package service

import (
	"context"
	"testing"

	"recharge-go/internal/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockPlatformAPIParamRepository 模拟仓库
type MockPlatformAPIParamRepository struct {
	mock.Mock
}

func (m *MockPlatformAPIParamRepository) Create(ctx context.Context, param *model.PlatformAPIParam) error {
	args := m.Called(ctx, param)
	return args.Error(0)
}

func (m *MockPlatformAPIParamRepository) Update(ctx context.Context, param *model.PlatformAPIParam) error {
	args := m.Called(ctx, param)
	return args.Error(0)
}

func (m *MockPlatformAPIParamRepository) Delete(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockPlatformAPIParamRepository) GetByID(ctx context.Context, id int64) (*model.PlatformAPIParam, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.PlatformAPIParam), args.Error(1)
}

func (m *MockPlatformAPIParamRepository) List(ctx context.Context, apiID int64, page, pageSize int) ([]*model.PlatformAPIParam, int64, error) {
	args := m.Called(ctx, apiID, page, pageSize)
	return args.Get(0).([]*model.PlatformAPIParam), args.Get(1).(int64), args.Error(2)
}

func TestPlatformAPIParamService(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockPlatformAPIParamRepository)
	service := NewPlatformAPIParamService(mockRepo)

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
	}

	// 测试创建
	t.Run("CreateParam", func(t *testing.T) {
		mockRepo.On("Create", ctx, param).Return(nil)
		err := service.CreateParam(ctx, param)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	// 测试创建 - 参数验证失败
	t.Run("CreateParam - Validation Failed", func(t *testing.T) {
		invalidParam := &model.PlatformAPIParam{
			APIID:   0,
			Name:    "",
			Code:    "",
			Cost:    -1,
			MinCost: -1,
			MaxCost: -1,
		}
		err := service.CreateParam(ctx, invalidParam)
		assert.Error(t, err)
	})

	// 测试更新
	t.Run("UpdateParam", func(t *testing.T) {
		param.ID = 1
		mockRepo.On("GetByID", ctx, param.ID).Return(param, nil)
		mockRepo.On("Update", ctx, param).Return(nil)
		err := service.UpdateParam(ctx, param)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	// 测试更新 - 参数不存在
	t.Run("UpdateParam - Not Found", func(t *testing.T) {
		param.ID = 999
		mockRepo.On("GetByID", ctx, param.ID).Return(nil, nil)
		err := service.UpdateParam(ctx, param)
		assert.Error(t, err)
		mockRepo.AssertExpectations(t)
	})

	// 测试删除
	t.Run("DeleteParam", func(t *testing.T) {
		param.ID = 1
		mockRepo.On("GetByID", ctx, param.ID).Return(param, nil)
		mockRepo.On("Delete", ctx, param.ID).Return(nil)
		err := service.DeleteParam(ctx, param.ID)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	// 测试获取
	t.Run("GetParam", func(t *testing.T) {
		param.ID = 1
		mockRepo.On("GetByID", ctx, param.ID).Return(param, nil)
		got, err := service.GetParam(ctx, param.ID)
		assert.NoError(t, err)
		assert.Equal(t, param, got)
		mockRepo.AssertExpectations(t)
	})

	// 测试列表
	t.Run("ListParams", func(t *testing.T) {
		params := []*model.PlatformAPIParam{param}
		mockRepo.On("List", ctx, int64(1), 1, 10).Return(params, int64(1), nil)
		got, total, err := service.ListParams(ctx, 1, 1, 10)
		assert.NoError(t, err)
		assert.Equal(t, params, got)
		assert.Equal(t, int64(1), total)
		mockRepo.AssertExpectations(t)
	})
}
