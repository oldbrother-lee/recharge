package controller

import (
	"net/http"
	"recharge-go/internal/model"
	"recharge-go/internal/service"
	"recharge-go/internal/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ProductTypeController 产品类型控制器
type ProductTypeController struct {
	service *service.ProductTypeService
}

// NewProductTypeController 创建产品类型控制器实例
func NewProductTypeController(service *service.ProductTypeService) *ProductTypeController {
	return &ProductTypeController{
		service: service,
	}
}

// List 获取产品类型列表
// @Summary 获取产品类型列表
// @Description 获取产品类型列表
// @Tags 产品类型管理
// @Accept json
// @Produce json
// @Param page query int true "页码"
// @Param page_size query int true "每页数量"
// @Param type_name query string false "类型名称"
// @Param typec_id query int false "类型分类ID"
// @Param status query int false "状态"
// @Param account_type query int false "充值账号类型"
// @Success 200 {object} utils.Response{data=model.ProductTypeListResponse}
// @Router /api/v1/product-type/list [get]
func (c *ProductTypeController) List(ctx *gin.Context) {
	var req model.ProductTypeListRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := c.service.List(&req)
	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(ctx, resp)
}

// Create 创建产品类型
// @Summary 创建产品类型
// @Description 创建产品类型
// @Tags 产品类型管理
// @Accept json
// @Produce json
// @Param request body model.ProductTypeCreateRequest true "创建产品类型请求"
// @Success 200 {object} utils.Response
// @Router /api/v1/product-type [post]
func (c *ProductTypeController) Create(ctx *gin.Context) {
	var req model.ProductTypeCreateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}

	err := c.service.Create(&req)
	if err != nil {
		if err == service.ErrProductTypeNameExists {
			utils.Error(ctx, http.StatusBadRequest, err.Error())
			return
		}
		if err == service.ErrProductTypeCategoryNotFound {
			utils.Error(ctx, http.StatusBadRequest, err.Error())
			return
		}
		utils.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(ctx, nil)
}

// Update 更新产品类型
// @Summary 更新产品类型
// @Description 更新产品类型
// @Tags 产品类型管理
// @Accept json
// @Produce json
// @Param id path int true "产品类型ID"
// @Param request body model.ProductTypeUpdateRequest true "更新产品类型请求"
// @Success 200 {object} utils.Response
// @Router /api/v1/product-type/{id} [put]
func (c *ProductTypeController) Update(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		utils.Error(ctx, http.StatusBadRequest, "无效的ID")
		return
	}

	var req model.ProductTypeUpdateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}
	req.ID = id

	err = c.service.Update(&req)
	if err != nil {
		if err == service.ErrProductTypeNotFound {
			utils.Error(ctx, http.StatusNotFound, err.Error())
			return
		}
		if err == service.ErrProductTypeNameExists {
			utils.Error(ctx, http.StatusBadRequest, err.Error())
			return
		}
		if err == service.ErrProductTypeCategoryNotFound {
			utils.Error(ctx, http.StatusBadRequest, err.Error())
			return
		}
		utils.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(ctx, nil)
}

// Delete 删除产品类型
// @Summary 删除产品类型
// @Description 删除产品类型
// @Tags 产品类型管理
// @Accept json
// @Produce json
// @Param id path int true "产品类型ID"
// @Success 200 {object} utils.Response
// @Router /api/v1/product-type/{id} [delete]
func (c *ProductTypeController) Delete(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		utils.Error(ctx, http.StatusBadRequest, "无效的ID")
		return
	}

	err = c.service.Delete(id)
	if err != nil {
		if err == service.ErrProductTypeNotFound {
			utils.Error(ctx, http.StatusNotFound, err.Error())
			return
		}
		utils.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(ctx, nil)
}

// GetByID 获取产品类型详情
// @Summary 获取产品类型详情
// @Description 获取产品类型详情
// @Tags 产品类型管理
// @Accept json
// @Produce json
// @Param id path int true "产品类型ID"
// @Success 200 {object} utils.Response{data=model.ProductType}
// @Router /api/v1/product-type/{id} [get]
func (c *ProductTypeController) GetByID(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		utils.Error(ctx, http.StatusBadRequest, "无效的ID")
		return
	}

	productType, err := c.service.GetByID(id)
	if err != nil {
		if err == service.ErrProductTypeNotFound {
			utils.Error(ctx, http.StatusNotFound, err.Error())
			return
		}
		utils.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(ctx, productType)
}

// ListCategories 获取产品类型分类列表
// @Summary 获取产品类型分类列表
// @Description 获取产品类型分类列表
// @Tags 产品类型管理
// @Accept json
// @Produce json
// @Param page query int true "页码"
// @Param page_size query int true "每页数量"
// @Param name query string false "分类名称"
// @Success 200 {object} utils.Response{data=model.ProductTypeCategoryListResponse}
// @Router /api/v1/product-type/categories [get]
func (c *ProductTypeController) ListCategories(ctx *gin.Context) {
	var req model.ProductTypeCategoryListRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := c.service.ListCategories(&req)
	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(ctx, resp)
}
