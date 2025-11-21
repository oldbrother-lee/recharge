-- 为 pull_sources 表添加 bind_user_id 字段，用于账号绑定
ALTER TABLE pull_sources 
ADD COLUMN bind_user_id BIGINT NULL COMMENT '绑定的本地用户ID' AFTER account_password;

-- 添加索引以提高查询性能
ALTER TABLE pull_sources 
ADD INDEX idx_bind_user_id (bind_user_id);

-- 添加外键约束（可选，确保数据完整性）
ALTER TABLE pull_sources 
ADD CONSTRAINT fk_pull_sources_bind_user 
FOREIGN KEY (bind_user_id) REFERENCES users(id) ON DELETE SET NULL;