# 外部订单API文档

## 概述

外部订单API为第三方系统提供充值订单的创建、查询和状态回调功能。所有API调用都需要进行签名验证以确保安全性。

## 基础信息

- **基础URL**: `https://your-domain.com`
- **请求格式**: JSON
- **响应格式**: JSON
- **字符编码**: UTF-8
- **签名算法**: MD5

## 认证方式

### API密钥

每个接入方都会分配一个唯一的API密钥组合：
- `app_id`: 应用ID
- `app_key`: 应用密钥（用于请求头认证）
- `app_secret`: 应用秘钥（用于签名计算）

### 请求头

所有API请求都需要在请求头中包含以下信息：

```http
X-API-Key: your_app_key
X-Signature: calculated_signature
Content-Type: application/json
```

## 签名算法

### 签名步骤

1. **参数收集**: 收集所有请求参数（包括请求体中的参数）
2. **参数过滤**: 过滤掉空值参数和签名参数本身
3. **参数排序**: 按参数名进行字典序排序
4. **参数拼接**: 按照 `key=value&key=value` 的格式拼接
5. **添加密钥**: 在拼接字符串末尾添加 `&key=app_secret`
6. **计算签名**: 根据签名类型计算哈希值
7. **转换大写**: 将签名结果转换为大写

### 签名示例

假设请求参数为：
```json
{
  "app_id": "test_app_001",
  "mobile": "13800138000",
  "product_id": 1,
  "out_trade_num": "ORDER_20231201_001",
  "amount": 10.00,
  "timestamp": 1701398400,
  "nonce": "abc123",
  "sign_type": "MD5"
}
```

app_secret为：`test_secret_123456`

**步骤1-4**: 拼接字符串
```
app_id=test_app_001&amount=10.00&mobile=13800138000&nonce=abc123&out_trade_num=ORDER_20231201_001&product_id=1&timestamp=1701398400
```

**步骤5**: 添加密钥
```
app_id=test_app_001&amount=10.00&mobile=13800138000&nonce=abc123&out_trade_num=ORDER_20231201_001&product_id=1&timestamp=1701398400&key=test_secret_123456
```

**步骤6-7**: 计算MD5并转大写
```
sign = MD5(待签名字符串).toUpperCase()
```

## API接口

### 1. 创建订单

**接口地址**: `POST /external/order`

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| app_id | string | 是 | 应用ID |
| mobile | string | 是 | 手机号码 |
| product_id | int64 | 是 | 产品ID |
| out_trade_num | string | 是 | 外部交易号（唯一） |
| amount | float64 | 是 | 充值金额 |
| biz_type | string | 否 | 业务类型 |
| notify_url | string | 否 | 回调通知URL |
| param1 | string | 否 | 扩展参数1 |
| param2 | string | 否 | 扩展参数2 |
| param3 | string | 否 | 扩展参数3 |
| customer_id | int64 | 否 | 外部客户ID |
| isp | int | 否 | 运营商 |
| remark | string | 否 | 备注 |
| timestamp | int64 | 是 | 时间戳（秒） |
| nonce | string | 是 | 随机字符串 |
| sign | string | 是 | 签名 |

**请求示例**:

```json
{
  "app_id": "test_app_001",
  "mobile": "13800138000",
  "product_id": 1,
  "out_trade_num": "ORDER_20231201_001",
  "amount": 10.00,
  "biz_type": "recharge",
  "notify_url": "https://your-domain.com/callback",
  "timestamp": 1701398400,
  "nonce": "abc123",
  "sign": "CALCULATED_SIGNATURE"
}
```

**响应示例**:

```json
{
  "code": 200,
  "message": "Success",
  "data": {
    "order_id": 123456,
    "order_number": "R202312010001",
    "out_trade_num": "ORDER_20231201_001",
    "status": 1,
    "status_desc": "待支付",
    "amount": 10.00,
    "create_time": 1701398400
  },
  "timestamp": 1701398400
}
```

### 2. 查询订单

**接口地址**: `GET /external/order/query`

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| app_id | string | 是 | 应用ID |
| out_trade_num | string | 否 | 外部交易号 |
| order_number | string | 否 | 内部订单号 |
| timestamp | int64 | 是 | 时间戳（秒） |
| nonce | string | 是 | 随机字符串 |
| sign | string | 是 | 签名 |

**注意**: `out_trade_num` 和 `order_number` 至少提供一个

**请求示例**:

```
GET /external/order/query?app_id=test_app_001&out_trade_num=ORDER_20231201_001&timestamp=1701398400&nonce=abc123&sign=CALCULATED_SIGNATURE
```

**响应示例**:

```json
{
  "code": 200,
  "message": "Success",
  "data": {
    "order_id": 123456,
    "order_number": "R202312010001",
    "out_trade_num": "ORDER_20231201_001",
    "status": 3,
    "status_desc": "成功",
    "amount": 10.00,
    "create_time": 1701398400
  },
  "timestamp": 1701398400
}
```

### 3. 状态回调（主动通知）

当订单状态发生变更时，系统会主动向接入方的回调地址发送通知。

**接口地址**: 接入方提供的回调URL

**请求方式**: `POST`

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| app_id | string | 是 | 应用ID |
| out_trade_num | string | 是 | 外部交易号 |
| order_number | string | 是 | 内部订单号 |
| status | int | 是 | 订单状态 |
| message | string | 否 | 状态描述 |
| timestamp | int64 | 是 | 时间戳（秒） |
| nonce | string | 是 | 随机字符串 |
| sign | string | 是 | 签名 |

**回调示例**:

```json
{
  "app_id": "test_app_001",
  "out_trade_num": "ORDER_20231201_001",
  "order_number": "R202312010001",
  "status": 3,
  "message": "充值成功",
  "timestamp": 1701398400,
  "nonce": "xyz789",
  "sign": "CALCULATED_SIGNATURE"
}
```

**期望响应**:

接入方需要返回以下格式的响应表示接收成功：

```json
{
  "code": 200,
  "message": "Success",
  "timestamp": 1701398400
}
```

## 订单状态说明

| 状态码 | 状态名称 | 说明 |
|--------|----------|------|
| 1 | 待支付 | 订单已创建，等待支付 |
| 2 | 待充值 | 订单已支付，等待充值 |
| 3 | 充值中 | 正在进行充值操作 |
| 4 | 成功 | 充值成功 |
| 5 | 失败 | 充值失败 |
| 6 | 已取消 | 订单已取消 |
| 7 | 已退款 | 订单已退款 |

## 错误码说明

| 错误码 | 说明 |
|--------|------|
| 200 | 成功 |
| 400 | 请求参数错误 |
| 401 | 认证失败 |
| 403 | 权限不足 |
| 404 | 资源不存在 |
| 429 | 请求频率超限 |
| 500 | 服务器内部错误 |

## 安全建议

1. **HTTPS**: 生产环境必须使用HTTPS协议
2. **时间戳**: 建议设置5分钟的时间窗口，超时请求将被拒绝
3. **随机数**: 每次请求使用不同的随机数，防止重放攻击
4. **IP白名单**: 配置IP白名单限制访问来源
5. **密钥安全**: 妥善保管app_secret，不要在客户端代码中暴露
6. **日志记录**: 记录所有API调用日志，便于问题排查

## SDK示例

### Go语言示例

```go
package main

import (
    "crypto/md5"
    "encoding/hex"
    "fmt"
    "sort"
    "strconv"
    "strings"
    "time"
)

type APIClient struct {
    AppID     string
    AppKey    string
    AppSecret string
    BaseURL   string
}

func (c *APIClient) GenerateSignature(params map[string]interface{}, signType string) string {
    // 过滤空值参数
    filteredParams := make(map[string]string)
    for k, v := range params {
        if v != nil && v != "" {
            filteredParams[k] = fmt.Sprintf("%v", v)
        }
    }
    
    // 参数排序
    keys := make([]string, 0, len(filteredParams))
    for k := range filteredParams {
        keys = append(keys, k)
    }
    sort.Strings(keys)
    
    // 拼接参数
    var paramPairs []string
    for _, k := range keys {
        paramPairs = append(paramPairs, fmt.Sprintf("%s=%s", k, filteredParams[k]))
    }
    paramString := strings.Join(paramPairs, "&")
    
    // 添加密钥
    signString := paramString + "&key=" + c.AppSecret
    
    // 计算签名
    switch strings.ToUpper(signType) {
    case "MD5":
        hash := md5.Sum([]byte(signString))
        return strings.ToUpper(hex.EncodeToString(hash[:]))
    default:
        return ""
    }
}

func (c *APIClient) CreateOrder(mobile string, productID int64, outTradeNum string, amount float64) {
    params := map[string]interface{}{
        "app_id":        c.AppID,
        "mobile":        mobile,
        "product_id":    productID,
        "out_trade_num": outTradeNum,
        "amount":        amount,
        "timestamp":     time.Now().Unix(),
        "nonce":         "abc123",
    }
    
    signature := c.GenerateSignature(params)
    params["sign"] = signature
    
    // 发送HTTP请求...
    fmt.Printf("Signature: %s\n", signature)
}
```

### PHP示例

```php
<?php
class APIClient {
    private $appId;
    private $appKey;
    private $appSecret;
    private $baseUrl;
    
    public function __construct($appId, $appKey, $appSecret, $baseUrl) {
        $this->appId = $appId;
        $this->appKey = $appKey;
        $this->appSecret = $appSecret;
        $this->baseUrl = $baseUrl;
    }
    
    public function generateSignature($params) {
        // 过滤空值参数
        $filteredParams = array_filter($params, function($value) {
            return $value !== null && $value !== '';
        });
        
        // 移除签名参数
        unset($filteredParams['sign']);
        
        // 参数排序
        ksort($filteredParams);
        
        // 拼接参数
        $paramString = http_build_query($filteredParams);
        
        // 添加密钥
        $signString = $paramString . '&key=' . $this->appSecret;
        
        // 计算MD5签名
        return strtoupper(md5($signString));
    }
    
    public function createOrder($mobile, $productId, $outTradeNum, $amount) {
        $params = [
            'app_id' => $this->appId,
            'mobile' => $mobile,
            'product_id' => $productId,
            'out_trade_num' => $outTradeNum,
            'amount' => $amount,
            'timestamp' => time(),
            'nonce' => 'abc123'
        ];
        
        $params['sign'] = $this->generateSignature($params);
        
        // 发送HTTP请求...
        echo "Signature: " . $params['sign'] . "\n";
    }
}
?>
```

## 测试环境

- **测试地址**: `https://test-api.your-domain.com`
- **测试账号**: 
  - app_id: `test_app_001`
  - app_key: `test_key_123456789`
  - app_secret: `test_secret_abcdefghijklmnopqrstuvwxyz123456`

## 联系我们

如有技术问题，请联系：
- 邮箱: api-support@your-domain.com
- 电话: 400-xxx-xxxx
- QQ群: xxxxxxxxx