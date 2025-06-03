package service

import (
	"context"
	"errors"
	"recharge-go/internal/model"
	"recharge-go/internal/repository"
	"time"
)

type UserTagService struct {
	userTagRepo         *repository.UserTagRepository
	userTagRelationRepo *repository.UserTagRelationRepository
}

func NewUserTagService(
	userTagRepo *repository.UserTagRepository,
	userTagRelationRepo *repository.UserTagRelationRepository,
) *UserTagService {
	return &UserTagService{
		userTagRepo:         userTagRepo,
		userTagRelationRepo: userTagRelationRepo,
	}
}

// CreateTag 创建用户标签
func (s *UserTagService) CreateTag(ctx context.Context, tag *model.UserTag) error {
	tag.CreatedAt = time.Now()
	tag.UpdatedAt = time.Now()
	return s.userTagRepo.Create(ctx, tag)
}

// UpdateTag 更新用户标签
func (s *UserTagService) UpdateTag(ctx context.Context, tag *model.UserTag) error {
	existingTag, err := s.userTagRepo.GetByID(ctx, tag.ID)
	if err != nil {
		return err
	}

	tag.CreatedAt = existingTag.CreatedAt
	tag.UpdatedAt = time.Now()
	return s.userTagRepo.Update(ctx, tag)
}

// DeleteTag 删除用户标签
func (s *UserTagService) DeleteTag(ctx context.Context, id int64) error {
	return s.userTagRepo.Delete(ctx, id)
}

// GetTag 获取用户标签
func (s *UserTagService) GetTag(ctx context.Context, id int64) (*model.UserTag, error) {
	return s.userTagRepo.GetByID(ctx, id)
}

// ListTags 获取用户标签列表
func (s *UserTagService) ListTags(ctx context.Context) ([]model.UserTag, error) {
	return s.userTagRepo.List(ctx)
}

// AssignUserTag 分配用户标签
func (s *UserTagService) AssignUserTag(ctx context.Context, userID, tagID int64) error {
	// 检查标签是否存在
	_, err := s.userTagRepo.GetByID(ctx, tagID)
	if err != nil {
		return errors.New("标签不存在")
	}

	// 创建标签关系
	relation := &model.UserTagRelation{
		UserID:    userID,
		TagID:     tagID,
		CreatedAt: time.Now(),
	}

	return s.userTagRelationRepo.Create(ctx, relation)
}

// RemoveUserTag 移除用户标签
func (s *UserTagService) RemoveUserTag(ctx context.Context, userID, tagID int64) error {
	return s.userTagRelationRepo.Delete(ctx, userID, tagID)
}

// GetUserTags 获取用户的所有标签
func (s *UserTagService) GetUserTags(ctx context.Context, userID int64) ([]model.UserTag, error) {
	return s.userTagRelationRepo.GetUserTags(ctx, userID)
}

// UpdateTagStatus 更新标签状态
func (s *UserTagService) UpdateTagStatus(ctx context.Context, id int64, status int) error {
	tag, err := s.userTagRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	tag.Status = status
	tag.UpdatedAt = time.Now()
	return s.userTagRepo.Update(ctx, tag)
}
