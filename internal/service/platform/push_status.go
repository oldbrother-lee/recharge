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
	default:
		return 0, fmt.Errorf("unsupported platform type: %s", account.Platform.Code)
	}

	params["sign"] = sign

	jsonData, err := json.Marshal(params)
	if err != nil {
		return 0, err
	}

	logger.Info("[GetPushStatus] 请求平台", "platform", account.Platform.Code, "url", url, "params", string(jsonData))
	resp, err := s.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		logger.Error("[GetPushStatus] 请求失败", "platform", account.Platform.Code, "error", err.Error())
		return 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("[GetPushStatus] 读取响应失败", "platform", account.Platform.Code, "error", err.Error())
		return 0, err
	}
	logger.Info("[GetPushStatus] 平台原始响应", "platform", account.Platform.Code, "body", string(body))

	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Status int `json:"status"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		logger.Error("[GetPushStatus] 响应解析失败", "platform", account.Platform.Code, "error", err.Error())
		return 0, err
	}
	if result.Code != 0 && result.Message != "请求成功" {
		logger.Error("[GetPushStatus] 平台返回错误", "platform", account.Platform.Code, "code", result.Code, "message", result.Message)
		return 0, fmt.Errorf("%s error: %s", account.Platform.Code, result.Message)
	}
	logger.Info("[GetPushStatus] 最终推单状态", "platform", account.Platform.Code, "status", result.Data.Status)
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
		logger.Info(fmt.Sprintf("kekebang Notify params: %+v", params))
		logger.Info(fmt.Sprintf("kekebang Notify appSecret: %s", account.AppSecret))
		sign = signature.GenerateKekebangNotifySign(params, account.AppSecret)
		url = fmt.Sprintf("%s/openapi/suppler/v1/edit-supply-switch-status", account.Platform.ApiURL)
	default:
		return fmt.Errorf("unsupported platform type: %s", account.Platform.Code)
	}

	params["sign"] = sign

	jsonData, err := json.Marshal(params)
	if err != nil {
		return err
	}

	logger.Info("[UpdatePushStatus] 请求平台", "platform", account.Platform.Code, "url", url, "params", string(jsonData))
	resp, err := s.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		logger.Error("[UpdatePushStatus] 请求失败", "platform", account.Platform.Code, "error", err.Error())
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("[UpdatePushStatus] 读取响应失败", "platform", account.Platform.Code, "error", err.Error())
		return err
	}
	logger.Info("[UpdatePushStatus] 平台原始响应", "platform", account.Platform.Code, "body", string(body))
	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		logger.Error("[UpdatePushStatus] 响应解析失败", "platform", account.Platform.Code, "error", err.Error())
		return err
	}
	if result.Code != 0 && result.Message != "请求成功" {
		logger.Error("[UpdatePushStatus] 平台返回错误", "platform", account.Platform.Code, "code", result.Code, "message", result.Message)
		return fmt.Errorf("%s error: %s", account.Platform.Code, result.Message)
	}
	logger.Info("[UpdatePushStatus] 最终推单状态", "platform", account.Platform.Code, "status", status)

	// 更新本地数据库 push_status 字段
	err = s.AccountRepo.GetDB().Model(&model.PlatformAccount{}).
		Where("id = ?", account.ID).
		Update("push_status", status).Error
	if err != nil {
		logger.Error("[UpdatePushStatus] 本地数据库更新失败", "account_id", account.ID, "error", err.Error())
		return err
	}

	return nil
}
