package platform

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"

    "recharge-go/internal/model"
    "recharge-go/internal/repository"
    "recharge-go/pkg/logger"
    "recharge-go/pkg/signature"
)

// PushStatusService 推单状态服务
type PushStatusService struct {
	AccountRepo *repository.PlatformAccountRepository
	httpClient  *http.Client
}

// NewPushStatusService 创建推单状态服务
func NewPushStatusService(accountRepo *repository.PlatformAccountRepository) *PushStatusService {
	return &PushStatusService{
		AccountRepo: accountRepo,
		httpClient:  &http.Client{},
	}
}

// GetPushStatus 获取推单状态
func (s *PushStatusService) GetPushStatus(account *model.PlatformAccount) (int, error) {
	var params map[string]interface{}
	var url string
	var sign string

	// 根据平台类型选择不同的实现
	switch account.Platform.Code {
	case "mifeng":
		params = map[string]interface{}{
			"app_key":   account.AppKey,
			"timestamp": time.Now().Unix(),
		}
		sign = signature.GenerateSign(params, account.AppSecret)
		url = fmt.Sprintf("%s/userapi/sgd/getSupplyGoodManageSwitch", account.Platform.ApiURL)
	case "kekebang":
		params = map[string]interface{}{
			"app_key":   account.AppKey,
			"timestamp": time.Now().Unix(),
		}
		sign = signature.GenerateKekebangSign(params, account.AppSecret)
		url = fmt.Sprintf("%s/openapi/suppler/v1/get-supply-switch-status", account.Platform.ApiURL)
	case "xianyinke":
		params = map[string]interface{}{
			"app_key":   account.AppKey,
			"timestamp": time.Now().Unix(),
		}
		sign = signature.GenerateXianyinkeSign(params, account.AppSecret)
		// 临时URL：参考蜜蜂，后续以实际为准
		url = fmt.Sprintf("%s/userapi/sgd/getSupplyGoodManageSwitch", account.Platform.ApiURL)
	default:
		return 0, fmt.Errorf("unsupported platform type: %s", account.Platform.Code)
	}

	params["sign"] = sign

    jsonData, err := json.Marshal(params)
    if err != nil {
        return 0, err
    }

    lg := logger.GetCategoryLogger("platform")
    previewLen := 256
    if len(jsonData) < previewLen {
        previewLen = len(jsonData)
    }
    lg.Info("获取推单状态请求",
        logger.StringV2("platform_code", account.Platform.Code),
        logger.StringV2("url", url),
        logger.IntV2("request_body_size", len(jsonData)),
        logger.StringV2("request_body_preview", string(jsonData[:previewLen])),
    )
    resp, err := s.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonData))
    if err != nil {
        lg.Error("获取推单状态请求失败",
            logger.StringV2("platform_code", account.Platform.Code),
            logger.StringV2("url", url),
            logger.ErrorV2(err),
        )
        return 0, err
    }
    defer resp.Body.Close()
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        lg.Error("获取推单状态读取响应失败",
            logger.StringV2("platform_code", account.Platform.Code),
            logger.StringV2("url", url),
            logger.ErrorV2(err),
        )
        return 0, err
    }
    respPreviewLen := 256
    if len(body) < respPreviewLen {
        respPreviewLen = len(body)
    }
    lg.Info("获取推单状态响应",
        logger.StringV2("platform_code", account.Platform.Code),
        logger.StringV2("url", url),
        logger.IntV2("status_code", resp.StatusCode),
        logger.IntV2("response_body_size", len(body)),
        logger.StringV2("response_body_preview", string(body[:respPreviewLen])),
    )

	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Status int `json:"status"`
		} `json:"data"`
	}
    if err := json.Unmarshal(body, &result); err != nil {
        lg.Error("获取推单状态响应解析失败",
            logger.StringV2("platform_code", account.Platform.Code),
            logger.StringV2("url", url),
            logger.ErrorV2(err),
        )
        return 0, err
    }
    if result.Code != 0 && result.Message != "请求成功" {
        lg.Warn("获取推单状态业务错误",
            logger.StringV2("platform_code", account.Platform.Code),
            logger.StringV2("url", url),
            logger.IntV2("code", result.Code),
            logger.StringV2("message", result.Message),
        )
        return 0, fmt.Errorf("%s error: %s", account.Platform.Code, result.Message)
    }
    lg.Info("获取推单状态成功",
        logger.StringV2("platform_code", account.Platform.Code),
        logger.IntV2("status", result.Data.Status),
    )
    return result.Data.Status, nil
}

// UpdatePushStatus 更新推单状态
func (s *PushStatusService) UpdatePushStatus(account *model.PlatformAccount, status int) error {
	var params map[string]interface{}
	var url string
	var sign string

	data := map[string]int{
		"status": status,
	}
	dataStr, err := json.Marshal(data)
	if err != nil {
		return err
	}

	// 根据平台类型选择不同的实现
	switch account.Platform.Code {
	case "mifeng":
		params = map[string]interface{}{
			"app_key":   account.AppKey,
			"data":      string(dataStr),
			"timestamp": time.Now().Unix(),
		}
		sign = signature.GenerateSign(params, account.AppSecret)
		url = fmt.Sprintf("%s/userapi/sgd/editSupplyGoodManageSwitch", account.Platform.ApiURL)
	case "kekebang":
		params = map[string]interface{}{
			"app_key":   account.AppKey,
			"data":      string(dataStr),
			"timestamp": time.Now().Unix(),
		}
    // 敏感信息不直接输出，使用长度代替
    logger.GetCategoryLogger("platform").Info("kekebang 通知参数准备",
        logger.IntV2("params_fields", len(params)),
        logger.IntV2("app_secret_length", len(account.AppSecret)),
    )
		sign = signature.GenerateKekebangNotifySign(params, account.AppSecret)
		url = fmt.Sprintf("%s/openapi/suppler/v1/edit-supply-switch-status", account.Platform.ApiURL)
	case "xianyinke":
		params = map[string]interface{}{
			"app_key":   account.AppKey,
			"data":      string(dataStr),
			"timestamp": time.Now().Unix(),
		}
		sign = signature.GenerateXianyinkeSign(params, account.AppSecret)
		// 临时URL：参考蜜蜂，后续以实际为准
		url = fmt.Sprintf("%s/userapi/sgd/editSupplyGoodManageSwitch", account.Platform.ApiURL)
	default:
		return fmt.Errorf("unsupported platform type: %s", account.Platform.Code)
	}

	params["sign"] = sign

    jsonData, err := json.Marshal(params)
    if err != nil {
        return err
    }

    lg := logger.GetCategoryLogger("platform")
    previewLen := 256
    if len(jsonData) < previewLen {
        previewLen = len(jsonData)
    }
    lg.Info("更新推单状态请求",
        logger.StringV2("platform_code", account.Platform.Code),
        logger.StringV2("url", url),
        logger.IntV2("request_body_size", len(jsonData)),
        logger.StringV2("request_body_preview", string(jsonData[:previewLen])),
    )
    resp, err := s.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonData))
    if err != nil {
        lg.Error("更新推单状态请求失败",
            logger.StringV2("platform_code", account.Platform.Code),
            logger.StringV2("url", url),
            logger.ErrorV2(err),
        )
        return err
    }
    defer resp.Body.Close()
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        lg.Error("更新推单状态读取响应失败",
            logger.StringV2("platform_code", account.Platform.Code),
            logger.StringV2("url", url),
            logger.ErrorV2(err),
        )
        return err
    }
    respPreviewLen := 256
    if len(body) < respPreviewLen {
        respPreviewLen = len(body)
    }
    lg.Info("更新推单状态响应",
        logger.StringV2("platform_code", account.Platform.Code),
        logger.StringV2("url", url),
        logger.IntV2("status_code", resp.StatusCode),
        logger.IntV2("response_body_size", len(body)),
        logger.StringV2("response_body_preview", string(body[:respPreviewLen])),
    )
	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
    if err := json.Unmarshal(body, &result); err != nil {
        lg.Error("更新推单状态响应解析失败",
            logger.StringV2("platform_code", account.Platform.Code),
            logger.StringV2("url", url),
            logger.ErrorV2(err),
        )
        return err
    }
    if result.Code != 0 && result.Message != "请求成功" {
        lg.Warn("更新推单状态业务错误",
            logger.StringV2("platform_code", account.Platform.Code),
            logger.StringV2("url", url),
            logger.IntV2("code", result.Code),
            logger.StringV2("message", result.Message),
        )
        return fmt.Errorf("%s error: %s", account.Platform.Code, result.Message)
    }
    lg.Info("更新推单状态成功",
        logger.StringV2("platform_code", account.Platform.Code),
        logger.IntV2("status", status),
    )

	// 更新本地数据库 push_status 字段
	err = s.AccountRepo.GetDB().Model(&model.PlatformAccount{}).
		Where("id = ?", account.ID).
		Update("push_status", status).Error
    if err != nil {
        lg.Error("更新推单状态本地数据库失败",
            logger.Int64V2("account_id", account.ID),
            logger.ErrorV2(err),
        )
        return err
    }

	return nil
}
