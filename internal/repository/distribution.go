package repository

import (
	"recharge-go/internal/model"
	"time"

	"gorm.io/gorm"
)

// DistributionRepository 分销仓储接口
type DistributionRepository interface {
	// 分销商相关
	CreateDistributor(distributor *model.Distributor) error
	GetDistributor(id int64) (*model.Distributor, error)
	UpdateDistributor(distributor *model.Distributor) error
	DeleteDistributor(id int64) error
	ListDistributors(page, pageSize int, status string) ([]model.Distributor, int64, error)
	GetDistributorStatistics(userID int64) (*model.DistributorStatistics, error)

	// 分销等级相关
	CreateGrade(grade *model.DistributionGrade) error
	UpdateGrade(grade *model.DistributionGrade) error
	DeleteGrade(id int64) error
	GetGrade(id int64) (*model.DistributionGrade, error)
	ListGrades() ([]*model.DistributionGrade, error)

	// 分销规则相关
	CreateRule(rule *model.DistributionRule) error
	UpdateRule(rule *model.DistributionRule) error
	DeleteRule(id int64) error
	GetRule(id int64) (*model.DistributionRule, error)
	ListRules(gradeID int64) ([]*model.DistributionRule, error)

	// 分销佣金相关
	CreateCommission(commission *model.DistributionCommission) error
	UpdateCommission(commission *model.DistributionCommission) error
	GetCommission(id int64) (*model.DistributionCommission, error)
	ListCommissions(userID int64, status int) ([]*model.DistributionCommission, error)

	// 提现相关
	CreateWithdrawal(withdrawal *model.DistributionWithdrawal) error
	UpdateWithdrawal(withdrawal *model.DistributionWithdrawal) error
	GetWithdrawal(id int64) (*model.DistributionWithdrawal, error)
	ListWithdrawals(userID int64, status int) ([]*model.DistributionWithdrawal, error)

	// 分销统计相关
	GetStatistics(userID int64) (*model.DistributionStatistics, error)
	UpdateStatistics(statistics *model.DistributionStatistics) error
	AddCommission(userID int64, amount float64) error
}

type distributionRepository struct {
	db *gorm.DB
}

func NewDistributionRepository(db *gorm.DB) DistributionRepository {
	return &distributionRepository{db: db}
}

// DistributionGrade 相关方法
func (r *distributionRepository) CreateGrade(grade *model.DistributionGrade) error {
	return r.db.Create(grade).Error
}

func (r *distributionRepository) UpdateGrade(grade *model.DistributionGrade) error {
	return r.db.Save(grade).Error
}

func (r *distributionRepository) DeleteGrade(id int64) error {
	return r.db.Delete(&model.DistributionGrade{}, id).Error
}

func (r *distributionRepository) GetGrade(id int64) (*model.DistributionGrade, error) {
	var grade model.DistributionGrade
	err := r.db.First(&grade, id).Error
	return &grade, err
}

func (r *distributionRepository) ListGrades() ([]*model.DistributionGrade, error) {
	var grades []*model.DistributionGrade
	err := r.db.Find(&grades).Error
	return grades, err
}

// DistributionRule 相关方法
func (r *distributionRepository) CreateRule(rule *model.DistributionRule) error {
	return r.db.Create(rule).Error
}

func (r *distributionRepository) UpdateRule(rule *model.DistributionRule) error {
	return r.db.Save(rule).Error
}

func (r *distributionRepository) DeleteRule(id int64) error {
	return r.db.Delete(&model.DistributionRule{}, id).Error
}

func (r *distributionRepository) GetRule(id int64) (*model.DistributionRule, error) {
	var rule model.DistributionRule
	err := r.db.First(&rule, id).Error
	return &rule, err
}

func (r *distributionRepository) ListRules(gradeID int64) ([]*model.DistributionRule, error) {
	var rules []*model.DistributionRule
	err := r.db.Where("grade_id = ?", gradeID).Find(&rules).Error
	return rules, err
}

// DistributionCommission 相关方法
func (r *distributionRepository) CreateCommission(commission *model.DistributionCommission) error {
	return r.db.Create(commission).Error
}

func (r *distributionRepository) UpdateCommission(commission *model.DistributionCommission) error {
	return r.db.Save(commission).Error
}

func (r *distributionRepository) GetCommission(id int64) (*model.DistributionCommission, error) {
	var commission model.DistributionCommission
	err := r.db.First(&commission, id).Error
	return &commission, err
}

func (r *distributionRepository) ListCommissions(userID int64, status int) ([]*model.DistributionCommission, error) {
	var commissions []*model.DistributionCommission
	query := r.db.Where("user_id = ?", userID)
	if status >= 0 {
		query = query.Where("status = ?", status)
	}
	err := query.Find(&commissions).Error
	return commissions, err
}

// DistributionWithdrawal 相关方法
func (r *distributionRepository) CreateWithdrawal(withdrawal *model.DistributionWithdrawal) error {
	return r.db.Create(withdrawal).Error
}

func (r *distributionRepository) UpdateWithdrawal(withdrawal *model.DistributionWithdrawal) error {
	return r.db.Save(withdrawal).Error
}

func (r *distributionRepository) GetWithdrawal(id int64) (*model.DistributionWithdrawal, error) {
	var withdrawal model.DistributionWithdrawal
	err := r.db.First(&withdrawal, id).Error
	return &withdrawal, err
}

func (r *distributionRepository) ListWithdrawals(userID int64, status int) ([]*model.DistributionWithdrawal, error) {
	var withdrawals []*model.DistributionWithdrawal
	query := r.db.Where("user_id = ?", userID)
	if status >= 0 {
		query = query.Where("status = ?", status)
	}
	err := query.Find(&withdrawals).Error
	return withdrawals, err
}

// DistributionStatistics 相关方法
func (r *distributionRepository) GetStatistics(userID int64) (*model.DistributionStatistics, error) {
	var statistics model.DistributionStatistics
	err := r.db.Where("user_id = ?", userID).First(&statistics).Error
	if err == gorm.ErrRecordNotFound {
		statistics.UserID = userID
		err = r.db.Create(&statistics).Error
	}
	return &statistics, err
}

func (r *distributionRepository) UpdateStatistics(statistics *model.DistributionStatistics) error {
	return r.db.Save(statistics).Error
}

func (r *distributionRepository) AddCommission(userID int64, amount float64) error {
	return r.db.Model(&model.DistributionStatistics{}).
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"total_commission": gorm.Expr("total_commission + ?", amount),
			"total_orders":     gorm.Expr("total_orders + 1"),
		}).Error
}

// DistributionDistributor 相关方法
func (r *distributionRepository) CreateDistributor(distributor *model.Distributor) error {
	return r.db.Create(distributor).Error
}

func (r *distributionRepository) GetDistributor(id int64) (*model.Distributor, error) {
	var distributor model.Distributor
	err := r.db.First(&distributor, id).Error
	if err != nil {
		return nil, err
	}
	return &distributor, nil
}

func (r *distributionRepository) UpdateDistributor(distributor *model.Distributor) error {
	return r.db.Save(distributor).Error
}

func (r *distributionRepository) DeleteDistributor(id int64) error {
	return r.db.Delete(&model.Distributor{}, id).Error
}

func (r *distributionRepository) ListDistributors(page, pageSize int, status string) ([]model.Distributor, int64, error) {
	var distributors []model.Distributor
	var total int64

	query := r.db.Model(&model.Distributor{})
	if status != "" {
		query = query.Where("status = ?", status)
	}

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Offset((page - 1) * pageSize).Limit(pageSize).Find(&distributors).Error
	if err != nil {
		return nil, 0, err
	}

	return distributors, total, nil
}

// GetDistributorStatistics 获取分销商统计信息
func (r *distributionRepository) GetDistributorStatistics(userID int64) (*model.DistributorStatistics, error) {
	// 初始化统计信息
	statistics := &model.DistributorStatistics{}

	// 获取总订单数和总佣金
	var totalOrders int64
	var totalCommission float64
	var totalAmount float64

	err := r.db.Model(&model.DistributionCommission{}).
		Where("user_id = ?", userID).
		Count(&totalOrders).Error
	if err != nil {
		return nil, err
	}

	err = r.db.Model(&model.DistributionCommission{}).
		Where("user_id = ? AND status = ?", userID, 1). // 1表示已结算
		Select("COALESCE(SUM(commission), 0) as total_commission, COALESCE(SUM(amount), 0) as total_amount").
		Row().Scan(&totalCommission, &totalAmount)
	if err != nil {
		return nil, err
	}

	// 获取本月订单数和佣金
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	var monthOrders int64
	var monthCommission float64
	var monthAmount float64

	err = r.db.Model(&model.DistributionCommission{}).
		Where("user_id = ? AND create_time >= ?", userID, startOfMonth).
		Count(&monthOrders).Error
	if err != nil {
		return nil, err
	}

	err = r.db.Model(&model.DistributionCommission{}).
		Where("user_id = ? AND status = ? AND create_time >= ?", userID, 1, startOfMonth).
		Select("COALESCE(SUM(commission), 0) as month_commission, COALESCE(SUM(amount), 0) as month_amount").
		Row().Scan(&monthCommission, &monthAmount)
	if err != nil {
		return nil, err
	}

	// 设置统计信息
	statistics.TotalOrders = totalOrders
	statistics.TotalCommission = totalCommission
	statistics.TotalAmount = totalAmount
	statistics.MonthOrders = monthOrders
	statistics.MonthCommission = monthCommission
	statistics.MonthAmount = monthAmount

	return statistics, nil
}
