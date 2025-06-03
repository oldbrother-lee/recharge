package service

import (
	"context"
	"recharge-go/internal/model"
	"recharge-go/internal/repository"
	"time"
)

// UserLogService 用户日志服务
type UserLogService struct {
	userLogRepo *repository.UserLogRepository
}

// NewUserLogService 创建用户日志服务
func NewUserLogService(userLogRepo *repository.UserLogRepository) *UserLogService {
	return &UserLogService{
		userLogRepo: userLogRepo,
	}
}

// CreateLog 创建用户日志
func (s *UserLogService) CreateLog(ctx context.Context, req *model.UserLogRequest) (*model.UserLogResponse, error) {
	log := &model.UserLog{
		UserID:    req.UserID,
		Action:    req.Action,
		TargetID:  req.TargetID,
		Content:   req.Content,
		IP:        req.IP,
		CreatedAt: time.Now(),
	}

	if err := s.userLogRepo.Create(ctx, log); err != nil {
		return nil, err
	}

	return &model.UserLogResponse{
		ID:        log.ID,
		UserID:    log.UserID,
		Action:    log.Action,
		TargetID:  log.TargetID,
		Content:   log.Content,
		IP:        log.IP,
		CreatedAt: log.CreatedAt,
	}, nil
}

// GetLogByID 获取日志详情
func (s *UserLogService) GetLogByID(ctx context.Context, id int64) (*model.UserLogResponse, error) {
	log, err := s.userLogRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return &model.UserLogResponse{
		ID:        log.ID,
		UserID:    log.UserID,
		Action:    log.Action,
		TargetID:  log.TargetID,
		Content:   log.Content,
		IP:        log.IP,
		CreatedAt: log.CreatedAt,
	}, nil
}

// ListLogs 获取日志列表
func (s *UserLogService) ListLogs(ctx context.Context, req *model.UserLogListRequest) (*model.UserLogListResponse, error) {
	logs, total, err := s.userLogRepo.List(ctx, req.UserID, req.TargetID, req.Action, req.Current, req.Size)
	if err != nil {
		return nil, err
	}

	var logResponses []model.UserLogResponse
	for _, log := range logs {
		logResponses = append(logResponses, model.UserLogResponse{
			ID:        log.ID,
			UserID:    log.UserID,
			Action:    log.Action,
			TargetID:  log.TargetID,
			Content:   log.Content,
			IP:        log.IP,
			CreatedAt: log.CreatedAt,
		})
	}

	return &model.UserLogListResponse{
		List:  logResponses,
		Total: total,
	}, nil
}
