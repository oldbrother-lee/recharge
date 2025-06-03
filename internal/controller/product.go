package controller

import (
	"fmt"
	"net/http"
	"recharge-go/internal/model"
	"recharge-go/internal/service"
	"recharge-go/internal/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ProductController struct {
	productService *service.ProductService
}

func NewProductController(productService *service.ProductService) *ProductController {
	return &ProductController{
		productService: productService,
	}
}

// List 获取商品列表
// @Summary 获取商品列表
// @Description 获取商品列表
// @Tags 商品管理
// @Accept json
// @Produce json
// @Param page query int true "页码"
// @Param page_size query int true "每页数量"
// @Param type query int false "商品类型"
// @Param category query int false "商品分类"
// @Param isp query string false "运营商"
// @Param status query int false "状态"
// @Success 200 {object} response.Response{data=model.ProductListResponse}
// @Router /api/v1/product/list [get]
func (c *ProductController) List(ctx *gin.Context) {
	var req model.ProductListRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		fmt.Printf("参数绑定错误: %v\n", err)
		utils.Error(ctx, http.StatusBadRequest, fmt.Sprintf("参数错误: %v", err))
		return
	}

	fmt.Printf("绑定后的参数: %+v\n", req)

	resp, err := c.productService.List(ctx.Request.Context(), &req)
	if err != nil {
		fmt.Printf("服务调用错误: %v\n", err)
		utils.Error(ctx, http.StatusInternalServerError, fmt.Sprintf("服务错误: %v", err))
		return
	}

	fmt.Println("请求处理成功")
	utils.Success(ctx, resp)
}

// GetByID 获取商品详情
// @Summary 获取商品详情
// @Description 获取商品详情
// @Tags 商品管理
// @Accept json
// @Produce json
// @Param id path int true "商品ID"
// @Success 200 {object} response.Response{data=model.ProductDetailResponse}
// @Router /api/v1/product/{id} [get]
func (c *ProductController) GetByID(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		utils.Error(ctx, http.StatusBadRequest, "invalid id")
		return
	}

	resp, err := c.productService.GetByID(ctx.Request.Context(), id)
	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(ctx, resp)
}

// Create 创建商品
// @Summary 创建商品
// @Description 创建商品
// @Tags 商品管理
// @Accept json
// @Produce json
// @Param product body model.ProductCreateRequest true "商品信息"
// @Success 200 {object} response.Response{data=model.Product}
// @Router /api/v1/product [post]
func (c *ProductController) Create(ctx *gin.Context) {
	var req model.ProductCreateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}

	product, err := c.productService.Create(ctx.Request.Context(), &req)
	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(ctx, product)
}

// Update 更新商品
// @Summary 更新商品
// @Description 更新商品
// @Tags 商品管理
// @Accept json
// @Produce json
// @Param id path int true "商品ID"
// @Param product body model.ProductUpdateRequest true "商品信息"
// @Success 200 {object} response.Response{data=model.Product}
// @Router /api/v1/product/{id} [put]
func (c *ProductController) Update(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		utils.Error(ctx, http.StatusBadRequest, "invalid id")
		return
	}

	var req model.ProductUpdateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}

	req.ID = id
	product, err := c.productService.Update(ctx.Request.Context(), &req)
	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(ctx, product)
}

// Delete 删除商品
// @Summary 删除商品
// @Description 删除商品
// @Tags 商品管理
// @Accept json
// @Produce json
// @Param id path int true "商品ID"
// @Success 200 {object} response.Response
// @Router /api/v1/product/{id} [delete]
func (c *ProductController) Delete(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		utils.Error(ctx, http.StatusBadRequest, "invalid id")
		return
	}

	err = c.productService.Delete(ctx.Request.Context(), id)
	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(ctx, nil)
}

// ListCategories 获取商品分类列表
// @Summary 获取商品分类列表
// @Description 获取商品分类列表
// @Tags 商品管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=model.ProductCategoryListResponse}
// @Router /api/v1/product/categories [get]
func (c *ProductController) ListCategories(ctx *gin.Context) {
	resp, err := c.productService.ListCategories(ctx.Request.Context())
	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(ctx, resp)
}

// CreateCategory 创建商品分类
// @Summary 创建商品分类
// @Description 创建商品分类
// @Tags 商品管理
// @Accept json
// @Produce json
// @Param category body model.ProductCategoryCreateRequest true "商品分类信息"
// @Success 200 {object} response.Response{data=model.ProductCategory}
// @Router /api/v1/product/categories [post]
func (c *ProductController) CreateCategory(ctx *gin.Context) {
	var category model.ProductCategory
	if err := ctx.ShouldBindJSON(&category); err != nil {
		utils.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}

	if err := c.productService.CreateCategory(ctx.Request.Context(), &category); err != nil {
		utils.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(ctx, nil)
}

// UpdateCategory 更新商品分类
// @Summary 更新商品分类
// @Description 更新商品分类
// @Tags 商品管理
// @Accept json
// @Produce json
// @Param id path int true "商品分类ID"
// @Param category body model.ProductCategory true "商品分类信息"
// @Success 200 {object} response.Response
// @Router /api/v1/product/categories/{id} [put]
func (c *ProductController) UpdateCategory(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		utils.Error(ctx, http.StatusBadRequest, "invalid id")
		return
	}

	var category model.ProductCategory
	if err := ctx.ShouldBindJSON(&category); err != nil {
		utils.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}

	category.ID = id
	if err := c.productService.UpdateCategory(ctx.Request.Context(), &category); err != nil {
		utils.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(ctx, nil)
}

// DeleteCategory 删除商品分类
// @Summary 删除商品分类
// @Description 删除商品分类
// @Tags 商品管理
// @Accept json
// @Produce json
// @Param id path int true "商品分类ID"
// @Success 200 {object} response.Response
// @Router /api/v1/product/categories/{id} [delete]
func (c *ProductController) DeleteCategory(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		utils.Error(ctx, http.StatusBadRequest, "invalid id")
		return
	}

	err = c.productService.DeleteCategory(ctx.Request.Context(), id)
	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(ctx, nil)
}

// ListTypes 获取商品类型列表
// @Summary 获取商品类型列表
// @Description 获取商品类型列表
// @Tags 商品管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=[]model.ProductType}
// @Router /api/v1/product/types [get]
func (c *ProductController) ListTypes(ctx *gin.Context) {
	types, err := c.productService.ListTypes(ctx.Request.Context())
	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(ctx, types)
}
