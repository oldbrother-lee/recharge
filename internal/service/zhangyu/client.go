package zhangyu

import (
	"bytes"
	"context"
	"crypto/rc4"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

    "recharge-go/internal/model"
    logger "recharge-go/pkg/log"
    "recharge-go/pkg/redis"
)

// Client 章鱼平台客户端
// 说明：不依赖外部可变结构，参数由调用方传入；token 缓存到 Redis
type Client struct {
	baseURL string
}

func NewClient(baseURL string) *Client {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "https://api.zy128.cn"
	}
	return &Client{baseURL: strings.TrimRight(baseURL, "/")}
}

// 章鱼平台响应RC4密钥（示例约定为平台固定ID字符串）
const zyResponseRC4Key = "QYkCe9Sv"

// ---- RC4 加/解密 ----
func rc4EncryptBase64(data []byte, key string) (string, error) {
	cipher, err := rc4.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}
	dst := make([]byte, len(data))
	cipher.XORKeyStream(dst, data)
	return base64.StdEncoding.EncodeToString(dst), nil
}

func rc4DecryptBase64ToBytes(enc string, key string) ([]byte, error) {
	b, err := base64.StdEncoding.DecodeString(enc)
	if err != nil {
		return nil, err
	}
	cipher, err := rc4.NewCipher([]byte(key))
	if err != nil {
		return nil, err
	}
	dst := make([]byte, len(b))
	cipher.XORKeyStream(dst, b)
	return dst, nil
}

// resolveRC4Key 解析RC4密钥
// 优先使用账号的 AppSecret（通常填写为 rc4Key）；
// 若未配置，则可从环境变量 ZHANGYU_RC4_KEY 读取；
// 最后兜底为平台ID字符串（部分环境按平台ID作为密钥）。
func resolveRC4Key(acc *model.PlatformAccount) string {
	if acc == nil {
		return ""
	}
	if v := strings.TrimSpace(acc.AppSecret); v != "" {
		return v
	}
	if v := strings.TrimSpace(os.Getenv("ZHANGYU_RC4_KEY")); v != "" {
		return v
	}
	if acc.PlatformID > 0 {
		return strconv.FormatInt(acc.PlatformID, 10)
	}
	return ""
}

// Login 登录并返回 token，同时写入 Redis 缓存（无过期或由上层控制TTL）
func (c *Client) Login(ctx context.Context, acc *model.PlatformAccount) (string, error) {
	rc4Key := resolveRC4Key(acc)
	devID := strings.TrimSpace(acc.AppKey)
	username := strings.TrimSpace(acc.AccountName)
	password := strings.TrimSpace(acc.AccountPassword)

	// 构造登录数据
	payload := struct {
		DevID string `json:"dev_id"`
		Body  string `json:"body"`
	}{}

	loginData := struct {
		Data struct {
			Username string `json:"username"`
			Password string `json:"password"`
		} `json:"data"`
	}{}
	loginData.Data.Username = username
	loginData.Data.Password = password
	raw, _ := json.Marshal(loginData)
	enc, err := rc4EncryptBase64(raw, rc4Key)
	if err != nil {
		return "", fmt.Errorf("rc4 encrypt failed: %v", err)
	}
	payload.DevID = devID
	payload.Body = enc

	reqBytes, _ := json.Marshal(payload)
	url := c.baseURL + "/api/openapi/login"
	resp, err := http.Post(url, "application/json; charset=utf-8", bytes.NewBuffer(reqBytes))
	if err != nil {
		return "", fmt.Errorf("http request failed: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	// 章鱼响应整体仍需 RC4 解密
	// 按示例约定，响应整体使用固定RC4密钥解密
	decBytes, err := rc4DecryptBase64ToBytes(string(body), zyResponseRC4Key)
	if err != nil {
		return "", fmt.Errorf("decrypt response failed: %v", err)
	}
	var loginResp struct {
		Data string `json:"data"`
		Msg  string `json:"msg"`
		Ret  int    `json:"ret"`
	}
	if err := json.Unmarshal(decBytes, &loginResp); err != nil {
		return "", fmt.Errorf("unmarshal login response failed: %v", err)
	}
	if loginResp.Ret != 0 {
		return "", fmt.Errorf("login failed: %s", loginResp.Msg)
	}
	token := strings.TrimSpace(loginResp.Data)
	if token == "" {
		return "", fmt.Errorf("empty token")
	}

	// 写入 Redis
	if rc := redis.GetClient(); rc != nil {
		key := fmt.Sprintf("zhangyu:token:%d:%s", acc.ID, username)
		_ = rc.Set(ctx, key, token, 0).Err()
		logger.WithContextCategory(ctx, "zhangyu").Info("写入Redis token",
			logger.StringV2("key", key))
	}
	return token, nil
}

// GetOrder 拉取订单（支持面值范围与省份）
func (c *Client) GetOrder(ctx context.Context, acc *model.PlatformAccount, token, flag, amount, maxAmount, prov string) (*ExternalOrder, error) {
	rc4Key := resolveRC4Key(acc)
	devID := strings.TrimSpace(acc.AppKey)

	// 构造请求
	payload := struct {
		DevID string `json:"dev_id"`
		Body  string `json:"body"`
	}{}

	// 内层JSON严格按示例结构：{"token":"...","data":{"amount":"...","max_amount":"...","prov":"..."}}
	inner := map[string]interface{}{
		"token": token,
		"flag":  flag,
		"data": map[string]string{
			"amount":     amount,
			"max_amount": maxAmount,
			"prov":       prov,
		},
	}
	raw, _ := json.Marshal(inner)
	enc, err := rc4EncryptBase64(raw, rc4Key)
	if err != nil {
		return nil, fmt.Errorf("rc4 encrypt failed: %v", err)
	}
	payload.DevID = devID
	payload.Body = enc

	// 精简：仅打印加密前的真实请求参数（内层JSON），与示例保持一致
	logger.WithContextCategory(ctx, "zhangyu").Info("拉单请求参数",
		logger.StringV2("url", c.baseURL+"/api/openapi/getOrder"),
		logger.StringV2("request_json", string(raw)),
	)

	reqBytes, _ := json.Marshal(payload)
	// 不再打印“发送拉单请求payload”日志，避免重复
	url := c.baseURL + "/api/openapi/getOrder"
	resp, err := http.Post(url, "application/json; charset=utf-8", bytes.NewBuffer(reqBytes))
	if err != nil {
		return nil, fmt.Errorf("http request failed: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	// 按示例约定，响应整体使用固定RC4密钥解密
	// 响应加密/解密内容预览日志
	rawRespPreview := string(body)
	if len(rawRespPreview) > 200 {
		rawRespPreview = rawRespPreview[:200]
	}
	decBytes, err := rc4DecryptBase64ToBytes(string(body), zyResponseRC4Key)
	if err != nil {
		return nil, fmt.Errorf("decrypt response failed: %v", err)
	}
	decRespPreview := string(decBytes)
	if len(decRespPreview) > 300 {
		decRespPreview = decRespPreview[:300]
	}
	logger.WithContextCategory(ctx, "zhangyu").Info("拉单响应解密预览",
		logger.IntV2("status_code", resp.StatusCode),
		logger.IntV2("encrypted_len", len(body)),
		logger.StringV2("encrypted_preview", rawRespPreview),
		logger.IntV2("decrypted_len", len(decBytes)),
		logger.StringV2("decrypted_preview", decRespPreview),
	)
	var orderResp struct {
		Data struct {
			ID         int    `json:"id"`
			Amount     int    `json:"amount"`
			Mobile     string `json:"mobile"`
			Operator   string `json:"operator"`
			OperatorID int    `json:"operator_id"`
			Prov       string `json:"prov"`
			Timeout    int64  `json:"timeout"`
		} `json:"data"`
		Msg string `json:"msg"`
		Ret int    `json:"ret"`
	}
	if err := json.Unmarshal(decBytes, &orderResp); err != nil {
		return nil, fmt.Errorf("unmarshal order response failed: %v", err)
	}
	if orderResp.Ret != 0 {
		return nil, fmt.Errorf("pull order failed: %s", orderResp.Msg)
	}
	// 无订单场景：ret=0但msg为“暂无订单”，或data为空/ID为0
	if strings.Contains(orderResp.Msg, "暂无订单") || orderResp.Data.ID == 0 || strings.TrimSpace(orderResp.Data.Mobile) == "" {
		logger.WithContextCategory(ctx, "zhangyu").Info("拉单返回暂无订单",
			logger.StringV2("msg", orderResp.Msg),
			logger.IntV2("ret", orderResp.Ret),
		)
		return nil, fmt.Errorf("no order")
	}

	eo := &ExternalOrder{
		ID:           strconv.Itoa(orderResp.Data.ID),
		Mobile:       orderResp.Data.Mobile,
		OperatorID:   orderResp.Data.OperatorID,
		Amount:       float64(orderResp.Data.Amount),
		ProvinceName: orderResp.Data.Prov,
		ExternalCode: flag,
	}
	return eo, nil
}

// ReportOrder 上报订单处理结果（一次）
func (c *Client) ReportOrder(ctx context.Context, acc *model.PlatformAccount, token, flag string, report ReportPayload) error {
	rc4Key := resolveRC4Key(acc)
	devID := strings.TrimSpace(acc.AppKey)

	payload := struct {
		DevID string `json:"dev_id"`
		Body  string `json:"body"`
	}{}

	reportData := struct {
		Token string `json:"token"`
		Flag  string `json:"flag"`
		Data  struct {
			ID              string `json:"id"`
			Result          string `json:"result"`
			Reason          string `json:"reason"`
			PtransID        string `json:"ptransId"`
			Cookie          string `json:"cookie"`
			OrderCreateTime string `json:"orderCreateTime"`
		} `json:"data"`
	}{}
	reportData.Token = token
	reportData.Flag = flag
	reportData.Data.ID = report.ID
	reportData.Data.Result = report.Result
	reportData.Data.Reason = report.Reason
	reportData.Data.PtransID = report.PtransID
	reportData.Data.Cookie = report.Cookie
	reportData.Data.OrderCreateTime = report.OrderCreateTime

	raw, _ := json.Marshal(reportData)
	enc, err := rc4EncryptBase64(raw, rc4Key)
	if err != nil {
		return fmt.Errorf("rc4 encrypt failed: %v", err)
	}
	payload.DevID = devID
	payload.Body = enc

	reqBytes, _ := json.Marshal(payload)
	url := c.baseURL + "/api/openapi/report"
	resp, err := http.Post(url, "application/json; charset=utf-8", bytes.NewBuffer(reqBytes))
	if err != nil {
		return fmt.Errorf("http request failed: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	// 按示例约定，响应整体使用固定RC4密钥解密
	decBytes, err := rc4DecryptBase64ToBytes(string(body), zyResponseRC4Key)
	if err != nil {
		return fmt.Errorf("decrypt response failed: %v", err)
	}
	var reportResp struct {
		Msg string `json:"msg"`
		Ret int    `json:"ret"`
	}
	if err := json.Unmarshal(decBytes, &reportResp); err != nil {
		return fmt.Errorf("unmarshal report response failed: %v", err)
	}
	if reportResp.Ret != 0 {
		return fmt.Errorf("report failed: %s", reportResp.Msg)
	}
	return nil
}

// LoadToken 从Redis读取缓存token
func (c *Client) LoadToken(ctx context.Context, acc *model.PlatformAccount) (string, error) {
	username := strings.TrimSpace(acc.AccountName)
	key := fmt.Sprintf("zhangyu:token:%d:%s", acc.ID, username)
	if rc := redis.GetClient(); rc != nil {
		v, err := rc.Get(ctx, key).Result()
		if err == nil && strings.TrimSpace(v) != "" {
			return v, nil
		}
	}
	return "", fmt.Errorf("token not found in redis")
}

// ReportPayload 上报载荷
type ReportPayload struct {
	ID              string
	Result          string
	Reason          string
	PtransID        string
	Cookie          string
	OrderCreateTime string
}

// ExternalOrder 为复用 pullorder.ExternalOrder 简化定义（避免循环依赖）
// 若需要统一，可在上层转换为 pullorder.ExternalOrder
type ExternalOrder struct {
	ID           string
	Mobile       string
	OperatorID   int
	Amount       float64
	ProvinceName string
	ExternalCode string
}
