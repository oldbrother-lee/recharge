# Chongzhi平台集成指南

## 概述

Chongzhi平台是一个充值服务平台，支持手机话费充值等服务。本文档介绍如何在系统中配置和使用Chongzhi平台。

## 平台特性

- **编码格式**: GBK编码
- **请求方式**: HTTP POST
- **数据格式**: XML
- **签名算法**: MD5
- **支持产品**: 手机话费充值

## 配置步骤

### 1. 数据库配置

执行迁移文件来添加平台配置：

```bash
# 执行SQL迁移文件
mysql -u username -p database_name < migrations/add_chongzhi_platform.sql
```

### 2. 平台账号配置

在管理后台或直接修改数据库中的平台账号信息：

```sql
UPDATE platform_accounts 
SET app_key = '你的实际app_key', 
    app_secret = '你的实际app_secret'
WHERE platform_id = (SELECT id FROM platforms WHERE code = 'chongzhi');
```

### 3. API接口配置

更新API接口地址：

```sql
UPDATE platform_apis 
SET url = '实际的API接口地址'
WHERE code = 'chongzhi';
```

### 4. 产品参数配置

根据实际情况调整产品参数：

```sql
-- 更新产品成本和价格
UPDATE platform_api_params pap
JOIN platform_apis pa ON pap.api_id = pa.id
SET pap.cost = 实际成本价格,
    pap.price = 实际销售价格
WHERE pa.code = 'chongzhi' AND pap.product_id = '产品ID';
```

## API接口说明

### 请求参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| userid | string | 是 | 商户ID |
| productid | string | 是 | 产品ID |
| price | string | 是 | 充值金额 |
| num | string | 是 | 充值数量（通常为1） |
| mobile | string | 是 | 手机号码 |
| spordertime | string | 是 | 订单时间（yyyyMMddHHmmss） |
| sporderid | string | 是 | 商户订单号 |
| sign | string | 是 | MD5签名 |

### 签名算法

```
签名字符串 = userid=值&productid=值&price=值&num=值&mobile=值&spordertime=值&sporderid=值&key=密钥
MD5签名 = MD5(签名字符串).toUpperCase()
```

### 响应格式

成功响应示例：
```xml
<?xml version="1.0" encoding="gb2312"?>
<order>
    <orderid>平台订单号</orderid>
    <productid>产品ID</productid>
    <num>1</num>
    <ordercash>充值金额</ordercash>
    <productname>产品名称</productname>
    <sporderid>商户订单号</sporderid>
    <mobile>手机号码</mobile>
    <merchantsubmittime>提交时间</merchantsubmittime>
    <resultno>0</resultno>
    <remark1>成功</remark1>
    <fundbalance>账户余额</fundbalance>
</order>
```

失败响应示例：
```xml
<?xml version="1.0" encoding="gb2312"?>
<order>
    <resultno>错误码</resultno>
    <remark1>错误信息</remark1>
</order>
```

### 状态码说明

| 状态码 | 说明 |
|--------|------|
| 0 | 成功 |
| 其他 | 失败（具体错误信息见remark1字段） |

## 使用示例

### 1. 提交充值订单

```go
// 获取平台管理器
manager := recharge.NewManager(db)

// 提交订单
err := manager.SubmitOrder(ctx, order, api, apiParam)
if err != nil {
    log.Printf("订单提交失败: %v", err)
    return
}

log.Printf("订单提交成功: %s", order.OrderNumber)
```

### 2. 查询订单状态

```go
// 查询订单状态
err := manager.QueryOrderStatus(ctx, order)
if err != nil {
    log.Printf("查询订单状态失败: %v", err)
    return
}

log.Printf("订单状态: %v", order.Status)
```

### 3. 处理回调通知

```go
// 处理平台回调
err := manager.HandleCallback(ctx, "chongzhi", callbackData)
if err != nil {
    log.Printf("处理回调失败: %v", err)
    return
}

log.Printf("回调处理成功")
```

## 注意事项

1. **编码格式**: 请求和响应都使用GBK编码，需要进行编码转换
2. **签名验证**: 严格按照签名算法生成和验证签名
3. **错误处理**: 根据返回的resultno判断请求是否成功
4. **超时设置**: 建议设置合理的请求超时时间（30秒）
5. **日志记录**: 记录详细的请求和响应日志便于问题排查

## 故障排查

### 常见问题

1. **签名错误**: 检查签名算法和参数顺序
2. **编码问题**: 确保使用GBK编码发送请求
3. **网络超时**: 检查网络连接和超时设置
4. **参数错误**: 验证所有必填参数是否正确传递

### 调试建议

1. 开启详细日志记录
2. 使用测试环境验证配置
3. 检查数据库中的配置信息
4. 联系平台技术支持获取帮助