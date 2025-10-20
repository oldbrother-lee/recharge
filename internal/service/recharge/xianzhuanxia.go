package recharge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"recharge-go/internal/model"
	"recharge-go/pkg/logger"
	"recharge-go/pkg/signature"
	"strconv"
	"time"

	"recharge-go/internal/repository"

	"gorm.io/gorm"
)

// XianzhuanxiaPlatform 闲赚侠平台实现
type XianzhuanxiaPlatform struct {
	platformRepo repository.PlatformRepository
}

// NewXianzhuanxiaPlatform 创建闲转侠平台实例
func NewXianzhuanxiaPlatform(db *gorm.DB) *XianzhuanxiaPlatform {
	return &XianzhuanxiaPlatform{
		platformRepo: repository.NewPlatformRepository(db),
	}
}

// GetName 获取平台名称
func (p *XianzhuanxiaPlatform) GetName() string {
	return "xianzhuanxia"
}

// getAPIKeyAndSecret 获取API密钥和密钥
func (p *XianzhuanxiaPlatform) getAPIKeyAndSecret(ctx context.Context, apiID uint) (string, string, string, error) {
	account, err := p.platformRepo.GetAccountByID(ctx, int64(apiID))
	if err != nil {
		return "", "", "", fmt.Errorf("获取平台账号信息失败: %v", err)
	}
	return account.AppKey, account.AppSecret, account.AccountName, nil
}

// SubmitOrderResult 提交订单结果
type SubmitOrderResult struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		OrderID string `json:"order_id"`
	} `json:"data"`
}

// QueryOrderStatusResult 查询订单状态结果
type QueryOrderStatusResult struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Status int `json:"status"`
	} `json:"data"`
}

// SubmitOrder 提交订单
func (p *XianzhuanxiaPlatform) SubmitOrder(ctx context.Context, order *model.Order, api *model.PlatformAPI, apiParam *model.PlatformAPIParam) error {
    l := logger.WithContextCategory(ctx, "recharge")
    if l != nil {
        l.Info("开始提交闲赚侠订单",
            logger.Int64V2("order_id", order.ID),
            logger.StringV2("order_number", order.OrderNumber),
            logger.StringV2("mobile", order.Mobile),
        )
    }

	// 获取API密钥和密钥
	appKey, _, accountName, err := p.getAPIKeyAndSecret(ctx, uint(api.AccountID))
	if err != nil {
		return fmt.Errorf("获取API密钥失败: %v", err)
	}

	// 构建请求参数
	params := map[string]string{
		"orderNo":     order.OrderNumber,
		"accountNum":  order.Mobile,
		"taskGoodsId": apiParam.ProductID,
		"ip":          "192.168.31.2",
		"notifyUrl":   api.CallbackURL,
		"maxWaitTime": strconv.Itoa(600),
	}

	// 生成签名
    authToken, _, err := signature.GenerateXianzhuanxiaSignature(params, appKey, accountName)
    if err != nil {
        if l != nil {
            l.Error("生成签名失败",
                logger.ErrorV2(err),
                logger.AnyV2("params_keys", func() []string { ks := make([]string, 0, len(params)); for k := range params { ks = append(ks, k) }; return ks }()),
            )
        }
        return fmt.Errorf("生成签名失败: %v", err)
    }

	// 发送请求
    jsonData, err := json.Marshal(params)
    if err != nil {
        if l != nil {
            l.Error("序列化请求参数失败",
                logger.ErrorV2(err),
                logger.AnyV2("params_keys", func() []string { ks := make([]string, 0, len(params)); for k := range params { ks = append(ks, k) }; return ks }()),
            )
        }
        return fmt.Errorf("序列化请求参数失败: %v", err)
    }

    req, err := http.NewRequestWithContext(ctx, "POST", api.URL, bytes.NewBuffer(jsonData))
    if err != nil {
        if l != nil {
            l.Error("创建HTTP请求失败",
                logger.ErrorV2(err),
                logger.StringV2("url", order.PlatformURL),
            )
        }
        return fmt.Errorf("创建HTTP请求失败: %v", err)
    }

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Auth_Token", authToken)
    if l != nil {
        l.Info("闲赚侠请求",
            logger.StringV2("method", req.Method),
            logger.StringV2("url", req.URL.String()),
            logger.IntV2("content_length", int(req.ContentLength)),
            logger.AnyV2("header_keys", func() []string { ks := make([]string, 0, len(req.Header)); for k := range req.Header { ks = append(ks, k) }; return ks }()),
        )
    }
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
    if err != nil {
        if l != nil {
            l.Error("发送HTTP请求失败",
                logger.ErrorV2(err),
                logger.StringV2("url", req.URL.String()),
            )
        }
        return fmt.Errorf("发送HTTP请求失败: %v", err)
    }
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
    if err != nil {
        if l != nil {
            l.Error("读取响应内容失败",
                logger.ErrorV2(err),
                logger.IntV2("status_code", resp.StatusCode),
            )
        }
        return fmt.Errorf("读取响应内容失败: %v", err)
    }

	// 记录平台原始响应内容
    if l != nil {
        l.Info("平台响应内容",
            logger.Int64V2("order_id", order.ID),
            logger.StringV2("order_number", order.OrderNumber),
            logger.StringV2("response_preview", func(b []byte) string { if len(b) > 300 { return string(b[:300]) + "..." } ; return string(b) }(body)),
        )
    }

	// 解析响应
	var result SubmitOrderResult
    if err := json.Unmarshal(body, &result); err != nil {
        if l != nil {
            l.Error("解析响应内容失败",
                logger.ErrorV2(err),
                logger.StringV2("body_preview", func(b []byte) string { if len(b) > 300 { return string(b[:300]) + "..." } ; return string(b) }(body)),
            )
        }
        return fmt.Errorf("解析响应内容失败: %v", err)
    }

    if result.Code != 0 {
        if l != nil {
            l.Error("提交订单失败",
                logger.StringV2("platform", "xianzhuanxia"),
                logger.Int64V2("order_id", order.ID),
                logger.StringV2("order_number", order.OrderNumber),
                logger.StringV2("error", result.Message),
                logger.StringV2("response_preview", func(b []byte) string { if len(b) > 300 { return string(b[:300]) + "..." } ; return string(b) }(body)),
            )
        }
        return fmt.Errorf("submit order failed: %v", result.Message)
    }

	// 更新订单信息
	order.APIOrderNumber = result.Data.OrderID
	order.APITradeNum = result.Data.OrderID

    if l != nil {
        l.Info("提交订单成功",
            logger.Int64V2("order_id", order.ID),
            logger.StringV2("order_number", order.OrderNumber),
            logger.StringV2("api_order_id", result.Data.OrderID),
        )
    }

	return nil
}

// QueryOrderStatus 查询订单状态
func (p *XianzhuanxiaPlatform) QueryOrderStatus(ctx context.Context, order *model.Order) (model.OrderStatus, error) {
    l := logger.WithContextCategory(ctx, "recharge")
    if l != nil {
        l.Info("开始查询闲赚侠订单状态",
            logger.Int64V2("order_id", order.ID),
            logger.StringV2("order_number", order.OrderNumber),
            logger.StringV2("api_order_id", order.APIOrderNumber),
        )
    }

	// 获取API密钥和密钥
	appKey, appSecret, _, err := p.getAPIKeyAndSecret(ctx, uint(order.APICurID))
	if err != nil {
		return 0, fmt.Errorf("获取API密钥失败: %v", err)
	}

	// 构建请求参数
	params := map[string]string{
		"user_id":   appSecret, // 使用SecretKey作为user_id
		"order_id":  order.APIOrderNumber,
		"timestamp": strconv.FormatInt(time.Now().Unix(), 10),
	}

	// 生成签名
    authToken, _, err := signature.GenerateXianzhuanxiaSignature(params, appKey, appSecret)
    if err != nil {
        if l != nil {
            l.Error("生成签名失败",
                logger.ErrorV2(err),
                logger.AnyV2("params_keys", func() []string { ks := make([]string, 0, len(params)); for k := range params { ks = append(ks, k) }; return ks }()),
            )
        }
        return 0, fmt.Errorf("生成签名失败: %v", err)
    }

	// 发送请求
    jsonData, err := json.Marshal(params)
    if err != nil {
        if l != nil {
            l.Error("序列化请求参数失败",
                logger.ErrorV2(err),
                logger.AnyV2("params_keys", func() []string { ks := make([]string, 0, len(params)); for k := range params { ks = append(ks, k) }; return ks }()),
            )
        }
        return 0, fmt.Errorf("序列化请求参数失败: %v", err)
    }

    req, err := http.NewRequestWithContext(ctx, "POST", order.PlatformURL+"/query", bytes.NewBuffer(jsonData))
    if err != nil {
        if l != nil {
            l.Error("创建HTTP请求失败",
                logger.ErrorV2(err),
                logger.StringV2("url", order.PlatformURL+"/query"),
            )
        }
        return 0, fmt.Errorf("创建HTTP请求失败: %v", err)
    }

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+authToken)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
    if err != nil {
        if l != nil {
            l.Error("发送HTTP请求失败",
                logger.ErrorV2(err),
                logger.StringV2("url", req.URL.String()),
            )
        }
        return 0, fmt.Errorf("发送HTTP请求失败: %v", err)
    }
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
    if err != nil {
        if l != nil {
            l.Error("读取响应内容失败",
                logger.ErrorV2(err),
                logger.IntV2("status_code", resp.StatusCode),
            )
        }
        return 0, fmt.Errorf("读取响应内容失败: %v", err)
    }

	var result QueryOrderStatusResult
    if err := json.Unmarshal(body, &result); err != nil {
        if l != nil {
            l.Error("解析响应内容失败",
                logger.ErrorV2(err),
                logger.StringV2("body_preview", func(b []byte) string { if len(b) > 300 { return string(b[:300]) + "..." } ; return string(b) }(body)),
            )
        }
        return 0, fmt.Errorf("解析响应内容失败: %v", err)
    }

    if result.Code != 0 {
        if l != nil {
            l.Error("查询订单状态失败",
                logger.IntV2("code", result.Code),
                logger.StringV2("message", result.Message),
            )
        }
        return 0, fmt.Errorf("查询订单状态失败: %s", result.Message)
    }

	switch result.Data.Status {
	case 2:
		return model.OrderStatusSuccess, nil
	case 3, 5:
		return model.OrderStatusFailed, nil
	case 1, 4:
		return model.OrderStatusRecharging, nil
	default:
		return model.OrderStatusRecharging, nil
	}
}

// ParseCallbackData 解析回调数据
func (p *XianzhuanxiaPlatform) ParseCallbackData(data []byte) (*model.CallbackData, error) {
	var callback struct {
		OrderID   string  `json:"order_id"`
		Status    int     `json:"status"`
		Message   string  `json:"message"`
		Amount    float64 `json:"amount"`
		Sign      string  `json:"sign"`
		Timestamp string  `json:"timestamp"`
	}

	if err := json.Unmarshal(data, &callback); err != nil {
		return nil, fmt.Errorf("解析回调数据失败: %v", err)
	}

	// 验证签名
	params := map[string]string{
		"order_id":  callback.OrderID,
		"status":    strconv.Itoa(callback.Status),
		"amount":    strconv.FormatFloat(callback.Amount, 'f', 2, 64),
		"timestamp": callback.Timestamp,
	}

	authToken, _, err := signature.GenerateXianzhuanxiaSignature(params, "", "") // 这里需要从订单中获取
	if err != nil {
		return nil, fmt.Errorf("生成签名失败: %v", err)
	}

	if authToken != callback.Sign {
		return nil, fmt.Errorf("签名验证失败")
	}

	// 转换状态
	var status string
	switch callback.Status {
	case 1:
		status = "success"
	case 2:
		status = "failed"
	default:
		status = "processing"
	}

	return &model.CallbackData{
		OrderID:       callback.OrderID,
		Status:        status,
		Message:       callback.Message,
		Amount:        strconv.FormatFloat(callback.Amount, 'f', 2, 64),
		Sign:          callback.Sign,
		Timestamp:     callback.Timestamp,
		TransactionID: "xianzhuanxia_" + callback.OrderID,
	}, nil
}

// QueryBalance 查询账户余额
func (p *XianzhuanxiaPlatform) QueryBalance(ctx context.Context, accountID int64) (float64, error) {
    l := logger.WithContextCategory(ctx, "recharge")
    if l != nil {
        l.Info("开始查询闲赚侠账户余额",
            logger.Int64V2("account_id", accountID),
        )
    }

	// 获取API密钥和密钥
	appKey, appSecret, accountName, err := p.getAPIKeyAndSecret(ctx, uint(accountID))
	if err != nil {
		return 0, fmt.Errorf("获取API密钥失败: %v", err)
	}

	// 获取平台API信息
	api, err := p.platformRepo.GetPlatformByCode(ctx, "xianzhuanxia")
	if err != nil {
		return 0, fmt.Errorf("获取平台API信息失败: %v", err)
	}

	// 构建请求参数
	params := map[string]string{
		"user_id":   appSecret, // 使用SecretKey作为user_id
		"timestamp": strconv.FormatInt(time.Now().Unix(), 10),
	}

	// 生成签名
    authToken, _, err := signature.GenerateXianzhuanxiaSignature(params, appKey, accountName)
    if err != nil {
        if l != nil {
            l.Error("生成签名失败",
                logger.ErrorV2(err),
                logger.AnyV2("params_keys", func() []string { ks := make([]string, 0, len(params)); for k := range params { ks = append(ks, k) }; return ks }()),
            )
        }
        return 0, fmt.Errorf("生成签名失败: %v", err)
    }

	// 发送请求
    jsonData, err := json.Marshal(params)
    if err != nil {
        if l != nil {
            l.Error("序列化请求参数失败",
                logger.ErrorV2(err),
                logger.AnyV2("params_keys", func() []string { ks := make([]string, 0, len(params)); for k := range params { ks = append(ks, k) }; return ks }()),
            )
        }
        return 0, fmt.Errorf("序列化请求参数失败: %v", err)
    }

    req, err := http.NewRequestWithContext(ctx, "POST", api.URL+"/query-balance", bytes.NewBuffer(jsonData))
    if err != nil {
        if l != nil {
            l.Error("创建HTTP请求失败",
                logger.ErrorV2(err),
                logger.StringV2("url", api.URL+"/query-balance"),
            )
        }
        return 0, fmt.Errorf("创建HTTP请求失败: %v", err)
    }

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Auth_Token", authToken)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
    if err != nil {
        if l != nil {
            l.Error("发送HTTP请求失败",
                logger.ErrorV2(err),
                logger.StringV2("url", req.URL.String()),
            )
        }
        return 0, fmt.Errorf("发送HTTP请求失败: %v", err)
    }
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
    if err != nil {
        if l != nil {
            l.Error("读取响应内容失败",
                logger.ErrorV2(err),
                logger.IntV2("status_code", resp.StatusCode),
            )
        }
        return 0, fmt.Errorf("读取响应内容失败: %v", err)
    }

	// 解析响应
	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Balance float64 `json:"balance"`
		} `json:"data"`
	}

    if err := json.Unmarshal(body, &result); err != nil {
        if l != nil {
            l.Error("解析响应内容失败",
                logger.ErrorV2(err),
                logger.StringV2("body_preview", func(b []byte) string { if len(b) > 300 { return string(b[:300]) + "..." } ; return string(b) }(body)),
            )
        }
        return 0, fmt.Errorf("解析响应内容失败: %v", err)
    }

    if result.Code != 0 {
        if l != nil {
            l.Error("查询余额失败",
                logger.StringV2("platform", "xianzhuanxia"),
                logger.Int64V2("account_id", accountID),
                logger.StringV2("error", result.Message),
            )
        }
        return 0, fmt.Errorf("查询余额失败: %s", result.Message)
    }

    if l != nil {
        l.Info("查询余额成功",
            logger.Int64V2("account_id", accountID),
            logger.Float64V2("balance", result.Data.Balance),
        )
    }

	return result.Data.Balance, nil
}
