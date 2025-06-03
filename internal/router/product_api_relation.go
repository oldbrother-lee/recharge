package router

import (
	"recharge-go/internal/controller"
	"recharge-go/internal/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterProductAPIRelationRoutes 注册商品接口关联路由
func RegisterProductAPIRelationRoutes(r *gin.RouterGroup, ctrl *controller.ProductAPIRelationController) {
	// 需要认证的路由组
	auth := r.Group("/product-api-relations")
	auth.Use(middleware.Auth())
	{
		// 创建商品接口关联
		auth.POST("", ctrl.Create)
		// 更新商品接口关联
		auth.PUT("/:id", ctrl.Update)
		// 删除商品接口关联
		auth.DELETE("/:id", ctrl.Delete)
		// 获取商品接口关联详情
		auth.GET("/:id", ctrl.GetByID)
		// 获取商品接口关联列表
		auth.GET("", ctrl.GetList)
	}
}
