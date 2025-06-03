package router

import (
	"recharge-go/internal/controller"
	"recharge-go/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRebateRoutes(r *gin.Engine, rebateController *controller.RebateController) {
	rebate := r.Group("/api/rebates")
	rebate.Use(middleware.Auth())
	{
		rebate.GET("", rebateController.GetRebates)
		rebate.GET("/:id", rebateController.GetRebate)
		rebate.PUT("/:id/status", rebateController.UpdateRebateStatus)
	}
}
