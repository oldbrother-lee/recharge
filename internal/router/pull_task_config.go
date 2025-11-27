package router

import (
	"recharge-go/internal/controller"
	"recharge-go/internal/repository"
	"recharge-go/pkg/database"

	"github.com/gin-gonic/gin"
)

// RegisterPullTaskConfigRoutes 统一拉单配置路由（仅管理员）
func RegisterPullTaskConfigRoutes(auth *gin.RouterGroup) {
	db := database.DB
	repo := repository.NewPullTaskConfigRepository(db)
	ctrl := controller.NewPullTaskConfigController(repo)

	r := auth.Group("/pull/configs")
	{
		r.POST("", ctrl.Create)
		r.PUT(":id", ctrl.Update)
		r.DELETE(":id", ctrl.Delete)
		r.GET("", ctrl.List)
	}
}
