package recharge

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"recharge-go/internal/model"
	"recharge-go/internal/repository"
	"recharge-go/pkg/logger"
	"recharge-go/pkg/signature"
	"strconv"
	"time"
	"strings"

	"gorm.io/gorm"
)

// MishiPlatform 秘史平台实现
type MishiPlatform struct {
	platformRepo repository.PlatformRepository
}

// NewMishiPlatform 创建秘史平台实例
func NewMishiPlatform(db *gorm.DB) *MishiPlatform {
	return &MishiPlatform{
		platformRepo: repository.NewPlatformRepository(db),
	}
}

// GetName 获取平台名称
func (p *MishiPlatform) GetName() string {
	return "mishi"
}

// getAPIKeyAndSecret 获取API密钥和密钥
func (p *MishiPlatform) getAPIKeyAndSecret(accountID int64) (string, string, string, error) {
	// accountIDStr := strconv.FormatInt(accountID, 10)
	account, err := p.platformRepo.GetPlatformAccountByID(accountID)
	if err != nil {
		return "", "", "", fmt.Errorf("获取平台账号信息失败: %v", err)
	}
	return account.AppKey, account.AppSecret, account.AccountName, nil
}

// convertOperatorCode 转换运营商编码
func convertOperatorCode(operatorCode string) string {
	switch operatorCode {
	case "1":
		return "1" // 移动
	case "3":
		return "2" // 联通
	case "2":
		return "3" // 电信
	case "虚拟":
		return "4" // 虚商
	case "国家电网":
		return "101" // 国家电网
	case "南方电网":
		return "102" // 南方电网
	case "中石化":
		return "104" // 中石化
	case "中石油":
		return "105" // 中石油
	case "腾讯":
		return "1000" // 腾讯
	case "爱奇艺":
		return "1001" // 爱奇艺
	case "优酷":
		return "1002" // 优酷
	case "抖音":
		return "1031" // 抖音
	default:
		return "1" // 默认移动
	}
}

// SubmitOrder 提交订单
func (p *MishiPlatform) SubmitOrder(ctx context.Context, order *model.Order, api *model.PlatformAPI, apiParam *model.PlatformAPIParam) error {
    l := logger.WithContextCategory(ctx, "recharge")
    if l != nil {
        l.Info("开始提交秘史订单",
            logger.Int64V2("order_id", order.ID),
            logger.StringV2("order_number", order.OrderNumber),
            logger.StringV2("mobile", order.Mobile),
        )
    }
	// 原先打印完整结构的调试输出改为结构化且不暴露敏感信息
    if l != nil {
        l.Info("[mishi] 提交参数上下文",
            logger.StringV2("api_code", api.Code),
            logger.Int64V2("api_id", api.ID),
            logger.Int64V2("platform_id", api.PlatformID),
            logger.Int64V2("account_id", api.AccountID),
            logger.Int64V2("param_id", apiParam.ID),
            logger.StringV2("product_id", apiParam.ProductID),
        )
    }
	// 获取API密钥和密钥
	_, appSecret, accountName, err := p.getAPIKeyAndSecret(api.AccountID)
	if err != nil {
		return fmt.Errorf("meishi 获取API密钥失败!!!: %v", err)
	}

	// 构建请求参数
	szTimeStamp := time.Now().Format("2006-01-02 15:04:05")
	params := url.Values{}
	params.Add("szAgentId", accountName)                                  // 客户id
	params.Add("szOrderId", order.OrderNumber)                            // 订单号
	params.Add("szPhoneNum", order.Mobile)                                // 充值手机号
	params.Add("nMoney", strconv.FormatInt(int64(order.Denom), 10))       // 充值金额
	params.Add("nSortType", convertOperatorCode(strconv.Itoa(order.ISP))) // 运营商编码
	params.Add("nProductClass", "1")                                      // 充值产品分类
	params.Add("nProductType", "1")                                       // 充值产品类型
	params.Add("szProductId", apiParam.ProductID)
	params.Add("szTimeStamp", szTimeStamp)

	// 生成签名（日志中避免泄露密钥）
	signStr := fmt.Sprintf("szAgentId=%s&szOrderId=%s&szPhoneNum=%s&nMoney=%s&nSortType=%s&nProductClass=%s&nProductType=%s&szTimeStamp=%s&szKey=%s",
		accountName, order.OrderNumber, order.Mobile, strconv.FormatInt(int64(order.Denom), 10),
		convertOperatorCode(strconv.Itoa(order.ISP)), "1", "1", szTimeStamp, appSecret)
    redacted := strings.ReplaceAll(signStr, appSecret, "****")
    if l != nil {
        l.Info("meishi 生成签名前",
            logger.StringV2("sign_preview", redacted),
            logger.BoolV2("contains_secret", true),
        )
    }
	sign := signature.GetMD5(signStr)
	params.Add("szVerifyString", sign)

	// 添加回调地址
	params.Add("szNotifyUrl", api.CallbackURL)

    // 发送请求前日志（不打印敏感参数值）
    paramKeys := func() []string { keys := make([]string, 0, len(params)); for k := range params { if k != "szVerifyString" && k != "szPhoneNum" { keys = append(keys, k) } }; return keys }()
    if l != nil {
        l.Info("meishi 发送请求",
            logger.StringV2("url", api.URL+"/api/submitorder"),
            logger.AnyV2("param_keys", paramKeys),
        )
    }
	// 发送请求
	respStr, err := p.sendRequest(ctx, api.URL+"/api/submitorder", params)
	if err != nil {
		return fmt.Errorf("发送请求失败: %v", err)
	}
    if l != nil {
        l.Info("meishi 收到响应",
            logger.StringV2("url", api.URL+"/api/submitorder"),
            logger.StringV2("resp_preview", func(s string) string { if len(s) > 300 { return s[:300] + "..." }; return s }(respStr)),
        )
    }
	// 解析响应
	var result MishiOrderResponseSubmit
	if err := json.Unmarshal([]byte(respStr), &result); err != nil {
		return fmt.Errorf("解析响应失败: %v", err)
	}

	// 处理响应
    if result.NRtn != 0 {
        if l != nil {
            l.Error("meishi 提交订单失败",
                logger.Int64V2("n_rtn", result.NRtn),
                logger.StringV2("szRtnCode", result.SzRtnCode),
            )
        }
        return fmt.Errorf("提交订单失败: %s", result.SzRtnCode)
    }

	// 更新订单信息
	// order.APIOrderNumber = result.SzOrderId
	// order.APITradeNum = result.SzOrderId

    if l != nil {
        l.Info("提交订单成功",
            logger.Int64V2("order_id", order.ID),
            logger.StringV2("order_number", order.OrderNumber),
            logger.StringV2("api_order_id", result.SzOrderId),
        )
    }

	return nil
}

// QueryOrderStatus 查询订单状态
func (p *MishiPlatform) QueryOrderStatus(ctx context.Context, order *model.Order) (model.OrderStatus, error) {
    l := logger.WithContextCategory(ctx, "recharge")
    if l != nil {
        l.Info("开始查询秘史订单状态",
            logger.Int64V2("order_id", order.ID),
            logger.StringV2("order_number", order.OrderNumber),
            logger.StringV2("api_order_id", order.APIOrderNumber),
        )
    }

	// 获取API密钥和密钥
	_, appSecret, accountName, err := p.getAPIKeyAndSecret(order.PlatformAccountID)
	if err != nil {
		return 0, fmt.Errorf("获取API密钥失败: %v", err)
	}

	// 构建请求参数
	params := url.Values{}
	params.Add("szAgentId", accountName)
	params.Add("szOrderId", order.OrderNumber)

	// 生成签名
	signStr := fmt.Sprintf("szAgentId=%s&szOrderId=%s&szKey=%s",
		accountName, order.OrderNumber, appSecret)
	sign := signature.GetMD5(signStr)
	params.Add("szVerifyString", sign)

	// 发送请求
	respStr, err := p.sendRequest(ctx, order.PlatformURL+"/query", params)
	if err != nil {
		return 0, fmt.Errorf("查询订单状态失败: %v", err)
	}

	// 解析响应
	var result MishiOrderResponseQuery
	if err := json.Unmarshal([]byte(respStr), &result); err != nil {
		return 0, fmt.Errorf("解析响应失败: %v", err)
	}

	// 处理响应
	if result.SzRtnCode != "success" {
		return 0, fmt.Errorf("查询订单状态失败: %s", result.SzRtnMsg)
	}

	// 转换状态
	var status model.OrderStatus
	switch result.SzRtnMsg {
	case "1":
		status = model.OrderStatusProcessing
	case "2":
		status = model.OrderStatusSuccess
	case "3":
		status = model.OrderStatusFailed
	default:
		status = model.OrderStatusProcessing
	}

	return status, nil
}

// mapOrderState 返回本地订单状态码和字符串
func (p *MishiPlatform) mapOrderState(nFlag string, orderID, orderNumber string) (int, string) {
	var status int
	var statusStr string
	switch nFlag {
	case "2":
		status = int(model.OrderStatusSuccess)
		statusStr = strconv.Itoa(status)
        logger.GetCategoryLogger("recharge").Info("【秘史订单状态】充值成功",
            logger.StringV2("order_id", orderID),
            logger.StringV2("order_number", orderNumber),
        )
	case "3":
		status = int(model.OrderStatusFailed)
		statusStr = strconv.Itoa(status)
        logger.GetCategoryLogger("recharge").Info("【秘史订单状态】充值失败",
            logger.StringV2("order_id", orderID),
            logger.StringV2("order_number", orderNumber),
        )
	default:
		status = int(model.OrderStatusProcessing)
		statusStr = strconv.Itoa(status)
        logger.GetCategoryLogger("recharge").Info("【秘史订单状态】处理中",
            logger.StringV2("order_id", orderID),
            logger.StringV2("order_number", orderNumber),
            logger.StringV2("nFlag", nFlag),
        )
	}
	return status, statusStr
}

// ParseCallbackData 解析回调数据
func (p *MishiPlatform) ParseCallbackData(data []byte) (*model.CallbackData, error) {
	// 先尝试 url.ParseQuery 解析表单格式
	form, err := url.ParseQuery(string(data))
	if err == nil && len(form) > 0 {
		// 检查必要字段是否存在
		if len(form["szOrderId"]) > 0 && len(form["nFlag"]) > 0 {
			_, statusStr := p.mapOrderState(form["nFlag"][0], form["szOrderId"][0], form["szOrderId"][0])
			callbackData := &model.CallbackData{
				OrderID:       form["szOrderId"][0],
				Status:        statusStr,
				Message:       getFormValue(form, "szRtnMsg"),
				Amount:        getFormValue(form, "fSalePrice"),
				Sign:          getFormValue(form, "szVerifyString"),
				OrderNumber:   form["szOrderId"][0],
				Timestamp:     "",
				TransactionID: "mishi_" + form["szOrderId"][0], // 使用平台前缀+订单号作为TransactionID
			}
            logger.GetCategoryLogger("recharge").Info("mishi回调解析完成(form)",
                logger.AnyV2("callback_data", callbackData),
            )
			return callbackData, nil
		}
	}
	// 如果不是表单格式，尝试 json 解析
	var req struct {
		SzOrderId      string `json:"szOrderId"`
		NFlag          string `json:"nFlag"`
		SzRtnMsg       string `json:"szRtnMsg"`
		FSalePrice     string `json:"fSalePrice"`
		SzVerifyString string `json:"szVerifyString"`
	}
    if err := json.Unmarshal(data, &req); err != nil {
        logger.GetCategoryLogger("recharge").Error("mishi回调参数解析失败",
            logger.ErrorV2(err),
            logger.StringV2("raw", string(data)),
        )
        return nil, errors.New("解析回调数据失败")
    }
	_, statusStr := p.mapOrderState(req.NFlag, req.SzOrderId, req.SzOrderId)
	callbackData := &model.CallbackData{
		OrderID:       req.SzOrderId,
		Status:        statusStr,
		Message:       req.SzRtnMsg,
		Amount:        req.FSalePrice,
		Sign:          req.SzVerifyString,
		OrderNumber:   req.SzOrderId,
		Timestamp:     "",
		TransactionID: "mishi_" + req.SzOrderId, // 使用平台前缀+订单号作为TransactionID
	}
    logger.GetCategoryLogger("recharge").Info("mishi回调解析完成(json)",
        logger.AnyV2("callback_data", callbackData),
    )
	return callbackData, nil
}

// getFormValue 安全地获取表单值
func getFormValue(form url.Values, key string) string {
	if values, exists := form[key]; exists && len(values) > 0 {
		return values[0]
	}
	return ""
}

// sendRequest 发送请求
func (p *MishiPlatform) sendRequest(ctx context.Context, url string, params url.Values) (string, error) {
	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBufferString(params.Encode()))
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %v", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// 发送请求
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %v", err)
	}
    preview := func(b []byte) string { if len(b) > 300 { return string(b[:300]) + "..." }; return string(b) }(body)
    if ctx != nil {
        l := logger.WithContextCategory(ctx, "recharge")
        if l != nil {
            l.Info("mishi 响应",
                logger.StringV2("url", url),
                logger.IntV2("status_code", resp.StatusCode),
                logger.StringV2("body_preview", preview),
            )
        }
    } else {
        logger.GetCategoryLogger("recharge").Info("mishi 响应",
            logger.StringV2("url", url),
            logger.IntV2("status_code", resp.StatusCode),
            logger.StringV2("body_preview", preview),
        )
    }
    return string(body), nil

}

// QueryBalance 查询账户余额
func (p *MishiPlatform) QueryBalance(ctx context.Context, accountID int64) (float64, error) {
    l := logger.WithContextCategory(ctx, "recharge")
    if l != nil {
        l.Info("开始查询秘史账户余额",
            logger.Int64V2("account_id", accountID),
        )
    }

	// 获取API密钥和密钥
	_, appSecret, accountName, err := p.getAPIKeyAndSecret(accountID)
	if err != nil {
		return 0, fmt.Errorf("获取API密钥失败: %v", err)
	}

	// 获取平台API信息
	api, err := p.platformRepo.GetPlatformByCode(ctx, "mishi")
	if err != nil {
		return 0, fmt.Errorf("获取平台API信息失败: %v", err)
	}

	// 构建请求参数
	params := url.Values{}
	params.Add("szAgentId", accountName)

	// 生成签名
	signStr := fmt.Sprintf("szAgentId=%s&szKey=%s", accountName, appSecret)
	sign := signature.GetMD5(signStr)
	params.Add("szVerifyString", sign)

	// 发送请求
	respStr, err := p.sendRequest(ctx, api.URL+"/api/old/queryBalance", params)
	if err != nil {
		return 0, fmt.Errorf("查询余额失败: %v", err)
	}

	// 解析响应
	var result MishiResponse
	if err := json.Unmarshal([]byte(respStr), &result); err != nil {
		return 0, fmt.Errorf("解析响应失败: %v", err)
	}

	// 处理响应
	if result.SzRtnCode != "success" {
		return 0, fmt.Errorf("查询余额失败: %s", result.SzRtnCode)
	}

    if l != nil {
        l.Info("查询余额成功",
            logger.Int64V2("account_id", accountID),
            logger.Float64V2("balance", result.FBalance),
        )
    }

	return result.FBalance, nil
}

// MishiResponse 秘史平台响应
type MishiResponse struct {
	SzRtnCode string  `json:"szRtnCode"`
	SzAgentId string  `json:"szAgentId"`
	FBalance  float64 `json:"fBalance"`
	FCredit   float64 `json:"fCredit"`
	NRtn      int     `json:"nRtn"`
}

type MishiOrderResponseQuery struct {
	SzRtnCode  string  `json:"szRtnCode"`
	SzOrderId  string  `json:"szAgentId"`
	FSalePrice float64 `json:"fBalance"`
	SzRtnMsg   string  `json:"fCredit"`
}

type MishiOrderResponseSubmit struct {
	NRtn       int64   `json:"nRtn"`
	SzRtnCode  string  `json:"szRtnCode"`
	SzOrderId  string  `json:"SzOrderId"`
	FSalePrice float64 `json:"fSalePrice"`
	FNBalance  float64 `json:"fNBalance"`
}
