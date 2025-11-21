# 最新 API 对接文档（统一版）

本文件汇总项目当前对外与内部相关的 HTTP API，涵盖外部订单接口、回调接口、公开与管理员手机查询接口，以及鉴权规则与统一响应结构。本文面向对接方与项目开发维护者，路径与参数均以当前代码实现为准。

- 基础前缀 Base URL: /api/v1
- 所有路径均在上述前缀之下，例如 POST /api/v1/external/order

## 一、鉴权与安全

- 外部 API 统一采用 API Key + 签名校验的方式：
  - 请求头：
    - X-API-Key: 分配的 App Key
    - X-Signature: 签名值（按签名算法生成）
  - 业务参数需包含：app_id, timestamp, nonce, sign（sign 与 X-Signature 等价，服务端同时支持以便兼容）
  - 服务端校验内容：
    - API Key 状态与 IP 白名单校验
    - 频率限制（按 app_id 分钟级限流）
    - 签名验证（移除 sign 后对参数进行字典序拼接 + app_secret + MD5 大写生成）

- 管理后台接口使用系统内部认证中间件（JWT），需在请求头携带有效的 Authorization: Bearer <token>

- 超时与重试：服务端默认请求超时时间为 30 秒（可在配置文件中调整）。

## 二、统一响应结构

无论成功或失败，均返回统一结构：

- 成功：
  - code: 200
  - message: "success" 或业务提示
  - data: 对应接口的数据对象
  - timestamp: 服务端时间戳（秒）

- 失败：
  - code: HTTP 状态码（如 400/401/404/429/500）
  - message: 错误说明
  - timestamp: 服务端时间戳（秒）

示例（创建订单失败）：
{
  "code": 400,
  "message": "请求参数错误",
  "timestamp": 1700000000
}

## 三、订单状态码（主要值）

- 0/未知: 未知状态（仅作为兜底显示）
- 1/待支付: 待支付
- 2/待充值: 待充值
- 3/充值中: 充值中
- 4/成功: 充值成功
- 5/失败: 充值失败
- 6/已退款: 已退款
- 7/已取消: 已取消

说明：项目还包含部分扩展状态（如部分充值、已拆单、处理中等），查询接口会返回具体数值与对应中文含义。

## 四、外部订单 API

说明：以下接口均在 /api/v1 前缀下，并需通过外部认证中间件（X-API-Key + X-Signature）。

1) 创建订单
- 方法: POST
- 路径: /external/order
- 请求体（JSON）字段：
  - app_id: string, required
  - mobile: string, required
  - product_id: int64, required
  - out_trade_num: string, required（外部交易号，唯一）
  - amount: number, required
  - biz_type: string, optional
  - notify_url: string, optional（订单状态变更回调地址）
  - param1/param2/param3: string, optional（扩展）
  - customer_id: int64, optional
  - isp: int, optional（运营商）
  - remark: string, optional
  - timestamp: int64, required（秒）
  - nonce: string, required
  - sign: string, required
- 响应（成功）：
{
  "code": 200,
  "message": "success",
  "data": {
    "order_number": "INTERNAL_ORDER_001",
    "out_trade_num": "EXT_001",
    "status": 1,
    "status_desc": "待支付",
    "amount": 100.0,
    "create_time": 1700000000
  },
  "timestamp": 1700000000
}

2) 查询订单
- 方法: GET
- 路径: /external/order/query
- 查询参数：
  - out_trade_num: string，与 order_number 至少传一个
  - order_number: string，与 out_trade_num 至少传一个
- 响应（成功）：与创建返回结构一致
- 失败：
  - 404: Order not found
  - 400: 缺少查询参数

3) 申请退款
- 方法: POST
- 路径: /external/order/refund
- 说明：需要传入与业务约定的退款参数（具体字段以退款控制器实现为准），同样走外部认证与签名校验。

## 五、外部回调 API

- 方法: POST
- 路径: /external/callback/order
- 鉴权：不走外部认证中间件，但会进行签名验证（要求对方按约定算法生成 sign）。
- 行为：
  - 解析回调请求体，记录日志
  - 根据外部交易号或内部订单号查询并更新订单状态
  - 对失败场景提供重试机制（队列）
  - 根据 notify_url 进行平台级通知（如配置）
- 响应：成功返回统一成功结构；失败返回统一错误结构

## 六、手机查询 API

说明：分为公开接口与管理员接口，两者路径不同，管理员接口需携带系统 JWT 授权。

1) 管理员接口（需认证）
- 基础路径: /phone
- 方法/路径：
  - POST /phone/balance       （查询余额/套餐等）
  - POST /phone/payment-records（查询缴费记录）
- 说明：仅管理员或具备相应权限的用户可访问。

2) 公开接口（无需认证）
- 基础路径: /public/phone
- 方法/路径：
  - POST /public/phone/balance
  - POST /public/phone/payment-records
- 说明：适用于向外提供的公共查询能力（如需对外开放）。

## 七、错误码与示例

- 400 Bad Request：请求参数错误、签名错误等
- 401 Unauthorized：API Key 无效或缺失
- 404 Not Found：资源不存在（如订单未找到）
- 429 Too Many Requests：触发限流
- 500 Internal Server Error：服务端异常

## 八、签名算法摘要（对接方须知）

- 参与签名的所有业务参数需去除空值，并按字典序（key 升序）拼接为 key=value 的形式，以 & 连接。
- 在上述拼接字符串末尾追加 &app_secret=YOUR_SECRET。
- 对最终字符串进行 MD5，取大写作为签名值。
- 将签名同时提供于：
  - 请求体字段：sign
  - 请求头字段：X-Signature

## 九、参考

- 外部订单控制器：负责创建与查询的响应结构与错误处理，含状态中文描述与统一错误返回。
- 外部认证中间件：X-API-Key 与签名校验、IP 白名单、限流与签名解析逻辑。
- 路由：统一前缀 /api/v1，外部订单路由组 /external/order，回调路由组 /external/callback；手机查询路由组 /phone 与 /public/phone。