# 退款金额一致性修复

## 问题描述

在充值服务的回调处理中发现了一个严重的金额不一致问题：

### 问题现象
- **扣款时**：使用 `order.Price` 字段作为扣款金额
- **退款时**：在充值失败回调中使用 `order.TotalPrice` 字段作为退款金额
- **结果**：当 `order.Price` 和 `order.TotalPrice` 不相等时，会导致多退或少退钱的问题

### 具体代码位置

**扣款代码** (`internal/service/recharge/base.go:62`)：
```go
// 扣除平台账号余额
if err := s.balanceService.DeductBalance(ctx, order.PlatformAccountID, order.Price, order.ID, "订单充值扣除"); err != nil {
    // ...
}
```

**退款代码** (`internal/service/recharge/base.go:145` - 修复前)：
```go
// 如果充值失败，退还余额
if callbackData.Status == "failed" {
    if err := s.balanceService.RefundBalance(ctx, order.PlatformAccountID, order.TotalPrice, order.ID, "充值失败退还"); err != nil {
        // ...
    }
}
```

### 风险评估

假设一个订单：
- `order.Price = 10.00` (实际扣款金额)
- `order.TotalPrice = 15.00` (订单总价)

在充值失败时：
- 扣款：10.00 元
- 退款：15.00 元
- **损失：5.00 元**

## 修复方案

### 修复原则
保持扣款和退款使用相同的金额字段，确保金额一致性。

### 修复内容

将充值失败回调中的退款金额从 `order.TotalPrice` 改为 `order.Price`：

```go
// 修复后的代码
if callbackData.Status == "failed" {
    if err := s.balanceService.RefundBalance(ctx, order.PlatformAccountID, order.Price, order.ID, "充值失败退还"); err != nil {
        logger.Error("退还余额失败",
            "error", err,
            "account_id", order.PlatformAccountID,
            "amount", order.Price) // 也修复了日志中的金额字段
        // ...
    }
}
```

### 修复文件
- `internal/service/recharge/base.go` (第145行和第148行)

## 一致性验证

### 扣款和退款场景对比

| 场景 | 扣款字段 | 退款字段 | 状态 |
|------|----------|----------|------|
| ProcessRechargeTask - API获取失败 | `order.Price` | `order.Price` | ✅ 一致 |
| ProcessRechargeTask - 订单提交失败 | `order.Price` | `order.Price` | ✅ 一致 |
| HandleCallback - 充值失败 (修复前) | `order.Price` | `order.TotalPrice` | ❌ 不一致 |
| HandleCallback - 充值失败 (修复后) | `order.Price` | `order.Price` | ✅ 一致 |

### 字段含义说明

根据 `internal/model/order.go` 中的定义：

```go
type Order struct {
    // ...
    Denom      float64 `json:"denom" gorm:"type:decimal(10,2);comment:面值"`
    TotalPrice float64 `json:"total_price" gorm:"type:decimal(10,2);comment:总价"`
    Price      float64 `json:"price" gorm:"type:decimal(10,2);comment:单价"`
    // ...
}
```

- `Price`: 单价，实际扣款金额
- `TotalPrice`: 总价，可能包含手续费、优惠等
- `Denom`: 面值

在充值业务中，实际扣款使用的是 `Price` 字段，因此退款也应该使用 `Price` 字段。

## 测试验证

创建了完整的测试用例 `test/refund_consistency_test.go` 来验证修复效果：

### 测试用例

1. **TestRefundConsistency**: 验证退款使用正确的字段
2. **TestDeductRefundAmountConsistency**: 验证扣款和退款金额一致性
3. **TestPriceVsTotalPriceDifference**: 说明Price和TotalPrice差异的影响

### 运行测试

```bash
cd /Users/lee/GolandProjects/recharge-go
go test ./test -v -run TestRefund
```

## 影响范围

### 修复影响
- ✅ 解决了充值失败退款金额不一致的问题
- ✅ 防止了多退款导致的资金损失
- ✅ 保持了所有退款场景的金额一致性

### 兼容性
- ✅ 不影响现有的扣款逻辑
- ✅ 不影响其他退款场景
- ✅ 向后兼容，不需要数据迁移

## 建议

### 代码审查建议
1. 在所有涉及金额的操作中，明确使用哪个字段
2. 建立扣款和退款金额一致性的代码规范
3. 添加单元测试覆盖所有金额操作场景

### 监控建议
1. 添加扣款和退款金额不匹配的监控告警
2. 定期审计余额变动日志的一致性
3. 监控异常的退款金额（如明显超出正常范围的退款）

## 总结

这个修复解决了一个可能导致资金损失的严重问题。通过确保扣款和退款使用相同的金额字段（`order.Price`），我们消除了金额不一致的风险，保护了平台的资金安全。