package router

import (
	"recharge-go/internal/controller"
	"recharge-go/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterDistributionRoutes(r *gin.Engine, ctrl *controller.DistributionController) {
	distribution := r.Group("/api/distribution")
	{
		// 分销等级管理
		distribution.POST("/grades", middleware.Auth(), ctrl.CreateGrade)
		distribution.PUT("/grades/:id", middleware.Auth(), ctrl.UpdateGrade)
		distribution.GET("/grades", middleware.Auth(), ctrl.ListGrades)

		// 分销规则管理
		distribution.POST("/rules", middleware.Auth(), ctrl.CreateRule)
		distribution.GET("/rules", middleware.Auth(), ctrl.ListRules)

		// 提现管理
		distribution.POST("/withdrawals", middleware.Auth(), ctrl.CreateWithdrawal)
		distribution.GET("/withdrawals", middleware.Auth(), ctrl.ListWithdrawals)

		// 分销统计
		distribution.GET("/statistics", middleware.Auth(), ctrl.GetStatistics)
	}
}
