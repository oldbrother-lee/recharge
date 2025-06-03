package controller

import (
	"net/http"
	"recharge-go/internal/service"
	"recharge-go/internal/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

type RewardController struct {
	rewardService service.RewardService
}

func NewRewardController(rewardService service.RewardService) *RewardController {
	return &RewardController{
		rewardService: rewardService,
	}
}

// GetRewards 获取奖励列表
func (c *RewardController) GetRewards(ctx *gin.Context) {
	userID, _ := ctx.Get("user_id")
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "10"))

	rewards, total, err := c.rewardService.GetRewardsByUserID(ctx, userID.(int64), page, pageSize)
	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(ctx, gin.H{
		"list":  rewards,
		"total": total,
	})
}

// GetReward 获取奖励详情
func (c *RewardController) GetReward(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		utils.Error(ctx, http.StatusBadRequest, "Invalid ID")
		return
	}

	reward, err := c.rewardService.GetRewardByID(ctx, id)
	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(ctx, reward)
}

// UpdateRewardStatus 更新奖励状态
func (c *RewardController) UpdateRewardStatus(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		utils.Error(ctx, http.StatusBadRequest, "Invalid ID")
		return
	}

	var req struct {
		Status int `json:"status" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}

	err = c.rewardService.UpdateRewardStatus(ctx, id, req.Status)
	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(ctx, nil)
}
