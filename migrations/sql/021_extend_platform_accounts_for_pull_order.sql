-- 扩展 platform_accounts 表，添加拉单功能相关字段

ALTER TABLE `platform_accounts` 
ADD COLUMN `account_password` varchar(255) DEFAULT NULL COMMENT '平台账号密码' AFTER `app_secret`,
ADD COLUMN `bind_user_id` bigint DEFAULT NULL COMMENT '绑定的本地用户ID' AFTER `account_password`,
ADD COLUMN `bind_user_name` varchar(50) DEFAULT NULL COMMENT '绑定用户名（冗余字段）' AFTER `bind_user_id`,
ADD COLUMN `enable_pull_order` tinyint(1) NOT NULL DEFAULT '0' COMMENT '是否启用拉单功能' AFTER `bind_user_name`,
ADD COLUMN `max_concurrency` int NOT NULL DEFAULT '1' COMMENT '最大并发拉取数' AFTER `enable_pull_order`,
ADD COLUMN `poll_interval_sec` int NOT NULL DEFAULT '10' COMMENT '默认轮询间隔秒' AFTER `max_concurrency`,
ADD COLUMN `pull_action` varchar(64) DEFAULT NULL COMMENT '拉单动作名' AFTER `poll_interval_sec`;

-- 添加索引
ALTER TABLE `platform_accounts` 
ADD INDEX `idx_bind_user_id` (`bind_user_id`),
ADD INDEX `idx_enable_pull_order` (`enable_pull_order`);