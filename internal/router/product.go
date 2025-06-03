package router

import (
	"recharge-go/internal/controller"
	"recharge-go/internal/middleware"
	"recharge-go/internal/service"

	"github.com/gin-gonic/gin"
)

// RegisterProductRoutes 注册商品相关路由
func RegisterProductRoutes(r *gin.RouterGroup, productController *controller.ProductController, userService *service.UserService) {
	product := r.Group("/product")
	{
		// 普通用户可访问的路由
		product.GET("/list", productController.List)
		product.GET("/:id", productController.GetByID)
		product.GET("/categories", productController.ListCategories)
		product.GET("/types", productController.ListTypes)
		// 需要超级管理员权限的路由
		adminProduct := product.Group("")
		adminProduct.Use(middleware.CheckSuperAdmin(userService))
		{
			adminProduct.POST("", productController.Create)
			adminProduct.PUT("/:id", productController.Update)
			adminProduct.DELETE("/:id", productController.Delete)
			adminProduct.POST("/category", productController.CreateCategory)
			adminProduct.PUT("/category/:id", productController.UpdateCategory)
			adminProduct.DELETE("/category/:id", productController.DeleteCategory)
		}
	}
}
