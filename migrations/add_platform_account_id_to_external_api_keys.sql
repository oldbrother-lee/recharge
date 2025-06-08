-- 为external_api_keys表添加platform_account_id字段
ALTER TABLE external_api_keys ADD COLUMN platform_account_id BIGINT NOT NULL DEFAULT 0 COMMENT '平台账号ID';

-- 添加索引
CREATE INDEX idx_external_api_keys_platform_account_id ON external_api_keys(platform_account_id);

-- 如果需要外键约束，可以取消注释下面的语句
-- ALTER TABLE external_api_keys ADD CONSTRAINT fk_external_api_keys_platform_account_id 
--   FOREIGN KEY (platform_account_id) REFERENCES platform_accounts(id);