package repository

import (
	"context"
	"errors"
	"fmt"
	"recharge-go/internal/model"

	"gorm.io/gorm"
)

// BalanceLogRepository 余额流水仓储
// 负责 balance_logs 表的增查和用户余额原子操作

type BalanceLogRepository struct {
	db *gorm.DB
}

func NewBalanceLogRepository(db *gorm.DB) *BalanceLogRepository {
	return &BalanceLogRepository{db: db}
}

// GetDB 获取数据库连接
func (r *BalanceLogRepository) GetDB() *gorm.DB {
	return r.db
}

// CreateLog 新增一条余额流水
func (r *BalanceLogRepository) CreateLog(ctx context.Context, log *model.BalanceLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

// fundFlowCol 统一字符串列 COLLATE，避免 UNION 混用排序规则 (Error 1271)
func fundFlowCol(expr string) string {
	return fmt.Sprintf("CONVERT(%s USING utf8mb4) COLLATE utf8mb4_unicode_ci", expr)
}

func fundFlowListSQL() string {
	c := fundFlowCol
	return `
SELECT * FROM (
  SELECT
    bl.id AS id,
    bl.user_id AS user_id,
    bl.order_id AS order_id,
    bl.platform_account_id AS platform_account_id,
    bl.platform_id AS platform_id,
    ` + c("COALESCE(bl.platform_code, '')") + ` AS platform_code,
    ` + c("COALESCE(bl.platform_name, '')") + ` AS platform_name,
    bl.amount AS amount,
    bl.type AS type,
    bl.style AS style,
    bl.balance AS balance,
    bl.balance_before AS balance_before,
    ` + c("COALESCE(bl.remark, '')") + ` AS remark,
    ` + c("COALESCE(bl.operator, '')") + ` AS operator,
    bl.created_at AS created_at,
    ` + c("?") + ` AS fund_source,
    ` + c("COALESCE(o.order_number, '')") + ` AS order_number,
    ` + c("COALESCE(o.out_trade_num, '')") + ` AS out_trade_num,
    ` + c("COALESCE(o.mobile, '')") + ` AS mobile,
    COALESCE(o.status, 0) AS order_status
  FROM balance_logs bl
  LEFT JOIN orders o ON bl.order_id = o.id
  WHERE bl.user_id = ?
  UNION ALL
  SELECT
    cl.id AS id,
    cl.user_id AS user_id,
    cl.order_id AS order_id,
    0 AS platform_account_id,
    0 AS platform_id,
    ` + c("''") + ` AS platform_code,
    ` + c("''") + ` AS platform_name,
    CASE
      WHEN cl.type = ? THEN (cl.credit_after - cl.credit_before)
      WHEN cl.type = ? THEN -cl.amount
      ELSE cl.amount
    END AS amount,
    CASE
      WHEN cl.type = ? THEN 2
      WHEN cl.type = ? AND (cl.credit_after - cl.credit_before) < 0 THEN 2
      ELSE 1
    END AS type,
    CASE
      WHEN cl.type = ? THEN ?
      WHEN cl.type = ? THEN ?
      ELSE ?
    END AS style,
    cl.credit_after AS balance,
    cl.credit_before AS balance_before,
    ` + c("COALESCE(cl.remark, '')") + ` AS remark,
    ` + c("COALESCE(cl.operator, '')") + ` AS operator,
    cl.created_at AS created_at,
    ` + c("?") + ` AS fund_source,
    ` + c("COALESCE(o.order_number, '')") + ` AS order_number,
    ` + c("COALESCE(o.out_trade_num, '')") + ` AS out_trade_num,
    ` + c("COALESCE(o.mobile, '')") + ` AS mobile,
    COALESCE(o.status, 0) AS order_status
  FROM credit_logs cl
  LEFT JOIN orders o ON cl.order_id = o.id
  WHERE cl.user_id = ?
) AS fund_flow
ORDER BY created_at DESC, id DESC
LIMIT ? OFFSET ?`
}

// ListLogs 查询用户余额流水（分页）
func (r *BalanceLogRepository) ListLogs(ctx context.Context, userID int64, offset, limit int) ([]model.BalanceLog, int64, error) {
	var logs []model.BalanceLog
	var total int64
	db := r.db.WithContext(ctx).Model(&model.BalanceLog{}).Where("user_id = ?", userID)
	db.Count(&total)
	err := db.Order("id desc").Offset(offset).Limit(limit).Find(&logs).Error
	return logs, total, err
}

// ListLogsWithOrder 查询用户资金流水（余额 + 授信，分页，关联订单信息）
func (r *BalanceLogRepository) ListLogsWithOrder(ctx context.Context, userID int64, offset, limit int) ([]model.BalanceLogWithOrder, int64, error) {
	var balanceCount, creditCount int64
	if err := r.db.WithContext(ctx).Model(&model.BalanceLog{}).Where("user_id = ?", userID).Count(&balanceCount).Error; err != nil {
		return nil, 0, err
	}
	if err := r.db.WithContext(ctx).Model(&model.CreditLog{}).Where("user_id = ?", userID).Count(&creditCount).Error; err != nil {
		return nil, 0, err
	}
	total := balanceCount + creditCount

	listSQL := fundFlowListSQL()

	var logs []model.BalanceLogWithOrder
	err := r.db.WithContext(ctx).Raw(listSQL,
		model.FundSourceBalance, userID,
		model.CreditTypeSet, model.CreditTypeUse,
		model.CreditTypeSet, model.CreditTypeUse,
		model.CreditTypeUse, model.BalanceStyleCreditUse,
		model.CreditTypeRestore, model.BalanceStyleCreditRestore,
		model.BalanceStyleCreditAdjust,
		model.FundSourceCredit, userID,
		limit, offset,
	).Scan(&logs).Error
	return logs, total, err
}

// AddBalance 用户余额增加（使用原子性更新避免竞态条件）
func (r *BalanceLogRepository) AddBalance(ctx context.Context, userID int64, amount float64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 使用原子性更新避免读取-计算-写入的竞态条件
		result := tx.Model(&model.User{}).
			Where("id = ?", userID).
			Update("balance", gorm.Expr("balance + ?", amount))
		
		if result.Error != nil {
			return result.Error
		}
		
		if result.RowsAffected == 0 {
			return errors.New("用户不存在")
		}
		
		return nil
	})
}

// SubBalance 用户余额扣减（使用原子性更新和余额校验避免竞态条件）
func (r *BalanceLogRepository) SubBalance(ctx context.Context, userID int64, amount float64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 使用原子性更新，同时在SQL层面校验余额充足
		// 只有当余额充足时才会更新，避免读取-计算-写入的竞态条件
		result := tx.Model(&model.User{}).
			Where("id = ? AND balance >= ?", userID, amount).
			Update("balance", gorm.Expr("balance - ?", amount))
		
		if result.Error != nil {
			return result.Error
		}
		
		if result.RowsAffected == 0 {
			// 检查是用户不存在还是余额不足
			var user model.User
			if err := tx.Where("id = ?", userID).First(&user).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return errors.New("用户不存在")
				}
				return err
			}
			return errors.New("余额不足")
		}
		
		return nil
	})
}

// DeleteByOrderIDs 批量删除余额日志（分批避免占位符过多导致 1390 错误）
func (r *BalanceLogRepository) DeleteByOrderIDs(ctx context.Context, orderIDs []int64) error {
	if len(orderIDs) == 0 {
		return nil
	}
	const batchSize = 1000
	for start := 0; start < len(orderIDs); start += batchSize {
		end := start + batchSize
		if end > len(orderIDs) {
			end = len(orderIDs)
		}
		ids := orderIDs[start:end]
		if err := r.db.WithContext(ctx).Where("order_id IN ?", ids).Delete(&model.BalanceLog{}).Error; err != nil {
			return err
		}
	}
	return nil
}
