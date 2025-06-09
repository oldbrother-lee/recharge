-- 外部API充值渠道配置脚本
-- 使用前请根据实际情况修改相关参数

-- 1. 创建平台记录
INSERT INTO platforms (name, code, status, api_url, created_at, updated_at) 
VALUES ('外部API充值平台', 'external_api', 1, 'http://target-system.com', NOW(), NOW());

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
    created_at, 
    updated_at
) VALUES (
    @platform_id,
    'external_api_account_001',
    'your_app_id_here', -- 请替换为实际的APP ID
    'your_app_secret_here', -- 请替换为实际的APP Secret
    'http://target-system.com', -- 请替换为实际的API地址
    10000.00, -- 初始余额，请根据需要调整
    1, -- 启用状态
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
    status,
    created_at,
    updated_at
) VALUES (
    @platform_id,
    @account_id,
    '外部API充值接口',
    'external_api_recharge',
    '/api/external/order', -- API接口路径
    'POST',
    1,
    NOW(),
    NOW()
);

-- 获取刚创建的API ID
SET @api_id = LAST_INSERT_ID();

-- 4. 创建API参数配置（示例：手机充值10元）
INSERT INTO platform_api_params (
    api_id,
    product_id,
    status,
    created_at,
    updated_at
) VALUES (
    @api_id,
    'MOBILE_10', -- 请替换为目标系统的实际商品ID
    1,
    NOW(),
    NOW()
);

-- 5. 配置商品与API的关联关系
-- 注意：需要根据实际的商品ID进行配置
INSERT INTO product_api_relations (
    product_id,
    api_id,
    priority,
    status,
    created_at,
    updated_at
) VALUES (
    1, -- 请替换为本系统的实际商品ID
    @api_id,
    1, -- 优先级，数字越小优先级越高
    1, -- 启用状态
    NOW(),
    NOW()
);

-- 查询配置结果
SELECT 
    p.id as platform_id,
    p.name as platform_name,
    p.code as platform_code,
    pa.id as account_id,
    pa.account_name,
    pa.app_key,
    pa.balance,
    api.id as api_id,
    api.name as api_name,
    api.url as api_url
FROM platforms p
JOIN platform_accounts pa ON p.id = pa.platform_id
JOIN platform_apis api ON pa.id = api.account_id
WHERE p.code = 'external_api'
ORDER BY p.id DESC
LIMIT 1;

-- 显示配置完成信息
SELECT 
    '外部API充值渠道配置完成' as message,
    @platform_id as platform_id,
    @account_id as account_id,
    @api_id as api_id;

-- 注意事项：
-- 1. 请根据实际情况修改 app_key、app_secret、api_url 等参数
-- 2. 请确保 product_id 与目标系统的商品ID匹配
-- 3. 请根据实际商品配置 product_api_relations 表
-- 4. 建议在测试环境先验证配置的正确性