-- 移除external_api_keys表的user_id字段
ALTER TABLE `external_api_keys` 
DROP INDEX `idx_user_id`,
DROP COLUMN `user_id`;