package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"recharge-go/internal/model"
	"recharge-go/pkg/logger"
	"recharge-go/pkg/signature"
)

// BeeService 蜜蜂平台服务
type BeeService struct {
	baseURL string
}

// NewBeeService 创建蜜蜂平台服务实例
func NewBeeService() *BeeService {
	return &BeeService{
		baseURL: "http://test.shop.center.mf178.cn",
	}
}

// BeeProduct 蜜蜂平台商品信息
type BeeProduct struct {
	GoodsID                int64                       `json:"goods_id"`
	GoodsMode              int                         `json:"goods_mode"`
	GoodsModeText          string                      `json:"goods_mode_text"`
	BID                    int                         `json:"b_id"`
	BName                  string                      `json:"b_name"`
	VenderID               int                         `json:"vender_id"`
	VenderName             string                      `json:"vender_name"`
	GoodsSku               string                      `json:"goods_sku"`
	GoodsName              string                      `json:"goods_name"`
	SourcePackLimitID      int                         `json:"source_pack_limit_id"`
	SourceLimitTxt         string                      `json:"source_limit_txt"`
	SpecValueIds           string                      `json:"spec_value_ids"`
	SpecName               string                      `json:"spec_name"`
	UserPayment            string                      `json:"user_payment"`
	UserQuoteMode          int                         `json:"user_quote_mode"`
	UserQuoteModeText      string                      `json:"user_quote_mode_text"`
	UserPaymentRange       string                      `json:"user_payment_range"`
	SupplyStatus           int                         `json:"supply_status"`
	NeedSetProv            bool                        `json:"need_set_prov"`
	IsBan                  int                         `json:"is_ban"`
	TemplatePrice          string                      `json:"template_price"`
	TemplateSettleDiscount float64                     `json:"template_settle_discount"`
	Rowspan                int                         `json:"rowspan"`
	LastTradedPrice        interface{}                 `json:"last_traded_price"`
	CardPendingNum         int                         `json:"card_pending_num"`
	UserQuoteStockInfo     *BeeUserQuoteStockInfo      `json:"user_quote_stock_info"`
	UserQuoteStockProvInfo []BeeUserQuoteStockProvInfo `json:"user_quote_stock_prov_info"`
	// ProvCodeConfig []BeeProvCodeConfig `json:"prov_code_config"`
	// OrderLimitConfig       *BeeOrderLimitConfig     `json:"order_limit_config"`
	// OrderLimitConfigFilm   *BeeOrderLimitConfigFilm `json:"order_limit_config_film"`
}

// BeeProvince 蜜蜂平台省份信息
type BeeProvince struct {
	Prov             string  `json:"prov"`
	UserQuotePayment float64 `json:"user_quote_payment"`
	ExternalCode     string  `json:"external_code"`
	Status           int     `json:"status"`
}

// BeeProductListResponse 蜜蜂平台商品列表响应
type BeeProductListResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// BeeProductListData 蜜蜂平台商品列表数据结构
type BeeProductListData struct {
	GoodsInfo            []BeeProduct             `json:"goods_info"`
	StatInfo             BeeStatInfo              `json:"stat_info"`
	UserVenderConfigInfo *BeeUserVenderConfigInfo `json:"user_vender_config_info"`
}

// BeeStatInfo 统计信息
type BeeStatInfo struct {
	Total         int `json:"total"`
	SupplyTotal   int `json:"supply_total"`
	UnsupplyTotal int `json:"unsupply_total"`
	Page          int `json:"page"`
	PageSize      int `json:"pageSize"`
}

// BeeUserVenderConfigInfo 用户供应商配置信息
type BeeUserVenderConfigInfo struct {
	RechargeIdInfo interface{} `json:"recharge_id_info"`
}

// BeeUpdatePriceRequest 更新商品价格请求
type BeeUpdatePriceRequest struct {
	GoodsID              int64         `json:"goods_id"`
	Status               int           `json:"status"`
	ProvLimitType        int           `json:"prov_limit_type"`
	UserQuoteType        int           `json:"user_quote_type"`
	ExternalCodeLinkType int           `json:"external_code_link_type"`
	UserQuotePayment     float64       `json:"user_quote_payment"`
	ExternalCode         string        `json:"external_code"`
	ProvInfo             []BeeProvince `json:"prov_info"`
}

// BeeUpdateProvinceRequest 更新省份配置请求
type BeeUpdateProvinceRequest struct {
	GoodsID int64    `json:"goods_id"`
	Provs   []string `json:"provs"`
}

// GetProductList 获取商品列表
func (s *BeeService) GetProductList(account *model.PlatformAccount, page, pageSize int) (*BeeProductListResponse, error) {
	// 构造data参数
	dataParams := map[string]interface{}{
		"b_id":     "6",
		"page":     page,
		"pageSize": pageSize,
	}

	dataJSON, err := json.Marshal(dataParams)
	if err != nil {
		return nil, fmt.Errorf("构造data参数失败: %v", err)
	}

	params := map[string]string{
		"data": string(dataJSON),
	}

	resp, err := s.makeRequest("/userapi/sgd/getSupplyGoodManageList", params, account)
	if err != nil {
		return nil, err
	}

	var result BeeProductListResponse
	err = json.Unmarshal(resp, &result)
	if err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	return &result, nil
}

// ParseProductListData 解析商品列表数据
func ParseProductListData(rawData []byte) (*BeeProductListData, error) {
	// 首先尝试解析为包含goods_info的对象格式
	var objData struct {
		GoodsInfo            []BeeProduct             `json:"goods_info"`
		StatInfo             *BeeStatInfo             `json:"stat_info"`
		UserVenderConfigInfo *BeeUserVenderConfigInfo `json:"user_vender_config_info"`
	}
	err := json.Unmarshal(rawData, &objData)
	if err == nil && objData.GoodsInfo != nil {
		// 对象格式解析成功
		result := &BeeProductListData{
			GoodsInfo:            objData.GoodsInfo,
			UserVenderConfigInfo: objData.UserVenderConfigInfo,
		}

		// 如果有统计信息则使用，否则生成默认统计信息
		if objData.StatInfo != nil {
			result.StatInfo = *objData.StatInfo
		} else {
			result.StatInfo = BeeStatInfo{
				Total:    len(objData.GoodsInfo),
				Page:     1,
				PageSize: len(objData.GoodsInfo),
			}
		}
		return result, nil
	}

	// 如果对象格式解析失败，尝试解析为数组格式
	var arrayData []BeeProduct
	err = json.Unmarshal(rawData, &arrayData)
	if err != nil {
		logger.Error("bee parse product list data", "error", err, "raw_data", string(rawData)[:min(len(rawData), 1000)])
		return nil, fmt.Errorf("无法解析数据格式: %v", err)
	}

	// 将数组格式转换为标准格式
	return &BeeProductListData{
		GoodsInfo: arrayData,
		StatInfo: BeeStatInfo{
			Total:    len(arrayData),
			Page:     1,
			PageSize: len(arrayData),
		},
	}, nil
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// max 返回两个整数中的较大值
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// UpdateProductPrice 更新商品价格
func (s *BeeService) UpdateProductPrice(account *model.PlatformAccount, req *BeeUpdatePriceRequest) error {
	// 构造商品数据结构，按照官方示例格式
	goods := map[string]interface{}{
		"goods_id":                req.GoodsID,
		"status":                  req.Status,
		"prov_limit_type":         req.ProvLimitType,
		"user_quote_type":         req.UserQuoteType,
		"external_code_link_type": req.ExternalCodeLinkType,
		"user_quote_payment":      req.UserQuotePayment,
		"external_code":           req.ExternalCode,
	}

	// 添加省份信息
	if len(req.ProvInfo) > 0 {
		goods["prov_info"] = req.ProvInfo
	}

	// 构造data参数，包装成goods数组
	dataStruct := map[string]interface{}{
		"goods": []interface{}{goods},
	}

	// 将data结构转换为JSON字符串
	dataJSON, err := json.Marshal(dataStruct)
	if err != nil {
		return fmt.Errorf("构造data参数失败: %v", err)
	}

	params := map[string]string{
		"data": string(dataJSON),
	}
	logger.Info("bee update price", "params", params)
	_, err = s.makeRequest("/userapi/sgd/editSupplyGoodManageStockWithProv", params, account)
	return err
}

// UpdateProductProvince 更新商品省份配置
func (s *BeeService) UpdateProductProvince(account *model.PlatformAccount, req *BeeUpdateProvinceRequest) error {
	params := map[string]string{
		"goods_id": strconv.FormatInt(req.GoodsID, 10),
	}

	// 添加省份列表
	provsJSON, _ := json.Marshal(req.Provs)
	params["provs"] = string(provsJSON)

	_, err := s.makeRequest("/userapi/sgd/editSupplyGoodManageProvCode", params, account)
	return err
}

// makeRequest 发起API请求
func (s *BeeService) makeRequest(endpoint string, params map[string]string, account *model.PlatformAccount) ([]byte, error) {
	// 添加公共参数
	params["app_key"] = account.AppKey
	params["timestamp"] = strconv.FormatInt(time.Now().Unix(), 10)

	// 转换参数类型以适配公共签名方法
	signParams := make(map[string]interface{})
	for k, v := range params {
		signParams[k] = v
	}

	// 生成签名
	sign := signature.GenerateSign(signParams, account.AppSecret)
	params["sign"] = sign

	// 构建请求URL
	requestURL := s.baseURL + endpoint

	// 构建POST数据
	data := url.Values{}
	for k, v := range params {
		data.Set(k, v)
	}

	// 发起请求
	resp, err := http.PostForm(requestURL, data)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}
	logger.Info("bee response", "body", string(body))
	return body, nil
}

// BeeUserQuoteStockInfo 用户报价库存信息
type BeeUserQuoteStockInfo struct {
	ID                   int         `json:"id"`
	UserQuotePayment     string      `json:"user_quote_payment"`
	UsableStock          int         `json:"usable_stock"`
	UserQuoteDiscount    interface{} `json:"user_quote_discount"`
	ProvLimitType        int         `json:"prov_limit_type"`
	UserQuoteType        int         `json:"user_quote_type"`
	ExternalCodeLinkType int         `json:"external_code_link_type"`
}

// BeeUserQuoteStockProvInfo 蜜蜂平台用户报价库存省份信息
type BeeUserQuoteStockProvInfo struct {
	ID                int         `json:"id"`
	QuoteID           int         `json:"quote_id"`
	GoodsID           int         `json:"goods_id"`
	Prov              string      `json:"prov"`
	ProvID            int         `json:"prov_id"`
	UserQuotePayment  string      `json:"user_quote_payment"`
	UserQuoteDiscount int         `json:"user_quote_discount"`
	ExternalCode      string      `json:"external_code"`
	Status            int         `json:"status"`
	LastTradedPrice   interface{} `json:"last_traded_price"`
	PinYin            string      `json:"pin_yin"`
}

// BeeProvCodeConfig 省份配置信息
type BeeProvCodeConfig struct {
	Prov              string      `json:"prov"`
	ProvID            int         `json:"prov_id"`
	UserQuotePayment  string      `json:"user_quote_payment"`
	UserQuoteDiscount interface{} `json:"user_quote_discount"`
	ExternalCode      string      `json:"external_code"`
	Status            string      `json:"status"`
}

// BeeOrderLimitConfig 订单限制配置
type BeeOrderLimitConfig struct {
	SourceLimit    int         `json:"source_limit"`
	SourceLimitTxt string      `json:"source_limit_txt"`
	PriceLimit     interface{} `json:"price_limit"`
	ExternalCode   string      `json:"external_code"`
	UserRejectTime interface{} `json:"user_reject_time"`
}

// BeeOrderLimitConfigFilm 订单限制配置影片
type BeeOrderLimitConfigFilm struct {
	Province   string `json:"province"`
	City       string `json:"city"`
	Film       string `json:"film"`
	Cinema     string `json:"cinema"`
	ChangeSeat string `json:"change_seat"`
	TicketNum  string `json:"ticket_num"`
}
