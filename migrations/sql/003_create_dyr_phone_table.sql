-- 创建手机归属地表
tCREATE TABLE IF NOT EXISTS `dyr_phone` (
    `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT 'ID',
    `phone_number` varchar(20) NOT NULL COMMENT '手机号码',
    `province` varchar(50) NOT NULL COMMENT '省份',
    `city` varchar(50) NOT NULL COMMENT '城市',
    `isp` varchar(50) NOT NULL COMMENT '运营商',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_phone_number` (`phone_number`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='手机归属地表'; 