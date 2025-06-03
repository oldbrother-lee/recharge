package router

import (
	"recharge-go/internal/controller"
	"recharge-go/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRewardRoutes(r *gin.Engine, rewardController *controller.RewardController) {
	reward := r.Group("/api/rewards")
	reward.Use(middleware.Auth())
	{
		reward.GET("", rewardController.GetRewards)
		reward.GET("/:id", rewardController.GetReward)
		reward.PUT("/:id/status", rewardController.UpdateRewardStatus)
	}
}
