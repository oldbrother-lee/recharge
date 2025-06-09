package middleware

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"recharge-go/internal/repository"
	"recharge-go/internal/utils"
	"recharge-go/pkg/signature"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ExternalAuthMiddleware 外部认证中间件结构体
type ExternalAuthMiddleware struct {
	apiKeyRepo         repository.ExternalAPIKeyRepository
	signatureValidator *signature.ExternalAPISignatureValidator
	rateLimit          map[string]*RateLimiter
}

// RateLimiter 简单的速率限制器
type RateLimiter struct {
	requests   []time.Time
	maxReqs    int
	timeWindow time.Duration
}

// NewExternalAuthMiddleware 创建外部认证中间件
func NewExternalAuthMiddleware(apiKeyRepo repository.ExternalAPIKeyRepository) *ExternalAuthMiddleware {
	return &ExternalAuthMiddleware{
		apiKeyRepo:         apiKeyRepo,
		signatureValidator: signature.NewExternalAPISignatureValidator(),
		rateLimit:          make(map[string]*RateLimiter),
	}
}

// ExternalAuth 外部认证中间件
func (m *ExternalAuthMiddleware) ExternalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 获取客户端IP
		clientIP := getClientIP(c)

		// 2. 获取API Key
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			utils.Error(c, http.StatusUnauthorized, "API Key is required")
			c.Abort()
			return
		}

		// 3. 验证API Key
		apiKeyInfo, err := m.apiKeyRepo.GetByAppKey(apiKey)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				utils.Error(c, http.StatusUnauthorized, "Invalid API Key")
			} else {
				utils.Error(c, http.StatusInternalServerError, "API Key validation failed")
			}
			c.Abort()
			return
		}

		// 4. 检查API Key状态
		if !apiKeyInfo.IsActive() {
			utils.Error(c, http.StatusUnauthorized, "API Key is inactive or expired")
			c.Abort()
			return
		}

		// 5. 检查IP白名单
		if !apiKeyInfo.IsIPAllowed(clientIP) {
			utils.Error(c, http.StatusForbidden, "IP not allowed")
			c.Abort()
			return
		}

		// 6. 检查请求频率限制
		if !m.checkRateLimit(apiKeyInfo.AppID, apiKeyInfo.RateLimit) {
			utils.Error(c, http.StatusTooManyRequests, "Rate limit exceeded")
			c.Abort()
			return
		}

		// 7. 验证签名
		if err := m.validateSignature(c, apiKeyInfo.AppSecret); err != nil {
			utils.Error(c, http.StatusUnauthorized, fmt.Sprintf("Signature validation failed: %v", err))
			c.Abort()
			return
		}

		// 8. 将API Key信息存储到上下文中
		c.Set("api_key_info", apiKeyInfo)
		c.Set("client_ip", clientIP)

		c.Next()
	}
}

// validateSignature 验证签名
func (m *ExternalAuthMiddleware) validateSignature(c *gin.Context, appSecret string) error {
	// 获取签名相关参数
	signature := c.GetHeader("X-Signature")
	if signature == "" {
		signature = c.Query("sign")
	}
	if signature == "" {
		return fmt.Errorf("signature is required")
	}

	// 根据请求方法获取参数
	var params map[string]interface{}

	switch c.Request.Method {
	case "GET":
		params = m.signatureValidator.ParseFormParams(c.Request.URL.Query())
	case "POST":
		contentType := c.GetHeader("Content-Type")
		if strings.Contains(contentType, "application/json") {
			// JSON格式
			body, err := io.ReadAll(c.Request.Body)
			if err != nil {
				return fmt.Errorf("failed to read request body")
			}
			// 重新设置body，以便后续处理
			c.Request.Body = io.NopCloser(strings.NewReader(string(body)))

			var jsonData map[string]interface{}
			if err := json.Unmarshal(body, &jsonData); err != nil {
				return fmt.Errorf("failed to parse JSON body")
			}
			params = m.signatureValidator.ParseJSONParams(jsonData)
		} else {
			// 表单格式
			if err := c.Request.ParseForm(); err != nil {
				return fmt.Errorf("failed to parse form data")
			}
			params = m.signatureValidator.ParseFormParams(c.Request.PostForm)
		}
	default:
		return fmt.Errorf("unsupported request method")
	}

	// 验证签名
	return m.signatureValidator.ValidateExternalAPISignature(params, signature, appSecret)
}

// checkRateLimit 检查请求频率限制
func (m *ExternalAuthMiddleware) checkRateLimit(appID string, maxReqs int) bool {
	now := time.Now()
	timeWindow := time.Minute // 1分钟时间窗口

	if m.rateLimit[appID] == nil {
		m.rateLimit[appID] = &RateLimiter{
			requests:   make([]time.Time, 0),
			maxReqs:    maxReqs,
			timeWindow: timeWindow,
		}
	}

	limiter := m.rateLimit[appID]

	// 清理过期的请求记录
	var validRequests []time.Time
	for _, reqTime := range limiter.requests {
		if now.Sub(reqTime) < timeWindow {
			validRequests = append(validRequests, reqTime)
		}
	}
	limiter.requests = validRequests

	// 检查是否超过限制
	if len(limiter.requests) >= maxReqs {
		return false
	}

	// 添加当前请求
	limiter.requests = append(limiter.requests, now)
	return true
}

// getClientIP 获取客户端真实IP
func getClientIP(c *gin.Context) string {
	// 尝试从各种头部获取真实IP
	if ip := c.GetHeader("X-Forwarded-For"); ip != "" {
		// X-Forwarded-For可能包含多个IP，取第一个
		if ips := strings.Split(ip, ","); len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	if ip := c.GetHeader("X-Real-IP"); ip != "" {
		return ip
	}

	if ip := c.GetHeader("X-Original-Forwarded-For"); ip != "" {
		return ip
	}

	// 从RemoteAddr获取
	if ip, _, err := net.SplitHostPort(c.Request.RemoteAddr); err == nil {
		return ip
	}

	return c.Request.RemoteAddr
}
