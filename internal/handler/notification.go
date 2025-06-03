package handler

import (
	"net/http"
	"recharge-go/internal/model/notification"
	notificationService "recharge-go/internal/service/notification"
	"recharge-go/internal/utils"
	"recharge-go/pkg/logger"

	"strconv"

	"github.com/gin-gonic/gin"
)

// NotificationHandler 通知处理器
type NotificationHandler struct {
	notificationService notificationService.NotificationService
}

// NewNotificationHandler 创建通知处理器
func NewNotificationHandler(notificationService notificationService.NotificationService) *NotificationHandler {
	return &NotificationHandler{
		notificationService: notificationService,
	}
}

// CreateNotificationRequest 创建通知请求
type CreateNotificationRequest struct {
	OrderID          int64  `json:"order_id" binding:"required"`
	PlatformCode     string `json:"platform_code" binding:"required"`
	NotificationType string `json:"notification_type" binding:"required"`
	Content          string `json:"content" binding:"required"`
}

// CreateNotification 创建通知
func (h *NotificationHandler) CreateNotification(c *gin.Context) {
	var req struct {
		OrderID          int64  `json:"order_id" binding:"required"`
		PlatformCode     string `json:"platform_code" binding:"required"`
		NotificationType string `json:"notification_type" binding:"required"`
		Content          string `json:"content" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "invalid request parameters")
		return
	}

	record := &notification.NotificationRecord{
		OrderID:          req.OrderID,
		PlatformCode:     req.PlatformCode,
		NotificationType: req.NotificationType,
		Content:          req.Content,
		Status:           1, // 待处理
	}

	if err := h.notificationService.CreateNotification(c.Request.Context(), record); err != nil {
		logger.Error("create notification failed", "error", err)
		utils.Error(c, http.StatusInternalServerError, "create notification failed")
		return
	}

	utils.Success(c, nil)
}

// GetNotificationStatus 获取通知状态
func (h *NotificationHandler) GetNotificationStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "invalid notification id")
		return
	}

	record, err := h.notificationService.GetNotificationStatus(c.Request.Context(), id)
	if err != nil {
		logger.Error("get notification status failed", "error", err)
		utils.Error(c, http.StatusInternalServerError, "get notification status failed")
		return
	}

	utils.Success(c, record)
}

// ListNotifications 获取通知列表
func (h *NotificationHandler) ListNotifications(c *gin.Context) {
	var req struct {
		OrderID          int64  `form:"order_id"`
		PlatformCode     string `form:"platform_code"`
		NotificationType string `form:"notification_type"`
		Status           int    `form:"status"`
		Page             int    `form:"page" binding:"required,min=1"`
		PageSize         int    `form:"page_size" binding:"required,min=1,max=100"`
	}

	if err := c.ShouldBindQuery(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "invalid request parameters")
		return
	}

	params := make(map[string]interface{})
	if req.OrderID > 0 {
		params["order_id"] = req.OrderID
	}
	if req.PlatformCode != "" {
		params["platform_code"] = req.PlatformCode
	}
	if req.NotificationType != "" {
		params["notification_type"] = req.NotificationType
	}
	if req.Status > 0 {
		params["status"] = req.Status
	}

	records, total, err := h.notificationService.ListNotifications(c.Request.Context(), params, req.Page, req.PageSize)
	if err != nil {
		logger.Error("list notifications failed", "error", err)
		utils.Error(c, http.StatusInternalServerError, "list notifications failed")
		return
	}

	utils.Success(c, gin.H{
		"total":     total,
		"records":   records,
		"page":      req.Page,
		"page_size": req.PageSize,
	})
}

// RetryFailedNotification 重试失败的通知
func (h *NotificationHandler) RetryFailedNotification(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "invalid notification id")
		return
	}

	if err := h.notificationService.RetryFailedNotification(c.Request.Context(), id); err != nil {
		logger.Error("retry failed notification failed", "error", err)
		utils.Error(c, http.StatusInternalServerError, "retry failed notification failed")
		return
	}

	utils.Success(c, nil)
}

// RegisterRoutes 注册路由
func (h *NotificationHandler) RegisterRoutes(r *gin.RouterGroup) {
	notification := r.Group("/notification")
	{
		notification.POST("", h.CreateNotification)
		notification.GET("/:id", h.GetNotificationStatus)
		notification.GET("", h.ListNotifications)
		notification.POST("/:id/retry", h.RetryFailedNotification)
	}
}
