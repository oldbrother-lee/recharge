package main

import (
	"crypto/rc4"
	"encoding/base64"
	"encoding/json"
	"fmt"
)

type RequestData struct {
	Action string      `json:"action"`
	Flag   string      `json:"flag"`
	Ver    string      `json:"ver"`
	Token  string      `json:"token"`
	Data   interface{} `json:"data"`
}

func rc4EncryptBase64(plaintext []byte, key string) (string, error) {
	c, err := rc4.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}
	out := make([]byte, len(plaintext))
	c.XORKeyStream(out, plaintext)
	return base64.StdEncoding.EncodeToString(out), nil
}

func printPayload(title string, payload RequestData, rc4Key string) {
	b, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("[%s] JSON序列化失败: %v\n", title, err)
		return
	}
	enc, err := rc4EncryptBase64(b, rc4Key)
	if err != nil {
		fmt.Printf("[%s] 加密失败: %v\n", title, err)
		return
	}
	fmt.Printf("\n[%s] 明文JSON:\n%s\n\n", title, string(b))
	fmt.Printf("[%s] 加密后报文(Base64(RC4(JSON))):\n%s\n\n", title, enc)
}

func PrintDzPayloadSamples() {
	// 固定参数（按你的示例）
	baseURL := "http://api.hdb666.com/api/phrecharge?devKey=10028"
	rc4Key := "IT8MYBDY"             // RC4密钥（示例用你的 app_key）
	username := "1863563749302"      // 登录账号
	password := "JpE9F7heOl1fP%S0"   // 登录密码
	loginToken := "TOKEN_FROM_LOGIN" // 登录返回的 token（打印拉单/通知用占位）

	fmt.Println("网关地址:", baseURL)

	// 1) 登录报文
	loginReq := RequestData{
		Action: "login",
		Flag:   "invite_dxfs",
		Ver:    "1.0.0.0",
		Token:  "",
		Data: map[string]string{
			"username": username,
			"password": password,
		},
	}
	printPayload("登录请求", loginReq, rc4Key)

	// 2) 拉单报文（为已配置的四个变体打印）
	pullAction := "orderinfo" // 按平台文档替换为实际动作名，如 orderlist/pull_order
	variants := []struct {
		isp  int
		face float64
	}{{1, 30.00}, {1, 50.00}, {2, 50.00}, {3, 50.00}}
	for _, v := range variants {
		pullReq := RequestData{
			Action: pullAction,
			Flag:   "invite_dxfs",
			Ver:    "1.0.0.0",
			Token:  loginToken,
			Data: map[string]interface{}{
				"isp":          v.isp,  // 1=移动，2=电信，3=联通
				"face_value":   v.face, // 面值
				"cursor_token": "",     // 首次为空；后续传增量游标
				"limit":        50,     // 拉取上限
			},
		}
		title := fmt.Sprintf("拉单请求(isp=%d face=%.2f)", v.isp, v.face)
		printPayload(title, pullReq, rc4Key)
	}

	// 3) 首次通知报文（示例：确认接单）
	notifyAction := "notify_accept" // 按平台文档替换为实际通知动作名
	notifyReq := RequestData{
		Action: notifyAction,
		Flag:   "invite_dxfs",
		Ver:    "1.0.0.0",
		Token:  loginToken,
		Data: map[string]interface{}{
			"order_id": "OUT_ORDER_ID_FROM_PULL",
			"status":   "accepted",
			"message":  "已接单，进入待充值",
		},
	}
	printPayload("首次通知请求", notifyReq, rc4Key)
}
