package repository

import (
	"context"
	"errors"
	"fmt"
	"recharge-go/internal/model"
	"recharge-go/pkg/logger"
	"strconv"
	"time"

	"gorm.io/gorm"
)

var (
	ErrOrderNotFound = errors.New("order not found")
)

// OrderRepository 订单仓库接口
type OrderRepository interface {
	// Create 创建订单
	Create(ctx context.Context, order *model.Order) error
	// GetByID 根据ID获取订单
	GetByID(ctx context.Context, id int64) (*model.Order, error)
	// GetByOrderNumber 根据订单号获取订单
	GetByOrderNumber(ctx context.Context, orderNumber string) (*model.Order, error)
	// GetByCustomerID 根据客户ID获取订单列表
	GetByCustomerID(ctx context.Context, customerID int64, page, pageSize int) ([]*model.Order, int64, error)
	// UpdateStatus 更新订单状态
	UpdateStatus(ctx context.Context, id int64, status model.OrderStatus) error
	// UpdatePayInfo 更新支付信息
	UpdatePayInfo(ctx context.Context, id int64, payWay int, serialNumber string) error
	// UpdateAPIInfo 更新API信息
	UpdateAPIInfo(ctx context.Context, id int64, apiID int64, apiName string, apiParamName string) error
	// UpdateFinishTime 更新完成时间
	UpdateFinishTime(ctx context.Context, id int64) error
	// UpdateRemark 更新备注
	UpdateRemark(ctx context.Context, id int64, remark string) error
	// Delete 删除订单
	Delete(ctx context.Context, id int64) error
	// GetOrderByOutTradeNum 根据外部交易号获取订单
	GetOrderByOutTradeNum(ctx context.Context, outTradeNum string) (*model.Order, error)
	// GetByOutTradeNum 根据外部交易号获取订单
	GetByOutTradeNum(ctx context.Context, outTradeNum string) (*model.Order, error)
	// GetOrders 获取订单列表
	GetOrders(ctx context.Context, params map[string]interface{}, page, pageSize int) ([]*model.Order, int64, error)
	// GetByStatus 根据状态获取订单列表
	GetByStatus(ctx context.Context, status model.OrderStatus) ([]*model.Order, error)
	// GetByOrderID 根据订单号获取订单
	GetByOrderID(ctx context.Context, orderID string) (*model.Order, error)
	// UpdatePlatformID 更新订单支付平台ID和API ID
	UpdatePlatformID(ctx context.Context, orderID int64, platformID *model.PlatformAPI, ParamID int64) error
	// DB 返回数据库连接
	DB() *gorm.DB
	// GetIDsByTimeRange 查询指定时间范围的订单ID
	GetIDsByTimeRange(ctx context.Context, start, end string) ([]int64, error)
	// DeleteByIDs 批量删除订单
	DeleteByIDs(ctx context.Context, ids []int64) (int64, error)
	// FindProductByPriceAndISP 根据价格、ISP和状态获取产品
	FindProductByPriceAndISP(price float64, isp int, status int) (*model.Product, error)
	// FindProductByPriceAndISPWithTolerance 根据价格、ISP和状态获取产品，支持价格误差容忍
	FindProductByPriceAndISPWithTolerance(price float64, isp int, status int, tolerance float64) (*model.Product, error)
	// FindProductByNameValueAndISP 根据产品名称数字部分、ISP和状态获取产品
	FindProductByNameValueAndISP(nameValue int, isp int, status int) (*model.Product, error)
	// UpdateStatusCAS 原子性地将订单状态从 oldStatus 更新为 newStatus，同时写入 api_id
	UpdateStatusCAS(ctx context.Context, id int64, oldStatus, newStatus model.OrderStatus, apiID int64) (bool, error)
	// UpdateStatusAndAPIID 更新订单状态和API ID
	UpdateStatusAndAPIID(ctx context.Context, id int64, status model.OrderStatus, apiID int64, usedAPIs string) error
	// GetOrderRealtimeStatistics 获取实时统计
	GetOrderRealtimeStatistics(ctx context.Context, userId int64) (*model.OrderStatisticsOverview, error)
	// GetOperatorRealtimeStatistics 获取运营商实时统计
	GetOperatorRealtimeStatistics(ctx context.Context, start, end time.Time) ([]model.OrderStatisticsOperator, error)
	// GetOperatorRealtimeStatisticsByUser 获取指定用户的运营商实时统计
	GetOperatorRealtimeStatisticsByUser(ctx context.Context, start, end time.Time, userId int64) ([]model.OrderStatisticsOperator, error)
	// GetOperatorOrderCount 按运营商分组统计订单总数
	GetOperatorOrderCount(ctx context.Context, start, end time.Time) ([]model.OperatorOrderCount, error)
	GetByUserID(ctx context.Context, userID int64, params map[string]interface{}, page, pageSize int) ([]*model.Order, int64, error)
	// SoftDeleteByID 软删除订单
	SoftDeleteByID(ctx context.Context, id int64) error
}

// OrderRepositoryImpl 订单仓库实现
type OrderRepositoryImpl struct {
	db *gorm.DB
}

// NewOrderRepository 创建订单仓库实例
func NewOrderRepository(db *gorm.DB) OrderRepository {
	if db == nil {
		panic("db is nil in NewOrderRepository")
	}
	return &OrderRepositoryImpl{db: db}
}

// DB 返回数据库连接
func (r *OrderRepositoryImpl) DB() *gorm.DB {
	return r.db
}

// GetByStatus 根据状态获取订单列表
func (r *OrderRepositoryImpl) GetByStatus(ctx context.Context, status model.OrderStatus) ([]*model.Order, error) {
	var orders []*model.Order
	if err := r.db.Where("status = ?", status).Find(&orders).Error; err != nil {
		return nil, err
	}
	return orders, nil
}

// Create 创建订单
func (r *OrderRepositoryImpl) Create(ctx context.Context, order *model.Order) error {
	return r.db.Create(order).Error
}

// GetByID 根据ID获取订单
func (r *OrderRepositoryImpl) GetByID(ctx context.Context, id int64) (*model.Order, error) {
	var order model.Order
	if err := r.db.Where("id = ? AND is_del = 0", id).First(&order).Error; err != nil {
		return nil, err
	}
	return &order, nil
}

// GetByOrderNumber 根据订单号获取订单
func (r *OrderRepositoryImpl) GetByOrderNumber(ctx context.Context, orderNumber string) (*model.Order, error) {
	var order model.Order
	if err := r.db.Where("order_number = ? AND is_del = 0", orderNumber).First(&order).Error; err != nil {
		return nil, err
	}
	return &order, nil
}

// GetByOutTradeNum 根据外部交易号获取订单
func (r *OrderRepositoryImpl) GetByOutTradeNum(ctx context.Context, outTradeNum string) (*model.Order, error) {
	var order model.Order
	if err := r.db.Where("order_number = ? AND is_del = 0", outTradeNum).First(&order).Error; err != nil {
		return nil, err
	}
	return &order, nil
}

// GetByCustomerID 根据客户ID获取订单列表
func (r *OrderRepositoryImpl) GetByCustomerID(ctx context.Context, customerID int64, page, pageSize int) ([]*model.Order, int64, error) {
	var orders []*model.Order
	var total int64

	if err := r.db.Model(&model.Order{}).Where("customer_id = ? AND is_del = 0", customerID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := r.db.Where("customer_id = ? AND is_del = 0", customerID).Offset(offset).Limit(pageSize).Find(&orders).Error; err != nil {
		return nil, 0, err
	}

	return orders, total, nil
}

// UpdateStatus 更新订单状态
func (r *OrderRepositoryImpl) UpdateStatus(ctx context.Context, id int64, status model.OrderStatus) error {
	return r.db.Model(&model.Order{}).Where("id = ?", id).Update("status", status).Error
}

// UpdatePayInfo 更新支付信息
func (r *OrderRepositoryImpl) UpdatePayInfo(ctx context.Context, id int64, payWay int, serialNumber string) error {
	return r.db.Model(&model.Order{}).Where("id = ?", id).Updates(map[string]interface{}{
		"pay_way":       payWay,
		"serial_number": serialNumber,
	}).Error
}

// UpdateAPIInfo 更新API信息
func (r *OrderRepositoryImpl) UpdateAPIInfo(ctx context.Context, id int64, apiID int64, apiName string, apiParamName string) error {
	return r.db.WithContext(ctx).
		Model(&model.Order{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"api_cur_id":       apiID,
			"api_cur_param_id": apiParamName,
		}).Error
}

// UpdateFinishTime 更新完成时间
func (r *OrderRepositoryImpl) UpdateFinishTime(ctx context.Context, id int64) error {
	return r.db.Model(&model.Order{}).Where("id = ?", id).Update("finish_time", time.Now()).Error
}

// UpdateRemark 更新备注
func (r *OrderRepositoryImpl) UpdateRemark(ctx context.Context, id int64, remark string) error {
	return r.db.Model(&model.Order{}).Where("id = ?", id).Update("remark", remark).Error
}

// Delete 删除订单
func (r *OrderRepositoryImpl) Delete(ctx context.Context, id int64) error {
	return r.db.Delete(&model.Order{}, id).Error
}

// GetOrderByOutTradeNum 根据外部交易号获取订单
func (r *OrderRepositoryImpl) GetOrderByOutTradeNum(ctx context.Context, outTradeNum string) (*model.Order, error) {
	var order model.Order
	if err := r.db.Where("out_trade_num = ? AND is_del = 0", outTradeNum).First(&order).Error; err != nil {
		return nil, err
	}
	return &order, nil
}

// GetOrders 获取订单列表
func (r *OrderRepositoryImpl) GetOrders(ctx context.Context, params map[string]interface{}, page, pageSize int) ([]*model.Order, int64, error) {
	var orders []*model.Order
	var total int64

	query := r.db.Model(&model.Order{}).Where("is_del = 0")

	// 添加查询条件
	for key, value := range params {
		// 将 interface{} 转换为 string
		strValue, ok := value.(string)
		if !ok || strValue == "" {
			continue
		}
		switch key {
		case "client":
			// 将字符串转换为整数
			clientID, err := strconv.ParseInt(strValue, 10, 64)
			if err != nil {
				logger.Error("解析client参数失败: %v", err)
				continue
			}
			if clientID > 0 {
				query = query.Where("client = ?", clientID)
			}
		case "status":
			// 将字符串转换为整数
			status, err := strconv.ParseInt(strValue, 10, 64)
			if err != nil {
				logger.Error("解析status参数失败: %v", err)
				continue
			}
			if status >= 0 {
				query = query.Where("status = ?", status)
			}
		case "order_number":
			query = query.Where("order_number LIKE ?", "%"+strValue+"%")
		case "mobile":
			query = query.Where("mobile LIKE ?", "%"+strValue+"%")
		case "start_time":
			query = query.Where("create_time >= ?", strValue)
		case "end_time":
			query = query.Where("create_time <= ?", strValue)
		case "platform_code":
			query = query.Where("platform_code = ?", strValue)
		default:
			// 对于其他字段，使用精确匹配
			query = query.Where(key+" = ?", strValue)
		}
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Order("create_time DESC").Offset(offset).Limit(pageSize).Find(&orders).Error; err != nil {
		return nil, 0, err
	}

	return orders, total, nil
}

// GetByOrderID 根据订单号获取订单
func (r *OrderRepositoryImpl) GetByOrderID(ctx context.Context, orderID string) (*model.Order, error) {
	var order model.Order
	if err := r.db.Where("order_number = ?", orderID).First(&order).Error; err != nil {
		return nil, err
	}
	return &order, nil
}

// UpdatePlatformID 更新订单支付平台ID和API ID
func (r *OrderRepositoryImpl) UpdatePlatformID(ctx context.Context, orderID int64, platformID *model.PlatformAPI, ParamID int64) error {
	return r.db.WithContext(ctx).
		Model(&model.Order{}).
		Where("id = ?", orderID).
		Updates(map[string]interface{}{
			"platform_id":      platformID.ID,
			"api_cur_id":       platformID.ID, // 使用相同的platformID作为api_id
			"api_cur_param_id": ParamID,
			"platform_name":    platformID.Name,
			"platform_code":    platformID.Code,
		}).Error
}

// GetIDsByTimeRange 查询指定时间范围的订单ID
func (r *OrderRepositoryImpl) GetIDsByTimeRange(ctx context.Context, start, end string) ([]int64, error) {
	var ids []int64
	err := r.db.Model(&model.Order{}).Debug().
		Where("create_time BETWEEN ? AND ?", start, end).
		Pluck("id", &ids).Error
	fmt.Println("ids########", ids)
	return ids, err
}

// DeleteByIDs 批量删除订单
func (r *OrderRepositoryImpl) DeleteByIDs(ctx context.Context, ids []int64) (int64, error) {
	res := r.db.Unscoped().Where("id IN ?", ids).Delete(&model.Order{})
	return res.RowsAffected, res.Error
}

// FindProductByPriceAndISP 根据价格、ISP和状态获取产品
func (r *OrderRepositoryImpl) FindProductByPriceAndISP(price float64, isp int, status int) (*model.Product, error) {
	var product model.Product
	err := r.db.Where("price = ? AND isp = ? AND status = ?", price, isp, status).First(&product).Error
	if err != nil {
		return nil, err
	}
	return &product, nil
}

// FindProductByPriceAndISPWithTolerance 根据价格、ISP和状态获取产品，支持价格误差容忍
func (r *OrderRepositoryImpl) FindProductByPriceAndISPWithTolerance(price float64, isp int, status int, tolerance float64) (*model.Product, error) {
	var product model.Product
	err := r.db.Where("ABS(price - ?) < ? AND isp = ? AND status = ?", price, tolerance, isp, status).First(&product).Error
	if err != nil {
		return nil, err
	}
	return &product, nil
}

// FindProductByNameValueAndISP 根据产品名称数字部分、ISP和状态获取产品
func (r *OrderRepositoryImpl) FindProductByNameValueAndISP(nameValue int, isp int, status int) (*model.Product, error) {
	var product model.Product
	// ISP字段存储为字符串，直接进行字符串相等比较
	ispStr := fmt.Sprintf("%d", isp)
	
	// 使用更灵活的正则表达式，支持中文前缀（如"中国移动"、"移动"等）
	// 匹配：中文字符后跟数字，或者数字在字符串末尾
	err := r.db.Where("name REGEXP ? AND isp = ? AND status = ?",
		fmt.Sprintf("[\\u4e00-\\u9fff]+%d($|[^0-9])", nameValue), ispStr, status).First(&product).Error
	
	if err != nil {
		return nil, err
	}
	return &product, nil
}

// UpdateStatusCAS 原子性地将订单状态从 oldStatus 更新为 newStatus，同时写入 api_id
func (r *OrderRepositoryImpl) UpdateStatusCAS(ctx context.Context, id int64, oldStatus, newStatus model.OrderStatus, apiID int64) (bool, error) {
	res := r.db.Model(&model.Order{}).
		Where("id = ? AND status = ? AND (api_id = 0 OR api_id IS NULL OR api_id = ?)", id, oldStatus, apiID).
		Updates(map[string]interface{}{
			"status": newStatus,
			"api_id": apiID,
		})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}

// UpdateStatusAndAPIID 更新订单状态和API ID
func (r *OrderRepositoryImpl) UpdateStatusAndAPIID(ctx context.Context, id int64, status model.OrderStatus, apiID int64, usedAPIs string) error {
	fmt.Println("即将执行 Updates")
	err := r.db.WithContext(ctx).
		Model(&model.Order{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":    status,
			"api_id":    apiID,
			"used_apis": usedAPIs,
		}).Error
	fmt.Println("Updates 执行完毕，err =", err)
	return err
}

// GetOrderRealtimeStatistics 获取实时统计
func (r *OrderRepositoryImpl) GetOrderRealtimeStatistics(ctx context.Context, userId int64) (*model.OrderStatisticsOverview, error) {
	var overview model.OrderStatisticsOverview

	baseQuery := r.db.Model(&model.Order{}).Where("is_del = 0")
	if userId > 0 {
		baseQuery = baseQuery.Where("customer_id = ?", userId)
	}

	// 获取总订单数
	if err := baseQuery.Count(&overview.Total.Total).Error; err != nil {
		return nil, err
	}

	// 获取昨日订单数
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	queryYesterday := r.db.Model(&model.Order{}).Where("is_del = 0")
	if userId > 0 {
		queryYesterday = queryYesterday.Where("customer_id = ?", userId)
	}
	if err := queryYesterday.Where("DATE(create_time) = ?", yesterday).
		Count(&overview.Total.Yesterday).Error; err != nil {
		return nil, err
	}

	// 获取今日订单数
	today := time.Now().Format("2006-01-02")
	queryToday := r.db.Model(&model.Order{}).Where("is_del = 0")
	if userId > 0 {
		queryToday = queryToday.Where("customer_id = ? and DATE(create_time) = ?", userId, today)
	}
	if err := queryToday.Where("DATE(create_time) = ?", today).
		Count(&overview.Total.Today).Error; err != nil {
		return nil, err
	}

	// 获取订单状态统计
	queryStatus := r.db.Model(&model.Order{}).Where("is_del = 0")
	if userId > 0 {
		queryStatus = queryStatus.Where("customer_id = ? and DATE(create_time) = ?", userId, today)
	}
	if err := queryStatus.Where("DATE(create_time) = ? ", today).
		Select(
			"COUNT(CASE WHEN status in (3,10) THEN 1 END) as processing",
			"COUNT(CASE WHEN status = 4 THEN 1 END) as success",
			"COUNT(CASE WHEN status = 5 THEN 1 END) as failed",
		).
		Scan(&overview.Status).Error; err != nil {
		return nil, err
	}

	//获取昨日订单状态统计
	queryStatusYesterday := r.db.Model(&model.Order{}).Where("is_del = 0")
	if userId > 0 {
		queryStatusYesterday = queryStatusYesterday.Where("customer_id = ? and DATE(create_time) = ?", userId, yesterday)
	}
	if err := queryStatusYesterday.Where("DATE(create_time) = ?", yesterday).
		Select(
			"COUNT(CASE WHEN status in (3,10) THEN 1 END) as yesterday_processing",
			"COUNT(CASE WHEN status = 4 THEN 1 END) as yesterday_success",
			"COUNT(CASE WHEN status = 5 THEN 1 END) as yesterday_failed",
		).
		Scan(&overview.YesterdayStatus).Error; err != nil {
		return nil, err
	}

	// 获取盈利统计
	queryProfit := r.db.Model(&model.Order{}).Where("is_del = 0")
	if userId > 0 {
		queryProfit = queryProfit.Where("customer_id = ? and DATE(create_time) = ?", userId, today)
	}
	if err := queryProfit.Where("DATE(create_time) = ? and status = 4", today).
		Select(
			"COALESCE(SUM(price), 0) as cost_amount",
			"COALESCE(SUM(price - const_price), 0) as profit_amount",
		).
		Scan(&overview.Profit).Error; err != nil {
		return nil, err
	}

	return &overview, nil
}

// GetOperatorRealtimeStatistics 获取运营商实时统计
func (r *OrderRepositoryImpl) GetOperatorRealtimeStatistics(ctx context.Context, start, end time.Time) ([]model.OrderStatisticsOperator, error) {
	var stats []model.OrderStatisticsOperator
	sql := `orders.isp, COUNT(*) as total_orders`
	err := r.db.Table("orders").
		Select(sql).
		Where("DATE(orders.create_time) = ? AND orders.isp IN (1,2,3)", start.Format("2006-01-02")).
		Group("orders.isp").
		Scan(&stats).Error
	if err != nil {
		return nil, err
	}

	return stats, nil
}

// GetOperatorRealtimeStatisticsByUser 获取指定用户的运营商实时统计
func (r *OrderRepositoryImpl) GetOperatorRealtimeStatisticsByUser(ctx context.Context, start, end time.Time, userId int64) ([]model.OrderStatisticsOperator, error) {
	var stats []model.OrderStatisticsOperator
	sql := `orders.isp, COUNT(*) as total_orders`
	err := r.db.Table("orders").
		Select(sql).
		Where("DATE(orders.create_time) = ? AND orders.isp IN (1,2,3) AND orders.customer_id = ?", start.Format("2006-01-02"), userId).
		Group("orders.isp").
		Scan(&stats).Error
	if err != nil {
		return nil, err
	}

	return stats, nil
}

// GetOperatorOrderCount 按运营商分组统计订单总数
func (r *OrderRepositoryImpl) GetOperatorOrderCount(ctx context.Context, start, end time.Time) ([]model.OperatorOrderCount, error) {
	var result []model.OperatorOrderCount
	err := r.db.Table("orders").
		Select("products.isp as operator, COUNT(*) as total").
		Joins("JOIN products ON orders.product_id = products.id").
		Where("DATE(orders.create_time) BETWEEN ? AND ?", start.Format("2006-01-02"), end.Format("2006-01-02")).
		Group("products.isp").
		Scan(&result).Error
	return result, err
}

// GetByUserID 根据用户ID获取订单列表
func (r *OrderRepositoryImpl) GetByUserID(ctx context.Context, userID int64, params map[string]interface{}, page, pageSize int) ([]*model.Order, int64, error) {
	var orders []*model.Order
	var total int64
	fmt.Println("userID#########################", userID)
	query := r.db.Model(&model.Order{}).Where("is_del = 0 AND customer_id = ?", userID)

	// 添加其他查询条件
	for key, value := range params {
		if key == "user_id" {
			continue // 跳过 user_id，因为已经作为基础条件
		}
		strValue, ok := value.(string)
		if !ok || strValue == "" {
			continue
		}
		switch key {
		case "client":
			clientID, err := strconv.ParseInt(strValue, 10, 64)
			if err != nil {
				logger.Error("解析client参数失败: %v", err)
				continue
			}
			if clientID > 0 {
				query = query.Where("client = ?", clientID)
			}
		case "status":
			status, err := strconv.ParseInt(strValue, 10, 64)
			if err != nil {
				logger.Error("解析status参数失败: %v", err)
				continue
			}
			if status >= 0 {
				query = query.Where("status = ?", status)
			}
		case "order_number":
			query = query.Where("order_number LIKE ?", "%"+strValue+"%")
		case "mobile":
			query = query.Where("mobile LIKE ?", "%"+strValue+"%")
		case "start_time":
			query = query.Where("create_time >= ?", strValue)
		case "end_time":
			query = query.Where("create_time <= ?", strValue)
		case "platform_code":
			query = query.Where("platform_code = ?", strValue)
		default:
			query = query.Where(key+" = ?", strValue)
		}
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Order("create_time DESC").Offset(offset).Limit(pageSize).Find(&orders).Error; err != nil {
		return nil, 0, err
	}

	return orders, total, nil
}

// SoftDeleteByID 软删除订单
func (r *OrderRepositoryImpl) SoftDeleteByID(ctx context.Context, id int64) error {
	return r.db.Model(&model.Order{}).Where("id = ?", id).Update("is_del", 1).Error
}
