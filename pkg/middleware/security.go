package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/time/rate"

	log "recharge-go/pkg/log"
	resp "recharge-go/pkg/utils/response"
)

// SecurityConfig 安全配置
type SecurityConfig struct {
	JWT       JWTConfig       `yaml:"jwt"`
	RateLimit RateLimitConfig `yaml:"rate_limit"`
	CORS      CORSConfig      `yaml:"cors"`
}

// JWTConfig JWT配置
type JWTConfig struct {
	Secret     string        `yaml:"secret"`
	Expiration time.Duration `yaml:"expiration"`
	Issuer     string        `yaml:"issuer"`
	SkipPaths  []string      `yaml:"skip_paths"`
}

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	Enabled      bool          `yaml:"enabled"`
	RPS          int           `yaml:"rps"`
	Burst        int           `yaml:"burst"`
	Window       time.Duration `yaml:"window"`
	SkipPaths    []string      `yaml:"skip_paths"`
	IncludePaths []string      `yaml:"include_paths"`
}

// CORSConfig CORS配置
type CORSConfig struct {
	AllowOrigins     []string `yaml:"allow_origins"`
	AllowMethods     []string `yaml:"allow_methods"`
	AllowHeaders     []string `yaml:"allow_headers"`
	ExposeHeaders    []string `yaml:"expose_headers"`
	AllowCredentials bool     `yaml:"allow_credentials"`
	MaxAge           int      `yaml:"max_age"`
	SkipPaths        []string `yaml:"skip_paths"`
}

// JWTClaims JWT声明
type JWTClaims struct {
	UserID   int64    `json:"user_id"`
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
	jwt.RegisteredClaims
}

// SecurityMiddleware 安全中间件管理器
type SecurityMiddleware struct {
	config   *SecurityConfig
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex // 保护limiters map的并发访问
	stopCh   chan struct{}
	wg       sync.WaitGroup
	stopOnce sync.Once
}

// NewSecurityMiddleware 创建安全中间件
func NewSecurityMiddleware(config *SecurityConfig) *SecurityMiddleware {
	sm := &SecurityMiddleware{
		config:   config,
		limiters: make(map[string]*rate.Limiter),
		stopCh:   make(chan struct{}),
	}

	// 启动后台清理goroutine
	sm.wg.Add(1)
	go sm.startCleanupRoutine()

	return sm
}

// JWTAuth JWT认证中间件
func (sm *SecurityMiddleware) JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查是否跳过认证
		if sm.shouldSkipJWT(c.Request.URL.Path) {
			c.Next()
			return
		}

		// 获取token
		token := sm.extractToken(c)
		if token == "" {
			log.WithContext(c.Request.Context()).Warn("JWT missing token",
				log.String("path", c.Request.URL.Path),
				log.String("method", c.Request.Method),
			)
			resp.ErrorWithCode(c, http.StatusUnauthorized, 1002, "Unauthorized", nil)
			c.Abort()
			return
		}

		// 验证token
		claims, err := sm.validateToken(token)
		if err != nil {
			log.WithContext(c.Request.Context()).Warn("JWT validation failed",
				log.String("token", token),
				log.ErrorV2(err),
			)
			resp.ErrorWithCode(c, http.StatusUnauthorized, 1002, "Unauthorized", nil)
			c.Abort()
			return
		}

		// 设置用户信息到上下文
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("roles", claims.Roles)
		// 为了兼容性，设置第一个角色为role
		if len(claims.Roles) > 0 {
			c.Set("role", claims.Roles[0])
		}

		// 添加到请求上下文
		ctx := context.WithValue(c.Request.Context(), "user_id", claims.UserID)
		ctx = context.WithValue(ctx, "username", claims.Username)
		ctx = context.WithValue(ctx, "roles", claims.Roles)
		if len(claims.Roles) > 0 {
			ctx = context.WithValue(ctx, "role", claims.Roles[0])
		}
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// RateLimit 限流中间件
func (sm *SecurityMiddleware) RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !sm.config.RateLimit.Enabled {
			c.Next()
			return
		}

		if !sm.shouldApplyRateLimit(c.Request.URL.Path) {
			c.Next()
			return
		}

		// 获取客户端标识（IP或用户ID）
		clientID := sm.getClientID(c)

		// 获取或创建限流器
		limiter := sm.getLimiter(clientID)

		// 检查是否允许请求
		if !limiter.Allow() {
			log.WithContext(c.Request.Context()).Warn("Rate limit exceeded",
				log.String("client_id", clientID),
				log.String("path", c.Request.URL.Path),
			)
			resp.ErrorWithCode(c, http.StatusTooManyRequests, 1006, "Too Many Requests", nil)
			c.Abort()
			return
		}

		c.Next()
	}
}

// CORS CORS中间件
func (sm *SecurityMiddleware) CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		if matchPath(c.Request.URL.Path, sm.config.CORS.SkipPaths) {
			c.Next()
			return
		}
		origin := c.Request.Header.Get("Origin")

		// 检查是否允许的源
		if sm.isAllowedOrigin(origin) {
			c.Header("Access-Control-Allow-Origin", origin)
		} else if len(sm.config.CORS.AllowOrigins) == 1 && sm.config.CORS.AllowOrigins[0] == "*" {
			c.Header("Access-Control-Allow-Origin", "*")
		}

		// 设置其他CORS头
		c.Header("Access-Control-Allow-Methods", strings.Join(sm.config.CORS.AllowMethods, ", "))
		c.Header("Access-Control-Allow-Headers", strings.Join(sm.config.CORS.AllowHeaders, ", "))

		if len(sm.config.CORS.ExposeHeaders) > 0 {
			c.Header("Access-Control-Expose-Headers", strings.Join(sm.config.CORS.ExposeHeaders, ", "))
		}

		if sm.config.CORS.AllowCredentials {
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		if sm.config.CORS.MaxAge > 0 {
			c.Header("Access-Control-Max-Age", strconv.Itoa(sm.config.CORS.MaxAge))
		}

		// 处理预检请求
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// RequestID 请求ID中间件
func (sm *SecurityMiddleware) RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}

		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		// 添加到请求上下文
		ctx := context.WithValue(c.Request.Context(), "request_id", requestID)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// Security 安全头中间件
func (sm *SecurityMiddleware) Security() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 设置安全头
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Header("Content-Security-Policy", "default-src 'self'")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		c.Next()
	}
}

// extractToken 提取token
func (sm *SecurityMiddleware) extractToken(c *gin.Context) string {
	// 从Authorization头提取
	auth := c.GetHeader("Authorization")
	if auth != "" && strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}

	// 从查询参数提取
	return c.Query("token")
}

// validateToken 验证token
func (sm *SecurityMiddleware) validateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(sm.config.JWT.Secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// shouldSkipJWT 检查是否跳过JWT认证
func (sm *SecurityMiddleware) shouldSkipJWT(path string) bool {
	return matchPath(path, sm.config.JWT.SkipPaths)
}

// shouldSkipRateLimit 检查是否跳过限流
func (sm *SecurityMiddleware) shouldSkipRateLimit(path string) bool {
	return matchPath(path, sm.config.RateLimit.SkipPaths)
}

func (sm *SecurityMiddleware) shouldApplyRateLimit(path string) bool {
	if len(sm.config.RateLimit.IncludePaths) > 0 {
		return matchPath(path, sm.config.RateLimit.IncludePaths)
	}
	return !matchPath(path, sm.config.RateLimit.SkipPaths)
}

// getClientID 获取客户端标识
func (sm *SecurityMiddleware) getClientID(c *gin.Context) string {
	// 优先从 Authorization 解析用户ID（即使 JWTAuth 尚未运行）
	token := sm.extractToken(c)
	if token != "" {
		if claims, err := sm.validateToken(token); err == nil {
			return fmt.Sprintf("user:%d", claims.UserID)
		}
	}
	// 回退：如果上下文已有 user_id（例如其他中间件已设置）
	if userID, exists := c.Get("user_id"); exists {
		return fmt.Sprintf("user:%s", userID)
	}
	// 最后回退到 IP 维度限流
	return fmt.Sprintf("ip:%s", c.ClientIP())
}

// getLimiter 获取或创建限流器
func (sm *SecurityMiddleware) getLimiter(clientID string) *rate.Limiter {
	// 先尝试读锁获取已存在的限流器
	sm.mu.RLock()
	if limiter, exists := sm.limiters[clientID]; exists {
		sm.mu.RUnlock()
		return limiter
	}
	sm.mu.RUnlock()

	// 使用写锁创建新的限流器
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// 双重检查，防止在获取写锁期间其他goroutine已经创建了限流器
	if limiter, exists := sm.limiters[clientID]; exists {
		return limiter
	}

	// 创建新的限流器
	limiter := rate.NewLimiter(rate.Limit(sm.config.RateLimit.RPS), sm.config.RateLimit.Burst)
	sm.limiters[clientID] = limiter

	return limiter
}

// startCleanupRoutine 启动清理例程
func (sm *SecurityMiddleware) startCleanupRoutine() {
	ticker := time.NewTicker(time.Hour) // 每小时清理一次
	defer ticker.Stop()
	defer sm.wg.Done()

	for {
		select {
		case <-ticker.C:
			sm.cleanupLimiters()
		case <-sm.stopCh:
			return
		}
	}
}

// cleanupLimiters 清理过期的限流器
func (sm *SecurityMiddleware) cleanupLimiters() {
	// 简单的清理策略：定期清理所有限流器
	// 实际应用中可以使用更复杂的LRU或TTL策略
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// 记录清理前的限流器数量
	oldCount := len(sm.limiters)
	sm.limiters = make(map[string]*rate.Limiter)

	// 记录清理日志
	if oldCount > 0 {
		log.L().Info("Rate limiters cleaned up",
			log.Int("cleaned_count", oldCount),
		)
	}
}

// isAllowedOrigin 检查是否允许的源
func (sm *SecurityMiddleware) isAllowedOrigin(origin string) bool {
	for _, allowedOrigin := range sm.config.CORS.AllowOrigins {
		if allowedOrigin == "*" || allowedOrigin == origin {
			return true
		}
	}
	return false
}

// generateRequestID 生成请求ID
func generateRequestID() string {
	// 简单的请求ID生成，实际应用中可以使用UUID
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// GenerateToken 生成JWT token
func (sm *SecurityMiddleware) GenerateToken(userID, username, role string) (string, error) {
	// 将userID转换为int64
	userIDInt, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return "", fmt.Errorf("invalid user ID: %v", err)
	}

	claims := &JWTClaims{
		UserID:   userIDInt,
		Username: username,
		Roles:    []string{role},
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    sm.config.JWT.Issuer,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(sm.config.JWT.Expiration)),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(sm.config.JWT.Secret))
}

func matchPath(path string, patterns []string) bool {
	for _, p := range patterns {
		if p == path {
			return true
		}
		if strings.HasSuffix(p, "*") {
			prefix := strings.TrimSuffix(p, "*")
			if strings.HasPrefix(path, prefix) {
				return true
			}
		}
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

// RequireRole 角色权限中间件
func (sm *SecurityMiddleware) RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("role")
		if !exists {
			resp.ErrorWithCode(c, http.StatusUnauthorized, 1002, "Unauthorized", nil)
			c.Abort()
			return
		}

		userRoleStr := userRole.(string)
		for _, role := range roles {
			if userRoleStr == role {
				c.Next()
				return
			}
		}

		log.WithContext(c.Request.Context()).Warn("Insufficient permissions",
			log.String("user_role", userRoleStr),
			log.Any("required_roles", roles),
		)

		resp.ErrorWithCode(c, http.StatusForbidden, 1003, "Forbidden", nil)
		c.Abort()
	}
}

// Stop 停止后台清理协程
func (sm *SecurityMiddleware) Stop() {
	if sm == nil {
		return
	}
	sm.stopOnce.Do(func() {
		close(sm.stopCh)
	})
	// 等待清理协程退出
	sm.wg.Wait()
}
