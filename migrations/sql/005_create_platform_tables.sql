-- 创建平台表
CREATE TABLE IF NOT EXISTS `platforms` (
    `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `name` varchar(50) NOT NULL COMMENT '平台名称',
    `code` varchar(20) NOT NULL COMMENT '平台代码',
    `api_url` varchar(255) NOT NULL COMMENT 'API地址',
    `description` varchar(255) DEFAULT NULL COMMENT '描述',
    `status` tinyint NOT NULL DEFAULT '1' COMMENT '状态：1-启用，0-禁用',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at` datetime DEFAULT NULL COMMENT '删除时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_code` (`code`),
    KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='平台表';

-- 创建平台账号表
CREATE TABLE IF NOT EXISTS `platform_accounts` (
    `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `platform_id` bigint NOT NULL COMMENT '平台ID',
    `account_name` varchar(50) NOT NULL COMMENT '账号名称',
    `type` tinyint NOT NULL DEFAULT '1' COMMENT '账号类型：1-测试账号，2-正式账号',
    `app_key` varchar(64) NOT NULL COMMENT 'AppKey',
    `app_secret` varchar(64) NOT NULL COMMENT 'AppSecret',
    `description` varchar(255) DEFAULT NULL COMMENT '描述',
    `daily_limit` decimal(10,2) NOT NULL DEFAULT '0.00' COMMENT '每日限额',
    `monthly_limit` decimal(10,2) NOT NULL DEFAULT '0.00' COMMENT '每月限额',
    `balance` decimal(10,2) NOT NULL DEFAULT '0.00' COMMENT '余额',
    `priority` int NOT NULL DEFAULT '0' COMMENT '优先级',
    `status` tinyint NOT NULL DEFAULT '1' COMMENT '状态：1-启用，0-禁用',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at` datetime DEFAULT NULL COMMENT '删除时间',
    PRIMARY KEY (`id`),
    KEY `idx_platform_id` (`platform_id`),
    KEY `idx_deleted_at` (`deleted_at`),
    CONSTRAINT `fk_platform_accounts_platform_id` FOREIGN KEY (`platform_id`) REFERENCES `platforms` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='平台账号表';

-- 创建平台API表
CREATE TABLE IF NOT EXISTS `platform_apis` (
    `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `platform_id` bigint NOT NULL COMMENT '平台ID',
    `api_name` varchar(50) NOT NULL COMMENT 'API名称',
    `api_code` varchar(50) NOT NULL COMMENT 'API代码',
    `api_path` varchar(255) NOT NULL COMMENT 'API路径',
    `method` varchar(10) NOT NULL DEFAULT 'POST' COMMENT '请求方法',
    `timeout` int NOT NULL DEFAULT '30' COMMENT '超时时间(秒)',
    `retry_times` int NOT NULL DEFAULT '3' COMMENT '重试次数',
    `status` tinyint(1) NOT NULL DEFAULT '1' COMMENT '状态：1-启用，0-禁用',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    KEY `idx_platform_id` (`platform_id`),
    CONSTRAINT `fk_platform_apis_platform_id` FOREIGN KEY (`platform_id`) REFERENCES `platforms` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='平台API表'; 