package router

import (
	"recharge-go/internal/controller"
	"recharge-go/internal/middleware"
	"recharge-go/internal/service"

	"github.com/gin-gonic/gin"
)

// RegisterProductTypeRoutes 注册产品类型相关路由
func RegisterProductTypeRoutes(r *gin.RouterGroup, productTypeController *controller.ProductTypeController, userService *service.UserService) {
	// 产品类型管理路由组
	productType := r.Group("/product-type")
	{
		// 需要超级管理员权限的路由
		admin := productType.Group("")
		admin.Use(middleware.CheckSuperAdmin(userService))
		{
			admin.GET("/list", productTypeController.List)                 // 获取产品类型列表
			admin.POST("", productTypeController.Create)                   // 创建产品类型
			admin.PUT("/:id", productTypeController.Update)                // 更新产品类型
			admin.DELETE("/:id", productTypeController.Delete)             // 删除产品类型
			admin.GET("/categories", productTypeController.ListCategories) // 获取产品类型分类列表
		}

		// 公共路由
		productType.GET("/:id", productTypeController.GetByID) // 获取产品类型详情
	}
}
