package service

import (
	"context"
	"errors"
	"recharge-go/internal/model"
	"recharge-go/internal/repository"
	"time"
)

type UserGradeService struct {
	userGradeRepo         *repository.UserGradeRepository
	userGradeRelationRepo *repository.UserGradeRelationRepository
}

func NewUserGradeService(
	userGradeRepo *repository.UserGradeRepository,
	userGradeRelationRepo *repository.UserGradeRelationRepository,
) *UserGradeService {
	return &UserGradeService{
		userGradeRepo:         userGradeRepo,
		userGradeRelationRepo: userGradeRelationRepo,
	}
}

// CreateGrade 创建用户等级
func (s *UserGradeService) CreateGrade(ctx context.Context, grade *model.UserGrade) error {
	grade.CreatedAt = time.Now()
	grade.UpdatedAt = time.Now()
	return s.userGradeRepo.Create(ctx, grade)
}

// UpdateGrade 更新用户等级
func (s *UserGradeService) UpdateGrade(ctx context.Context, grade *model.UserGrade) error {
	existingGrade, err := s.userGradeRepo.GetByID(ctx, grade.ID)
	if err != nil {
		return err
	}

	grade.CreatedAt = existingGrade.CreatedAt
	grade.UpdatedAt = time.Now()
	return s.userGradeRepo.Update(ctx, grade)
}

// DeleteGrade 删除用户等级
func (s *UserGradeService) DeleteGrade(ctx context.Context, id int64) error {
	return s.userGradeRepo.Delete(ctx, id)
}

// GetGrade 获取用户等级
func (s *UserGradeService) GetGrade(ctx context.Context, id int64) (*model.UserGrade, error) {
	return s.userGradeRepo.GetByID(ctx, id)
}

// ListGrades 获取用户等级列表
func (s *UserGradeService) ListGrades(ctx context.Context) ([]model.UserGrade, error) {
	return s.userGradeRepo.List(ctx)
}

// AssignUserGrade 分配用户等级
func (s *UserGradeService) AssignUserGrade(ctx context.Context, userID, gradeID int64) error {
	// 检查等级是否存在
	_, err := s.userGradeRepo.GetByID(ctx, gradeID)
	if err != nil {
		return errors.New("等级不存在")
	}

	// 删除旧的等级关系
	err = s.userGradeRelationRepo.Delete(ctx, userID, 0) // 0 表示删除该用户的所有等级关系
	if err != nil {
		return err
	}

	// 创建新的等级关系
	relation := &model.UserGradeRelation{
		UserID:    userID,
		GradeID:   gradeID,
		CreatedAt: time.Now(),
	}

	return s.userGradeRelationRepo.Create(ctx, relation)
}

// GetUserGrade 获取用户的等级
func (s *UserGradeService) GetUserGrade(ctx context.Context, userID int64) (*model.UserGrade, error) {
	return s.userGradeRelationRepo.GetUserGrade(ctx, userID)
}

// RemoveUserGrade 移除用户等级
func (s *UserGradeService) RemoveUserGrade(ctx context.Context, userID, gradeID int64) error {
	return s.userGradeRelationRepo.Delete(ctx, userID, gradeID)
}

// UpdateGradeStatus 更新等级状态
func (s *UserGradeService) UpdateGradeStatus(ctx context.Context, id int64, status int) error {
	grade, err := s.userGradeRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	grade.Status = status
	grade.UpdatedAt = time.Now()
	return s.userGradeRepo.Update(ctx, grade)
}
