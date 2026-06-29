package controller

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"recharge-go/internal/repository"
	"recharge-go/internal/service"
	"recharge-go/pkg/log"
	logger "recharge-go/pkg/log"
	"recharge-go/pkg/signature"
	resp "recharge-go/pkg/utils/response"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
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
	log.WithContextCategory(ctx.Request.Context(), "callback").Info("平台账号加载成功", log.StringV2("userid", userID), log.AnyV2("account", account))

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
	log.WithContextCategory(ctx.Request.Context(), "callback").Info("收到签名信息", log.StringV2("sign", sign))
	// 使用账号的AppSecret验证签名
	// if !verifySignature(body, sign, account.AppSecret) {
	// 	ctx.JSON(http.StatusBadRequest, gin.H{
	// 		"code": "1001",
	// 		"msg":  "invalid sign",
	// 	})
	// 	return
	// }

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
	logger.WithContextCategory(ctx.Request.Context(), "callback").Info("收到秘史平台回调", logger.StringV2("userid", ctx.Param("userid")))
	// 1. 获取userid
	userIDStr := ctx.Param("userid")
	if userIDStr == "" {
		resp.ErrorWithCode(ctx, http.StatusInternalServerError, 400, "缺少userid", nil)
		return
	}

	// 获取 appkey 等信息
	account, err := c.platformRepo.GetPlatformAccountByAccountName(userIDStr)
	if err != nil {
		logger.WithContextCategory(ctx.Request.Context(), "callback").Error("返回：401 平台账号不存在", logger.ErrorV2(err))
		resp.ErrorWithCode(ctx, http.StatusUnauthorized, 400, "平台账号不存在", nil)
		return
	}

	// 读取原始请求体
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		resp.Error(ctx, 400, "读取请求体失败")
		return
	}

	// 解析参数
	form, err := url.ParseQuery(string(body))
	if err != nil {
		resp.Error(ctx, 400, "参数解析失败")
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
	// 追加：按“原始字符串”拼接一版签名串（不做数值转型），用于对比上游是否以原值拼接
	signStrRaw := fmt.Sprintf(
		"szAgentId=%s&szOrderId=%s&szPhoneNum=%s&nDemo=%s&fSalePrice=%s&nFlag=%s&szKey=%s",
		params["szAgentId"],
		params["szOrderId"],
		params["szPhoneNum"],
		params["nDemo"],
		params["fSalePrice"],
		params["nFlag"],
		account.AppSecret,
	)

	// 计算两种串的MD5（小写）与对方回调的签名（小写），仅记录对比，不拦截
	calcSignNormalized := signature.GetMD5(signStr)
	calcSignRaw := signature.GetMD5(signStrRaw)
	receivedSign := strings.ToLower(params["szVerifyString"])

	// 对签名串进行脱敏（隐藏密钥）
	maskedSignStr := strings.ReplaceAll(signStr, account.AppSecret, "******")
	maskedSignStrRaw := strings.ReplaceAll(signStrRaw, account.AppSecret, "******")

	// 灰度策略：启用“原样拼接”作为当前生效的签名计算方式，但不拦截验签失败
	// 对签名串进行脱敏（隐藏密钥）
	// maskedSignStr := strings.ReplaceAll(signStr, account.AppSecret, "******")
	// maskedSignStrRaw := strings.ReplaceAll(signStrRaw, account.AppSecret, "******")

	signMode := "strict_raw"
	activeSignStrMasked := maskedSignStrRaw
	activeSignMD5 := calcSignRaw
	equalWithActive := activeSignMD5 == receivedSign

	// 构造安全参数日志（脱敏手机号）
	safeParams := map[string]string{}
	for k, v := range params {
		if k == "szPhoneNum" && len(v) >= 7 {
			safeParams[k] = v[:3] + "****" + v[len(v)-4:]
		} else {
			safeParams[k] = v
		}
	}

	logger.WithContextCategory(ctx.Request.Context(), "callback").Info("【mishi 回调签名调试】",
		logger.StringV2("userid", userIDStr),
		logger.StringV2("content_type", ctx.GetHeader("Content-Type")),
		logger.StringV2("user_agent", ctx.GetHeader("User-Agent")),
		logger.StringV2("sign_mode", signMode),
		logger.AnyV2("received_params", safeParams),
		logger.StringV2("received_sign", receivedSign),
		logger.StringV2("active_sign_str_masked", activeSignStrMasked),
		logger.StringV2("active_sign_md5", activeSignMD5),
		logger.BoolV2("equal_with_active", equalWithActive),
		logger.StringV2("our_sign_str_normalized_masked", maskedSignStr),
		logger.StringV2("our_sign_md5_normalized", calcSignNormalized),
		logger.StringV2("our_sign_str_raw_masked", maskedSignStrRaw),
		logger.StringV2("our_sign_md5_raw", calcSignRaw),
		logger.BoolV2("equal_with_normalized", calcSignNormalized == receivedSign),
		logger.BoolV2("equal_with_raw", calcSignRaw == receivedSign),
	)

	// 严格模式：若按原样拼接校验不通过，则直接返回 401
	if !equalWithActive {
		logger.WithContextCategory(ctx.Request.Context(), "callback").Error("mishi 回调签名校验失败",
			logger.StringV2("userid", userIDStr),
			logger.StringV2("received_sign", receivedSign),
			logger.StringV2("expected_sign", activeSignMD5),
		)
		resp.ErrorWithCode(ctx, http.StatusUnauthorized, 401, "签名校验失败", nil)
		return
	}

	logger.WithContextCategory(ctx.Request.Context(), "callback").Info("签名字符串", logger.StringV2("sign_str", signStr))
	// 业务处理交给 service
	if err := c.rechargeService.HandleCallback(ctx, "mishi", body); err != nil {
		logger.WithContextCategory(ctx.Request.Context(), "callback").Error("返回：500 处理回调失败", logger.ErrorV2(err))
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "failed to process callback"})
		return
	}
	ctx.String(200, "success")
}

// HandleChongzhiCallback 处理充值平台回调
func (c *CallbackController) HandleChongzhiCallback(ctx *gin.Context) {
	// 1. 获取userid
	logger.WithContextCategory(ctx.Request.Context(), "callback").Info("处理充值平台回调!!!")
	userIDStr := ctx.Param("userid")
	if userIDStr == "" {
		logger.WithContextCategory(ctx.Request.Context(), "callback").Error("处理充值平台回调 返回：400 缺少userid")
		resp.Error(ctx, 400, "缺少userid")
		return
	}

	// 2. 验证平台账号是否存在
	_, err := c.platformRepo.GetPlatformAccountByAccountName(userIDStr)
	if err != nil {
		logger.WithContextCategory(ctx.Request.Context(), "callback").Error("处理充值平台回调 返回：400 平台账号不存在", logger.ErrorV2(err))
		resp.Error(ctx, 400, "平台账号不存在")
		return
	}

	// 3. 读取原始请求体
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		logger.WithContextCategory(ctx.Request.Context(), "callback").Error("处理充值平台回调 返回：400 读取请求体失败", logger.ErrorV2(err))
		resp.Error(ctx, 400, "读取请求体失败")
		return
	}

	// 打印原始回调数据用于调试
	logger.WithContextCategory(ctx.Request.Context(), "callback").Info("收到充值平台回调数据",
		logger.StringV2("userid", userIDStr),
		logger.StringV2("raw_body", string(body)),
		logger.StringV2("content_type", ctx.GetHeader("Content-Type")),
		logger.StringV2("user_agent", ctx.GetHeader("User-Agent")),
	)

	// 4. 调用 service 层处理业务（chongzhi平台的签名验证在ParseCallbackData中处理）
	err = c.rechargeService.HandleCallback(ctx, "chongzhi", body)
	if err != nil {
		logger.WithContextCategory(ctx.Request.Context(), "callback").Error("处理充值平台回调 返回：500 处理回调失败", logger.ErrorV2(err))
		resp.Error(ctx, 500, err.Error())
		return
	}

	// 5. 返回成功（根据文档要求返回OK字符）
	logger.WithContextCategory(ctx.Request.Context(), "callback").Info("处理充值平台回调 返回：200 成功")
	ctx.String(200, "OK")
}

// HandleDayuanrenCallback 处理大猿人平台回调
func (c *CallbackController) HandleDayuanrenCallback(ctx *gin.Context) {
	// 1. 获取userid
	logger.WithContextCategory(ctx.Request.Context(), "callback").Info("处理大猿人平台回调!!!")
	userIDStr := ctx.Param("userid")
	if userIDStr == "" {
		logger.WithContextCategory(ctx.Request.Context(), "callback").Error("处理大猿人平台回调 返回：400 缺少userid")
		resp.Error(ctx, 400, "缺少userid")
		return
	}

	// 2. 获取平台账号信息
	account, err := c.platformRepo.GetPlatformAccountByAccountName(userIDStr)
	if err != nil {
		logger.WithContextCategory(ctx.Request.Context(), "callback").Error("处理大猿人平台回调 返回：400 平台账号不存在", logger.ErrorV2(err))
		resp.Error(ctx, 400, "平台账号不存在")
		return
	}
	logger.WithContextCategory(ctx.Request.Context(), "callback").Info("平台账号加载成功", logger.StringV2("userid", userIDStr), logger.AnyV2("account", account))

	// 3. 读取原始请求体
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		logger.WithContextCategory(ctx.Request.Context(), "callback").Error("处理大猿人平台回调 返回：400 读取请求体失败", logger.ErrorV2(err))
		resp.Error(ctx, 400, "读取请求体失败")
		return
	}

	// 4. 解析表单参数
	form, err := url.ParseQuery(string(body))
	if err != nil {
		logger.WithContextCategory(ctx.Request.Context(), "callback").Error("处理大猿人平台回调 返回：400 解析参数失败", logger.ErrorV2(err))
		resp.Error(ctx, 400, "解析参数失败")
		return
	}
	params := make(map[string]string)
	for k, v := range form {
		if len(v) > 0 {
			params[k] = v[0]
		}
	}
	logger.WithContextCategory(ctx.Request.Context(), "callback").Info("大猿人回调参数", logger.AnyV2("params", params))
	// 5. 签名校验
	// if !signature.VerifyDayuanrenSign(params, account.AppSecret) {
	// 	logger.Error("处理大猿人平台回调 返回：400 签名校验失败")
	// 	utils.Error(ctx, 400, "签名校验失败")
	// 	return
	// }

	// 6. 调用 service 层处理业务
	err = c.rechargeService.HandleCallback(ctx, "dayuanren", body)
	if err != nil {
		logger.WithContextCategory(ctx.Request.Context(), "callback").Error("处理大猿人平台回调 返回：500 处理回调失败", logger.ErrorV2(err))
		resp.Error(ctx, 500, err.Error())
		return
	}

	// 7. 返回成功
	logger.WithContextCategory(ctx.Request.Context(), "callback").Info("处理大猿人平台回调 返回：200 成功")
	ctx.String(200, "success")
}

// HandlePayc2Callback 处理 payc2 平台回调
func (c *CallbackController) HandlePayc2Callback(ctx *gin.Context) {
	// 1. 获取userid
	logger.WithContextCategory(ctx.Request.Context(), "callback").Info("处理 payc2 平台回调!!!")
	userIDStr := ctx.Param("userid")
	if userIDStr == "" {
		logger.WithContextCategory(ctx.Request.Context(), "callback").Error("处理 payc2 平台回调 返回：400 缺少userid")
		resp.Error(ctx, 400, "缺少userid")
		return
	}

	// 2. 验证平台账号是否存在
	_, err := c.platformRepo.GetPlatformAccountByAccountName(userIDStr)
	if err != nil {
		logger.WithContextCategory(ctx.Request.Context(), "callback").Error("处理 payc2 平台回调 返回：400 平台账号不存在", logger.ErrorV2(err))
		resp.Error(ctx, 400, "平台账号不存在")
		return
	}

	// 3. 读取原始请求体
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		logger.WithContextCategory(ctx.Request.Context(), "callback").Error("处理 payc2 平台回调 返回：400 读取请求体失败", logger.ErrorV2(err))
		resp.Error(ctx, 400, "读取请求体失败")
		return
	}

	// 4. 解析表单参数
	form, err := url.ParseQuery(string(body))
	if err != nil {
		logger.WithContextCategory(ctx.Request.Context(), "callback").Error("处理 payc2 平台回调 返回：400 解析参数失败", logger.ErrorV2(err))
		resp.Error(ctx, 400, "解析参数失败")
		return
	}
	params := make(map[string]string)
	for k, v := range form {
		if len(v) > 0 {
			params[k] = v[0]
		}
	}

	// 打印原始回调数据用于调试
	logger.WithContextCategory(ctx.Request.Context(), "callback").Info("收到 payc2 平台回调数据",
		logger.StringV2("userid", userIDStr),
		logger.StringV2("raw_body", string(body)),
		logger.StringV2("content_type", ctx.GetHeader("Content-Type")),
		logger.StringV2("user_agent", ctx.GetHeader("User-Agent")),
	)

	// 5. 调用 service 层处理业务（payc2平台的签名验证在ParseCallbackData中处理）
	err = c.rechargeService.HandleCallback(ctx, "payc2", body)
	if err != nil {
		logger.WithContextCategory(ctx.Request.Context(), "callback").Error("处理 payc2 平台回调 返回：500 处理回调失败", logger.ErrorV2(err))
		resp.Error(ctx, 500, err.Error())
		return
	}

	// 6. 返回成功（根据文档要求返回ok字符）
	logger.WithContextCategory(ctx.Request.Context(), "callback").Info("处理 payc2 平台回调 返回：200 成功")
	ctx.String(200, "ok")
}

// HandleLingshiCallback 处理灵石平台回调
func (c *CallbackController) HandleLingshiCallback(ctx *gin.Context) {
	// 1. 获取userid
	logger.WithContextCategory(ctx.Request.Context(), "callback").Info("处理灵石平台回调!!!")
	userIDStr := ctx.Param("userid")
	if userIDStr == "" {
		logger.WithContextCategory(ctx.Request.Context(), "callback").Error("处理灵石平台回调 返回：400 缺少userid")
		resp.Error(ctx, 400, "缺少userid")
		return
	}

	// 2. 验证平台账号是否存在 (可选，ParseCallbackData 会根据 appId 验证)
	_, err := c.platformRepo.GetPlatformAccountByAccountName(userIDStr)
	if err != nil {
		logger.WithContextCategory(ctx.Request.Context(), "callback").Error("处理灵石平台回调 返回：400 平台账号不存在", logger.ErrorV2(err))
		resp.Error(ctx, 400, "平台账号不存在")
		return
	}

	// 3. 读取原始请求体
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		logger.WithContextCategory(ctx.Request.Context(), "callback").Error("处理灵石平台回调 返回：400 读取请求体失败", logger.ErrorV2(err))
		resp.Error(ctx, 400, "读取请求体失败")
		return
	}

	// 打印原始回调数据用于调试
	logger.WithContextCategory(ctx.Request.Context(), "callback").Info("收到灵石平台回调数据",
		logger.StringV2("userid", userIDStr),
		logger.StringV2("raw_body", string(body)),
		logger.StringV2("content_type", ctx.GetHeader("Content-Type")),
		logger.StringV2("user_agent", ctx.GetHeader("User-Agent")),
	)

	// 4. 调用 service 层处理业务（灵石平台的签名验证在ParseCallbackData中处理）
	err = c.rechargeService.HandleCallback(ctx, "lingshi", body)
	if err != nil {
		logger.WithContextCategory(ctx.Request.Context(), "callback").Error("处理灵石平台回调 返回：500 处理回调失败", logger.ErrorV2(err))
		resp.Error(ctx, 500, err.Error())
		return
	}

	// 5. 返回成功（根据文档要求返回纯文本 "success"）
	logger.WithContextCategory(ctx.Request.Context(), "callback").Info("处理灵石平台回调 返回：200 成功")
	ctx.String(200, "success")
}

// HandleKasushouCallback 处理卡速售平台回调
func (c *CallbackController) HandleKasushouCallback(ctx *gin.Context) {
	// 1. 获取userid
	logger.WithContextCategory(ctx.Request.Context(), "callback").Info("处理卡速售平台回调!!!")
	userIDStr := ctx.Param("userid")
	if userIDStr == "" {
		logger.WithContextCategory(ctx.Request.Context(), "callback").Error("处理卡速售平台回调 返回：400 缺少userid")
		resp.Error(ctx, 400, "缺少userid")
		return
	}

	// 2. 验证平台账号是否存在
	_, err := c.platformRepo.GetPlatformAccountByAccountName(userIDStr)
	if err != nil {
		logger.WithContextCategory(ctx.Request.Context(), "callback").Error("处理卡速售平台回调 返回：400 平台账号不存在", logger.ErrorV2(err))
		resp.Error(ctx, 400, "平台账号不存在")
		return
	}

	// 3. 读取原始请求体
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		logger.WithContextCategory(ctx.Request.Context(), "callback").Error("处理卡速售平台回调 返回：400 读取请求体失败", logger.ErrorV2(err))
		resp.Error(ctx, 400, "读取请求体失败")
		return
	}

	// 打印原始回调数据用于调试
	logger.WithContextCategory(ctx.Request.Context(), "callback").Info("收到卡速售平台回调数据",
		logger.StringV2("userid", userIDStr),
		logger.StringV2("raw_body", string(body)),
		logger.StringV2("content_type", ctx.GetHeader("Content-Type")),
		logger.StringV2("user_agent", ctx.GetHeader("User-Agent")),
	)

	// 4. 调用 service 层处理业务（卡速售平台的签名验证在 ParseCallbackData 和 VerifyKasushouCallback 中处理）
	err = c.rechargeService.HandleCallback(ctx, "kasushou", body)
	if err != nil {
		logger.WithContextCategory(ctx.Request.Context(), "callback").Error("处理卡速售平台回调 返回：500 处理回调失败", logger.ErrorV2(err))
		resp.Error(ctx, 500, err.Error())
		return
	}

	// 5. 返回成功（根据文档要求返回纯文本 "ok"）
	logger.WithContextCategory(ctx.Request.Context(), "callback").Info("处理卡速售平台回调 返回：200 成功")
	ctx.String(200, "ok")
}

// HandleTurboCallback 处理 Turbo 平台回调（JSON Body）
func (c *CallbackController) HandleTurboCallback(ctx *gin.Context) {
	userIDStr := ctx.Param("userid")
	if userIDStr == "" {
		resp.Error(ctx, 400, "缺少userid")
		return
	}
	if _, err := c.platformRepo.GetPlatformAccountByAccountName(userIDStr); err != nil {
		logger.WithContextCategory(ctx.Request.Context(), "callback").Error("Turbo 回调账号不存在", logger.ErrorV2(err))
		resp.Error(ctx, 400, "平台账号不存在")
		return
	}
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		resp.Error(ctx, 400, "读取请求体失败")
		return
	}
	logger.WithContextCategory(ctx.Request.Context(), "callback").Info("收到 Turbo 回调",
		logger.StringV2("userid", userIDStr),
		logger.StringV2("raw_preview", truncateStr(string(body), 800)),
	)
	if err := c.rechargeService.HandleCallback(ctx.Request.Context(), "turbo", body); err != nil {
		logger.WithContextCategory(ctx.Request.Context(), "callback").Error("Turbo 回调处理失败", logger.ErrorV2(err))
		resp.Error(ctx, 500, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"status": "success", "message": "ok"})
}

// HandleXingchenCallback 处理兴辰网络平台回调（form/json）
func (c *CallbackController) HandleXingchenCallback(ctx *gin.Context) {
	userIDStr := ctx.Param("userid")
	if userIDStr == "" {
		resp.Error(ctx, 400, "缺少userid")
		return
	}
	if _, err := c.platformRepo.GetPlatformAccountByAccountName(userIDStr); err != nil {
		logger.WithContextCategory(ctx.Request.Context(), "callback").Error("兴辰网络回调账号不存在", logger.ErrorV2(err))
		resp.Error(ctx, 400, "平台账号不存在")
		return
	}
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		resp.Error(ctx, 400, "读取请求体失败")
		return
	}
	logger.WithContextCategory(ctx.Request.Context(), "callback").Info("收到兴辰网络回调",
		logger.StringV2("userid", userIDStr),
		logger.StringV2("raw_preview", truncateStr(string(body), 800)),
	)
	if err := c.rechargeService.HandleCallback(ctx.Request.Context(), "xingchen", body); err != nil {
		logger.WithContextCategory(ctx.Request.Context(), "callback").Error("兴辰网络回调处理失败", logger.ErrorV2(err))
		resp.Error(ctx, 500, err.Error())
		return
	}
	ctx.String(http.StatusOK, "OK")
}

func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// HandleShangtengCallback 处理商腾科技平台回调
func (c *CallbackController) HandleShangtengCallback(ctx *gin.Context) {
	// 1. 获取userid
	logger.WithContextCategory(ctx.Request.Context(), "callback").Info("处理商腾科技平台回调!!!")
	userIDStr := ctx.Param("userid")
	if userIDStr == "" {
		logger.WithContextCategory(ctx.Request.Context(), "callback").Error("处理商腾科技平台回调 返回：400 缺少userid")
		resp.Error(ctx, 400, "缺少userid")
		return
	}

	// 2. 验证平台账号是否存在
	_, err := c.platformRepo.GetPlatformAccountByAccountName(userIDStr)
	if err != nil {
		logger.WithContextCategory(ctx.Request.Context(), "callback").Error("处理商腾科技平台回调 返回：400 平台账号不存在", logger.ErrorV2(err))
		resp.Error(ctx, 400, "平台账号不存在")
		return
	}

	// 3. 读取原始请求体
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		logger.WithContextCategory(ctx.Request.Context(), "callback").Error("处理商腾科技平台回调 返回：400 读取请求体失败", logger.ErrorV2(err))
		resp.Error(ctx, 400, "读取请求体失败")
		return
	}

	// 调试日志
	logger.WithContextCategory(ctx.Request.Context(), "callback").Info("收到商腾科技平台回调数据",
		logger.StringV2("userid", userIDStr),
		logger.StringV2("raw_body", string(body)),
		logger.StringV2("content_type", ctx.GetHeader("Content-Type")),
		logger.StringV2("user_agent", ctx.GetHeader("User-Agent")),
	)

	// 4. 业务处理
	err = c.rechargeService.HandleCallback(ctx, "shangteng", body)
	if err != nil {
		logger.WithContextCategory(ctx.Request.Context(), "callback").Error("处理商腾科技平台回调 返回：500 处理回调失败", logger.ErrorV2(err))
		resp.Error(ctx, 500, err.Error())
		return
	}

	// 5. 返回成功（统一响应）
	logger.WithContextCategory(ctx.Request.Context(), "callback").Info("处理商腾科技平台回调 返回：200 成功")
	resp.Success(ctx, "ok")
}

// HandleKayixinCallback 处理卡易信 API 3.0 订单回调
func (c *CallbackController) HandleKayixinCallback(ctx *gin.Context) {
	appID := strings.TrimSpace(ctx.Param("userid"))
	if appID == "" {
		resp.Error(ctx, 400, "缺少客户编号")
		return
	}

	account, err := c.platformRepo.GetPlatformAccountByAccountName(appID)
	if err != nil {
		logger.WithContextCategory(ctx.Request.Context(), "callback").Error("卡易信回调账号不存在", logger.ErrorV2(err))
		resp.Error(ctx, 400, "平台账号不存在")
		return
	}

	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		resp.Error(ctx, 400, "读取请求体失败")
		return
	}
	rawBody := string(body)

	headerAppID := strings.TrimSpace(ctx.GetHeader("X-App-Id"))
	version := strings.TrimSpace(ctx.GetHeader("X-Version"))
	timestamp := strings.TrimSpace(ctx.GetHeader("X-Timestamp"))
	sign := strings.TrimSpace(ctx.GetHeader("X-Signature"))
	if version == "" {
		version = signature.KayixinAPIVersion
	}
	if headerAppID != "" && headerAppID != appID {
		logger.WithContextCategory(ctx.Request.Context(), "callback").Warn("卡易信回调路径与 Header X-App-Id 不一致",
			logger.StringV2("path_app_id", appID),
			logger.StringV2("header_app_id", headerAppID),
		)
	}
	verifyAppID := appID
	if headerAppID != "" {
		verifyAppID = headerAppID
	}
	if !signature.KayixinVerify(verifyAppID, account.AppSecret, version, timestamp, rawBody, sign) {
		logger.WithContextCategory(ctx.Request.Context(), "callback").Error("卡易信回调验签失败",
			logger.StringV2("app_id", verifyAppID),
		)
		resp.Error(ctx, 400, "签名验证失败")
		return
	}

	logger.WithContextCategory(ctx.Request.Context(), "callback").Info("收到卡易信回调",
		logger.StringV2("app_id", appID),
		logger.StringV2("raw_preview", truncateStr(rawBody, 800)),
	)

	if err := c.rechargeService.HandleCallback(ctx.Request.Context(), "kayixin", body); err != nil {
		logger.WithContextCategory(ctx.Request.Context(), "callback").Error("卡易信回调处理失败", logger.ErrorV2(err))
		resp.Error(ctx, 500, err.Error())
		return
	}
	ctx.String(http.StatusOK, "ok")
}
