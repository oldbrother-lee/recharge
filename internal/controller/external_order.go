package controller

import (
	"net/http"
	"recharge-go/internal/model"
	"recharge-go/internal/service"
	"recharge-go/internal/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ExternalOrderController struct {
	orderService service.OrderService
}

func NewExternalOrderController(orderService service.OrderService) *ExternalOrderController {
	return &ExternalOrderController{orderService: orderService}
}

type ExternalOrderCreateRequest struct {
	Mobile      string  `json:"mobile" binding:"required"`
	ProductID   int64   `json:"product_id" binding:"required"`
	OutTradeNum string  `json:"out_trade_num" binding:"required"`
	Amount      float64 `json:"amount" binding:"required"`
	BizType     string  `json:"biz_type"`
	NotifyURL   string  `json:"notify_url"`
	Param1      string  `json:"param1"`
	Param2      string  `json:"param2"`
	Param3      string  `json:"param3"`
	CustomerID  int64   `json:"customer_id"` // 可选，若有外部客户体系
	ISP         int     `json:"isp"`         // 可选，运营商
	Remark      string  `json:"remark"`
}

func (c *ExternalOrderController) CreateOrder(ctx *gin.Context) {
	var req ExternalOrderCreateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}

	// 可选：签名校验、IP白名单等安全措施

	order := &model.Order{
		Mobile:      req.Mobile,
		ProductID:   req.ProductID,
		OutTradeNum: req.OutTradeNum,
		TotalPrice:  req.Amount,
		Price:       req.Amount,
		Status:      model.OrderStatusPendingPayment,
		IsDel:       0,
		Param1:      req.Param1,
		Param2:      req.Param2,
		Param3:      req.Param3,
		CustomerID:  req.CustomerID,
		ISP:         req.ISP,
		Remark:      req.Remark,
		Client:      2, // 例如2代表外部API
	}

	if err := c.orderService.CreateOrder(ctx, order); err != nil {
		utils.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(ctx, gin.H{
		"order_id":     order.ID,
		"order_number": order.OrderNumber,
		"status":       order.Status,
		"create_time":  order.CreateTime,
	})
}

// GetOrder 获取订单
func (c *ExternalOrderController) GetOrder(ctx *gin.Context) {
	id := ctx.Param("id")
	orderID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		utils.Error(ctx, http.StatusBadRequest, "invalid order id")
		return
	}

	order, err := c.orderService.GetOrderByID(ctx, orderID)
	if err != nil {
		utils.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(ctx, order)
}
