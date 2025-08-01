# Payc2平台集成指南

## 概述

Payc2是一个话费充值平台，支持移动、联通、电信三大运营商的话费充值服务。本文档介绍如何在充值系统中集成payc2平台。

## 平台特性

- **支持运营商**: 移动(YD)、联通(LT)、电信(DX)
- **充值方式**: 话费直充
- **签名算法**: MD5
- **接口协议**: HTTP POST (application/x-www-form-urlencoded)
- **API版本**: v1.240318

## 配置步骤

### 1. 数据库配置

执行SQL脚本创建平台配置：

```bash
mysql -u username -p database_name < scripts/setup_payc2_platform.sql
```

### 2. 修改配置参数

在执行SQL脚本前，请修改以下参数：

- **商户号** (`app_key`): 替换为payc2提供的实际商户号
- **商户秘钥** (`app_secret`): 替换为payc2提供的实际签名密钥
- **回调地址** (`callback_url`): 替换为您的实际回调地址
- **API地址** (`url`): 确认payc2的实际API地址

### 3. 验证配置

执行以下SQL查询验证配置是否正确：

```sql
SELECT 
    p.name as platform_name,
    pa.account_name,
    pa.app_key,
    api.url as api_url
FROM platforms p
LEFT JOIN platform_accounts pa ON p.id = pa.platform_id
LEFT JOIN platform_apis api ON p.id = api.platform_id
WHERE p.code = 'payc2';
```

## API接口说明

### 充值接口

**接口地址**: `{baseURL}/apis/wof/order/create_phone`  
**请求方式**: POST  
**Content-Type**: application/x-www-form-urlencoded

#### 请求参数

| 参数名 | 必填 | 类型 | 说明 |
|--------|------|------|------|
| merch | 是 | string | 商户号 |
| orderNo | 否 | string | 商户订单号（最长50字符） |
| amount | 是 | int | 订单金额（整数，如：100） |
| notifyUrl | 是 | string | 回调通知地址 |
| timeoutSecond | 否 | int | 超时时间（秒，默认1800） |
| phone | 是 | string | 手机号 |
| telco | 是 | string | 运营商（DX:电信, YD:移动, LT:联通） |
| sign | 是 | string | 签名 |

#### 响应参数

| 参数名 | 类型 | 说明 |
|--------|------|------|
| errcode | int | 错误码（0表示成功） |
| errmsg | string | 描述信息 |
| datas | object | 数据对象 |
| datas.uid | string | 系统订单ID |
| datas.orderNo | string | 商户订单号 |

## 签名算法

### 签名步骤

1. 所有请求参数按照ASCII码的升序进行排序
2. 按照 `key1=value1&key2=value2` 进行组合
3. 最后加上商户秘钥（`&key=商户秘钥`）
4. 进行MD5运算

### 签名示例

假设商户号和签名密钥分别为：`1000` 和 `ad6360f2d7de4b1e915a3035437c4743`

请求参数：
```
merch=1000
orderNo=10000000076
amount=100
product=1000
notifyUrl=http://127.0.0.1:10980/notify/demo/form
clientIp=
```

排序后的字符串：
```
amount=100&clientIp=&merch=1000&notifyUrl=http://127.0.0.1:10980/notify/demo/form&orderNo=10000000076&product=1000
```

加上签名秘钥：
```
amount=100&clientIp=&merch=1000&notifyUrl=http://127.0.0.1:10980/notify/demo/form&orderNo=10000000076&product=1000&key=ad6360f2d7de4b1e915a3035437c4743
```

MD5运算结果：
```
d2c2bb545d50b48403ad1be2b03dd82a
```

## 运营商识别

系统会根据手机号前缀自动识别运营商：

### 电信 (DX)
- 133, 149, 153, 173, 177, 180, 181, 189, 199

### 联通 (LT)
- 130, 131, 132, 155, 156, 166, 175, 176, 185, 186

### 移动 (YD)
- 其他号段默认为移动

## 使用示例

### 1. 提交充值订单

```go
// 获取平台管理器
manager := recharge.NewManager(db)

// 加载平台
if err := manager.LoadPlatforms(); err != nil {
    log.Printf("加载平台失败: %v", err)
    return
}

// 提交订单
err := manager.SubmitOrder(ctx, order, api, apiParam)
if err != nil {
    log.Printf("订单提交失败: %v", err)
    return
}

log.Printf("订单提交成功: %s", order.OrderNumber)
```

### 2. 处理回调通知

```go
// 解析回调数据
callbackData, err := platform.ParseCallbackData(requestBody)
if err != nil {
    log.Printf("解析回调数据失败: %v", err)
    return
}

// 处理订单状态更新
if callbackData.Status == model.OrderStatusSuccess {
    // 充值成功处理逻辑
    log.Printf("订单充值成功: %s", callbackData.OrderNumber)
} else {
    // 充值失败处理逻辑
    log.Printf("订单充值失败: %s", callbackData.Message)
}
```

## 回调处理

系统会自动处理 payc2 平台的异步回调通知：

### 回调配置
- **回调地址**: `http://your-domain/api/callback/payc2/{userid}`
- **回调方式**: HTTP POST
- **数据格式**: application/x-www-form-urlencoded
- **返回要求**: HTTP状态码200，返回字符串"ok"

### 回调参数
| 参数名 | 必填 | 类型 | 说明 |
|--------|------|------|------|
| merch | 是 | string | 商户编号 |
| uid | 是 | string | 系统订单ID |
| orderNo | 否 | string | 商户订单号 |
| amount | 是 | int | 订单金额 |
| amountPaid | 是 | int | 已充金额 |
| stateAmount | 是 | int | 金额状态：0零充值，1已全充，3部分充，4已超充 |
| stateOver | 是 | int | 结束状态：0未结束，1已结束 |
| sign | 是 | string | 签名 |

### 回调处理逻辑
1. **签名验证**: 自动验证回调数据的MD5签名
2. **状态判断**: 根据stateAmount和stateOver判断订单状态
3. **订单更新**: 自动更新订单状态和充值结果
4. **重试机制**: 如果返回非"ok"字符串，系统会重新回调

### 状态映射
- `stateAmount=0`: 零充值 → 订单失败
- `stateAmount=1`: 已全充 → 订单成功
- `stateAmount=3`: 部分充 → 部分成功
- `stateAmount=4`: 已超充 → 订单成功
- `stateOver=1 && stateAmount=0`: 订单已结束且零充值 → 订单失败

## 注意事项

1. **签名验证**: 所有接口请求和回调都需要进行签名验证
2. **参数处理**: 传递的参数值即便为空，也必须参与签名
3. **运营商识别**: 系统会自动根据手机号识别运营商
4. **超时设置**: 建议设置合理的超时时间（默认30分钟）
5. **错误处理**: 需要根据返回的错误码进行相应的错误处理
6. **回调安全**: 回调地址需要支持HTTPS，确保数据传输安全

## 故障排查

### 常见问题

1. **签名验证失败**
   - 检查参数排序是否正确
   - 确认商户秘钥是否正确
   - 验证参数值是否包含特殊字符

2. **订单提交失败**
   - 检查API地址是否正确
   - 确认网络连接是否正常
   - 验证请求参数是否完整

3. **API响应错误**
   - **HTTP 520错误**: 平台服务器内部错误，建议稍后重试
   - **非JSON响应**: 平台返回错误信息而非标准JSON格式
   - **errCode非0**: 业务逻辑错误，查看errMsg获取具体原因

4. **回调处理异常**
   - 检查回调地址是否可访问
   - 确认回调签名验证逻辑
   - 验证回调数据格式

### 错误处理机制

系统已优化错误处理逻辑：
- 自动检查HTTP状态码
- 处理非JSON格式的错误响应
- 提供详细的错误信息用于调试

### 日志查看

系统会记录详细的操作日志，可通过以下方式查看：

```bash
# 查看payc2相关日志
grep "payc2" /path/to/logfile

# 查看签名相关日志
grep "签名" /path/to/logfile
```

## 技术支持

如遇到技术问题，请联系：
- 平台技术支持
- 查看系统日志
- 参考API文档