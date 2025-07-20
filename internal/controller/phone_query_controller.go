package controller

import (
	"net/http"
	"strconv"

	"recharge-go/internal/model"
	"recharge-go/internal/service"
	"recharge-go/pkg/logger"
	"recharge-go/pkg/utils/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// PhoneQueryController 手机查询控制器
type PhoneQueryController struct {
	phoneQueryService service.PhoneQueryService
}

// NewPhoneQueryController 创建手机查询控制器
func NewPhoneQueryController(phoneQueryService service.PhoneQueryService) *PhoneQueryController {
	return &PhoneQueryController{
		phoneQueryService: phoneQueryService,
	}
}

// QueryBalance 查询手机余额
// @Summary 查询手机余额
// @Description 查询指定手机号的余额信息
// @Tags 手机查询
// @Accept json
// @Produce json
// @Param request body model.PhoneBalanceRequest true "查询请求"
// @Success 200 {object} response.Response{data=model.PhoneBalanceResponse} "查询成功"
// @Failure 400 {object} response.Response "请求参数错误"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/phone/balance [post]
func (c *PhoneQueryController) QueryBalance(ctx *gin.Context) {
	var req model.PhoneBalanceRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		logger.Log.Error("绑定余额查询请求参数失败", zap.Error(err))
		response.Error(ctx, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}

	// 获取重试次数参数（可选）
	maxRetries := 1
	if retryStr := ctx.Query("max_retries"); retryStr != "" {
		if retry, err := strconv.Atoi(retryStr); err == nil && retry > 0 && retry <= 5 {
			maxRetries = retry
		}
	}

	logger.Log.Info("开始处理余额查询请求",
		zap.String("phone", req.Phone),
		zap.String("isp_type", req.ISPType),
		zap.Int("max_retries", maxRetries),
	)

	// 调用服务层查询余额
	var result *model.PhoneBalanceResponse
	var err error

	if maxRetries > 1 {
		result, err = c.phoneQueryService.QueryBalanceWithRetry(ctx, req.Phone, req.ISPType, maxRetries)
	} else {
		result, err = c.phoneQueryService.QueryBalance(ctx, req.Phone, req.ISPType)
	}

	if err != nil {
		logger.Log.Error("查询余额失败",
			zap.String("phone", req.Phone),
			zap.String("isp_type", req.ISPType),
			zap.Error(err),
		)
		response.Error(ctx, http.StatusInternalServerError, "查询余额失败: "+err.Error())
		return
	}

	// 检查API返回的错误码
	if result.ErrCode != 0 {
		logger.Log.Warn("第三方API返回错误",
			zap.String("phone", req.Phone),
			zap.Int("errcode", result.ErrCode),
			zap.String("errmsg", result.ErrMsg),
		)
		response.Error(ctx, http.StatusBadRequest, result.ErrMsg)
		return
	}

	logger.Log.Info("余额查询成功",
		zap.String("phone", req.Phone),
		zap.String("balance", result.Data),
	)

	response.Success(ctx, result)
}

// QueryPaymentRecords 查询缴费记录
// @Summary 查询缴费记录
// @Description 查询指定手机号的缴费记录（仅支持移动和联通）
// @Tags 手机查询
// @Accept json
// @Produce json
// @Param request body model.PaymentRecordRequest true "查询请求"
// @Success 200 {object} response.Response{data=model.PaymentRecordResponse} "查询成功"
// @Failure 400 {object} response.Response "请求参数错误"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/phone/payment-records [post]
func (c *PhoneQueryController) QueryPaymentRecords(ctx *gin.Context) {
	var req model.PaymentRecordRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		logger.Log.Error("绑定缴费记录查询请求参数失败", zap.Error(err))
		response.Error(ctx, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}

	// 获取重试次数参数（可选）
	maxRetries := 1
	if retryStr := ctx.Query("max_retries"); retryStr != "" {
		if retry, err := strconv.Atoi(retryStr); err == nil && retry > 0 && retry <= 5 {
			maxRetries = retry
		}
	}

	logger.Log.Info("开始处理缴费记录查询请求",
		zap.String("phone", req.Phone),
		zap.String("isp_type", req.ISPType),
		zap.Int("max_retries", maxRetries),
	)

	// 调用服务层查询缴费记录
	var result *model.PaymentRecordResponse
	var err error

	if maxRetries > 1 {
		result, err = c.phoneQueryService.QueryPaymentRecordsWithRetry(ctx, req.Phone, req.ISPType, maxRetries)
	} else {
		result, err = c.phoneQueryService.QueryPaymentRecords(ctx, req.Phone, req.ISPType)
	}

	if err != nil {
		logger.Log.Error("查询缴费记录失败",
			zap.String("phone", req.Phone),
			zap.String("isp_type", req.ISPType),
			zap.Error(err),
		)
		response.Error(ctx, http.StatusInternalServerError, "查询缴费记录失败: "+err.Error())
		return
	}

	// 检查API返回的错误码
	if result.ErrCode != 0 {
		logger.Log.Warn("第三方API返回错误",
			zap.String("phone", req.Phone),
			zap.Int("errcode", result.ErrCode),
			zap.String("errmsg", result.ErrMsg),
		)
		response.Error(ctx, http.StatusBadRequest, result.ErrMsg)
		return
	}

	records := result.GetRecords()
	logger.Log.Info("缴费记录查询成功",
		zap.String("phone", req.Phone),
		zap.Int("record_count", len(records)),
	)

	response.Success(ctx, result)
}