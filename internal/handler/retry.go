package handler

import (
	"net/http"
	"recharge-go/internal/service"
	"strconv"

	"github.com/gin-gonic/gin"
)

type RetryHandler struct {
	retryService *service.RetryService
}

func NewRetryHandler(retryService *service.RetryService) *RetryHandler {
	return &RetryHandler{
		retryService: retryService,
	}
}

// TriggerRetry 手动触发重试
func (h *RetryHandler) TriggerRetry(c *gin.Context) {
	recordID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的重试记录ID",
		})
		return
	}

	if err := h.retryService.TriggerRetry(c.Request.Context(), recordID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "触发重试失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "重试已触发",
	})
}

func (h *RetryHandler) RegisterRoutes(r *gin.RouterGroup) {
	retry := r.Group("/retry")
	{
		retry.POST("/:id/trigger", h.TriggerRetry)
	}
}
