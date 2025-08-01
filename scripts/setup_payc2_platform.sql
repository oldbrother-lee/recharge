-- payc2充值平台配置脚本
-- 使用前请根据实际情况修改相关参数

-- 1. 创建平台记录
INSERT INTO platforms (name, code, status, api_url, created_at, updated_at) 
VALUES ('payc2充值平台', 'payc2', 1, 'https://api.payc2.com', NOW(), NOW());

-- 获取刚创建的平台ID（请根据实际情况替换）
SET @platform_id = LAST_INSERT_ID();

-- 2. 创建平台账号
INSERT INTO platform_accounts (
    platform_id, 
    account_name, 
    app_key, 
    app_secret, 
    api_url,
    balance, 
    status, 
    type,
    description,
    daily_limit,
    monthly_limit,
    priority,
    created_at, 
    updated_at
) VALUES (
    @platform_id, 
    'payc2主账号', 
    '1000', -- 商户号，请替换为实际值
    'ad6360f2d7de4b1e915a3035437c4743', -- 商户秘钥，请替换为实际值
    'https://api.payc2.com/apis/wof/order/create_phone', -- API地址
    0.00, 
    1, -- 启用状态
    'recharge', -- 账号类型
    'payc2话费充值账号',
    100000.00, -- 日限额
    3000000.00, -- 月限额
    1, -- 优先级
    NOW(), 
    NOW()
);

-- 获取刚创建的账号ID
SET @account_id = LAST_INSERT_ID();

-- 3. 创建平台API配置
INSERT INTO platform_apis (
    platform_id,
    account_id,
    name,
    code,
    url,
    method,
    content_type,
    timeout,
    callback_url,
    status,
    retry_count,
    retry_delay,
    description,
    created_at,
    updated_at
) VALUES (
    @platform_id,
    @account_id,
    'payc2话费充值接口',
    'payc2_phone_recharge',
    'https://api.payc2.com/apis/wof/order/create_phone',
    'POST',
    'application/x-www-form-urlencoded',
    30, -- 超时时间（秒）
    'http://your-domain.com/callback/payc2', -- 回调地址，请替换为实际值
    1, -- 启用状态
    3, -- 重试次数
    3, -- 重试间隔（秒）
    'payc2平台话费充值API接口',
    NOW(),
    NOW()
);

-- 获取刚创建的API ID
SET @api_id = LAST_INSERT_ID();

-- 4. 创建API参数配置（可选）
INSERT INTO platform_api_params (
    api_id,
    param_name,
    param_value,
    param_type,
    is_required,
    description,
    created_at,
    updated_at
) VALUES 
(@api_id, 'timeoutSecond', '1800', 'int', 0, '订单超时时间（秒）', NOW(), NOW()),
(@api_id, 'version', 'v1.240318', 'string', 0, 'API版本号', NOW(), NOW());

-- 5. 查询创建结果
SELECT 
    p.id as platform_id,
    p.name as platform_name,
    p.code as platform_code,
    pa.id as account_id,
    pa.account_name,
    pa.app_key,
    api.id as api_id,
    api.name as api_name,
    api.url as api_url
FROM platforms p
LEFT JOIN platform_accounts pa ON p.id = pa.platform_id
LEFT JOIN platform_apis api ON p.id = api.platform_id
WHERE p.code = 'payc2';

-- 使用说明：
-- 1. 执行此脚本前，请确保数据库表结构已创建
-- 2. 请根据实际情况修改商户号(app_key)和商户秘钥(app_secret)
-- 3. 请修改回调地址(callback_url)为您的实际域名
-- 4. 可根据需要调整限额、优先级等参数
-- 5. 执行完成后，可通过管理后台或API接口进一步配置