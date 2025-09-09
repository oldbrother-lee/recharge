package controller

import (
	"fmt"
	"net/http"
	"strconv"

	"recharge-go/internal/model"
	"recharge-go/internal/repository"
	"recharge-go/internal/service"
	"recharge-go/pkg/database"
	"recharge-go/pkg/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// XianyinkeOrderController 闲赢客订单控制器
type XianyinkeOrderController struct {
	orderService    service.OrderService
	rechargeService service.RechargeService
}

// NewXianyinkeOrderController 创建闲赢客订单控制器
func NewXianyinkeOrderController(orderService service.OrderService, rechargeService service.RechargeService) *XianyinkeOrderController {
	return &XianyinkeOrderController{
		orderService:    orderService,
		rechargeService: rechargeService,
	}
}

// CreateOrder 接收闲赢客推送订单
// 路由：POST /api/v1/xianyinke/order/:userid
// 成功：{"result":"success","user_no":"<外部订单号>"}
// 失败：{"result":"fail"}
func (c *XianyinkeOrderController) CreateOrder(ctx *gin.Context) {
	userid := ctx.Param("userid")

	// 1) 通过 userid 查询平台账号与平台信息
	accountRepo := repository.NewPlatformRepository(database.DB)
	account, err := accountRepo.GetPlatformAccountByAccountName(userid)
	if err != nil || account == nil {
		logger.Log.Error("[xianyinke] 查询平台账号失败", zap.Error(err), zap.String("userid", userid))
		ctx.JSON(http.StatusOK, gin.H{"result": "fail"})
		return
	}
	platform, err := accountRepo.GetPlatformByID(account.PlatformID)
	if err != nil || platform == nil {
		ctx.JSON(http.StatusOK, gin.H{"result": "fail"})
		return
	}

	// 2) 解析请求体
	var req model.XianyinkeOrderRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		logger.Error("[xianyinke] 解析请求参数失败: %v", err)
		ctx.JSON(http.StatusOK, gin.H{"result": "fail"})
		return
	}

	// 3) 验签
	// raw, _ := json.Marshal(req)
	// params := map[string]interface{}{}
	// _ = json.Unmarshal(raw, &params)
	// // VerifyXianyinkeSign 内部会忽略 sign 字段
	// if !signature.VerifyXianyinkeSign(params, req.Sign, account.AppSecret) {
	// 	logger.Info("[xianyinke] 签名验证失败", zap.Any("params", params))
	// 	ctx.JSON(http.StatusOK, gin.H{"result": "fail"})
	// 	return
	// }

	// 4) 幂等：以 id 作为外部订单号
	outTradeNum := strconv.FormatInt(req.ID, 10)
	existOrder, err := c.orderService.GetOrderByOutTradeNum(ctx, outTradeNum)
	if err != nil && err != gorm.ErrRecordNotFound {
		logger.Log.Error("[xianyinke] 查询订单失败", zap.Error(err), zap.String("out_trade_num", outTradeNum))
		ctx.JSON(http.StatusOK, gin.H{"result": "fail"})
		return
	}
	if existOrder != nil {
		ctx.JSON(http.StatusOK, gin.H{"result": "success", "user_no": outTradeNum})
		return
	}

	// 5) 从 chan_pro_code 解析内部商品ID
	productID, err := strconv.ParseInt(req.ChanProCode, 10, 64)
	if err != nil {
		logger.Log.Error("[xianyinke] chan_pro_code 非法", zap.Error(err), zap.String("chan_pro_code", req.ChanProCode))
		ctx.JSON(http.StatusOK, gin.H{"result": "fail"})
		return
	}

	// 6) 校验商品存在并获取价格
	var product model.Product
	if err := database.DB.Model(&model.Product{}).Where("id = ?", productID).First(&product).Error; err != nil {
		logger.Log.Error("[xianyinke] 商品不存在", zap.Error(err), zap.Int64("product_id", productID))
		ctx.JSON(http.StatusOK, gin.H{"result": "fail"})
		return
	}

	// 6.1) 校验平台账号是否已绑定用户，避免空指针
	if account.BindUserID == nil {
		logger.Log.Warn("[xianyinke] 平台账号未绑定用户，拒绝创建订单", zap.Int64("platform_account_id", account.ID))
		ctx.JSON(http.StatusOK, gin.H{"result": "fail"})
		return
	}

	// 6.2) 按文档将 product_id(运营商编码) 与 province_id(地区编码) 转换为系统定义
	isp := getISPFromXianyinkeProductID(req.ProductID)
	if isp == 0 {
		logger.Log.Warn("[xianyinke] 未识别的运营商编码", zap.Int64("product_id", req.ProductID))
	}
	accountLocation := xianyinkeProvinceMap[req.ProvinceID]
	if accountLocation == "" {
		logger.Log.Warn("[xianyinke] 未识别的省份编码", zap.String("province_id", req.ProvinceID))
	}

	// 7) 组装订单并创建
	denom := float64(req.MarketPrice)
	order := &model.Order{
		Mobile:            req.Account,
		Denom:             denom,
		Price:             product.Price,
		ProductID:         productID,
		Status:            model.OrderStatusPendingRecharge,
		Client:            3,
		OutTradeNum:       outTradeNum,
		Remark:            fmt.Sprintf("闲赢客订单，商品ID：%d", productID),
		PlatformAccountID: account.ID,
		CustomerID:        *account.BindUserID,
		PlatformId:        platform.ID,
		PlatformCode:      platform.Code,
		PlatformName:      platform.Name,
		ISP:               isp,
		AccountLocation:   accountLocation,
	}

	if err := c.orderService.CreateOrder(ctx, order); err != nil {
		logger.Error(fmt.Sprintf("[xianyinke] 创建订单失败: %v, 订单: %v", err, order))
		ctx.JSON(http.StatusOK, gin.H{"result": "创建订单失败"})
		return
	}

	// 8) 入充值队列
	if err := c.rechargeService.CreateRechargeTask(ctx, order.ID); err != nil {
		logger.Error("[xianyinke] 创建充值任务失败: %v", err)
		ctx.JSON(http.StatusOK, gin.H{"result": "fail"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"result": "success", "user_no": outTradeNum})
}

// getISPFromXianyinkeProductID 将闲赢客的 product_id(运营商编码) 转为系统ISP编码
// 文档: 415-移动, 418-联通, 419-电信
// 系统: 1-移动, 2-电信, 3-联通
func getISPFromXianyinkeProductID(productID int64) int {
	switch productID {
	case 415:
		return 1 // 移动 -> 1
	case 418:
		return 3 // 联通 -> 3
	case 419:
		return 2 // 电信 -> 2
	default:
		return 0 // 未知
	}
}

// xianyinkeProvinceMap 省份编码到名称的映射
var xianyinkeProvinceMap = map[string]string{
	"11": "北京",
	"12": "天津",
	"13": "河北",
	"14": "山西",
	"15": "内蒙古",
	"21": "辽宁",
	"22": "吉林",
	"23": "黑龙江",
	"31": "上海",
	"32": "江苏",
	"33": "浙江",
	"34": "安徽",
	"35": "福建",
	"36": "江西",
	"37": "山东",
	"41": "河南",
	"42": "湖北",
	"43": "湖南",
	"44": "广东",
	"45": "广西",
	"46": "海南",
	"50": "重庆",
	"51": "四川",
	"52": "贵州",
	"53": "云南",
	"54": "西藏",
	"61": "陕西",
	"62": "甘肃",
	"63": "青海",
	"64": "宁夏",
	"65": "新疆",
}
