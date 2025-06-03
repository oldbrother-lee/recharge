# 订单 API 文档

## 创建订单

### 请求
```http
POST /api/v1/order/create
Content-Type: application/json
Authorization: Bearer {token}
```

### 请求参数
```json
{
    "product_id": 123,           // 商品ID
    "customer_id": 456,          // 客户ID
    "quantity": 1,               // 购买数量
    "total_amount": 100.00,      // 总金额
    "payment_method": "alipay",  // 支付方式
    "remark": "备注信息"          // 备注
}
```

### 响应
```json
{
    "code": 0,
    "message": "success",
    "data": {
        "id": 789,              // 订单ID
        "order_number": "ORD202403150001",  // 订单编号
        "product_id": 123,
        "customer_id": 456,
        "quantity": 1,
        "total_amount": 100.00,
        "status": "pending_payment",  // 订单状态
        "payment_method": "alipay",
        "remark": "备注信息",
        "created_at": "2024-03-15T10:00:00Z",
        "updated_at": "2024-03-15T10:00:00Z"
    }
}
```

## 获取订单详情

### 请求
```http
GET /api/v1/order/:id
Authorization: Bearer {token}
```

### 响应
```json
{
    "code": 0,
    "message": "success",
    "data": {
        "id": 789,
        "order_number": "ORD202403150001",
        "product_id": 123,
        "customer_id": 456,
        "quantity": 1,
        "total_amount": 100.00,
        "status": "pending_payment",
        "payment_method": "alipay",
        "remark": "备注信息",
        "created_at": "2024-03-15T10:00:00Z",
        "updated_at": "2024-03-15T10:00:00Z"
    }
}
```

## 获取订单列表

### 请求
```http
GET /api/v1/order/list
Authorization: Bearer {token}
```

### 查询参数
- `customer_id`: 客户ID（可选）
- `status`: 订单状态（可选）
- `start_time`: 开始时间（可选）
- `end_time`: 结束时间（可选）
- `page`: 页码（可选，默认1）
- `page_size`: 每页数量（可选，默认10）

### 响应
```json
{
    "code": 0,
    "message": "success",
    "data": {
        "total": 100,
        "items": [
            {
                "id": 789,
                "order_number": "ORD202403150001",
                "product_id": 123,
                "customer_id": 456,
                "quantity": 1,
                "total_amount": 100.00,
                "status": "pending_payment",
                "payment_method": "alipay",
                "remark": "备注信息",
                "created_at": "2024-03-15T10:00:00Z",
                "updated_at": "2024-03-15T10:00:00Z"
            }
        ]
    }
}
```

## 更新订单状态

### 请求
```http
PUT /api/v1/order/:id/status
Content-Type: application/json
Authorization: Bearer {token}
```

### 请求参数
```json
{
    "status": "recharging",  // 新状态
    "remark": "状态更新备注"  // 备注（可选）
}
```

### 响应
```json
{
    "code": 0,
    "message": "success",
    "data": null
}
```

## 处理订单支付

### 请求
```http
POST /api/v1/order/:id/payment
Content-Type: application/json
Authorization: Bearer {token}
```

### 请求参数
```json
{
    "payment_id": "PAY123456",  // 支付ID
    "payment_time": "2024-03-15T10:00:00Z",  // 支付时间
    "remark": "支付备注"  // 备注（可选）
}
```

### 响应
```json
{
    "code": 0,
    "message": "success",
    "data": null
}
```

## 处理订单充值

### 请求
```http
POST /api/v1/order/:id/recharge
Content-Type: application/json
Authorization: Bearer {token}
```

### 请求参数
```json
{
    "api_id": 123,  // API ID
    "api_order_number": "API123456",  // API 订单号
    "api_trade_num": "TRADE123456",  // API 交易号
    "remark": "充值备注"  // 备注（可选）
}
```

### 响应
```json
{
    "code": 0,
    "message": "success",
    "data": null
}
```

## 处理订单成功

### 请求
```http
POST /api/v1/order/:id/success
Content-Type: application/json
Authorization: Bearer {token}
```

### 请求参数
```json
{
    "remark": "成功备注"  // 备注（可选）
}
```

### 响应
```json
{
    "code": 0,
    "message": "success",
    "data": null
}
```

## 处理订单失败

### 请求
```http
POST /api/v1/order/:id/fail
Content-Type: application/json
Authorization: Bearer {token}
```

### 请求参数
```json
{
    "remark": "失败原因"  // 失败原因
}
```

### 响应
```json
{
    "code": 0,
    "message": "success",
    "data": null
}
```

## 处理订单退款

### 请求
```http
POST /api/v1/order/:id/refund
Content-Type: application/json
Authorization: Bearer {token}
```

### 请求参数
```json
{
    "refund_amount": 100.00,  // 退款金额
    "refund_reason": "退款原因",  // 退款原因
    "remark": "退款备注"  // 备注（可选）
}
```

### 响应
```json
{
    "code": 0,
    "message": "success",
    "data": null
}
```

## 处理订单取消

### 请求
```http
POST /api/v1/order/:id/cancel
Content-Type: application/json
Authorization: Bearer {token}
```

### 请求参数
```json
{
    "cancel_reason": "取消原因",  // 取消原因
    "remark": "取消备注"  // 备注（可选）
}
```

### 响应
```json
{
    "code": 0,
    "message": "success",
    "data": null
}
```

## 处理订单拆单

### 请求
```http
POST /api/v1/order/:id/split
Content-Type: application/json
Authorization: Bearer {token}
```

### 请求参数
```json
{
    "split_orders": [  // 拆单后的订单列表
        {
            "quantity": 1,  // 数量
            "amount": 100.00  // 金额
        }
    ],
    "remark": "拆单备注"  // 备注（可选）
}
```

### 响应
```json
{
    "code": 0,
    "message": "success",
    "data": {
        "split_order_ids": [789, 790]  // 拆单后的订单ID列表
    }
}
```

## 处理订单部分充值

### 请求
```http
POST /api/v1/order/:id/partial
Content-Type: application/json
Authorization: Bearer {token}
```

### 请求参数
```json
{
    "recharged_quantity": 1,  // 已充值数量
    "remaining_quantity": 1,  // 剩余数量
    "remark": "部分充值备注"  // 备注（可选）
}
```

### 响应
```json
{
    "code": 0,
    "message": "success",
    "data": null
}
```

## 错误码说明

| 错误码 | 说明 |
|--------|------|
| 0 | 成功 |
| 1001 | 参数错误 |
| 1002 | 订单不存在 |
| 1003 | 订单状态错误 |
| 1004 | 权限不足 |
| 1005 | 系统错误 |

## 订单状态说明

| 状态 | 说明 |
|------|------|
| pending_payment | 待支付 |
| pending_recharge | 待充值 |
| recharging | 充值中 |
| success | 充值成功 |
| failed | 充值失败 |
| refunded | 已退款 |
| cancelled | 已取消 |
| partial | 部分充值 |
| split | 已拆单 | 