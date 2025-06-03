package middleware

import (
	"fmt"
	"recharge-go/internal/service"
	"recharge-go/internal/utils"

	"github.com/gin-gonic/gin"
)

func CheckSuperAdmin(userService *service.UserService) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userID := ctx.GetInt64("user_id")
		fmt.Println(userID, "userID")
		// 获取用户角色
		userWithRoles, err := userService.GetUserWithRoles(ctx, userID)
		if err != nil {
			utils.Error(ctx, 500, "Failed to get user roles")
			ctx.Abort()
			return
		}
		fmt.Println(userWithRoles, "ttttttt")
		// 检查是否是超级管理员
		isSuperAdmin := false
		for _, role := range userWithRoles.Roles {
			if role.Code == "SUPER_ADMIN" {
				isSuperAdmin = true
				break
			}
		}

		if !isSuperAdmin {
			utils.Error(ctx, 403, "Permission denied")
			ctx.Abort()
			return
		}

		ctx.Next()
	}
}
