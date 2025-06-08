-- 创建外部API密钥表
CREATE TABLE IF NOT EXISTS `external_api_keys` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT 'ID',
  `app_id` varchar(64) NOT NULL COMMENT '应用ID',
  `app_key` varchar(128) NOT NULL COMMENT '应用密钥',
  `app_secret` varchar(256) NOT NULL COMMENT '应用秘钥',
  `app_name` varchar(128) NOT NULL COMMENT '应用名称',
  `description` varchar(255) DEFAULT NULL COMMENT '应用描述',
  `status` tinyint(1) NOT NULL DEFAULT '1' COMMENT '状态 1:启用 0:禁用',
  `ip_whitelist` text COMMENT 'IP白名单,逗号分隔',
  `notify_url` varchar(512) DEFAULT NULL COMMENT '回调通知URL',
  `rate_limit` int(11) NOT NULL DEFAULT '1000' COMMENT '每分钟请求限制',
  `expire_time` datetime DEFAULT NULL COMMENT '过期时间',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` datetime DEFAULT NULL COMMENT '删除时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_app_id` (`app_id`),
  UNIQUE KEY `uk_app_key` (`app_key`),
  KEY `idx_status` (`status`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='外部API密钥表';

-- 更新外部订单日志表结构
ALTER TABLE `external_order_logs` 
ADD COLUMN IF NOT EXISTS `app_id` varchar(64) DEFAULT NULL COMMENT '应用ID' AFTER `id`,
ADD COLUMN IF NOT EXISTS `out_trade_num` varchar(64) DEFAULT NULL COMMENT '外部交易号' AFTER `app_id`,
ADD COLUMN IF NOT EXISTS `order_id` bigint(20) DEFAULT NULL COMMENT '内部订单ID' AFTER `out_trade_num`,
ADD COLUMN IF NOT EXISTS `order_number` varchar(32) DEFAULT NULL COMMENT '内部订单号' AFTER `order_id`,
ADD COLUMN IF NOT EXISTS `action` varchar(32) DEFAULT NULL COMMENT '操作类型:create,query,callback' AFTER `order_number`,
ADD COLUMN IF NOT EXISTS `request_data` text COMMENT '请求数据' AFTER `action`,
ADD COLUMN IF NOT EXISTS `response_data` text COMMENT '响应数据' AFTER `request_data`,
ADD COLUMN IF NOT EXISTS `status` int(11) DEFAULT NULL COMMENT '状态:1成功,0失败' AFTER `response_data`,
ADD COLUMN IF NOT EXISTS `error_msg` varchar(512) DEFAULT NULL COMMENT '错误信息' AFTER `status`,
ADD COLUMN IF NOT EXISTS `client_ip` varchar(45) DEFAULT NULL COMMENT '客户端IP' AFTER `error_msg`,
ADD COLUMN IF NOT EXISTS `user_agent` varchar(512) DEFAULT NULL COMMENT '用户代理' AFTER `client_ip`,
ADD COLUMN IF NOT EXISTS `process_time` int(11) DEFAULT NULL COMMENT '处理时间(毫秒)' AFTER `user_agent`,
ADD COLUMN IF NOT EXISTS `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间' AFTER `created_at`,
ADD COLUMN IF NOT EXISTS `deleted_at` datetime DEFAULT NULL COMMENT '删除时间' AFTER `updated_at`;

-- 添加索引
ALTER TABLE `external_order_logs`
ADD INDEX IF NOT EXISTS `idx_app_id` (`app_id`),
ADD INDEX IF NOT EXISTS `idx_out_trade_num` (`out_trade_num`),
ADD INDEX IF NOT EXISTS `idx_order_id` (`order_id`),
ADD INDEX IF NOT EXISTS `idx_order_number` (`order_number`),
ADD INDEX IF NOT EXISTS `idx_action` (`action`),
ADD INDEX IF NOT EXISTS `idx_status` (`status`),
ADD INDEX IF NOT EXISTS `idx_created_at` (`created_at`),
ADD INDEX IF NOT EXISTS `idx_deleted_at` (`deleted_at`);

-- 插入示例API密钥数据
INSERT IGNORE INTO `external_api_keys` (
  `app_id`, 
  `app_key`, 
  `app_secret`, 
  `app_name`, 
  `description`, 
  `status`, 
  `ip_whitelist`, 
  `notify_url`, 
  `rate_limit`
) VALUES (
  'test_app_001',
  'test_key_123456789',
  'test_secret_abcdefghijklmnopqrstuvwxyz123456',
  '测试应用',
  '用于测试的外部API应用',
  1,
  '127.0.0.1,::1',
  'http://localhost:8080/callback',
  1000
);