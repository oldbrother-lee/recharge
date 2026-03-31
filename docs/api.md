# 充值下单对接 API 文档（外部接入专用）

本文档仅面向外部系统对接我方充值下单能力。

- Base URL：`/api/v1`
- 生产环境域名由我方单独提供

## 1. 接入鉴权

外部接口统一使用 `API Key + 签名`。

### 1.1 请求头

- `X-API-Key`：我方分配的 `app_key`
- `X-Signature`：签名值（推荐与请求参数里的 `sign` 一致）
- `Content-Type: application/json`

### 1.2 业务必传签名字段

以下字段需参与业务请求（且参与签名）：

- `app_id`
- `timestamp`（秒级时间戳）
- `nonce`（随机串，建议 16~32 位）
- `sign`

其他字段若传入则参与签名，不传或为空则不参与（见第 2 节）。

### 1.3 服务端校验规则

- API Key 是否有效、是否启用
- 调用 IP 是否在白名单
- 频率限制（按 app_id 分钟级）
- 签名是否正确
- 时间戳是否过期（默认允许 ±300 秒）

## 2. 签名算法（含示例）

签名规则按当前服务实现：

1. 取所有业务参数，移除 `sign` / `signature`
2. 过滤空值参数（`nil` 或空字符串）
3. 按参数名 ASCII 升序排序
4. 拼接为 `k=v&k2=v2...`（**值为原始字符串，不做 URL 编码**）
5. 末尾追加 `&key=APP_SECRET`
6. 对最终字符串做 MD5，并转大写，得到签名

**注意**：拼签名字符串时，参数值用**原始值**，不要对 `notify_url` 等做 URL 编码（不要用 `%3A`、`%2F` 等）。与请求体 JSON 里该字段的取值保持一致即可。

### 2.1 下单签名示例

**重要：参与签名的参数 = 请求体中「实际传入且非空」的所有字段**（排除 `sign`/`signature`）。  
**传了哪些字段，签名字段就必须包含哪些**；未传或空值不参与。例如传了 `notify_url`，则签名字符串里必须包含 `notify_url=xxx`，否则服务端验签会失败。

**示例一：仅必填字段（未传 notify_url）**

请求体（不含 sign）：

```json
{
  "app_id": "demo_app_001",
  "mobile": "13800138000",
  "product_id": 101,
  "out_trade_num": "EXT202603090001",
  "amount": 100,
  "timestamp": 1760000000,
  "nonce": "abc123xyz"
}
```

拼接串（`APP_SECRET = demo_secret_123456`）：

```text
amount=100&app_id=demo_app_001&mobile=13800138000&nonce=abc123xyz&out_trade_num=EXT202603090001&product_id=101&timestamp=1760000000&key=demo_secret_123456
```

MD5 大写：`5DE5885CE664A2FA497F0263FD318036`

**示例二：含可选字段 notify_url（传了就必须参与签名）**

请求体（不含 sign）：

```json
{
  "app_id": "demo_app_001",
  "mobile": "13800138000",
  "product_id": 101,
  "out_trade_num": "EXT202603090001",
  "amount": 100,
  "notify_url": "https://partner.example.com/callback/recharge",
  "timestamp": 1760000000,
  "nonce": "abc123xyz"
}
```

拼接串（注意多了 `notify_url`，值与 body 中一致、不 URL 编码）：

```text
amount=100&app_id=demo_app_001&mobile=13800138000&nonce=abc123xyz&notify_url=https://partner.example.com/callback/recharge&out_trade_num=EXT202603090001&product_id=101&timestamp=1760000000&key=demo_secret_123456
```

然后对该串做 MD5 并转大写得到 `sign`。**只要 body 里传了 `notify_url`，算签名时必须把该键值对一起参与排序和拼接，否则验签不通过。**

## 3. 下单接口

### 3.1 创建订单

- 方法：`POST`
- 路径：`/external/order`
- 鉴权：需要 `X-API-Key` + `X-Signature`

**请求体字段说明：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| app_id | string | 是 | 应用ID，需与 API Key 对应 |
| mobile | string | 是 | 充值手机号 |
| product_id | int64 | 是 | 商品ID |
| out_trade_num | string | 是 | 外部交易号，全局唯一 |
| amount | number | 是 | 金额，参与签名；**实际扣款以商品价格为准** |
| timestamp | int64 | 是 | 秒级时间戳 |
| nonce | string | 是 | 随机串 |
| sign | string | 是 | 签名值 |
| notify_url | string | 否 | 状态回调地址 |
| isp | int | 否 | 运营商 |
| remark | string | 否 | 备注 |

请求体示例（含必填与常用可选字段）：

```json
{
  "app_id": "demo_app_001",
  "mobile": "13800138000",
  "product_id": 101,
  "out_trade_num": "EXT202603090001",
  "amount": 100,
  "timestamp": 1760000000,
  "nonce": "abc123xyz",
  "sign": "5DE5885CE664A2FA497F0263FD318036",
  "notify_url": "https://partner.example.com/callback/recharge",
  "isp": 1,
  "remark": "test"
}
```


成功响应示例：

```json
{
  "code": 200,
  "message": "Success",
  "data": {
    "order_number": "R202603090001",
    "out_trade_num": "EXT202603090001",
    "status": 1,
    "status_desc": "待支付",
    "amount": 100,
    "create_time": 1760000001
  },
  "timestamp": 1760000001
}
```

备注：

- `out_trade_num` 必须全局唯一。
- 若重复下单，系统会返回已存在订单信息（`code=200`，`message=Order already exists`）。
- **amount** 必传且参与签名；订单实际扣款金额以**商品（product_id）价格**为准。
### 3.2 查询订单

- 方法：`GET`
- 路径：`/external/order/query`
- 鉴权：需要 `X-API-Key + X-Signature`
- 查询参数：`app_id`、`timestamp`、`nonce`、`sign`，以及以下二选一：
  - `out_trade_num`
  - `order_number`

请求示例：

```text
/api/v1/external/order/query?app_id=demo_app_001&out_trade_num=EXT202603090001&timestamp=1760000300&nonce=qwe789&sign=64F9FCCDFA62733C1FD293349C414B12
```

成功响应与创建订单一致。

### 3.3 申请退款

- 方法：`POST`
- 路径：`/external/order/refund`
- 鉴权：需要 `X-API-Key` + `X-Signature`
- 权限：仅允许对**本 API Key 所属用户创建的订单**发起退款（订单归属校验），否则返回 `403 无权限操作该订单`。
- 可退款状态：仅**待充值**可申请退款。申请后订单变为**待退款审核**，不直接退款；需**管理员审核通过**后才执行退款，审核拒绝则订单恢复为待充值。**失败**订单由系统自动退款，不可重复申请；**成功**不允许退款。

请求体：

```json
{
  "app_id": "demo_app_001",
  "out_trade_num": "EXT202603090001",
  "reason": "user request",
  "timestamp": 1760000600,
  "nonce": "refund001",
  "sign": "签名值"
}
```

成功响应示例（申请成功、待审核）：

```json
{
  "code": 200,
  "message": "退款申请已提交，待管理员审核",
  "data": {
    "order_number": "R202603090001",
    "out_trade_num": "EXT202603090001",
    "amount": 100,
    "status": "pending_review"
  }
}
```

管理员审核通过后，订单状态变为已退款（`status=6`）；审核拒绝则订单恢复为待充值。对接方可通过**查询订单**接口轮询订单状态。

## 4. 我方主动回调（notify_url）

当订单状态变为终态时，我方会向你在下单时传入的 `notify_url` 发起 `POST` 回调。

- 回调时机：`成功` 或 `失败`
- 请求体格式：`application/json`

请求体字段：

- `app_id`
- `out_trade_num`
- `status`（成功=4，失败=5）
- `timestamp`
- `nonce`
- `sign`

回调示例：

```json
{
  "app_id": "demo_app_001",
  "out_trade_num": "EXT202603090001",
  "status": 4,
  "timestamp": 1760001200,
  "nonce": "cb_1760001200",
  "sign": "回调签名"
}
```

你方建议返回 HTTP 200，响应体可自定义（建议 JSON）。

## 5. 状态码说明

- `1`：待支付
- `2`：待充值
- `3`：充值中
- `4`：成功
- `5`：失败
- `6`：已退款
- `7`：已取消
- `11`：待退款审核（用户已申请退款，等管理员审核）

## 6. 常见错误码

- `400`：参数错误（如缺少必要字段）
- `401`：鉴权失败（API Key 或签名错误）
- `402`：余额不足
- `403`：IP 不在白名单
- `404`：订单不存在
- `429`：请求过于频繁
- `500`：服务内部错误

## 7. 对接建议

1. 先完成签名工具封装，再联调下单/查询
2. `out_trade_num` 请使用你方唯一业务单号
3. 所有请求建议设置超时与重试机制（幂等场景重试）
4. 回调接口请做幂等处理（按 `out_trade_num` 去重）
