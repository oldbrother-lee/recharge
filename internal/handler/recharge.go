package handler

import (
    "recharge-go/internal/service"
    "recharge-go/pkg/logger"

    "github.com/gin-gonic/gin"
)

// RechargeHandler 充值处理器
type RechargeHandler struct {
	rechargeService service.RechargeService
}

// NewRechargeHandler 创建充值处理器
func NewRechargeHandler(rechargeService service.RechargeService) *RechargeHandler {
	return &RechargeHandler{
		rechargeService: rechargeService,
	}
}

// HandleCallback 处理平台回调
func (h *RechargeHandler) HandleCallback(c *gin.Context) {
    platform := c.Param("platform")
    if platform == "" {
        c.JSON(400, gin.H{
            "code": "1001",
            "msg":  "platform is required",
        })
        return
    }

	// 读取请求体
    data, err := c.GetRawData()
    if err != nil {
        logger.Error("读取回调请求体失败: %v", err)
        c.JSON(400, gin.H{
            "code": "1001",
            "msg":  "invalid request body",
        })
        return
    }

    logger.WithContextCategory(c.Request.Context(), "callback").Info("收到平台回调",
        logger.StringV2("platform", platform),
        logger.StringV2("raw_body", string(data)),
        logger.StringV2("content_type", c.ContentType()),
        logger.StringV2("user_agent", c.Request.UserAgent()),
    )

	// 处理回调
	if err := h.rechargeService.HandleCallback(c.Request.Context(), platform, data); err != nil {
		logger.Error("处理回调失败: %v", err)
		c.JSON(200, gin.H{
			"code": "1002",
			"msg":  "handle callback failed",
		})
		return
	}

	// 返回成功响应
	c.JSON(200, gin.H{
		"code": "0000",
		"msg":  "success",
	})
}
