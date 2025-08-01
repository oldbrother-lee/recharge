package middleware

import (
	"recharge-go/internal/repository"
	"recharge-go/internal/utils"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AuthMiddleware 认证中间件结构体
type AuthMiddleware struct {
	userRepo *repository.UserRepository
}

// NewAuthMiddleware 创建认证中间件
func NewAuthMiddleware(db *gorm.DB) *AuthMiddleware {
	return &AuthMiddleware{
		userRepo: repository.NewUserRepository(db),
	}
}

// Auth 认证中间件
func (a *AuthMiddleware) Auth() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authHeader := ctx.GetHeader("Authorization")
		if authHeader == "" {
			utils.Error(ctx, 401, "Authorization header is required")
			ctx.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			utils.Error(ctx, 401, "Authorization header format must be Bearer {token}")
			ctx.Abort()
			return
		}

		claims, err := utils.ValidateJWT(parts[1], false)
		if err != nil {
			utils.Error(ctx, 401, "Invalid token")
			ctx.Abort()
			return
		}

		// 验证用户当前状态
		user, err := a.userRepo.GetByID(ctx, claims.UserID)
		if err != nil {
			utils.Error(ctx, 401, "User not found")
			ctx.Abort()
			return
		}

		// 检查用户是否被禁用
		if user.Status != 1 {
			utils.Error(ctx, 401, "Account has been disabled")
			ctx.Abort()
			return
		}

		ctx.Set("user_id", claims.UserID)
		ctx.Set("username", claims.Username)
		ctx.Set("roles", claims.Roles)
		ctx.Next()
	}
}

// globalAuthMiddleware 全局认证中间件实例
var globalAuthMiddleware *AuthMiddleware

// InitGlobalAuthMiddleware 初始化全局认证中间件
func InitGlobalAuthMiddleware(db *gorm.DB) {
	globalAuthMiddleware = NewAuthMiddleware(db)
}

// Auth 兼容旧版本的全局认证函数
// 现在会检查用户状态
func Auth() gin.HandlerFunc {
	if globalAuthMiddleware == nil {
		// 如果全局中间件未初始化，返回一个错误处理函数
		return func(ctx *gin.Context) {
			utils.Error(ctx, 500, "Authentication middleware not initialized")
			ctx.Abort()
		}
	}
	return globalAuthMiddleware.Auth()
}
