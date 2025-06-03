package middleware

import (
	"recharge-go/internal/utils"
	"strings"

	"github.com/gin-gonic/gin"
)

func Auth() gin.HandlerFunc {
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

		ctx.Set("user_id", claims.UserID)
		ctx.Set("username", claims.Username)
		ctx.Set("roles", claims.Roles)
		ctx.Next()
	}
}
