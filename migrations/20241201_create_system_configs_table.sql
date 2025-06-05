-- 创建系统配置表
CREATE TABLE IF NOT EXISTS `system_configs` (
    `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `config_key` varchar(100) NOT NULL COMMENT '配置键',
    `config_value` text COMMENT '配置值',
    `config_desc` varchar(255) DEFAULT NULL COMMENT '配置描述',
    `config_type` varchar(50) DEFAULT 'string' COMMENT '配置类型(string,number,boolean,json)',
    `is_enabled` tinyint(1) DEFAULT 1 COMMENT '是否启用(0:禁用,1:启用)',
    `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_config_key` (`config_key`),
    KEY `idx_is_enabled` (`is_enabled`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='系统配置表';

-- 插入默认系统配置
INSERT INTO `system_configs` (`config_key`, `config_value`, `config_desc`, `config_type`) VALUES
('system_name', '充值系统', '系统名称', 'string'),
('system_version', '1.0.0', '系统版本', 'string'),
('system_description', '一个功能强大的充值管理系统', '系统描述', 'string'),
('maintenance_mode', 'false', '维护模式', 'boolean'),
('max_upload_size', '10485760', '最大上传文件大小(字节)', 'number'),
('session_timeout', '3600', '会话超时时间(秒)', 'number')
ON DUPLICATE KEY UPDATE
`config_value` = VALUES(`config_value`),
`updated_at` = CURRENT_TIMESTAMP;