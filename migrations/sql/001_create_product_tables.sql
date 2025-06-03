-- 商品分类表
CREATE TABLE IF NOT EXISTS `product_categories` (
    `id` bigint unsigned NOT NULL AUTO_INCREMENT,
    `name` varchar(100) NOT NULL COMMENT '分类名称',
    `sort` int NOT NULL DEFAULT '0' COMMENT '排序',
    `status` tinyint(1) NOT NULL DEFAULT '1' COMMENT '状态：1-正常 0-禁用',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_sort` (`sort`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='商品分类表';

-- 商品表
CREATE TABLE IF NOT EXISTS `products` (
    `id` bigint unsigned NOT NULL AUTO_INCREMENT,
    `name` varchar(255) NOT NULL COMMENT '商品名称',
    `description` varchar(255) DEFAULT NULL COMMENT '商品描述',
    `price` decimal(10,2) NOT NULL DEFAULT '0.00' COMMENT '商品价格',
    `type` tinyint NOT NULL DEFAULT '1' COMMENT '商品类型',
    `isp` varchar(10) NOT NULL DEFAULT '1,2,3' COMMENT '运营商',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `status` tinyint(1) NOT NULL DEFAULT '1' COMMENT '状态：1-正常 0-禁用',
    `sort` int NOT NULL DEFAULT '0' COMMENT '排序',
    `api_enabled` tinyint(1) NOT NULL DEFAULT '0' COMMENT 'API开关',
    `remark` varchar(255) DEFAULT NULL COMMENT '备注',
    `category_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '分类ID',
    `operator_tag` varchar(50) DEFAULT NULL COMMENT '运营商标签',
    `max_price` decimal(10,2) NOT NULL DEFAULT '0.00' COMMENT '最高价格',
    `voucher_price` varchar(255) DEFAULT NULL COMMENT '面值价格',
    `voucher_name` varchar(255) DEFAULT NULL COMMENT '面值名称',
    `show_style` tinyint NOT NULL DEFAULT '1' COMMENT '显示样式',
    `api_fail_style` tinyint NOT NULL DEFAULT '1' COMMENT 'API失败处理方式',
    `allow_provinces` text COMMENT '允许的省份',
    `allow_cities` text COMMENT '允许的城市',
    `forbid_provinces` text COMMENT '禁止的省份',
    `forbid_cities` text COMMENT '禁止的城市',
    `api_delay` decimal(10,2) NOT NULL DEFAULT '0.00' COMMENT 'API延迟',
    `grade_ids` varchar(500) DEFAULT NULL COMMENT '等级ID列表',
    `api_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '接码API ID',
    `api_param_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '接码API参数ID',
    `is_decode` tinyint(1) NOT NULL DEFAULT '0' COMMENT '是否是接码产品',
    `created_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    KEY `idx_category` (`category_id`),
    KEY `idx_status` (`status`),
    KEY `idx_sort` (`sort`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='商品表';

-- 商品规格表
CREATE TABLE IF NOT EXISTS `product_specs` (
    `id` bigint unsigned NOT NULL AUTO_INCREMENT,
    `product_id` bigint unsigned NOT NULL COMMENT '商品ID',
    `name` varchar(100) NOT NULL COMMENT '规格名称',
    `value` varchar(255) NOT NULL COMMENT '规格值',
    `price` decimal(10,2) NOT NULL DEFAULT '0.00' COMMENT '规格价格',
    `stock` int NOT NULL DEFAULT '0' COMMENT '库存',
    `sort` int NOT NULL DEFAULT '0' COMMENT '排序',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_product` (`product_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='商品规格表';

-- 会员等级表
CREATE TABLE IF NOT EXISTS `member_grades` (
    `id` bigint unsigned NOT NULL AUTO_INCREMENT,
    `name` varchar(50) NOT NULL COMMENT '等级名称',
    `grade_type` tinyint NOT NULL DEFAULT '1' COMMENT '等级类型',
    `sort` int NOT NULL DEFAULT '0' COMMENT '排序',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_sort` (`sort`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='会员等级表';

-- 商品会员价格表
CREATE TABLE IF NOT EXISTS `product_grade_prices` (
    `id` bigint unsigned NOT NULL AUTO_INCREMENT,
    `product_id` bigint unsigned NOT NULL COMMENT '商品ID',
    `grade_id` bigint unsigned NOT NULL COMMENT '等级ID',
    `price` decimal(10,2) NOT NULL DEFAULT '0.00' COMMENT '价格',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_product_grade` (`product_id`,`grade_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='商品会员价格表';