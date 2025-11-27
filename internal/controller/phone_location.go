package controller

import (
    "net/http"
    "strconv"

    "recharge-go/internal/model"
    "recharge-go/internal/service"
    resp "recharge-go/pkg/utils/response"

    "github.com/gin-gonic/gin"
)

type PhoneLocationController struct {
	service *service.PhoneLocationService
}

func NewPhoneLocationController(service *service.PhoneLocationService) *PhoneLocationController {
	return &PhoneLocationController{
		service: service,
	}
}

// List 获取手机归属地列表
func (c *PhoneLocationController) List(ctx *gin.Context) {
	var req model.PhoneLocationListRequest
    if err := ctx.ShouldBindQuery(&req); err != nil {
        resp.Error(ctx, http.StatusBadRequest, "参数错误")
        return
    }

    data, err := c.service.List(&req)
    if err != nil {
        resp.Error(ctx, http.StatusInternalServerError, "获取手机归属地列表失败")
        return
    }

    resp.Success(ctx, data)
}

// Create 创建手机归属地
func (c *PhoneLocationController) Create(ctx *gin.Context) {
	var location model.PhoneLocation
    if err := ctx.ShouldBindJSON(&location); err != nil {
        resp.Error(ctx, http.StatusBadRequest, "无效的输入数据")
        return
    }

	if err := c.service.Create(&location); err != nil {
		if err == service.ErrPhoneNumberExists {
            resp.Error(ctx, http.StatusBadRequest, "手机号已存在")
            return
        }
        resp.Error(ctx, http.StatusInternalServerError, "创建手机归属地失败")
        return
    }

    resp.Success(ctx, location)
}

// Update 更新手机归属地
func (c *PhoneLocationController) Update(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
    if err != nil {
        resp.Error(ctx, http.StatusBadRequest, "无效的ID")
        return
    }

	var location model.PhoneLocation
    if err := ctx.ShouldBindJSON(&location); err != nil {
        resp.Error(ctx, http.StatusBadRequest, "无效的输入数据")
        return
    }
	location.ID = id

	if err := c.service.Update(&location); err != nil {
        if err == service.ErrRecordNotFound {
            resp.Error(ctx, http.StatusNotFound, "手机归属地不存在")
            return
        }
        resp.Error(ctx, http.StatusInternalServerError, "更新手机归属地失败")
        return
    }

    resp.Success(ctx, location)
}

// Delete 删除手机归属地
func (c *PhoneLocationController) Delete(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
    if err != nil {
        resp.Error(ctx, http.StatusBadRequest, "无效的ID")
        return
    }

    if err := c.service.Delete(id); err != nil {
        if err == service.ErrRecordNotFound {
            resp.Error(ctx, http.StatusNotFound, "手机归属地不存在")
            return
        }
        resp.Error(ctx, http.StatusInternalServerError, "删除手机归属地失败")
        return
    }

    resp.Success(ctx, "删除成功")
}
