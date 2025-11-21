-- 创建得众平台任务配置表
CREATE TABLE IF NOT EXISTS `dz_task_configs` (
    `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `platform_id` bigint(20) NOT NULL COMMENT '平台ID',
    `platform_name` varchar(255) NOT NULL COMMENT '平台名称',
    `platform_account_id` bigint(20) NOT NULL COMMENT '平台账号ID',
    `platform_account` varchar(255) NOT NULL COMMENT '平台账号',
    `product_id` varchar(64) NOT NULL COMMENT '产品ID',
    `product_name` varchar(255) NOT NULL COMMENT '产品名称',
    `isp` int(11) NOT NULL COMMENT '运营商 1:移动 2:电信 3:联通',
    `face_value` int(11) NOT NULL COMMENT '面值',
    `poll_interval_sec` int(11) NOT NULL DEFAULT 30 COMMENT '轮询间隔秒数',
    `concurrency` int(11) NOT NULL DEFAULT 1 COMMENT '并发度',
    `enabled` int(11) NOT NULL DEFAULT 1 COMMENT '状态 1:启用 0:禁用',
    `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    KEY `idx_platform_id` (`platform_id`),
    KEY `idx_platform_account_id` (`platform_account_id`),
    KEY `idx_product_id` (`product_id`),
    KEY `idx_enabled` (`enabled`),
    KEY `idx_isp_face_value` (`isp`, `face_value`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='得众平台任务配置表';