package router

import (
	"recharge-go/internal/controller"

	"github.com/gin-gonic/gin"
)

func RegisterUserGradeRoutes(r *gin.RouterGroup, c *controller.UserGradeController) {
	// 用户等级管理
	gradeGroup := r.Group("/user-grades")
	{
		gradeGroup.GET("/list", c.List)
		gradeGroup.POST("", c.Create)
		gradeGroup.PUT("/:id", c.Update)
		gradeGroup.DELETE("/:id", c.Delete)
		gradeGroup.GET("/:id", c.Get)
		gradeGroup.POST("/assign", c.AssignUserGrade)
		gradeGroup.GET("/user/:user_id", c.GetUserGrade)
		gradeGroup.POST("/remove", c.RemoveUserGrade)
		gradeGroup.PUT("/:id/status", c.UpdateGradeStatus)
	}
}
