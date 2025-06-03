package handler

import (
	"recharge-go/internal/model"
	"recharge-go/internal/repository"
	"recharge-go/internal/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

type PlatformHandler struct {
	platformRepo repository.PlatformRepository
}

func NewPlatformHandler(platformRepo repository.PlatformRepository) *PlatformHandler {
	return &PlatformHandler{
		platformRepo: platformRepo,
	}
}

// List 获取平台列表
func (h *PlatformHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	req := &model.PlatformListRequest{
		Page:     page,
		PageSize: pageSize,
	}

	platforms, total, err := h.platformRepo.ListPlatforms(req)
	if err != nil {
		utils.Error(c, 500, "Internal server error")
		return
	}

	utils.Success(c, gin.H{
		"list":  platforms,
		"total": total,
	})
}

// Get 获取平台详情
func (h *PlatformHandler) Get(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.Error(c, 400, "Invalid ID")
		return
	}

	platform, err := h.platformRepo.GetPlatformByID(id)
	if err != nil {
		utils.Error(c, 500, "Internal server error")
		return
	}

	utils.Success(c, platform)
}

// Create 创建平台
func (h *PlatformHandler) Create(c *gin.Context) {
	var platform model.Platform
	if err := c.ShouldBindJSON(&platform); err != nil {
		utils.Error(c, 400, "Invalid parameters")
		return
	}

	if err := h.platformRepo.CreatePlatform(&platform); err != nil {
		utils.Error(c, 500, "Internal server error")
		return
	}

	utils.Success(c, platform)
}

// Update 更新平台
func (h *PlatformHandler) Update(c *gin.Context) {
	var platform model.Platform
	if err := c.ShouldBindJSON(&platform); err != nil {
		utils.Error(c, 400, "Invalid parameters")
		return
	}

	if err := h.platformRepo.UpdatePlatform(&platform); err != nil {
		utils.Error(c, 500, "Internal server error")
		return
	}

	utils.Success(c, platform)
}

// Delete 删除平台
func (h *PlatformHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.Error(c, 400, "Invalid ID")
		return
	}

	if err := h.platformRepo.Delete(id); err != nil {
		utils.Error(c, 500, "Internal server error")
		return
	}

	utils.Success(c, nil)
}
