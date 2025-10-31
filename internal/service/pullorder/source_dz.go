package pullorder

import (
	"context"
	"crypto/rc4"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"os"
	"sync"
	"strconv"
	"recharge-go/pkg/redis"

	"recharge-go/internal/model"
	"recharge-go/internal/repository"
	"recharge-go/internal/utils"
	"recharge-go/pkg/logger"
)

// DzPullPlatform 得众拉单平台实现
 type DzPullPlatform struct {
	repo         *repository.PullSourceRepositoryImpl
	orderService repository.OrderRepository // 仅占位，实际应为 service.OrderService；此处骨架不调用
	// token 缓存：按 sourceID 复用
	tokenCache map[int64]string
	tokenMu    sync.RWMutex
}

func NewDzPullPlatform(repo *repository.PullSourceRepositoryImpl) *DzPullPlatform {
	return &DzPullPlatform{repo: repo}
}

// 缓存访问辅助方法
func (p *DzPullPlatform) getToken(sourceID int64) string {
	p.tokenMu.RLock()
	defer p.tokenMu.RUnlock()
	if p.tokenCache == nil { return "" }
	return p.tokenCache[sourceID]
}

func (p *DzPullPlatform) setToken(sourceID int64, token string) {
	p.tokenMu.Lock()
	defer p.tokenMu.Unlock()
	if p.tokenCache == nil { p.tokenCache = make(map[int64]string) }
	p.tokenCache[sourceID] = token
}

func (p *DzPullPlatform) Code() string { return "dz" }
func (p *DzPullPlatform) Name() string { return "得众" }

// RC4 加密为 Base64
func rc4EncryptBase64(plaintext []byte, key string) (string, error) {
	c, err := rc4.NewCipher([]byte(key))
	if err != nil { return "", err }
	out := make([]byte, len(plaintext))
	c.XORKeyStream(out, plaintext)
	return base64.StdEncoding.EncodeToString(out), nil
}

// RC4 解密 Base64
func rc4DecryptBase64(b64 string, key string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil { return "", err }
	c, err := rc4.NewCipher([]byte(key))
	if err != nil { return "", err }
	out := make([]byte, len(data))
	c.XORKeyStream(out, data)
	return string(out), nil
}

// Pull 按变体ID拉取订单（统一：Redis共享token，必要时登录并写回Redis）
func (p *DzPullPlatform) Pull(ctx context.Context, variantID int64) ([]ExternalOrder, error) {
	logger.InfoV2("[DZ] ===== 开始变体拉单 =====", logger.Int64V2("variant_id", variantID))

	// 1) 读取变体与源配置
	variant, err := p.repo.GetVariantByID(ctx, variantID)
	if err != nil {
		logger.ErrorLogV2("[DZ] 读取变体失败", logger.ErrorV2(err), logger.Int64V2("variant_id", variantID))
		return nil, fmt.Errorf("读取变体失败: %w", err)
	}
	if variant == nil {
		logger.ErrorLogV2("[DZ] 变体不存在", logger.Int64V2("variant_id", variantID))
		return nil, nil
	}
	
	fmt.Printf("[DEBUG] 读取到变体: ID=%d, SourceID=%d, ISP=%d, FaceValue=%.2f\n", variant.ID, variant.SourceID, variant.ISP, variant.FaceValue)
	
	source, err := p.repo.GetSourceByID(ctx, variant.SourceID)
	if err != nil { 
		logger.ErrorLogV2("[DZ] 读取源失败", logger.ErrorV2(err), logger.Int64V2("source_id", variant.SourceID))
		return nil, fmt.Errorf("读取源失败: %w", err) 
	}
	if source == nil {
		logger.ErrorLogV2("[DZ] 拉单源不存在", logger.Int64V2("source_id", variant.SourceID))
		return nil, nil
	}

	fmt.Printf("[DEBUG] 读取到源: ID=%d, Code=%s, BaseURL=%s, AppKey=%s, AccountName=%s, AccountPassword=%s\n", 
		source.ID, source.Code, source.BaseURL, maskSecret(source.AppKey), source.AccountName, maskSecret(source.AccountPassword))

	baseURL := source.BaseURL
	rc4Key := source.AppKey
	username := source.AccountName
	if baseURL == "" || rc4Key == "" || username == "" {
		errMsg := fmt.Sprintf("源配置不完整: base_url=%s, app_key=%s, account_name=%s", baseURL, maskSecret(rc4Key), username)
		logger.ErrorLogV2("[DZ] " + errMsg)
		return nil, fmt.Errorf("源配置不完整: base_url/app_key/account_name 不能为空")
	}

	logger.InfoV2("[DZ] 变体与源信息",
		logger.Int64V2("source_id", source.ID),
		logger.StringV2("source_code", source.Code),
		logger.StringV2("source_name", source.Name),
		logger.StringV2("base_url", baseURL),
		logger.StringV2("account", username),
		logger.StringV2("app_key_mask", maskSecret(rc4Key)),
		logger.IntV2("isp", variant.ISP),
		logger.Float64V2("face_value", variant.FaceValue),
		logger.StringV2("cursor", variant.Cursor),
	)

	// 在 Pull 内共享的请求结构类型
	type RequestData struct {
		Action string      `json:"action"`
		Flag   string      `json:"flag"`
		Ver    string      `json:"ver"`
		Token  string      `json:"token"`
		Data   interface{} `json:"data"`
	}

	// 优先复用Redis缓存 token；若不存在则内存缓存；最后登录一次
	var token string
	redisKey := fmt.Sprintf("dz:token:%d:%s", source.ID, username)
	if rc := redis.GetClient(); rc != nil {
		if v, err := rc.Get(ctx, redisKey).Result(); err == nil && strings.TrimSpace(v) != "" {
			token = v
			logger.InfoV2("[DZ] 复用Redis缓存token", logger.StringV2("token_mask", maskSecret(v)))
		}
	}
	if strings.TrimSpace(token) == "" {
		// 次选：内存缓存
		token = strings.TrimSpace(p.getToken(source.ID))
		if token != "" {
			logger.InfoV2("[DZ] 复用内存缓存token", logger.StringV2("token_mask", maskSecret(token)))
		}
	}
	if strings.TrimSpace(token) == "" {
		// 登录（仅初次）
		type LoginData struct { Username string `json:"username"`; Password string `json:"password"` }
		type RequestData struct {
			Action string      `json:"action"`
			Flag   string      `json:"flag"`
			Ver    string      `json:"ver"`
			Token  string      `json:"token"`
			Data   interface{} `json:"data"`
		}

		// 密码优先从数据库字段读取，其次环境变量 RECHARGE_DZ_PASSWORD
		password := strings.TrimSpace(source.AccountPassword)
		if password == "" {
			envPwd := strings.TrimSpace(os.Getenv("RECHARGE_DZ_PASSWORD"))
			if envPwd != "" {
				password = envPwd
				logger.InfoV2("[DZ] 使用环境变量提供的密码", logger.StringV2("source", "RECHARGE_DZ_PASSWORD"))
			}
		}
		if password == "" {
			logger.ErrorLogV2("[DZ] 未配置平台登录密码", logger.StringV2("hint", "可在 pull_sources.account_password 入库或设置 RECHARGE_DZ_PASSWORD"))
			return nil, fmt.Errorf("缺少登录密码: 请在数据库 pull_sources.account_password 或环境变量 RECHARGE_DZ_PASSWORD 配置")
		}
		// 构造登录请求并加密
		loginReq := RequestData{ Action: "login", Flag: "invite_dxfs", Ver: "1.0.0.0", Token: "", Data: LoginData{Username: username, Password: password} }
		loginJSON, _ := json.Marshal(loginReq)
		loginEnc, err := rc4EncryptBase64(loginJSON, rc4Key)
		if err != nil { return nil, fmt.Errorf("登录报文加密失败: %w", err) }
		logger.InfoV2("[DZ] 登录请求预览", logger.AnyV2("payload", loginReq))
		loginRespDec, err := p.postAndDecrypt(baseURL, loginEnc, rc4Key)
		if err != nil { return nil, fmt.Errorf("登录HTTP失败: %w", err) }
		logger.InfoV2("[DZ] 登录响应解密", logger.StringV2("json", loginRespDec))
		var loginResp map[string]any
		if err := json.Unmarshal([]byte(loginRespDec), &loginResp); err != nil {
			logger.ErrorLogV2("[DZ] 登录响应JSON解析失败", logger.ErrorV2(err))
			return nil, fmt.Errorf("登录响应解析失败: %w", err)
		}
		token = getString(loginResp, "data")
		if token == "" { token = getString(loginResp, "token") }
		if token == "" { return nil, fmt.Errorf("登录未返回token") }
		logger.InfoV2("[DZ] 获取到登录token", logger.StringV2("token", maskSecret(token)))
		p.setToken(source.ID, token)
		// 写入 Redis（无过期）
		if rc := redis.GetClient(); rc != nil {
			_ = rc.Set(ctx, redisKey, token, 0).Err()
			logger.InfoV2("[DZ] token已写入Redis(无过期)", logger.StringV2("key", redisKey))
		}
	} else {
		logger.InfoV2("[DZ] 复用缓存token", logger.StringV2("token", maskSecret(token)))
	}

	// 3) 拉单（根据配置的动作名构造payload）
	pullAction := source.PullAction
	if strings.TrimSpace(pullAction) == "" { pullAction = "get" }
	logger.InfoV2("[DZ] 使用拉单动作", logger.StringV2("action", pullAction))

	var pullReq RequestData
	if pullAction == "get" {
		// 示例中的获取订单结构
		type GetData struct {
			Amount    string `json:"amount"`
			MaxAmount string `json:"max_amount"`
			Operator  int    `json:"operator"`
			Discount  int    `json:"discount"`
			Prov      string `json:"prov"`
		}
		amountStr := fmt.Sprintf("%g", variant.FaceValue)
		pullReq = RequestData{
			Action: pullAction,
			Flag:   "invite_dxfs",
			Ver:    "1.0.0.0",
			Token:  token,
			Data: GetData{
				Amount:    amountStr,
				MaxAmount: amountStr,
				Operator:  variant.ISP,
				Discount:  0,
				Prov:      "",
			},
		}
	} else {
		// 兼容此前实现（如 orderinfo/orderlist）
		type PullData struct {
			ISP         int     `json:"isp"`
			FaceValue   float64 `json:"face_value"`
			CursorToken string  `json:"cursor_token"`
			Limit       int     `json:"limit"`
		}
		pullReq = RequestData{
			Action: pullAction,
			Flag:   "invite_dxfs",
			Ver:    "1.0.0.0",
			Token:  token,
			Data:   PullData{ISP: variant.ISP, FaceValue: variant.FaceValue, CursorToken: variant.Cursor, Limit: 50},
		}
	}

	pullJSON, _ := json.Marshal(pullReq)
	pullEnc, err := rc4EncryptBase64(pullJSON, rc4Key)
	if err != nil { return nil, fmt.Errorf("拉单报文加密失败: %w", err) }
	logger.InfoV2("[DZ] 拉单请求参数预览", logger.AnyV2("payload", pullReq))

	pullRespDec, err := p.postAndDecrypt(baseURL, pullEnc, rc4Key)
	if err != nil { return nil, fmt.Errorf("拉单HTTP失败: %w", err) }
	logger.InfoV2("[DZ] 拉单响应解密", 
		logger.StringV2("json", pullRespDec),
		logger.Int64V2("source_id", source.ID),
		logger.Int64V2("variant_id", variant.ID),
		logger.IntV2("isp", variant.ISP),
		logger.Float64V2("face_value", variant.FaceValue),
		logger.StringV2("source_name", source.Name))

	// 解析拉单响应为外部订单
	var resp map[string]any
	if err := json.Unmarshal([]byte(pullRespDec), &resp); err != nil {
		logger.ErrorLogV2("[DZ] 拉单响应JSON解析失败", logger.ErrorV2(err))
		return nil, fmt.Errorf("拉单响应解析失败: %w", err)
	}
	if v, ok := resp["ret"].(float64); !ok || int(v) != 0 {
		msg := ""
		if s, ok := resp["msg"].(string); ok { msg = s }
		logger.InfoV2("[DZ] 拉单返回非成功", 
			logger.StringV2("msg", msg),
			logger.Int64V2("source_id", source.ID),
			logger.Int64V2("variant_id", variant.ID),
			logger.IntV2("isp", variant.ISP),
			logger.Float64V2("face_value", variant.FaceValue),
			logger.StringV2("source_name", source.Name))
		// 若疑似 token 失效，尝试重登并重试一次
		if strings.Contains(msg, "token") || strings.Contains(msg, "登录") || strings.Contains(msg, "未登录") || strings.Contains(msg, "失效") {
			logger.InfoV2("[DZ] 检测到可能的token失效，尝试重登并重试一次")
			p.setToken(source.ID, "")
			// 重登
			type LoginData struct { Username string `json:"username"`; Password string `json:"password"` }
			password := strings.TrimSpace(source.AccountPassword)
			if password == "" {
				if envPwd := strings.TrimSpace(os.Getenv("RECHARGE_DZ_PASSWORD")); envPwd != "" { password = envPwd }
			}
			loginReq := RequestData{ Action: "login", Flag: "invite_dxfs", Ver: "1.0.0.0", Token: "", Data: LoginData{Username: username, Password: password} }
			loginJSON, _ := json.Marshal(loginReq)
			loginEnc, err := rc4EncryptBase64(loginJSON, rc4Key)
			if err == nil {
				loginRespDec, err := p.postAndDecrypt(baseURL, loginEnc, rc4Key)
				if err == nil {
					var loginResp map[string]any
					if json.Unmarshal([]byte(loginRespDec), &loginResp) == nil {
						token = getString(loginResp, "data")
						if token == "" { token = getString(loginResp, "token") }
						if token == "" { return nil, fmt.Errorf("登录未返回token") }
						logger.InfoV2("[DZ] 获取到登录token", logger.StringV2("token", maskSecret(token)))
						p.setToken(source.ID, token)
						// 写入 Redis（无过期）
						if rc := redis.GetClient(); rc != nil {
							_ = rc.Set(ctx, redisKey, token, 0).Err()
							logger.InfoV2("[DZ] token已写入Redis(无过期)", logger.StringV2("key", redisKey))
						}
						// 重新构造拉单请求并重试
						if pullAction == "get" {
							type GetData struct { Amount string `json:"amount"`; MaxAmount string `json:"max_amount"`; Operator int `json:"operator"`; Discount int `json:"discount"`; Prov string `json:"prov"` }
							amountStr := fmt.Sprintf("%g", variant.FaceValue)
							pullReq = RequestData{ Action: pullAction, Flag: "invite_dxfs", Ver: "1.0.0.0", Token: token, Data: GetData{ Amount: amountStr, MaxAmount: amountStr, Operator: variant.ISP, Discount: 0, Prov: "" } }
						} else {
							type PullData struct { ISP int `json:"isp"`; FaceValue float64 `json:"face_value"`; CursorToken string `json:"cursor_token"`; Limit int `json:"limit"` }
							pullReq = RequestData{ Action: pullAction, Flag: "invite_dxfs", Ver: "1.0.0.0", Token: token, Data: PullData{ ISP: variant.ISP, FaceValue: variant.FaceValue, CursorToken: variant.Cursor, Limit: 50 } }
						}
						pullJSON2, _ := json.Marshal(pullReq)
						pullEnc2, err := rc4EncryptBase64(pullJSON2, rc4Key)
						if err == nil {
							pullRespDec2, err := p.postAndDecrypt(baseURL, pullEnc2, rc4Key)
							if err == nil {
								var resp2 map[string]any
								if json.Unmarshal([]byte(pullRespDec2), &resp2) == nil {
									if v2, ok := resp2["ret"].(float64); ok && int(v2) == 0 {
										resp = resp2
										logger.InfoV2("[DZ] 重登后拉单成功")
									} else {
										logger.InfoV2("[DZ] 重登后拉单仍失败")
									}
								}
							}
						}
					} else {
						return nil, fmt.Errorf("登录响应解析失败")
					}
				} else {
					return nil, fmt.Errorf("登录响应解析失败")
				}
			}
		}
		// 若未能成功重试，则返回空
		if v2, ok := resp["ret"].(float64); !ok || int(v2) != 0 { return []ExternalOrder{}, nil }
	}
	data, ok := resp["data"].(map[string]any)
	if !ok || data == nil {
		return []ExternalOrder{}, nil
	}
	// 提取字段并映射
	amount := 0.0
	if av, ok := data["amount"].(float64); ok { amount = av }
	operatorID := 0
	if ov, ok := data["operator_id"].(float64); ok { operatorID = int(ov) }
	mobile := ""
	if mv, ok := data["mobile"].(string); ok { mobile = mv }
	discount := 0.0
	if dv, ok := data["discount"].(string); ok {
		if f, err := strconv.ParseFloat(dv, 64); err == nil { discount = f }
	}
	province := ""
	if pv, ok := data["prov"].(string); ok { province = pv }
	// id 可能为数字，转换为不带科学计数法的字符串
	orderID := ""
	if idv, ok := data["id"].(float64); ok {
		orderID = fmt.Sprintf("%.0f", idv)
	}
	if orderID == "" { // 兜底：直接格式化
		orderID = fmt.Sprintf("%v", data["id"])
	}

	ext := ExternalOrder{
		ID:           orderID,
		Mobile:       mobile,
		OperatorID:   operatorID,
		Amount:       amount,
		Discount:     discount,
		ProvinceName: province,
	}
	logger.InfoV2("[DZ] 拉单解析到外部订单", 
		logger.StringV2("out_trade_num", ext.ID), 
		logger.Float64V2("amount", ext.Amount), 
		logger.IntV2("operator_id", ext.OperatorID),
		logger.Int64V2("source_id", source.ID),
		logger.Int64V2("variant_id", variant.ID),
		logger.IntV2("isp", variant.ISP),
		logger.Float64V2("face_value", variant.FaceValue),
		logger.StringV2("source_name", source.Name))
	return []ExternalOrder{ext}, nil
}

func (p *DzPullPlatform) postAndDecrypt(url string, enc string, rc4Key string) (string, error) {
	// 简化实现：发送请求并解密响应
	resp, err := http.Post(url, "application/x-www-form-urlencoded", strings.NewReader(enc))
	if err != nil { return "", fmt.Errorf("发送请求失败: %w", err) }
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil { return "", fmt.Errorf("读取响应失败: %w", err) }

	// 响应可能是JSON字符串或直接Base64
	var base64Data string
	if len(body) > 0 && body[0] == '"' && body[len(body)-1] == '"' {
		if err := json.Unmarshal(body, &base64Data); err != nil { return "", fmt.Errorf("JSON解析失败: %w", err) }
	} else {
		base64Data = string(body)
	}

	// 解密
	data, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil { return "", fmt.Errorf("Base64解码失败: %w", err) }
	c, err := rc4.NewCipher([]byte(rc4Key))
	if err != nil { return "", fmt.Errorf("RC4初始化失败: %w", err) }
	out := make([]byte, len(data))
	c.XORKeyStream(out, data)
	return string(out), nil
}

// 辅助函数：取最小值
func min(a, b int) int {
	if a < b { return a }
	return b
}

// MapToOrder 将得众订单映射为内部订单模型
func (p *DzPullPlatform) MapToOrder(ctx context.Context, ext ExternalOrder, productID int64, customerID int64) (*model.Order, error) {
	isp := utils.DzOperatorIDToCode(ext.OperatorID)
	if isp == 0 {
		return nil, fmt.Errorf("未知运营商: operator_id=%d", ext.OperatorID)
	}

	order := &model.Order{
		CustomerID:       customerID,
		Mobile:           ext.Mobile,
		ProductID:        productID,
		Denom:            ext.Amount,
		TotalPrice:       0,
		Price:            0,
		OfficialPayment:  0,
		UserQuotePayment: ext.Discount,
		UserPayment:      0,
		IsPay:            1,
		PayTime:          ptrTime(time.Now()),
		Remark:           "得众拉单",
		ISP:              isp,
		AccountLocation:  ext.ProvinceName,
		OutTradeNum:      ext.ID,
		Client:           3,
		PlatformName:     p.Name(),
		PlatformCode:     p.Code(),
	}
	return order, nil
}

func ptrTime(t time.Time) *time.Time { return &t }

// 辅助：掩码密钥
func maskSecret(s string) string {
	if len(s) <= 6 { return "***" }
	return s[:6] + "***"
}

// 辅助：安全获取字符串字段
func getString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok { return s }
	}
	return ""
}