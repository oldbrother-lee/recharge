# TransactionID 统一标准设计

## 设计原则

### 1. TransactionID 优先级规则

为了确保回调重复检查的准确性和换通道重试的正确处理，TransactionID应按以下优先级选择：

**优先级1：平台交易流水号**
- 如果平台提供专门的交易流水号/凭证号，优先使用
- 例如：`transaction_id`, `trade_no`, `proof`, `uid` 等

**优先级2：平台订单号**
- 如果没有专门的交易流水号，使用平台返回的订单号
- 例如：`platform_order_id`, `orderid` 等

**优先级3：我方订单号**
- 作为最后的备选方案，使用我方的订单号
- 例如：`out_trade_no`, `sporderid` 等

### 2. 命名规范

```go
// 统一的TransactionID设置模式
TransactionID: getPlatformTransactionID(callbackData)
```

### 3. 实现标准

每个平台都应该实现一个内部方法来获取最合适的TransactionID：

```go
func (p *PlatformName) getPlatformTransactionID(data map[string]string) string {
    // 优先级1：平台交易流水号
    if transactionID := data["platform_transaction_field"]; transactionID != "" {
        return transactionID
    }
    
    // 优先级2：平台订单号
    if platformOrderID := data["platform_order_field"]; platformOrderID != "" {
        return platformOrderID
    }
    
    // 优先级3：我方订单号（备选）
    return data["our_order_field"]
}
```

## 各平台具体实现

### 1. chongzhi平台
```go
// 当前实现：TransactionID: callbackReq.OrderID
// 符合标准：✅ 使用平台订单号
```

### 2. payc2平台
```go
// 当前实现：TransactionID: values.Get("uid")
// 符合标准：✅ 使用平台交易流水号
```

### 3. kekebang平台
```go
// 当前实现：TransactionID: resp.Proof
// 符合标准：✅ 使用平台凭证号
```

### 4. xianzhuanxia平台
```go
// 当前实现：TransactionID: callback.OrderID
// 符合标准：✅ 使用平台订单号
```

### 5. external_api平台
```go
// 当前实现：TransactionID: orderID
// 符合标准：✅ 使用订单号
```

### 6. mishi平台
```go
// 当前实现：✅ TransactionID: "mishi_" + form["szOrderId"][0]
// 符合标准：✅ 使用平台前缀+订单号，避免跨平台冲突
```

### 7. dayuanren平台
```go
// 当前实现：✅ TransactionID: "dayuanren_" + params["out_trade_num"]
// 符合标准：✅ 使用平台前缀+订单号，避免跨平台冲突
```

## 修复状态

1. **✅ 已完成**：所有平台都已添加带平台前缀的TransactionID设置
2. **✅ 已完成**：更新了所有平台的TransactionID格式为 "platform_" + orderID
3. **✅ 已完成**：更新了测试用例以反映新的TransactionID格式
4. **⚠️ 待验证**：需要在生产环境验证修复后的重复检查机制

### 修复内容
- mishi平台：TransactionID = "mishi_" + szOrderId
- dayuanren平台：TransactionID = "dayuanren_" + out_trade_num
- chongzhi平台：TransactionID = "chongzhi_" + orderID
- xianzhuanxia平台：TransactionID = "xianzhuanxia_" + orderID
- payc2平台：TransactionID = "payc2_" + uid
- external_api平台：TransactionID = "external_api_" + orderID

## 注意事项

1. **向后兼容**：修改不应影响现有正常工作的平台
2. **唯一性保证**：TransactionID必须在同一订单的不同回调中保持一致
3. **非空检查**：如果所有候选字段都为空，应该记录警告日志
4. **长度限制**：考虑数据库字段长度限制（建议最大255字符）