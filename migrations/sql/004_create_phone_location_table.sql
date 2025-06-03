-- 创建手机归属地表
CREATE TABLE IF NOT EXISTS `phone_locations` (
    `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `phone_number` varchar(20) NOT NULL COMMENT '手机号',
    `province` varchar(50) NOT NULL COMMENT '省份',
    `city` varchar(50) NOT NULL COMMENT '城市',
    `isp` varchar(50) NOT NULL COMMENT '运营商',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_phone_number` (`phone_number`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='手机归属地表'; 