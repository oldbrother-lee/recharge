package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"recharge-go/internal/model"
	"recharge-go/internal/repository"
	"recharge-go/internal/service"
	"recharge-go/internal/utils"
	"recharge-go/pkg/database"
	"recharge-go/pkg/logger"
	"recharge-go/pkg/signature"
	"recharge-go/pkg/utils/response"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// KekebangOrderController 可客帮订单控制器
type KekebangOrderController struct {
	orderService    service.OrderService
	rechargeService service.RechargeService
}

// NewKekebangOrderController 创建可客帮订单控制器
func NewKekebangOrderController(orderService service.OrderService, rechargeService service.RechargeService) *KekebangOrderController {
	return &KekebangOrderController{
		orderService:    orderService,
		rechargeService: rechargeService,
	}
}

func (c *KekebangOrderController) verifyProductExists(productID int64) (*model.Product, error) {
	fmt.Printf("[kekebang] 开始验证产品是否存在, 产品ID: %d\n", productID)

	var product model.Product
	err := database.DB.Model(&model.Product{}).
		Where("id = ?", productID).
		First(&product).Error

	if err != nil {
		fmt.Printf("[kekebang] 验证产品失败: %v\n", err)
		return nil, err
	}

	fmt.Printf("[kekebang] 产品验证通过, 产品ID: %d\n", productID)
	return &product, nil
}

// CreateOrder 创建订单
func (c *KekebangOrderController) CreateOrder(ctx *gin.Context) {
	userid := ctx.Param("userid")
	// 1. 查询 platform_accounts 表，找到 account_name = userid 的账号
	accountRepo := repository.NewPlatformRepository(database.DB)
	account, err := accountRepo.GetPlatformAccountByAccountName(userid)
	if err != nil || account == nil {
		utils.Error(ctx, http.StatusBadRequest, "无效的账号标识")
		return
	}

	// 2. 可通过 account.PlatformID 查询平台信息
	platform, err := accountRepo.GetPlatformByID(account.PlatformID)
	if err != nil || platform == nil {
		utils.Error(ctx, http.StatusBadRequest, "无效的平台")
		return
	}

	var req model.KekebangOrderRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		logger.Error("【解析请求参数失败】error: %v", err)
		response := gin.H{
			"code":    "FAIL",
			"message": "参数错误",
			"data":    gin.H{},
		}
		ctx.JSON(http.StatusOK, response)
		return
	}

	// 记录原始请求数据
	logger.Info(fmt.Sprintf("【收到可客帮订单请求】request: %+v", req))
	//先检查订单是否存在
	order, err := c.orderService.GetOrderByOutTradeNum(ctx, strconv.FormatInt(req.UserOrderID, 10))
	if err != nil && err != gorm.ErrRecordNotFound {
		logger.Log.Error("查询订单失败",
			zap.Error(err),
			zap.String("order_id", strconv.FormatInt(req.UserOrderID, 10)))
		response := gin.H{
			"code":    "FAIL",
			"message": "产品不存在",
			"data":    gin.H{},
		}
		ctx.JSON(http.StatusOK, response)
		return
	}
	if order != nil {
		response := gin.H{
			"code":    "FAIL",
			"message": "订单已存在",
			"data": gin.H{
				"createTime": order.CreateTime.Format("2006-01-02T15:04:05+0800"),
				"orderId":    req.UserOrderID,
				"orderNo":    order.OrderNumber,
			},
		}
		ctx.JSON(http.StatusOK, response)
		return
	}
	productID, err := strconv.ParseInt(req.OuterGoodsCode, 10, 64)
	if err != nil {
		logger.Error(fmt.Sprintf("【产品编码转换失败】error: %v", err))
		utils.Error(ctx, 500, "产品编码转换失败")
		return
	}
	product, err := c.verifyProductExists(productID)
	if err != nil {
		logger.Log.Error("产品验证失败",
			zap.Error(err),
			zap.Int64("product_id", productID),
			zap.String("request_id", ctx.GetString("request_id")))
		response := gin.H{
			"code":    "FAIL",
			"message": "产品不存在",
			"data":    gin.H{},
		}
		ctx.JSON(http.StatusOK, response)
		return
	}
	// 验证签名
	// if !c.verifySign(req) {
	// 	logger.Error("【签名验证失败】request: %+v", req)
	// 	response.Error(ctx, 400, "签名验证失败")
	// 	return
	// }

	// 创建订单
	order = &model.Order{
		Mobile:            req.Target,
		Denom:             req.Datas.Amount,
		Price:             product.Price,
		ProductID:         productID, // 需要根据 OuterGoodsCode 查询对应的商品ID
		Status:            model.OrderStatusPendingRecharge,
		Client:            3,                                          // 标识为自动取单任务，保持待充值状态
		OutTradeNum:       strconv.FormatInt(req.UserOrderID, 10),     // 外部交易号
		ISP:               getISPFromOperatorID(req.Datas.OperatorID), // 根据运营商ID获取ISP
		Param1:            req.Datas.ProvCode,                         // 省份代码
		Param2:            req.GoodsID,                                // 商品名称
		Param3:            req.GoodsName,                              // 外部商品编码
		Remark:            fmt.Sprintf("可客帮订单，商品ID：%s", req.GoodsID),
		AccountLocation:   req.Datas.ProvCode,
		PlatformAccountID: account.ID,
		CustomerID:        *account.BindUserID,
		PlatformId:        platform.ID,
		PlatformCode:      platform.Code,
		PlatformName:      platform.Name,
	}

	// 调用订单服务创建订单
	if err := c.orderService.CreateOrder(ctx, order); err != nil {
		logger.Error(fmt.Sprintf("【创建订单失败】error: %v", err))
		response := gin.H{
			"code":    "FAIL",
			"message": "产品不存在",
			"data":    gin.H{},
		}
		ctx.JSON(http.StatusOK, response)
		return
	}

	// 创建充值任务
	if err := c.rechargeService.CreateRechargeTask(ctx, order.ID); err != nil {
		logger.Error("【创建充值任务失败】error: %v", err)
		utils.Error(ctx, 500, "创建充值任务失败")
		return
	}
	response := gin.H{
		"code":    "SUCCESS",
		"message": "订单创建成功",
		"data": gin.H{
			"order_id": order.OutTradeNum,
			"status":   2,
		},
	}
	ctx.JSON(http.StatusOK, response)
}

// verifySign 验证签名
func (c *KekebangOrderController) verifySign(req model.KekebangOrderRequest) bool {
	// 将请求参数转换为 map
	jsonData, err := json.Marshal(req)
	if err != nil {
		logger.Error("【序列化请求参数失败】error: %v", err)
		return false
	}

	var params map[string]interface{}
	if err := json.Unmarshal(jsonData, &params); err != nil {
		logger.Error("【反序列化请求参数失败】error: %v", err)
		return false
	}

	// TODO: 从配置或数据库获取 secretKey
	secretKey := "ab4e90e8bd504a5a8d290b5d4b8235c9"

	// 验证签名
	return signature.VerifyKekebangSign(params, req.Sign, secretKey)
}

// getISPFromOperatorID 根据运营商ID获取ISP
func getISPFromOperatorID(operatorID string) int {
	//中国移动
	operatorName := strings.TrimPrefix(operatorID, "中国")
	switch operatorName {
	case "1":
		return 1 // 移动
	case "2":
		return 3 // 联通
	case "3":
		return 2 // 电信
	case "移动":
		return 1 // 移动
	case "联通":
		return 3 // 联通
	case "电信":
		return 2 // 电信
	case "虚拟":
		return 4 // 虚拟
	default:
		return 0 // 未知
	}
}

// QueryOrder 查询订单
func (c *KekebangOrderController) QueryOrder(ctx *gin.Context) {
	var req struct {
		AppKey      string `json:"app_key" binding:"required"`       // 应用密钥
		UserOrderID int64  `json:"user_order_id" binding:"required"` // 用户订单ID
		Sign        string `json:"sign" binding:"required"`          // 签名
		Timestamp   int64  `json:"timestamp" binding:"required"`     // 时间戳
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		logger.Error("【解析请求参数失败】error: %v", err)
		response.Error(ctx, 400, "解析请求参数失败")
		return
	}

	// 记录原始请求数据
	logger.Info("【收到可客帮订单查询请求】request: %+v", req)
	userid := ctx.Param("userid")
	// 1. 查询 platform_accounts 表，找到 account_name = userid 的账号
	accountRepo := repository.NewPlatformRepository(database.DB)
	account, err := accountRepo.GetPlatformAccountByAccountName(userid)
	if err != nil || account == nil {
		logger.Error("【无效的账号标识】userid: %s", userid)
		utils.Error(ctx, http.StatusBadRequest, "无效的账号标识")
		return
	}
	// 验证签名
	params := map[string]interface{}{
		"app_key":       req.AppKey,
		"user_order_id": req.UserOrderID,
		"timestamp":     req.Timestamp,
	}

	// TODO: 从配置或数据库获取 secretKey
	secretKey := account.AppSecret

	if !signature.VerifyKekebangSign(params, req.Sign, secretKey) {
		logger.Error("【签名验证失败】request: %+v", req)
		utils.Error(ctx, 400, "签名验证失败")
		return
	}

	// 查询订单
	order, err := c.orderService.GetOrderByOutTradeNum(ctx, strconv.FormatInt(req.UserOrderID, 10))
	if err != nil {
		// 判断是否为记录不存在错误
		if err.Error() == "record not found" {
			logger.Info("【订单不存在】order_id: %d", req.UserOrderID)
			response := gin.H{
				"code":    0,
				"message": "success",
				"data": gin.H{
					"status":   3,
					"rsp_info": "订单不存在或已失效",
					"rsp_time": time.Now().Unix(),
				},
			}
			ctx.JSON(http.StatusOK, response)
			return
		}
		// 其他数据库错误
		logger.Error("【查询订单失败】error: %v", err)
		utils.Error(ctx, 500, "查询订单失败")
		return
	}

	if order == nil {
		logger.Info("【订单查询结果为空】order_id: %d", req.UserOrderID)
		response := gin.H{
			"code":    0,
			"message": "success",
			"data": gin.H{
				"status":   3,
				"rsp_info": "订单不存在或已失效",
				"rsp_time": time.Now().Unix(),
			},
		}
		ctx.JSON(http.StatusOK, response)
		return
	}

	// 转换订单状态为可客帮状态
	status, rsp_info := getKekebangOrderStatusAndInfo(order)
	response := gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"status":   status,
			"rsp_info": rsp_info,
			"rsp_time": time.Now().Unix(),
		},
	}
	ctx.JSON(http.StatusOK, response)
}
func getKekebangOrderStatusAndInfo(order *model.Order) (int, string) {
	switch order.Status {
	case model.OrderStatusPendingPayment, model.OrderStatusPendingRecharge, model.OrderStatusRecharging:
		return 1, "充值中"
	case model.OrderStatusSuccess:
		return 2, "充值成功"
	case model.OrderStatusFailed:
		return 3, order.Remark
	case model.OrderStatusRefunded:
		return 4, "已退款"
	case model.OrderStatusCancelled:
		return 3, "订单已取消"
	case model.OrderStatusPartial:
		return 3, "部分充值"
	case model.OrderStatusSplit:
		return 3, "订单已拆单"
	default:
		return 0, "未知状态"
	}
}

// convertOrderStatus 转换订单状态为可客帮状态
func convertOrderStatus(status model.OrderStatus) string {
	switch status {
	case model.OrderStatusPendingPayment:
		return "pending"
	case model.OrderStatusPendingRecharge:
		return "pending"
	case model.OrderStatusRecharging:
		return "processing"
	case model.OrderStatusSuccess:
		return "success"
	case model.OrderStatusFailed:
		return "failed"
	case model.OrderStatusRefunded:
		return "refunded"
	case model.OrderStatusCancelled:
		return "cancelled"
	case model.OrderStatusPartial:
		return "partial"
	case model.OrderStatusSplit:
		return "split"
	case model.OrderStatusProcessing:
		return "processing"
	default:
		return "unknown"
	}
}

// getFinishTime 获取完成时间
func getFinishTime(t *time.Time) int64 {
	if t == nil {
		return 0
	}
	return t.Unix()
}
