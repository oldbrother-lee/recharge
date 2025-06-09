# 外部API充值渠道配置指南

本文档说明如何在充值平台中新增一个使用外部API创建订单的充值渠道。

## 概述

外部API充值渠道允许系统通过调用本系统的外部API来创建充值订单，实现系统间的充值功能。这种方式适用于多系统部署的场景，其中一个系统作为充值渠道为另一个系统提供充值服务。

## 配置步骤

### 1. 创建平台记录

首先需要在数据库中创建一个新的平台记录：

```sql
INSERT INTO platforms (name, code, status, api_url, created_at, updated_at) 
VALUES ('外部API充值平台', 'external_api', 1, 'http://target-system.com', NOW(), NOW());
```

### 2. 创建平台账号

为新平台创建账号，配置API密钥：

```sql
INSERT INTO platform_accounts (
    platform_id, 
    account_name, 
    app_key, 
    app_secret, 
    api_url,
    balance, 
    status, 
    created_at, 
    updated_at
) VALUES (
    1, -- 替换为实际的platform_id
    'external_api_account_001',
    'your_app_id_here', -- 目标系统分配的APP ID
    'your_app_secret_here', -- 目标系统分配的APP Secret
    'http://target-system.com', -- 目标系统的API地址
    10000.00, -- 初始余额
    1, -- 启用状态
    NOW(),
    NOW()
);
```

### 3. 创建平台API配置

配置平台的API接口信息：

```sql
INSERT INTO platform_apis (
    platform_id,
    account_id,
    name,
    code,
    url,
    method,
    status,
    created_at,
    updated_at
) VALUES (
    1, -- 替换为实际的platform_id
    1, -- 替换为实际的account_id
    '外部API充值接口',
    'external_api_recharge',
    '/api/external/order', -- 相对路径，会与account的api_url拼接
    'POST',
    1,
    NOW(),
    NOW()
);
```

### 4. 创建API参数配置

为API接口配置参数：

```sql
INSERT INTO platform_api_params (
    api_id,
    product_id,
    status,
    created_at,
    updated_at
) VALUES (
    1, -- 替换为实际的api_id
    'MOBILE_10', -- 目标系统的商品ID
    1,
    NOW(),
    NOW()
);
```

### 5. 配置商品与API的关联关系

将本系统的商品与外部API关联：

```sql
INSERT INTO product_api_relations (
    product_id,
    api_id,
    priority,
    status,
    created_at,
    updated_at
) VALUES (
    1, -- 本系统的商品ID
    1, -- 外部API的ID
    1, -- 优先级
    1, -- 启用状态
    NOW(),
    NOW()
);
```

## 使用Go代码配置

也可以通过Go代码来配置：

```go
package main

import (
    "context"
    "recharge-go/internal/model"
    "recharge-go/internal/service"
)

func setupExternalAPIChannel(platformService *service.PlatformService) error {
    ctx := context.Background()
    
    // 1. 创建平台
    platformReq := &model.PlatformCreateRequest{
        Name:   "外部API充值平台",
        Code:   "external_api",
        Status: 1,
        ApiURL: "http://target-system.com",
    }
    
    if err := platformService.CreatePlatform(platformReq); err != nil {
        return err
    }
    
    // 2. 创建平台账号
    accountReq := &model.PlatformAccountCreateRequest{
        PlatformID:  1, // 替换为实际的platform_id
        AccountName: "external_api_account_001",
        AppKey:      "your_app_id_here",
        AppSecret:   "your_app_secret_here",
        ApiURL:      "http://target-system.com",
        Balance:     10000.00,
        Status:      1,
    }
    
    if err := platformService.CreatePlatformAccount(accountReq); err != nil {
        return err
    }
    
    return nil
}
```

## 配置说明

### 平台配置参数

- **name**: 平台显示名称
- **code**: 平台代码，必须为 `external_api`
- **status**: 平台状态，1为启用，0为禁用
- **api_url**: 目标系统的基础API地址

### 账号配置参数

- **app_key**: 目标系统分配的APP ID
- **app_secret**: 目标系统分配的APP Secret
- **api_url**: 目标系统的API地址（可以与平台的api_url相同）
- **balance**: 账号余额

### API配置参数

- **url**: API接口的相对路径
- **method**: HTTP方法，通常为POST
- **product_id**: 目标系统的商品ID

## 工作流程

1. **订单创建**: 当有充值订单需要处理时，系统会根据商品配置选择外部API渠道
2. **API调用**: 系统使用配置的APP ID和Secret调用目标系统的外部API
3. **签名验证**: 使用MD5签名确保请求的安全性
4. **状态同步**: 通过回调或主动查询同步订单状态
5. **余额管理**: 自动管理平台账号余额

## 安全考虑

1. **API密钥管理**: 确保APP Secret的安全存储，不要在代码中硬编码
2. **签名验证**: 所有API请求都使用MD5签名验证
3. **HTTPS通信**: 生产环境建议使用HTTPS协议
4. **访问控制**: 限制API访问的IP地址范围

## 监控和日志

系统会自动记录：
- API调用日志
- 订单状态变更日志
- 余额变动日志
- 错误和异常日志

## 故障排查

常见问题及解决方案：

1. **签名验证失败**: 检查APP Secret配置是否正确
2. **API调用超时**: 检查目标系统的网络连接
3. **余额不足**: 检查平台账号余额是否充足
4. **商品不存在**: 检查目标系统的商品ID配置

## 扩展性

该实现支持：
- 多个外部API系统接入
- 动态配置管理
- 负载均衡和故障转移
- 实时监控和告警