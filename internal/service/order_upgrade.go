package service

import (
	"context"
	"recharge-go/internal/model"
	"recharge-go/internal/repository"
	"recharge-go/internal/utils"
	"time"
)

type OrderUpgradeService interface {
	CreateOrder(ctx context.Context, order *model.OrderUpgrade) error
	GetOrderByID(ctx context.Context, id int64) (*model.OrderUpgrade, error)
	GetOrdersByUserID(ctx context.Context, userID int64, page, pageSize int) ([]*model.OrderUpgrade, int64, error)
	UpdateOrderStatus(ctx context.Context, id int64, status int) error
	ProcessOrderPayment(ctx context.Context, orderID int64, payWay int, serialNumber string) error
}

type orderUpgradeService struct {
	orderUpgradeRepo      repository.OrderUpgradeRepository
	rebateRepo            repository.RebateRepository
	rewardRepo            repository.RewardRepository
	userRepo              repository.UserRepository
	userGradeRepo         repository.UserGradeRepository
	userGradeRelationRepo repository.UserGradeRelationRepository
}

func NewOrderUpgradeService(
	orderUpgradeRepo repository.OrderUpgradeRepository,
	rebateRepo repository.RebateRepository,
	rewardRepo repository.RewardRepository,
	userRepo repository.UserRepository,
	userGradeRepo repository.UserGradeRepository,
	userGradeRelationRepo repository.UserGradeRelationRepository,
) OrderUpgradeService {
	return &orderUpgradeService{
		orderUpgradeRepo:      orderUpgradeRepo,
		rebateRepo:            rebateRepo,
		rewardRepo:            rewardRepo,
		userRepo:              userRepo,
		userGradeRepo:         userGradeRepo,
		userGradeRelationRepo: userGradeRelationRepo,
	}
}

func (s *orderUpgradeService) CreateOrder(ctx context.Context, order *model.OrderUpgrade) error {
	// 生成订单号
	order.OrderNumber = time.Now().Format("20060102150405") + "-" + utils.RandString(6)
	order.CreatedAt = time.Now()
	order.UpdatedAt = time.Now()
	return s.orderUpgradeRepo.Create(ctx, order)
}

func (s *orderUpgradeService) GetOrderByID(ctx context.Context, id int64) (*model.OrderUpgrade, error) {
	return s.orderUpgradeRepo.GetByID(ctx, id)
}

func (s *orderUpgradeService) GetOrdersByUserID(ctx context.Context, userID int64, page, pageSize int) ([]*model.OrderUpgrade, int64, error) {
	return s.orderUpgradeRepo.GetByUserID(ctx, userID, page, pageSize)
}

func (s *orderUpgradeService) UpdateOrderStatus(ctx context.Context, id int64, status int) error {
	return s.orderUpgradeRepo.UpdateStatus(ctx, id, status)
}

func (s *orderUpgradeService) ProcessOrderPayment(ctx context.Context, orderID int64, payWay int, serialNumber string) error {
	// 获取订单信息
	order, err := s.orderUpgradeRepo.GetByID(ctx, orderID)
	if err != nil {
		return err
	}

	// 更新订单支付信息
	order.PayWay = payWay
	order.SerialNumber = serialNumber
	order.IsPay = 1
	order.PayTime = time.Now()
	order.UpdatedAt = time.Now()

	// 更新订单状态
	err = s.orderUpgradeRepo.UpdateStatus(ctx, orderID, 1)
	if err != nil {
		return err
	}

	// 更新用户等级
	// 获取用户当前等级
	currentGrade, err := s.userGradeRelationRepo.GetUserGrade(ctx, order.UserID)
	if err != nil {
		return err
	}

	// 删除旧的等级关系
	if currentGrade != nil {
		err = s.userGradeRelationRepo.Delete(ctx, order.UserID, currentGrade.ID)
		if err != nil {
			return err
		}
	}

	// 创建新的等级关系
	err = s.userGradeRelationRepo.Create(ctx, &model.UserGradeRelation{
		UserID:  order.UserID,
		GradeID: order.GradeID,
	})
	if err != nil {
		return err
	}

	// 创建返利记录
	if order.IsRebate == 1 {
		rebate := &model.Rebate{
			UserID:  order.UserID,
			OrderID: orderID,
			Amount:  order.RebatePrice,
			Status:  0,
			Remark:  "用户升级返利",
		}
		err = s.rebateRepo.Create(ctx, rebate)
		if err != nil {
			return err
		}
	}

	// 创建奖励记录
	if order.RewardPrice > 0 {
		reward := &model.Reward{
			UserID:  order.UserID,
			OrderID: orderID,
			Amount:  order.RewardPrice,
			Type:    1, // 升级奖励
			Status:  0,
			Remark:  "用户升级奖励",
		}
		err = s.rewardRepo.Create(ctx, reward)
		if err != nil {
			return err
		}
	}

	return nil
}
