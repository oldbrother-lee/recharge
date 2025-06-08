package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"recharge-go/internal/model"
	"recharge-go/internal/repository"
	"recharge-go/internal/service"
	"recharge-go/internal/utils"
	"recharge-go/pkg/database"
	"recharge-go/pkg/logger"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type MF178OrderController struct {
	orderService    service.OrderService
	rechargeService service.RechargeService
}

func NewMF178OrderController(
	orderService service.OrderService,
	rechargeService service.RechargeService,
) *MF178OrderController {
	return &MF178OrderController{
		orderService:    orderService,
		rechargeService: rechargeService,
	}
}

// Int64String 支持字符串和数字类型的互转
type Int64String int64

// UnmarshalJSON 实现自定义 JSON 解析
func (i *Int64String) UnmarshalJSON(data []byte) error {
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case float64:
		*i = Int64String(value)
	case string:
		// 如果是字符串，尝试转换为 int64
		if value == "" {
			*i = 0
			return nil
		}
		// 先尝试解析为 float64，再转换为 int64
		floatVal, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		*i = Int64String(floatVal)
	default:
		*i = 0
	}
	return nil
}

// MarshalJSON 实现自定义 JSON 序列化
func (i Int64String) MarshalJSON() ([]byte, error) {
	return json.Marshal(int64(i))
}

// 定义请求结构体，完全匹配外部系统推送的数据格式
type MF178OrderRequest struct {
	AppKey      string `json:"app_key" binding:"required"`       // 应用密钥
	UserOrderID int64  `json:"user_order_id" binding:"required"` // 用户订单号
	Datas       struct {
		OperatorID string  `json:"operator_id"` // 运营商
		ProvCode   string  `json:"prov_code"`   // 省份
		Amount     float64 `json:"amount"`      // 金额
	} `json:"datas" binding:"required"`
	VenderID         int         `json:"vender_id" binding:"required"`          // 供应商ID
	Target           string      `json:"target" binding:"required"`             // 目标手机号
	GoodsID          int64       `json:"goods_id" binding:"required"`           // 商品ID
	GoodsName        string      `json:"goods_name" binding:"required"`         // 商品名称
	OuterGoodsCode   string      `json:"outer_goods_code" binding:"required"`   // 外部商品编码
	OfficialPayment  Int64String `json:"official_payment" binding:"required"`   // 官方支付金额
	UserQuoteType    Int64String `json:"user_quote_type" binding:"required"`    // 用户报价类型
	UserQuotePayment Int64String `json:"user_quote_payment" binding:"required"` // 用户报价支付金额，改为 string 类型
	UserPayment      Int64String `json:"user_payment" binding:"required"`       // 用户支付金额
	Timestamp        int64       `json:"timestamp" binding:"required"`          // 时间戳
	Sign             string      `json:"sign" binding:"required"`               // 签名
}

// CreateOrder 创建订单
func (c *MF178OrderController) CreateOrder(ctx *gin.Context) {
	logger.Log.Info("开始处理创建订单请求")

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

	// 1. 读取请求体
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		logger.Log.Error("读取请求体失败",
			zap.Error(err),
			zap.String("request_id", ctx.GetString("request_id")))
		utils.Error(ctx, http.StatusBadRequest, "读取请求体失败")
		return
	}
	// 恢复请求体
	ctx.Request.Body = io.NopCloser(bytes.NewBuffer(body))
	logger.Log.Info("原始请求体",
		zap.String("body", string(body)),
		zap.String("request_id", ctx.GetString("request_id")))

	// 2. 解析请求参数
	var req MF178OrderRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		logger.Log.Error("参数绑定失败",
			zap.Error(err),
			zap.String("body", string(body)))
		response := gin.H{
			"code":    "FAIL",
			"message": "参数绑定失败",
			"data":    gin.H{},
		}
		ctx.JSON(http.StatusOK, response)
		return
	}
	logger.Log.Info("请求参数",
		zap.Any("request", req),
		zap.String("request_id", ctx.GetString("request_id")))

	// 3. 检查订单是否已存在
	order, err := c.orderService.GetOrderByOutTradeNum(ctx, strconv.FormatInt(req.UserOrderID, 10))
	if err != nil && err != gorm.ErrRecordNotFound {
		logger.Log.Error("查询订单失败",
			zap.Error(err),
			zap.String("order_id", strconv.FormatInt(req.UserOrderID, 10)))
		utils.Error(ctx, http.StatusInternalServerError, "查询订单失败1")
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

	// // 4. 转换金额字段
	// officialPayment, err := strconv.ParseFloat(req.OfficialPayment, 64)
	// if err != nil {
	// 	logger.Log.Error("官方支付金额转换失败",
	// 		zap.Error(err),
	// 		zap.String("original_value", req.OfficialPayment),
	// 		zap.String("request_id", ctx.GetString("request_id")))
	// 	utils.Error(ctx, http.StatusBadRequest, "无效的官方支付金额")
	// 	return
	// }
	// logger.Log.Info("官方支付金额",
	// 	zap.Float64("amount", officialPayment),
	// 	zap.String("request_id", ctx.GetString("request_id")))

	// userPayment, err := strconv.ParseFloat(req.UserPayment, 64)
	// if err != nil {
	// 	logger.Log.Error("用户支付金额转换失败",
	// 		zap.Error(err),
	// 		zap.String("original_value", req.UserPayment),
	// 		zap.String("request_id", ctx.GetString("request_id")))
	// 	utils.Error(ctx, http.StatusBadRequest, "无效的用户支付金额")
	// 	return
	// }

	// // 转换用户报价支付金额
	// userQuotePayment, err := strconv.ParseFloat(req.UserQuotePayment, 64)
	// if err != nil {
	// 	logger.Log.Error("用户报价支付金额转换失败",
	// 		zap.Error(err),
	// 		zap.String("original_value", req.UserQuotePayment),
	// 		zap.String("request_id", ctx.GetString("request_id")))
	// 	utils.Error(ctx, http.StatusBadRequest, "无效的用户报价支付金额")
	// 	return
	// }
	// 将浮点数转换为整数
	// userQuotePaymentInt := int(userQuotePayment)

	// 5. 记录订单信息到日志文件
	logger.Log.Info("收到新订单请求",
		zap.String("platform", "mf178"),
		zap.Int64("order_id", req.UserOrderID),
		zap.String("mobile", req.Target),
		zap.String("outer_goods_code", req.OuterGoodsCode),
		zap.String("operator_id", req.Datas.OperatorID),
		zap.Float64("amount", float64(req.OfficialPayment)),
		zap.String("raw_data", string(body)),
		zap.String("app_key", req.AppKey),
		zap.Int("vender_id", req.VenderID),
		zap.Int64("goods_id", req.GoodsID),
		zap.String("goods_name", req.GoodsName),
		zap.Float64("official_payment", float64(req.OfficialPayment)),
		zap.Int("user_quote_type", int(req.UserQuoteType)),
		zap.Int("user_quote_payment", int(req.UserQuotePayment)),
		zap.Float64("user_payment", float64(req.UserPayment)),
		zap.Int64("timestamp", req.Timestamp))

	// 6. 转换产品编码
	productID, err := strconv.ParseInt(req.OuterGoodsCode, 10, 64)
	if err != nil {
		logger.Log.Error("产品编码转换失败",
			zap.Error(err),
			zap.String("original_code", req.OuterGoodsCode),
			zap.String("request_id", ctx.GetString("request_id")))
		utils.Error(ctx, http.StatusBadRequest, "无效的产品编码")
		return
	}
	logger.Log.Info("产品编码转换成功",
		zap.Int64("product_id", productID),
		zap.String("request_id", ctx.GetString("request_id")))

	// 7. 验证产品是否存在
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
	fmt.Printf("product-------: %+v", product)
	logger.Log.Info("产品验证通过",
		zap.Int64("product_id", productID),
		zap.String("request_id", ctx.GetString("request_id")))

	//运营商转换
	isp := 1
	if req.Datas.OperatorID == "移动" {
		isp = 1
	} else if req.Datas.OperatorID == "电信" {
		isp = 2
	} else if req.Datas.OperatorID == "联通" {
		isp = 3
	}
	fmt.Printf("platform-------: %+v", platform)
	order = &model.Order{
		Mobile:            req.Target,
		ProductID:         productID,
		OutTradeNum:       strconv.FormatInt(req.UserOrderID, 10),
		Denom:             req.Datas.Amount,
		OfficialPayment:   float64(req.OfficialPayment),
		UserQuotePayment:  float64(req.UserQuotePayment),
		UserPayment:       float64(req.UserPayment),
		Price:             product.Price,
		Status:            model.OrderStatusPendingRecharge,
		IsDel:             0,
		Client:            3,   // 3代表MF178
		ISP:               isp, // 1代表移动
		Param1:            req.Datas.OperatorID,
		AccountLocation:   req.Datas.ProvCode,
		Param3:            req.GoodsName,
		PlatformAccountID: account.ID,
		CustomerID:        *account.BindUserID,
		PlatformId:        platform.ID,
		PlatformCode:      platform.Code,
		PlatformName:      platform.Name,
	}
	logger.Log.Info("准备创建订单",
		zap.Any("order", order),
		zap.String("request_id", ctx.GetString("request_id")))

	if err := c.orderService.CreateOrder(ctx, order); err != nil {
		logger.Log.Error("创建订单失败",
			zap.Error(err),
			zap.String("request_id", ctx.GetString("request_id")))
		response := gin.H{
			"code":    "FAIL",
			"message": "创建订单失败",
			"data":    gin.H{},
		}
		ctx.JSON(http.StatusOK, response)
		return
	}
	logger.Log.Info("订单创建成功",
		zap.Any("order", order),
		zap.String("request_id", ctx.GetString("request_id")))

	// 9. 创建充值任务
	logger.Log.Info("【准备创建充值任务】",
		zap.Int64("order_id", order.ID),
		zap.String("order_number", order.OrderNumber))

	if err := c.rechargeService.CreateRechargeTask(ctx, order.ID); err != nil {
		logger.Log.Error("【创建充值任务失败】",
			zap.Int64("order_id", order.ID),
			zap.String("order_number", order.OrderNumber),
			zap.Error(err))
		// 这里可以选择是否返回错误，因为订单已经创建成功
	} else {
		logger.Log.Info("【创建充值任务成功】",
			zap.Int64("order_id", order.ID),
			zap.String("order_number", order.OrderNumber))
	}

	// 10. 返回成功响应
	response := gin.H{
		"message": "创建订单成功",
		"code":    "SUCCESS",
		"data": gin.H{
			"createTime": order.CreateTime.Format("2006-01-02T15:04:05+0800"),
			"orderId":    req.UserOrderID,
			"orderNo":    order.OrderNumber,
		},
	}
	logger.Log.Info("返回响应",
		zap.Any("response", response),
		zap.String("request_id", ctx.GetString("request_id")))
	ctx.JSON(http.StatusOK, response)
}

// QueryOrder 查询订单状态
func (c *MF178OrderController) QueryOrder(ctx *gin.Context) {
	if logger.Log == nil {
		// 如果 logger 未初始化，使用 fmt 打印
		fmt.Printf("[MF178Order] 开始处理查询订单请求\n")
	} else {
		logger.Log.Info("开始处理查询订单请求",
			zap.String("request_id", ctx.GetString("request_id")))
	}

	// 读取原始请求体
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		if logger.Log == nil {
			fmt.Printf("[MF178Order] 读取请求体失败: %v\n", err)
		} else {
			logger.Log.Error("读取请求体失败",
				zap.Error(err),
				zap.String("request_id", ctx.GetString("request_id")))
		}
		response := gin.H{
			"code":    1,
			"message": "读取请求体失败",
			"data":    gin.H{},
		}
		ctx.JSON(http.StatusOK, response)
		return
	}
	// 恢复请求体
	ctx.Request.Body = io.NopCloser(bytes.NewBuffer(body))
	logger.Log.Info("原始请求体",
		zap.String("body", string(body)),
		zap.String("request_id", ctx.GetString("request_id")))

	var req struct {
		AppKey      string `json:"app_key" binding:"required"`       // 应用密钥
		UserOrderID int64  `json:"user_order_id" binding:"required"` // 用户订单号
		Timestamp   int64  `json:"timestamp" binding:"required"`     // 时间戳
		Sign        string `json:"sign" binding:"required"`          // 签名
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		logger.Log.Error("参数绑定失败",
			zap.Error(err),
			zap.String("body", string(body)),
			zap.String("request_id", ctx.GetString("request_id")))
		response := gin.H{
			"code":    1,
			"message": "参数绑定失败",
			"data":    gin.H{},
		}
		ctx.JSON(http.StatusOK, response)
		return
	}

	logger.Log.Info("开始查询订单",
		zap.Int64("order_id", req.UserOrderID),
		zap.String("request_id", ctx.GetString("request_id")))

	// 查询订单
	order, err := c.orderService.GetOrderByOutTradeNum(ctx, strconv.FormatInt(req.UserOrderID, 10))
	if err != nil {
		logger.Log.Info("查询订单失败",
			zap.Error(err),
			zap.Int64("order_id", req.UserOrderID),
			zap.String("request_id", ctx.GetString("request_id")))
		response := gin.H{
			"code":    1,
			"message": "查询订单失败",
			"data":    gin.H{},
		}
		ctx.JSON(http.StatusOK, response)
		return
	}

	if order == nil {
		response := gin.H{
			"code":    1,
			"message": "订单不存在",
			"data":    gin.H{},
		}
		ctx.JSON(http.StatusOK, response)
		return
	}

	// 根据订单状态获取状态码和描述
	status, rspInfo := getOrderStatusAndInfo(order)
	logger.Log.Info("获取订单状态",
		zap.Int("status_code", status),
		zap.String("status_info", rspInfo),
		zap.Int64("order_id", req.UserOrderID),
		zap.String("request_id", ctx.GetString("request_id")))

	// 构建成功响应
	response := gin.H{
		"code":    0,
		"message": "",
		"data": gin.H{
			"status":   status,
			"rsp_info": rspInfo,
			"rsp_time": time.Now().Unix(),
		},
	}

	logger.Log.Info("查询订单成功",
		zap.Any("response", response),
		zap.Int64("order_id", req.UserOrderID),
		zap.String("request_id", ctx.GetString("request_id")))
	ctx.JSON(http.StatusOK, response)
}

// verifyProductExists 验证产品是否存在
func (c *MF178OrderController) verifyProductExists(productID int64) (*model.Product, error) {
	fmt.Printf("[MF178Order] 开始验证产品是否存在, 产品ID: %d\n", productID)

	var product model.Product
	err := database.DB.Model(&model.Product{}).
		Where("id = ?", productID).
		First(&product).Error

	if err != nil {
		fmt.Printf("[MF178Order] 验证产品失败: %v\n", err)
		return nil, err
	}

	fmt.Printf("[MF178Order] 产品验证通过, 产品ID: %d\n", productID)
	return &product, nil
}

// getOrderStatusAndInfo 根据订单状态获取状态码和描述
func getOrderStatusAndInfo(order *model.Order) (int, string) {
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
