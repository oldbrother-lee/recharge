package service

import (
	"errors"
	"recharge-go/internal/model"
	"recharge-go/internal/repository"
)

type DistributionService struct {
	repo repository.DistributionRepository
}

func NewDistributionService(repo repository.DistributionRepository) *DistributionService {
	return &DistributionService{repo: repo}
}

// CreateDistributor 创建分销商
func (s *DistributionService) CreateDistributor(req *model.DistributorRequest) error {
	distributor := &model.Distributor{
		UserID:      req.UserID,
		Name:        req.Name,
		Phone:       req.Phone,
		Commission:  req.Commission,
		Status:      req.Status,
		Description: req.Description,
	}
	return s.repo.CreateDistributor(distributor)
}

// GetDistributor 获取分销商详情
func (s *DistributionService) GetDistributor(id int64) (*model.Distributor, error) {
	return s.repo.GetDistributor(id)
}

// UpdateDistributor 更新分销商
func (s *DistributionService) UpdateDistributor(id int64, req *model.DistributorRequest) error {
	distributor := &model.Distributor{
		ID:          id,
		UserID:      req.UserID,
		Name:        req.Name,
		Phone:       req.Phone,
		Commission:  req.Commission,
		Status:      req.Status,
		Description: req.Description,
	}
	return s.repo.UpdateDistributor(distributor)
}

// DeleteDistributor 删除分销商
func (s *DistributionService) DeleteDistributor(id int64) error {
	return s.repo.DeleteDistributor(id)
}

// ListDistributors 获取分销商列表
func (s *DistributionService) ListDistributors(req *model.DistributorListRequest) (*model.DistributorListResponse, error) {
	distributors, total, err := s.repo.ListDistributors(req.Page, req.PageSize, req.Status)
	if err != nil {
		return nil, err
	}

	return &model.DistributorListResponse{
		Total: total,
		List:  distributors,
	}, nil
}

// GetDistributorStatistics 获取分销商统计信息
func (s *DistributionService) GetDistributorStatistics(id int64) (*model.DistributorStatistics, error) {
	return s.repo.GetDistributorStatistics(id)
}

// DistributionGrade 相关方法
func (s *DistributionService) CreateGrade(grade *model.DistributionGrade) error {
	return s.repo.CreateGrade(grade)
}

func (s *DistributionService) UpdateGrade(grade *model.DistributionGrade) error {
	return s.repo.UpdateGrade(grade)
}

func (s *DistributionService) DeleteGrade(id int64) error {
	return s.repo.DeleteGrade(id)
}

func (s *DistributionService) GetGrade(id int64) (*model.DistributionGrade, error) {
	return s.repo.GetGrade(id)
}

func (s *DistributionService) ListGrades() ([]*model.DistributionGrade, error) {
	return s.repo.ListGrades()
}

// DistributionRule 相关方法
func (s *DistributionService) CreateRule(rule *model.DistributionRule) error {
	// 验证分销等级是否存在
	_, err := s.repo.GetGrade(rule.GradeID)
	if err != nil {
		return errors.New("分销等级不存在")
	}
	return s.repo.CreateRule(rule)
}

func (s *DistributionService) UpdateRule(rule *model.DistributionRule) error {
	return s.repo.UpdateRule(rule)
}

func (s *DistributionService) DeleteRule(id int64) error {
	return s.repo.DeleteRule(id)
}

func (s *DistributionService) GetRule(id int64) (*model.DistributionRule, error) {
	return s.repo.GetRule(id)
}

func (s *DistributionService) ListRules(gradeID int64) ([]*model.DistributionRule, error) {
	return s.repo.ListRules(gradeID)
}

// DistributionCommission 相关方法
func (s *DistributionService) CreateCommission(commission *model.DistributionCommission) error {
	return s.repo.CreateCommission(commission)
}

func (s *DistributionService) UpdateCommission(commission *model.DistributionCommission) error {
	return s.repo.UpdateCommission(commission)
}

func (s *DistributionService) GetCommission(id int64) (*model.DistributionCommission, error) {
	return s.repo.GetCommission(id)
}

func (s *DistributionService) ListCommissions(userID int64, status int) ([]*model.DistributionCommission, error) {
	return s.repo.ListCommissions(userID, status)
}

// DistributionWithdrawal 相关方法
func (s *DistributionService) CreateWithdrawal(withdrawal *model.DistributionWithdrawal) error {
	// 检查用户是否有足够的佣金
	statistics, err := s.repo.GetStatistics(withdrawal.UserID)
	if err != nil {
		return err
	}
	if statistics.TotalCommission < withdrawal.Amount {
		return errors.New("佣金余额不足")
	}
	return s.repo.CreateWithdrawal(withdrawal)
}

func (s *DistributionService) UpdateWithdrawal(withdrawal *model.DistributionWithdrawal) error {
	return s.repo.UpdateWithdrawal(withdrawal)
}

func (s *DistributionService) GetWithdrawal(id int64) (*model.DistributionWithdrawal, error) {
	return s.repo.GetWithdrawal(id)
}

func (s *DistributionService) ListWithdrawals(userID int64, status int) ([]*model.DistributionWithdrawal, error) {
	return s.repo.ListWithdrawals(userID, status)
}

// DistributionStatistics 相关方法
func (s *DistributionService) GetStatistics(userID int64) (*model.DistributionStatistics, error) {
	return s.repo.GetStatistics(userID)
}

func (s *DistributionService) UpdateStatistics(statistics *model.DistributionStatistics) error {
	return s.repo.UpdateStatistics(statistics)
}

func (s *DistributionService) AddCommission(userID int64, amount float64) error {
	return s.repo.AddCommission(userID, amount)
}

// 计算订单佣金
func (s *DistributionService) CalculateCommission(orderID int64, userID int64, amount float64, productType int) (float64, error) {
	// 获取用户的分销等级
	grade, err := s.repo.GetGrade(userID)
	if err != nil {
		return 0, err
	}

	// 获取对应的佣金规则
	rules, err := s.repo.ListRules(grade.ID)
	if err != nil {
		return 0, err
	}

	// 查找适用的佣金规则
	var applicableRule *model.DistributionRule
	for _, rule := range rules {
		if rule.ProductType == productType &&
			amount >= rule.MinAmount &&
			(rule.MaxAmount == 0 || amount <= rule.MaxAmount) {
			applicableRule = rule
			break
		}
	}

	if applicableRule == nil {
		return 0, nil
	}

	// 计算佣金
	commission := amount * applicableRule.Commission
	return commission, nil
}
