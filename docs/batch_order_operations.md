# 批量订单操作接口文档

本文档描述了新增的三个批量订单操作接口的使用方法。

## 接口列表

### 1. 批量删除订单

**接口地址：** `POST /api/v1/order/batch-delete`

**请求参数：**
```json
{
  "order_ids": [123, 456, 789]
}
```

**响应示例：**
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "success_count": 2,
    "failed_count": 1,
    "total_count": 3,
    "errors": [
      "订单order_id_3删除失败: 订单不存在"
    ]
  }
}
```

### 2. 批量设置订单成功

**接口地址：** `POST /api/v1/order/batch-success`

**请求参数：**
```json
{
  "order_ids": [123, 456, 789]
}
```

**响应示例：**
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "success_count": 3,
    "failed_count": 0,
    "total_count": 3
  }
}
```

### 3. 批量设置订单失败

**接口地址：** `POST /api/v1/order/batch-fail`

**请求参数：**
```json
{
  "order_ids": [123, 456, 789],
  "remark": "系统维护，订单处理失败"
}
```

**响应示例：**
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "success_count": 2,
    "failed_count": 1,
    "total_count": 3,
    "errors": [
      "订单456设置失败失败: 订单状态不允许修改"
    ]
  }
}
```

## 注意事项

1. **权限要求：** 这些接口需要相应的用户权限，请确保调用者具有订单管理权限。

2. **参数验证：**
   - `order_ids` 字段为必填项，不能为空数组
   - 批量设置失败接口的 `remark` 字段为必填项

3. **错误处理：**
   - 接口采用部分成功模式，即使部分订单操作失败，成功的订单仍会被处理
   - 失败的订单会在响应的 `errors` 字段中详细说明失败原因

4. **性能考虑：**
   - 建议单次批量操作的订单数量不超过100个
   - 大量订单操作可能会影响系统性能，请合理控制批量大小

5. **日志记录：**
   - 所有批量操作都会记录详细的操作日志
   - 失败的操作会记录错误信息，便于问题排查

## 前端集成

前端代码已经实现了对这些接口的调用，位于 `web/src/views/order/components/OrderList.vue` 文件中：

- `confirmBatchDelete()` - 调用批量删除接口
- `confirmBatchSuccess()` - 调用批量设置成功接口  
- `confirmBatchFail()` - 调用批量设置失败接口

前端会自动处理响应结果，显示操作成功数量和失败信息。