package recharge

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"recharge-go/internal/model"
	"recharge-go/internal/repository"
	"recharge-go/pkg/logger"
	"recharge-go/pkg/signature"
	"recharge-go/pkg/utils"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
	"gorm.io/gorm"
)

// ChongzhiPlatform 充值平台
type ChongzhiPlatform struct {
	platformRepo repository.PlatformRepository
	signer       *signature.ChongzhiSignature
}

// OrderResponse 解析返回的XML
type OrderResponse struct {
	XMLName            xml.Name `xml:"order"`
	OrderID            string   `xml:"orderid"`
	ProductID          string   `xml:"productid"`
	Num                string   `xml:"num"`
	OrderCash          string   `xml:"ordercash"`
	ProductName        string   `xml:"productname"`
	SpOrderID          string   `xml:"sporderid"`
	Mobile             string   `xml:"mobile"`
	MerchantSubmitTime string   `xml:"merchantsubmittime"`
	ResultNo           string   `xml:"resultno"`
	Remark1            string   `xml:"remark1"`
	FundBalance        string   `xml:"fundbalance"`
}

// NewChongzhiPlatform 创建平台实例
func NewChongzhiPlatform(db *gorm.DB) *ChongzhiPlatform {
	return &ChongzhiPlatform{
		platformRepo: repository.NewPlatformRepository(db),
		signer:       signature.NewChongzhiSignature(),
	}
}

// GetName 获取平台名称
func (p *ChongzhiPlatform) GetName() string {
	return "chongzhi"
}

// getAPIKeyAndSecret 获取API密钥和密钥
func (p *ChongzhiPlatform) getAPIKeyAndSecret(accountID int64) (string, string, error) {
	account, err := p.platformRepo.GetAccountByID(context.Background(), accountID)
	if err != nil {
		return "", "", fmt.Errorf("获取平台账号信息失败: %v", err)
	}
	return account.AccountName, account.AppKey, nil
}

// SubmitOrder 提交订单
func (p *ChongzhiPlatform) SubmitOrder(ctx context.Context, order *model.Order, api *model.PlatformAPI, apiParam *model.PlatformAPIParam) error {
	logger.Info(fmt.Sprintf("【开始提交订单】order_number: %s", order.OrderNumber))

	// 获取API密钥
	userid, key, err := p.getAPIKeyAndSecret(api.AccountID)
	if err != nil {
		return fmt.Errorf("get api key and secret failed: %v", err)
	}

	// 构造请求参数
	productid := apiParam.ProductID
	price := fmt.Sprintf("%v", order.Denom) // 转换为字符串
	num := "1"
	mobile := order.Mobile
	spordertime := time.Now().Format("20060102150405")
	sporderid := order.OrderNumber

	// 生成签名
	// 记录签名字符串用于调试
	signStr := fmt.Sprintf("userid=%s&productid=%s&price=%s&num=%s&mobile=%s&spordertime=%s&sporderid=%s&key=%s",
		userid, productid, price, num, mobile, spordertime, sporderid, key)
	logger.Info(fmt.Sprintf("签名字符串: %s", signStr))
	sign := p.signer.GenerateSign(userid, productid, price, num, mobile, spordertime, sporderid, key)
	logger.Info(fmt.Sprintf("生成签名: %s", sign))

	// 构造表单参数
	form := url.Values{}
	form.Set("userid", userid)
	form.Set("productid", productid)
	form.Set("price", price)
	form.Set("num", num)
	form.Set("mobile", mobile)
	form.Set("spordertime", spordertime)
	form.Set("sporderid", sporderid)
	form.Set("sign", sign)
	form.Set("back_url", api.CallbackURL)

	// 记录请求参数
	logger.Info(fmt.Sprintf("请求参数: userid=%s, productid=%s, price=%s, mobile=%s, spordertime=%s, sporderid=%s, key=%s, back_url=%s",
		userid, productid, price, mobile, spordertime, sporderid, key, api.CallbackURL))

	// 编码为GBK
	formStr := form.Encode()
	logger.Info(fmt.Sprintf("表单数据(UTF-8): %s", formStr))
	gbkBody, err := utils.EncodeGBK(formStr)
	if err != nil {
		logger.Error(fmt.Sprintf("GBK encode error: %v", err))
		return fmt.Errorf("GBK encode error: %v", err)
	}
	logger.Info(fmt.Sprintf("GBK编码后长度: %d bytes", len(gbkBody)))

	// 发送POST请求
	logger.Info(fmt.Sprintf("请求URL: %s", api.URL))
	req, err := http.NewRequest("POST", api.URL, bytes.NewReader(gbkBody))
	if err != nil {
		logger.Error(fmt.Sprintf("NewRequest error: %v", err))
		return fmt.Errorf("NewRequest error: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=gbk")
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; RechargeBot/1.0)")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Connection", "close")
	logger.Info(fmt.Sprintf("请求头: Content-Type=%s, User-Agent=%s",
		req.Header.Get("Content-Type"), req.Header.Get("User-Agent")))

	client := &http.Client{Timeout: time.Duration(api.Timeout) * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logger.Error(fmt.Sprintf("HTTP request error: %v", err))
		return fmt.Errorf("HTTP request error: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应body（GBK转UTF-8）
	gbkResp, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Error(fmt.Sprintf("ReadAll error: %v", err))
		return fmt.Errorf("ReadAll error: %v", err)
	}

	logger.Info(fmt.Sprintf("HTTP状态码: %d, 响应长度: %d", resp.StatusCode, len(gbkResp)))

	// 记录原始响应的十六进制内容（前100字节）
	if len(gbkResp) > 0 {
		maxLen := len(gbkResp)
		if maxLen > 100 {
			maxLen = 100
		}
		logger.Info(fmt.Sprintf("原始响应(hex前%d字节): %x", maxLen, gbkResp[:maxLen]))
		logger.Info(fmt.Sprintf("原始响应(string前%d字节): %s", maxLen, string(gbkResp[:maxLen])))
	} else {
		logger.Error("响应体完全为空")
	}

	// 检查响应是否为空
	if len(gbkResp) == 0 {
		// 记录更多调试信息
		logger.Error(fmt.Sprintf("空响应调试信息 - URL: %s, 超时设置: %d秒, 请求体长度: %d",
			api.URL, api.Timeout, len(gbkBody)))

		// 检查响应头
		for name, values := range resp.Header {
			for _, value := range values {
				logger.Info(fmt.Sprintf("响应头: %s = %s", name, value))
			}
		}

		return fmt.Errorf("服务器返回空响应")
	}

	utf8Reader := transform.NewReader(bytes.NewReader(gbkResp), simplifiedchinese.GBK.NewDecoder())
	utf8Resp, err := ioutil.ReadAll(utf8Reader)
	if err != nil {
		logger.Error(fmt.Sprintf("GBK decode error: %v", err))
		return fmt.Errorf("GBK decode error: %v", err)
	}

	// 解析XML
	xmlContent := string(utf8Resp)
	xmlContent = strings.Replace(xmlContent, `encoding="gb2312"`, `encoding="UTF-8"`, 1)

	var orderResp OrderResponse
	err = xml.Unmarshal([]byte(xmlContent), &orderResp)
	if err != nil {
		logger.Error(fmt.Sprintf("XML unmarshal error: %v", err))
		return fmt.Errorf("XML unmarshal error: %v", err)
	}

	// 检查业务结果
	if orderResp.ResultNo != "0" {
		logger.Error(fmt.Sprintf("业务错误: 错误码=%s, 错误信息=%s", orderResp.ResultNo, orderResp.Remark1))
		return fmt.Errorf("业务错误: %s", orderResp.Remark1)
	}

	logger.Info(fmt.Sprintf("【提交订单成功】order_number: %s, platform_order_id: %s", order.OrderNumber, orderResp.OrderID))
	return nil
}

// QueryOrderStatus 查询订单状态
func (p *ChongzhiPlatform) QueryOrderStatus(order *model.Order) (model.OrderStatus, error) {
	logger.Info(fmt.Sprintf("【开始查询订单状态】order_id: %d, order_number: %s", order.ID, order.OrderNumber))

	// 这里需要根据实际的查询接口实现
	// 暂时返回处理中状态
	return model.OrderStatusRecharging, nil
}

// CallbackRequest 回调请求参数结构
type CallbackRequest struct {
	UserID             string `form:"userid" json:"userid"`
	OrderID            string `form:"orderid" json:"orderid"`
	SpOrderID          string `form:"sporderid" json:"sporderid"`
	MerchantSubmitTime string `form:"merchantsubmittime" json:"merchantsubmittime"`
	ResultNo           string `form:"resultno" json:"resultno"`
	Sign               string `form:"sign" json:"sign"`
	ParValue           string `form:"parvalue" json:"parvalue"`
	Remark1            string `form:"remark1" json:"remark1"`
	FundBalance        string `form:"fundbalance" json:"fundbalance"`
}

// ParseCallbackData 解析回调数据
func (p *ChongzhiPlatform) ParseCallbackData(data []byte) (*model.CallbackData, error) {
	logger.Info("开始解析充值平台回调数据", "原始数据长度", len(data), "原始数据", string(data))

	// 先尝试解析为GBK编码的表单数据
	utf8Reader := transform.NewReader(bytes.NewReader(data), simplifiedchinese.GBK.NewDecoder())
	utf8Data, err := ioutil.ReadAll(utf8Reader)
	if err != nil {
		// 如果GBK解码失败，直接使用原始数据
		logger.Info("GBK解码失败，使用原始数据", "错误", err)
		utf8Data = data
	} else {
		logger.Info("GBK解码成功", "解码后数据", string(utf8Data))
	}

	// 解析表单参数
	form, err := url.ParseQuery(string(utf8Data))
	if err != nil {
		logger.Error("解析表单参数失败", "错误", err, "数据", string(utf8Data))
		return nil, fmt.Errorf("parse callback form data failed: %v", err)
	}

	// 打印所有解析到的参数
	logger.Info("解析到的表单参数:")
	for key, values := range form {
		for _, value := range values {
			logger.Info(fmt.Sprintf("  %s = %s", key, value))
		}
	}

	// 提取参数
	callbackReq := CallbackRequest{
		UserID:             form.Get("userid"),
		OrderID:            form.Get("orderid"),
		SpOrderID:          form.Get("sporderid"),
		MerchantSubmitTime: form.Get("merchantsubmittime"),
		ResultNo:           form.Get("resultno"),
		Sign:               form.Get("sign"),
		ParValue:           form.Get("parvalue"),
		Remark1:            form.Get("remark1"),
		FundBalance:        form.Get("fundbalance"),
	}

	// 验证必要参数
	if callbackReq.UserID == "" || callbackReq.OrderID == "" || callbackReq.SpOrderID == "" ||
		callbackReq.MerchantSubmitTime == "" || callbackReq.ResultNo == "" || callbackReq.Sign == "" {
		return nil, fmt.Errorf("missing required callback parameters")
	}

	// 签名验证
	// 根据文档：sign=MD5(userid=xxxx&orderid=xxxxxxx&sporderid=xxxxx&merchantsubmittime=xxxxx&resultno=xxxxx&key=xxxxxxx).toUpperCase()
	// 获取对应账号的key进行验证
	logger.Info("开始签名验证", "userid", callbackReq.UserID)
	account, err := p.platformRepo.GetPlatformAccountByAccountName(callbackReq.UserID)
	if err != nil {
		logger.Error("获取平台账号失败", "错误", err, "userid", callbackReq.UserID)
		return nil, fmt.Errorf("get platform account failed: %v", err)
	}
	logger.Info("获取到平台账号", "account_name", account.AccountName, "app_secret长度", len(account.AppSecret))

	signStr := fmt.Sprintf("userid=%s&orderid=%s&sporderid=%s&merchantsubmittime=%s&resultno=%s&key=%s",
		callbackReq.UserID, callbackReq.OrderID, callbackReq.SpOrderID,
		callbackReq.MerchantSubmitTime, callbackReq.ResultNo, account.AppKey)
	logger.Info("构建签名字符串", "signStr", signStr)

	expectedSign := strings.ToUpper(signature.GetMD5(signStr))
	logger.Info("签名验证结果", "期望签名", expectedSign, "实际签名", callbackReq.Sign, "是否匹配", expectedSign == callbackReq.Sign)

	if expectedSign != callbackReq.Sign {
		logger.Error("签名验证失败", "期望签名", expectedSign, "实际签名", callbackReq.Sign, "签名字符串", signStr)
		return nil, fmt.Errorf("invalid signature: expected %s, got %s", expectedSign, callbackReq.Sign)
	}
	logger.Info("签名验证成功")

	// 映射订单状态（根据文档：1=成功，9=失败）
	status := model.OrderStatusFailed
	if callbackReq.ResultNo == "1" {
		status = model.OrderStatusSuccess
	}

	return &model.CallbackData{
		OrderID:       callbackReq.SpOrderID,
		OrderNumber:   callbackReq.SpOrderID,
		Status:        strconv.Itoa(int(status)),
		Message:       callbackReq.Remark1,
		CallbackType:  "order_status",
		Amount:        callbackReq.ParValue,
		Timestamp:     callbackReq.MerchantSubmitTime,
		TransactionID: callbackReq.OrderID,
	}, nil
}

// BalanceResponse 余额查询响应结构
type BalanceResponse struct {
	XMLName  xml.Name `xml:"user"`
	UserID   string   `xml:"userid"`
	Balance  string   `xml:"balance"`
	ResultNo string   `xml:"resultno"`
}

// QueryBalance 查询账户余额
func (p *ChongzhiPlatform) QueryBalance(ctx context.Context, accountID int64) (float64, error) {
	logger.Info(fmt.Sprintf("【开始查询账户余额】account_id: %d", accountID))

	// 获取平台账号信息
	account, err := p.platformRepo.GetPlatformAccountByID(accountID)
	if err != nil {
		logger.Error(fmt.Sprintf("获取平台账号失败: %v", err))
		return 0, fmt.Errorf("get platform account failed: %v", err)
	}

	// 获取平台API配置（使用固定的API ID或通过其他方式获取）
	// 这里需要根据实际情况获取余额查询API的配置
	// 暂时使用硬编码的URL，实际应该从配置中获取
	balanceURL := "http://120.26.111.198:9086/searchbalance.do"
	timeout := 30 // 默认30秒超时

	// 构建签名字符串: sign=MD5(userid=xxxx&key=xxxxxxx).toUpperCase()
	signStr := fmt.Sprintf("userid=%s&key=%s", account.AccountName, account.AppKey)
	sign := strings.ToUpper(signature.GetMD5(signStr))
	logger.Info(fmt.Sprintf("构建余额查询签名: signStr=%s, sign=%s", signStr, sign))

	// 构建请求参数
	form := url.Values{}
	form.Set("userid", account.AccountName)
	form.Set("sign", sign)

	// 编码为GBK
	formStr := form.Encode()
	logger.Info(fmt.Sprintf("余额查询表单数据(UTF-8): %s", formStr))
	gbkBody, err := utils.EncodeGBK(formStr)
	if err != nil {
		logger.Error(fmt.Sprintf("GBK encode error: %v", err))
		return 0, fmt.Errorf("GBK encode error: %v", err)
	}

	// 发送POST请求
	logger.Info(fmt.Sprintf("余额查询请求URL: %s", balanceURL))
	req, err := http.NewRequest("POST", balanceURL, bytes.NewReader(gbkBody))
	if err != nil {
		logger.Error(fmt.Sprintf("NewRequest error: %v", err))
		return 0, fmt.Errorf("NewRequest error: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=gbk")
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; RechargeBot/1.0)")

	client := &http.Client{Timeout: time.Duration(timeout) * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logger.Error(fmt.Sprintf("HTTP request error: %v", err))
		return 0, fmt.Errorf("HTTP request error: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应body（GBK转UTF-8）
	gbkResp, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Error(fmt.Sprintf("ReadAll error: %v", err))
		return 0, fmt.Errorf("ReadAll error: %v", err)
	}

	logger.Info(fmt.Sprintf("余额查询HTTP状态码: %d, 响应长度: %d", resp.StatusCode, len(gbkResp)))

	// 检查响应是否为空
	if len(gbkResp) == 0 {
		logger.Error("余额查询返回空响应")
		return 0, fmt.Errorf("服务器返回空响应")
	}

	utf8Reader := transform.NewReader(bytes.NewReader(gbkResp), simplifiedchinese.GBK.NewDecoder())
	utf8Resp, err := ioutil.ReadAll(utf8Reader)
	if err != nil {
		logger.Error(fmt.Sprintf("GBK decode error: %v", err))
		return 0, fmt.Errorf("GBK decode error: %v", err)
	}

	// 解析XML
	xmlContent := string(utf8Resp)
	xmlContent = strings.Replace(xmlContent, `encoding="gb2312"`, `encoding="UTF-8"`, 1)
	logger.Info(fmt.Sprintf("余额查询响应XML: %s", xmlContent))

	var balanceResp BalanceResponse
	err = xml.Unmarshal([]byte(xmlContent), &balanceResp)
	if err != nil {
		logger.Error(fmt.Sprintf("XML unmarshal error: %v", err))
		return 0, fmt.Errorf("XML unmarshal error: %v", err)
	}

	// 检查业务结果
	if balanceResp.ResultNo != "1" {
		logger.Error(fmt.Sprintf("余额查询失败: resultno=%s", balanceResp.ResultNo))
		return 0, fmt.Errorf("余额查询失败: resultno=%s", balanceResp.ResultNo)
	}

	// 转换余额为浮点数
	balance, err := strconv.ParseFloat(balanceResp.Balance, 64)
	if err != nil {
		logger.Error(fmt.Sprintf("余额转换失败: %v", err))
		return 0, fmt.Errorf("余额转换失败: %v", err)
	}

	logger.Info(fmt.Sprintf("【查询账户余额成功】account_id: %d, balance: %.2f", accountID, balance))
	return balance, nil
}
