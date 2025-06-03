package router

import (
	"recharge-go/internal/controller"

	"github.com/gin-gonic/gin"
)

func RegisterStatisticsRoutes(r *gin.RouterGroup, statisticsController *controller.StatisticsController) {
	statistics := r.Group("/statistics")
	{
		order := statistics.Group("/order")
		{
			order.GET("/overview", statisticsController.GetOrderOverview)
			order.GET("/operator", statisticsController.GetOperatorStatistics)
			order.GET("/daily", statisticsController.GetDailyStatistics)
			order.GET("/trend", statisticsController.GetTrendStatistics)
			order.GET("/realtime", statisticsController.GetOrderRealtimeStatistics)
			order.GET("/isp-count", statisticsController.GetOperatorOrderCount)
			order.POST("/trigger", statisticsController.TriggerStatistics)
		}
	}
}
