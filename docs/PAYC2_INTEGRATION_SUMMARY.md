# Payc2充值平台集成总结

## 概述

本文档总结了payc2充值平台的完整集成过程，包括代码实现、配置文件和使用说明。

## 集成内容

### 1. 核心文件

#### 签名处理器
- **文件**: `internal/signature/payc2.go`
- **功能**: 实现payc2平台的MD5签名算法
- **特性**:
  - ASCII码升序参数排序
  - 支持空值参数签名
  - 自动运营商识别
  - 回调签名验证

#### 平台处理器
- **文件**: `internal/service/recharge/payc2.go`
- **功能**: 实现Platform接口，处理充值业务逻辑
- **方法**:
  - `SubmitOrder`: 提交充值订单
  - `QueryOrderStatus`: 查询订单状态
  - `ParseCallbackData`: 解析回调数据
  - `QueryBalance`: 查询账户余额

#### 平台注册
- **文件**: `internal/service/recharge/manager.go`
- **修改**: 在平台管理器中注册payc2平台
- **变更**:
  - 添加平台类型映射
  - 添加平台实例创建逻辑

### 2. 配置文件

#### 数据库配置脚本
- **文件**: `scripts/setup_payc2_platform.sql`
- **功能**: 创建平台、账号和API配置记录
- **包含**:
  - 平台基础信息
  - 账号配置（商户号、密钥等）
  - API接口配置
  - 参数配置

#### 集成文档
- **文件**: `docs/payc2_platform_setup.md`
- **内容**: 详细的集成指南和使用说明
- **包含**:
  - 配置步骤
  - API接口说明
  - 签名算法详解
  - 使用示例
  - 故障排查

### 3. 测试文件

#### 单元测试
- **文件**: `test/payc2_test.go`
- **覆盖**:
  - 签名生成测试
  - 运营商识别测试
  - 平台创建测试
  - 请求参数构建测试
  - 回调数据解析测试
  - 性能基准测试

## 技术特性

### 签名算法
- **类型**: MD5
- **排序**: ASCII码升序
- **格式**: `key1=value1&key2=value2&key=密钥`
- **特点**: 空值参与签名，未传递参数不参与签名

### 运营商识别
- **电信(DX)**: 133, 149, 153, 173, 177, 180, 181, 189, 199
- **联通(LT)**: 130, 131, 132, 155, 156, 166, 175, 176, 185, 186
- **移动(YD)**: 其他号段（默认）

### API接口
- **地址**: `/apis/wof/order/create_phone`
- **方法**: POST
- **格式**: application/x-www-form-urlencoded
- **超时**: 30秒（可配置）

## 配置参数

### 必需参数
- `merch`: 商户号
- `amount`: 充值金额（整数）
- `notifyUrl`: 回调地址
- `phone`: 手机号
- `telco`: 运营商代码
- `sign`: 签名

### 可选参数
- `orderNo`: 商户订单号
- `timeoutSecond`: 超时时间（默认1800秒）

## 使用流程

### 1. 数据库配置
```bash
# 执行配置脚本
mysql -u username -p database_name < scripts/setup_payc2_platform.sql
```

### 2. 修改配置
- 更新商户号和密钥
- 设置回调地址
- 调整限额和优先级

### 3. 系统集成
```go
// 创建平台管理器
manager := recharge.NewManager(db)

// 加载平台
if err := manager.LoadPlatforms(); err != nil {
    log.Fatal(err)
}

// 提交订单
err := manager.SubmitOrder(ctx, order, api, apiParam)
```

### 4. 回调处理
```go
// 解析回调数据
callbackData, err := platform.ParseCallbackData(requestBody)
if err != nil {
    // 处理解析错误
}

// 更新订单状态
if callbackData.Status == model.OrderStatusSuccess {
    // 充值成功处理
}
```

## 测试验证

### 运行测试
```bash
# 运行单元测试
go test ./test/payc2_test.go -v

# 运行性能测试
go test ./test/payc2_test.go -bench=.
```

### 验证签名
```bash
# 使用示例参数验证签名生成
# 期望结果: d2c2bb545d50b48403ad1be2b03dd82a
```

## 安全考虑

### 签名安全
- 商户密钥严格保密
- 所有请求必须签名验证
- 回调数据签名验证

### 网络安全
- 使用HTTPS传输
- 设置合理的超时时间
- 实现重试机制

### 数据安全
- 敏感信息加密存储
- 日志脱敏处理
- 访问权限控制

## 监控和日志

### 关键日志
- 订单提交日志
- 签名生成日志
- 回调处理日志
- 错误异常日志

### 监控指标
- 订单成功率
- 响应时间
- 错误率
- 回调及时性

## 故障排查

### 常见问题
1. **签名验证失败**
   - 检查参数排序
   - 验证密钥正确性
   - 确认参数完整性

2. **订单提交失败**
   - 检查网络连接
   - 验证API地址
   - 确认参数格式

3. **回调处理异常**
   - 检查回调地址
   - 验证签名逻辑
   - 确认数据格式

### 调试方法
- 查看详细日志
- 使用测试工具
- 对比API文档
- 联系技术支持

## 版本信息

- **API版本**: v1.240318
- **签名版本**: v1.240228
- **集成日期**: 2024年
- **维护状态**: 活跃维护

## 后续优化

### 功能增强
- 支持批量充值
- 增加余额查询
- 优化错误处理
- 完善监控告警

### 性能优化
- 签名算法优化
- 连接池管理
- 缓存机制
- 异步处理

## 联系方式

如有技术问题，请联系：
- 开发团队
- 技术文档
- 在线支持