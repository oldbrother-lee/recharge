package controller

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"recharge-go/internal/repository"
	"recharge-go/internal/service"
	"recharge-go/internal/utils"
	"recharge-go/pkg/logger"
	"recharge-go/pkg/signature"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type CallbackController struct {
	rechargeService service.RechargeService
	platformRepo    repository.PlatformRepository
	orderRepo       repository.OrderRepository
}

func NewCallbackController(rechargeService service.RechargeService, platformRepo repository.PlatformRepository, orderRepo repository.OrderRepository) *CallbackController {
	return &CallbackController{
		rechargeService: rechargeService,
		platformRepo:    platformRepo,
		orderRepo:       orderRepo,
	}
}

// KekebangCallbackRequest 客帮帮回调请求结构
type KekebangCallbackRequest struct {
	OrderID    string `json:"order_id"`    // 平台订单号
	TerraceID  string `json:"terrace_id"`  // 没用
	Account    string `json:"account"`     // 充值账号
	Time       string `json:"time"`        // 回调时间
	Amount     string `json:"amount"`      // 充值金额
	OrderState string `json:"order_state"` // 订单状态
	Sign       string `json:"sign"`        // 签名
}

// MishiCallbackRequest 秘史平台回调参数
type MishiCallbackRequest struct {
	SzAgentId      string  `form:"szAgentId" json:"szAgentId"`
	SzOrderId      string  `form:"szOrderId" json:"szOrderId"`
	SzPhoneNum     string  `form:"szPhoneNum" json:"szPhoneNum"`
	NDemo          float64 `form:"nDemo" json:"nDemo"`
	FSalePrice     float64 `form:"fSalePrice" json:"fSalePrice"`
	NFlag          int     `form:"nFlag" json:"nFlag"`
	SzRtnMsg       string  `form:"szRtnMsg" json:"szRtnMsg"`
	SzVerifyString string  `form:"szVerifyString" json:"szVerifyString"`
}

// HandleKekebangCallback 处理客帮帮回调
func (c *CallbackController) HandleKekebangCallback(ctx *gin.Context) {
	// 从URL中获取userid
	userID := ctx.Param("userid")
	if userID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "missing userid"})
		return
	}

	// 获取账号信息
	account, err := c.platformRepo.GetPlatformAccountByAccountName(userID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "failed to get account info"})
		return
	}

	// 读取请求体
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "failed to read request body"})
		return
	}

	// 解析请求体
	var data map[string]interface{}

	if err := json.Unmarshal(body, &data); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid request body"})
		return
	}
	// 获取签名
	sign, ok := data["sign"].(string)
	if !ok {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code": "1001",
			"msg":  "invalid sign",
		})
		return
	}

	// 使用账号的AppSecret验证签名
	if !verifySignature(body, sign, account.AppSecret) {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code": "1001",
			"msg":  "invalid sign",
		})
		return
	}

	// 处理回调
	if err := c.rechargeService.HandleCallback(ctx, "kekebang", body); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "failed to process callback"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"code": 200, "message": "success"})
}

// verifySignature 验证签名
func verifySignature(body []byte, sign string, secretKey string) bool {
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return false
	}
	return signature.VerifyKekebangSign(data, sign, secretKey)
}

// HandleMishiCallback 处理秘史平台回调
func (c *CallbackController) HandleMishiCallback(ctx *gin.Context) {
	fmt.Printf("[mishi] 处理秘史平台回调!!!\n")
	// 1. 获取userid
	userIDStr := ctx.Param("userid")
	if userIDStr == "" {
		utils.ErrorWithStatus(ctx, 500, 400, "缺少userid")
		return
	}

	// 获取 appkey 等信息
	account, err := c.platformRepo.GetPlatformAccountByAccountName(userIDStr)
	if err != nil {
		logger.Error("返回：401 平台账号不存在", zap.Error(err))
		utils.ErrorWithStatus(ctx, 401, 400, "平台账号不存在")
		return
	}

	// 读取原始请求体
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		utils.Error(ctx, 400, "读取请求体失败")
		return
	}

	// 解析参数
	form, err := url.ParseQuery(string(body))
	if err != nil {
		utils.Error(ctx, 400, "参数解析失败")
		return
	}
	params := make(map[string]string)
	for k, v := range form {
		if len(v) > 0 {
			params[k] = v[0]
		}
	}

	// 签名校验
	nDemo, _ := strconv.ParseFloat(params["nDemo"], 64)
	fSalePrice, _ := strconv.ParseFloat(params["fSalePrice"], 64)
	nFlag, _ := strconv.Atoi(params["nFlag"])

	signStr := fmt.Sprintf(
		"szAgentId=%s&szOrderId=%s&szPhoneNum=%s&nDemo=%v&fSalePrice=%.1f&nFlag=%d&szKey=%s",
		params["szAgentId"],
		params["szOrderId"],
		params["szPhoneNum"],
		nDemo,
		fSalePrice,
		nFlag,
		account.AppSecret,
	)
	if signature.GetMD5(signStr) != params["szVerifyString"] {
		logger.Error("返回：401 签名校验失败")
		utils.ErrorWithStatus(ctx, 401, 401, "签名校验失败")
		return
	}

	// 业务处理交给 service
	if err := c.rechargeService.HandleCallback(ctx, "mishi", body); err != nil {
		logger.Error("返回：500 处理回调失败", zap.Error(err))
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "failed to process callback"})
		return
	}
	ctx.String(200, "success")
}

// HandleChongzhiCallback 处理充值平台回调
func (c *CallbackController) HandleChongzhiCallback(ctx *gin.Context) {
	// 1. 获取userid
	logger.Info("处理充值平台回调!!!")
	userIDStr := ctx.Param("userid")
	if userIDStr == "" {
		logger.Error("处理充值平台回调 返回：400 缺少userid")
		utils.Error(ctx, 400, "缺少userid")
		return
	}

	// 2. 验证平台账号是否存在
	_, err := c.platformRepo.GetPlatformAccountByAccountName(userIDStr)
	if err != nil {
		logger.Error("处理充值平台回调 返回：400 平台账号不存在", zap.Error(err))
		utils.Error(ctx, 400, "平台账号不存在")
		return
	}

	// 3. 读取原始请求体
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		logger.Error("处理充值平台回调 返回：400 读取请求体失败", zap.Error(err))
		utils.Error(ctx, 400, "读取请求体失败")
		return
	}

	// 打印原始回调数据用于调试
	logger.Info("收到充值平台回调数据", 
		zap.String("userid", userIDStr),
		zap.String("raw_body", string(body)),
		zap.String("content_type", ctx.GetHeader("Content-Type")),
		zap.String("user_agent", ctx.GetHeader("User-Agent")),
	)

	// 4. 调用 service 层处理业务（chongzhi平台的签名验证在ParseCallbackData中处理）
	err = c.rechargeService.HandleCallback(ctx, "chongzhi", body)
	if err != nil {
		logger.Error("处理充值平台回调 返回：500 处理回调失败", zap.Error(err))
		utils.Error(ctx, 500, err.Error())
		return
	}

	// 5. 返回成功（根据文档要求返回OK字符）
	logger.Info("处理充值平台回调 返回：200 成功")
	ctx.String(200, "OK")
}

// HandleDayuanrenCallback 处理大猿人平台回调
func (c *CallbackController) HandleDayuanrenCallback(ctx *gin.Context) {
	// 1. 获取userid
	logger.Info("处理大猿人平台回调!!!")
	userIDStr := ctx.Param("userid")
	if userIDStr == "" {
		logger.Error("处理大猿人平台回调 返回：400 缺少userid")
		utils.Error(ctx, 400, "缺少userid")
		return
	}

	// 2. 获取平台账号信息
	account, err := c.platformRepo.GetPlatformAccountByAccountName(userIDStr)
	if err != nil {
		logger.Error("处理大猿人平台回调 返回：400 平台账号不存在", zap.Error(err))
		utils.Error(ctx, 400, "平台账号不存在")
		return
	}

	// 3. 读取原始请求体
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		logger.Error("处理大猿人平台回调 返回：400 读取请求体失败", zap.Error(err))
		utils.Error(ctx, 400, "读取请求体失败")
		return
	}

	// 4. 解析表单参数
	form, err := url.ParseQuery(string(body))
	if err != nil {
		logger.Error("处理大猿人平台回调 返回：400 解析参数失败", zap.Error(err))
		utils.Error(ctx, 400, "解析参数失败")
		return
	}
	params := make(map[string]string)
	for k, v := range form {
		if len(v) > 0 {
			params[k] = v[0]
		}
	}

	// 5. 签名校验
	if !signature.VerifyDayuanrenSign(params, account.AppSecret) {
		logger.Error("处理大猿人平台回调 返回：400 签名校验失败")
		utils.Error(ctx, 400, "签名校验失败")
		return
	}

	// 6. 调用 service 层处理业务
	err = c.rechargeService.HandleCallback(ctx, "dayuanren", body)
	if err != nil {
		logger.Error("处理大猿人平台回调 返回：500 处理回调失败", zap.Error(err))
		utils.Error(ctx, 500, err.Error())
		return
	}

	// 7. 返回成功
	logger.Info("处理大猿人平台回调 返回：200 成功")
	ctx.String(200, "success")
}
