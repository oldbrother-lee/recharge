package handler

import (
    "recharge-go/internal/repository"
    resp "recharge-go/pkg/utils/response"
    "strconv"

    "github.com/gin-gonic/gin"
)

type TaskOrderHandler struct {
	taskOrderRepo *repository.TaskOrderRepository
}

func NewTaskOrderHandler(taskOrderRepo *repository.TaskOrderRepository) *TaskOrderHandler {
	return &TaskOrderHandler{
		taskOrderRepo: taskOrderRepo,
	}
}

// List 获取任务订单列表
func (h *TaskOrderHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	orders, total, err := h.taskOrderRepo.List(page, pageSize)
    if err != nil {
        resp.Error(c, 500, "Internal server error")
        return
    }

    resp.Success(c, gin.H{
        "list":  orders,
        "total": total,
    })
}

// GetByOrderNumber 根据订单号获取任务订单
func (h *TaskOrderHandler) GetByOrderNumber(c *gin.Context) {
	orderNumber := c.Param("order_number")
	order, err := h.taskOrderRepo.GetByOrderNumber(orderNumber)
    if err != nil {
        resp.Error(c, 500, "Internal server error")
        return
    }

    resp.Success(c, order)
}
