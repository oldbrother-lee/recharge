-- 为external_api_keys表添加user_id字段
ALTER TABLE `external_api_keys` 
ADD COLUMN `user_id` bigint(20) DEFAULT NULL COMMENT '用户ID' AFTER `id`,
ADD INDEX `idx_user_id` (`user_id`);