package controller

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"recharge-go/internal/model"
	"recharge-go/internal/repository"
	"recharge-go/pkg/utils/response"

	"github.com/gin-gonic/gin"
)

// ExternalAPIKeyController API密钥管理控制器
type ExternalAPIKeyController struct {
	apiKeyRepo repository.ExternalAPIKeyRepository
	userRepo   *repository.UserRepository
}

// NewExternalAPIKeyController 创建API密钥管理控制器
func NewExternalAPIKeyController(apiKeyRepo repository.ExternalAPIKeyRepository, userRepo *repository.UserRepository) *ExternalAPIKeyController {
	return &ExternalAPIKeyController{
		apiKeyRepo: apiKeyRepo,
		userRepo:   userRepo,
	}
}

// CreateAPIKey 为当前用户创建API密钥
// @Summary 创建API密钥
// @Description 为当前用户创建新的API密钥
// @Tags API密钥管理
// @Accept json
// @Produce json
// @Param request body CreateAPIKeyRequest true "创建请求"
// @Success 200 {object} response.Response{data=model.ExternalAPIKey}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/external-api-keys [post]
func (c *ExternalAPIKeyController) CreateAPIKey(ctx *gin.Context) {
	// 获取当前用户ID
	userID, exists := ctx.Get("user_id")
	if !exists {
		response.Error(ctx, http.StatusUnauthorized, "用户未登录")
		return
	}

	userIDInt, ok := userID.(int64)
	if !ok {
		response.Error(ctx, http.StatusInternalServerError, "用户ID格式错误")
		return
	}

	// 检查用户是否已有API密钥
	existingKeys, _, err := c.apiKeyRepo.GetByUserID(userIDInt, 0, 10)
	if err != nil {
		response.Error(ctx, http.StatusInternalServerError, "查询API密钥失败")
		return
	}

	// 检查是否已存在该用户的API密钥（限制每个用户只能有一个API密钥）
	if len(existingKeys) > 0 {
		response.Error(ctx, http.StatusBadRequest, "用户已存在API密钥")
		return
	}

	var req CreateAPIKeyRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		req.AppName = fmt.Sprintf("user_%d_api", userIDInt) // 默认名称
		req.Description = "用户API密钥"
	}

	// 生成API密钥
	apiKey := &model.ExternalAPIKey{
		UserID:      userIDInt,
		AppID:       generateNumericAppID(),
		AppKey:      generateAppKey(),
		AppSecret:   generateAppSecret(),
		AppName:     req.AppName,
		Description: req.Description,
		Status:      1,    // 默认启用
		RateLimit:   1000, // 默认限制
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// 保存到数据库
	if err := c.apiKeyRepo.Create(apiKey); err != nil {
		response.Error(ctx, http.StatusInternalServerError, "创建API密钥失败")
		return
	}

	response.Success(ctx, apiKey)
}

// GetMyAPIKeys 获取当前用户的API密钥
// @Summary 获取我的API密钥
// @Description 获取当前用户的API密钥（每个用户只能有一个）
// @Tags API密钥管理
// @Produce json
// @Success 200 {object} response.Response{data=model.ExternalAPIKey}
// @Failure 401 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/external-api-keys/my [get]
func (c *ExternalAPIKeyController) GetMyAPIKeys(ctx *gin.Context) {
	// 获取当前用户ID
	userID, exists := ctx.Get("user_id")
	if !exists {
		response.Error(ctx, http.StatusUnauthorized, "用户未登录")
		return
	}

	userIDInt, ok := userID.(int64)
	if !ok {
		response.Error(ctx, http.StatusInternalServerError, "用户ID格式错误")
		return
	}

	// 直接根据用户ID查询API密钥（只取第一个，因为每个用户只能有一个）
	userKeys, _, err := c.apiKeyRepo.GetByUserID(userIDInt, 0, 1)
	if err != nil {
		response.Error(ctx, http.StatusInternalServerError, "查询API密钥失败")
		return
	}

	// 如果用户没有API密钥，返回null
	if len(userKeys) == 0 {
		response.Success(ctx, nil)
		return
	}

	// 返回用户的API密钥
	response.Success(ctx, userKeys[0])
}

// RegenerateAPIKey 重新生成API密钥
// @Summary 重新生成API密钥
// @Description 重新生成指定的API密钥
// @Tags API密钥管理
// @Produce json
// @Param id path int true "API密钥ID"
// @Success 200 {object} response.Response{data=model.ExternalAPIKey}
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/external-api-keys/{id}/regenerate [post]
func (c *ExternalAPIKeyController) RegenerateAPIKey(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.Error(ctx, http.StatusBadRequest, "无效的ID")
		return
	}

	// 获取当前用户ID
	userID, exists := ctx.Get("user_id")
	if !exists {
		response.Error(ctx, http.StatusUnauthorized, "用户未登录")
		return
	}

	userIDInt, ok := userID.(int64)
	if !ok {
		response.Error(ctx, http.StatusInternalServerError, "用户ID格式错误")
		return
	}

	// 获取API密钥
	allKeys, _, err := c.apiKeyRepo.List(0, 1000)
	if err != nil {
		response.Error(ctx, http.StatusInternalServerError, "查询API密钥失败")
		return
	}

	var apiKey *model.ExternalAPIKey
	for _, key := range allKeys {
		if key.ID == id && key.UserID == userIDInt {
			apiKey = key
			break
		}
	}

	if apiKey == nil {
		response.Error(ctx, http.StatusNotFound, "API密钥不存在")
		return
	}

	// 重新生成密钥
	apiKey.AppKey = generateAppKey()
	apiKey.AppSecret = generateAppSecret()
	apiKey.UpdatedAt = time.Now()

	// 更新到数据库
	if err := c.apiKeyRepo.Update(apiKey); err != nil {
		response.Error(ctx, http.StatusInternalServerError, "更新API密钥失败")
		return
	}

	response.Success(ctx, apiKey)
}

// UpdateAPIKeyStatus 更新API密钥状态
// @Summary 更新API密钥状态
// @Description 启用或禁用API密钥
// @Tags API密钥管理
// @Accept json
// @Produce json
// @Param id path int true "API密钥ID"
// @Param request body UpdateStatusRequest true "状态更新请求"
// @Success 200 {object} response.Response{data=model.ExternalAPIKey}
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/external-api-keys/{id}/status [put]
func (c *ExternalAPIKeyController) UpdateAPIKeyStatus(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.Error(ctx, http.StatusBadRequest, "无效的ID")
		return
	}

	// 获取当前用户ID
	userID, exists := ctx.Get("user_id")
	if !exists {
		response.Error(ctx, http.StatusUnauthorized, "用户未登录")
		return
	}

	userIDInt, ok := userID.(int64)
	if !ok {
		response.Error(ctx, http.StatusInternalServerError, "用户ID格式错误")
		return
	}

	var req UpdateStatusRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.Error(ctx, http.StatusBadRequest, "参数错误")
		return
	}

	// 获取API密钥
	allKeys, _, err := c.apiKeyRepo.List(0, 1000)
	if err != nil {
		response.Error(ctx, http.StatusInternalServerError, "查询API密钥失败")
		return
	}

	var apiKey *model.ExternalAPIKey
	for _, key := range allKeys {
		if key.ID == id && key.UserID == userIDInt {
			apiKey = key
			break
		}
	}

	if apiKey == nil {
		response.Error(ctx, http.StatusNotFound, "API密钥不存在")
		return
	}

	// 更新状态
	apiKey.Status = req.Status
	apiKey.UpdatedAt = time.Now()

	// 更新到数据库
	if err := c.apiKeyRepo.Update(apiKey); err != nil {
		response.Error(ctx, http.StatusInternalServerError, "更新API密钥状态失败")
		return
	}

	response.Success(ctx, apiKey)
}

// 请求结构体
type CreateAPIKeyRequest struct {
	AppName     string `json:"app_name" binding:"max=128"`
	Description string `json:"description" binding:"max=255"`
}

type UpdateStatusRequest struct {
	Status int `json:"status"`
}

// 生成纯数字的AppID (10位)
func generateNumericAppID() string {
	return fmt.Sprintf("%010d", time.Now().Unix()%10000000000)
}

// 生成AppKey (32位随机字符串)
func generateAppKey() string {
	return generateRandomString(32)
}

// 生成AppSecret (64位随机字符串)
func generateAppSecret() string {
	return generateRandomString(64)
}

// 生成指定长度的随机字符串
func generateRandomString(length int) string {
	bytes := make([]byte, length/2)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
