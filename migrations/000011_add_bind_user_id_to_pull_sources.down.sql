-- 回滚：移除 pull_sources 表的 bind_user_id 字段

-- 删除外键约束
ALTER TABLE pull_sources 
DROP FOREIGN KEY fk_pull_sources_bind_user;

-- 删除索引
ALTER TABLE pull_sources 
DROP INDEX idx_bind_user_id;

-- 删除字段
ALTER TABLE pull_sources 
DROP COLUMN bind_user_id;