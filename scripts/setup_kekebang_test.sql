-- 创建kekebang测试平台和账号

-- 1. 创建平台记录
INSERT INTO platforms (name, code, status, api_url, created_at, updated_at) 
VALUES ('客帮帮充值平台', 'kekebang', 1, 'http://api.kekebang.com', NOW(), NOW())
ON DUPLICATE KEY UPDATE 
    name = VALUES(name),
    status = VALUES(status),
    api_url = VALUES(api_url),
    updated_at = NOW();

-- 获取平台ID
SET @platform_id = (SELECT id FROM platforms WHERE code = 'kekebang');

-- 3. 创建测试订单记录
INSERT INTO orders (
    order_number,
    user_id,
    platform_id,
    platform_account_id,
    phone_number,
    amount,
    status,
    created_at,
    updated_at
) VALUES (
    'P20250804170530tgeCit', -- 测试订单号
    1, -- 用户ID
    @platform_id, -- 平台ID
    (SELECT id FROM platform_accounts WHERE account_name = 'kekebang_test'), -- 平台账号ID
    '13800138000', -- 手机号
    100.00, -- 充值金额
    2, -- 状态：待充值
    NOW(),
    NOW()
) ON DUPLICATE KEY UPDATE 
    status = VALUES(status),
    updated_at = NOW();

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
    'kekebang_test', -- 测试账号名
    'test_app_key', -- 测试APP Key
    'test_app_secret', -- 测试APP Secret
    'http://api.kekebang.com', -- API地址
    10000.00, -- 初始余额
    1, -- 启用状态
    'recharge', -- 账号类型
    'kekebang测试账号',
    100000.00, -- 日限额
    3000000.00, -- 月限额
    1, -- 优先级
    NOW(),
    NOW()
) ON DUPLICATE KEY UPDATE 
    app_key = VALUES(app_key),
    app_secret = VALUES(app_secret),
    status = VALUES(status),
    updated_at = NOW();

SELECT 'kekebang测试平台和账号创建完成' as result;