# 手机查询API文档

## 概述

手机查询API提供了余额查询和缴费记录查询功能，支持移动、联通、电信三大运营商。该功能通过对接第三方API实现，为平台提供实时的手机号码相关信息查询服务。

## 配置说明

### 配置文件

在 `configs/config.yaml` 中配置第三方API信息：

```yaml
third_party_api:
  base_url: "http://35.220.200.84:18080"  # 第三方API基础URL
  merchant_id: "your_merchant_id"         # 商户号（需要替换为实际值）
  token: "your_token"                     # 访问令牌（需要替换为实际值）
  timeout: 30                             # 请求超时时间（秒）
```

### 环境变量（可选）

也可以通过环境变量配置：

```bash
export THIRD_PARTY_API_BASE_URL="http://35.220.200.84:18080"
export THIRD_PARTY_API_MERCHANT_ID="your_merchant_id"
export THIRD_PARTY_API_TOKEN="your_token"
```

## API接口

### 1. 余额查询

#### 接口信息
- **URL**: `/api/v1/phone/balance`
- **方法**: POST
- **认证**: 需要管理员权限
- **Content-Type**: application/json

#### 请求参数

```json
{
  "phone": "13800138000",
  "isp_type": "yd"
}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| phone | string | 是 | 手机号码，11位数字 |
| isp_type | string | 是 | 运营商类型：dx(电信)、yd(移动)、lt(联通) |

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| max_retries | int | 否 | 最大重试次数，范围1-5，默认1 |

#### 响应示例

**成功响应**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "errcode": 0,
    "errmsg": "success",
    "datas": "98.50"
  }
}
```

**失败响应**:
```json
{
  "code": 400,
  "message": "查询失败: 手机号码不存在",
  "data": null
}
```

### 2. 余额查询（表单方式）

#### 接口信息
- **URL**: `/api/v1/phone/balance/form`
- **方法**: POST
- **认证**: 需要管理员权限
- **Content-Type**: application/x-www-form-urlencoded

#### 请求参数

```
phone=13800138000&isp_type=yd
```

### 3. 缴费记录查询

#### 接口信息
- **URL**: `/api/v1/phone/payment-records`
- **方法**: POST
- **认证**: 需要管理员权限
- **Content-Type**: application/json

#### 请求参数

```json
{
  "phone": "13800138000",
  "isp_type": "yd"
}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| phone | string | 是 | 手机号码，11位数字 |
| isp_type | string | 是 | 运营商类型：yd(移动)、lt(联通)，**不支持电信** |

#### 响应示例

**移动用户响应**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "errcode": 0,
    "errmsg": "success",
    "datas": [
      {
        "payTimeStamp": 1640995200000,
        "payAmount": "50.00",
        "payTime": "2022-01-01 08:00:00"
      }
    ]
  }
}
```

**联通用户响应**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "errcode": 0,
    "errmsg": "success",
    "data": [
      {
        "payTimeStamp": 1640995200000,
        "payAmount": "30.00",
        "payTime": "2022-01-01 08:00:00",
        "channel": "支付宝"
      }
    ]
  }
}
```

### 4. 缴费记录查询（表单方式）

#### 接口信息
- **URL**: `/api/v1/phone/payment-records/form`
- **方法**: POST
- **认证**: 需要管理员权限
- **Content-Type**: application/x-www-form-urlencoded

## 错误码说明

| 错误码 | 说明 |
|--------|------|
| 0 | 成功 |
| 1001 | 参数错误 |
| 1002 | 手机号码不存在 |
| 1003 | 运营商类型不支持 |
| 1004 | 查询频率过高 |
| 1005 | 系统维护中 |
| 9999 | 系统错误 |

## 使用示例

### cURL示例

**余额查询**:
```bash
curl -X POST "http://localhost:8080/api/v1/phone/balance" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your_jwt_token" \
  -d '{
    "phone": "13800138000",
    "isp_type": "yd"
  }'
```

**缴费记录查询**:
```bash
curl -X POST "http://localhost:8080/api/v1/phone/payment-records" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your_jwt_token" \
  -d '{
    "phone": "13800138000",
    "isp_type": "yd"
  }'
```

### JavaScript示例

```javascript
// 余额查询
const queryBalance = async (phone, ispType) => {
  try {
    const response = await fetch('/api/v1/phone/balance', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`
      },
      body: JSON.stringify({
        phone: phone,
        isp_type: ispType
      })
    });
    
    const result = await response.json();
    if (result.code === 200) {
      console.log('余额:', result.data.datas);
    } else {
      console.error('查询失败:', result.message);
    }
  } catch (error) {
    console.error('请求错误:', error);
  }
};

// 缴费记录查询
const queryPaymentRecords = async (phone, ispType) => {
  try {
    const response = await fetch('/api/v1/phone/payment-records', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`
      },
      body: JSON.stringify({
        phone: phone,
        isp_type: ispType
      })
    });
    
    const result = await response.json();
    if (result.code === 200) {
      const records = result.data.datas || result.data.data || [];
      console.log('缴费记录:', records);
    } else {
      console.error('查询失败:', result.message);
    }
  } catch (error) {
    console.error('请求错误:', error);
  }
};
```

## 注意事项

1. **权限要求**: 所有接口都需要管理员权限，普通用户无法访问
2. **运营商支持**: 
   - 余额查询：支持移动(yd)、联通(lt)、电信(dx)
   - 缴费记录查询：仅支持移动(yd)、联通(lt)
3. **重试机制**: 支持自动重试，可通过 `max_retries` 参数控制重试次数
4. **超时设置**: 默认30秒超时，可在配置文件中调整
5. **频率限制**: 建议控制查询频率，避免触发第三方API的频率限制
6. **数据格式**: 移动和联通的缴费记录响应格式略有不同，已在代码中统一处理

## 集成到充值流程

### 充值前查询余额

```go
// 在充值服务中集成余额查询
func (s *RechargeService) PreRechargeCheck(phone, ispType string) error {
    // 查询当前余额
    balance, err := s.phoneQueryService.QueryBalance(ctx, phone, ispType)
    if err != nil {
        return fmt.Errorf("查询余额失败: %v", err)
    }
    
    if balance.ErrCode != 0 {
        return fmt.Errorf("余额查询返回错误: %s", balance.ErrMsg)
    }
    
    // 记录充值前余额
    s.logger.Info("充值前余额", 
        zap.String("phone", phone),
        zap.String("balance", balance.Data),
    )
    
    return nil
}
```

### 充值后验证

```go
// 充值完成后验证
func (s *RechargeService) PostRechargeVerify(phone, ispType string, expectedAmount float64) error {
    // 等待一段时间让余额更新
    time.Sleep(30 * time.Second)
    
    // 查询充值后余额
    balance, err := s.phoneQueryService.QueryBalanceWithRetry(ctx, phone, ispType, 3)
    if err != nil {
        return fmt.Errorf("验证充值结果失败: %v", err)
    }
    
    // 验证余额是否增加
    // 这里需要根据实际业务逻辑实现
    
    return nil
}
```

## 监控和日志

系统会自动记录以下信息：
- 请求和响应的详细日志
- 请求耗时统计
- 错误率监控
- 第三方API调用状态

可以通过日志文件或监控系统查看相关信息。