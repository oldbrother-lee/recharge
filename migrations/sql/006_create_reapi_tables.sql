-- 平台接口表
CREATE TABLE IF NOT EXISTS `platform_apis` (
    `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `name` varchar(50) NOT NULL COMMENT '平台名称',
    `param1` varchar(255) DEFAULT NULL COMMENT '配置1(商户ID)',
    `param2` varchar(255) DEFAULT NULL COMMENT '配置2(秘钥)',
    `param3` varchar(255) DEFAULT NULL COMMENT '配置3(回调地址)',
    `param4` varchar(255) DEFAULT NULL COMMENT '配置4(接口地址)',
    `param5` varchar(255) DEFAULT NULL COMMENT '配置5',
    `remark` text COMMENT '配置说明',
    `api_remark` text COMMENT '套餐参数说明',
    `status` tinyint NOT NULL DEFAULT '1' COMMENT '状态：1-启用，0-禁用',
    `is_deleted` tinyint NOT NULL DEFAULT '0' COMMENT '是否删除：0-未删除，1-已删除',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    KEY `idx_status` (`status`),
    KEY `idx_is_deleted` (`is_deleted`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='平台接口表';

-- 接口参数表
CREATE TABLE IF NOT EXISTS `platform_api_params` (
    `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `api_id` bigint NOT NULL COMMENT '接口ID',
    `name` varchar(50) NOT NULL COMMENT '参数名称',
    `code` varchar(50) NOT NULL COMMENT '参数代码',
    `value` varchar(255) DEFAULT NULL COMMENT '参数值',
    `description` varchar(255) DEFAULT NULL COMMENT '参数描述',
    `allow_provinces` text COMMENT '允许的省份',
    `allow_cities` text COMMENT '允许的城市',
    `forbid_provinces` text COMMENT '禁止的省份',
    `forbid_cities` text COMMENT '禁止的城市',
    `sort` int NOT NULL DEFAULT '0' COMMENT '排序',
    `status` tinyint NOT NULL DEFAULT '1' COMMENT '状态：1-启用，0-禁用',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    KEY `idx_api_id` (`api_id`),
    KEY `idx_status` (`status`),
    CONSTRAINT `fk_platform_api_params_api_id` FOREIGN KEY (`api_id`) REFERENCES `platform_apis` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='接口参数表';

-- 商品接口关联表
CREATE TABLE IF NOT EXISTS `product_api_relations` (
    `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `product_id` bigint NOT NULL COMMENT '商品ID',
    `api_id` bigint NOT NULL COMMENT '接口ID',
    `param_id` bigint NOT NULL COMMENT '参数ID',
    `sort` int NOT NULL DEFAULT '0' COMMENT '排序',
    `status` tinyint NOT NULL DEFAULT '1' COMMENT '状态：1-启用，0-禁用',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    KEY `idx_product_id` (`product_id`),
    KEY `idx_api_id` (`api_id`),
    KEY `idx_param_id` (`param_id`),
    CONSTRAINT `fk_product_api_relations_product_id` FOREIGN KEY (`product_id`) REFERENCES `products` (`id`) ON DELETE CASCADE,
    CONSTRAINT `fk_product_api_relations_api_id` FOREIGN KEY (`api_id`) REFERENCES `platform_apis` (`id`) ON DELETE CASCADE,
    CONSTRAINT `fk_product_api_relations_param_id` FOREIGN KEY (`param_id`) REFERENCES `platform_api_params` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='商品接口关联表';

-- 接口调用日志表
CREATE TABLE IF NOT EXISTS `api_call_logs` (
    `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `api_id` bigint NOT NULL COMMENT '接口ID',
    `request_url` varchar(255) NOT NULL COMMENT '请求URL',
    `request_method` varchar(10) NOT NULL COMMENT '请求方法',
    `request_params` text COMMENT '请求参数',
    `response_data` text COMMENT '响应数据',
    `status_code` int NOT NULL COMMENT '状态码',
    `error_message` varchar(255) DEFAULT NULL COMMENT '错误信息',
    `duration` int NOT NULL COMMENT '耗时(毫秒)',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    PRIMARY KEY (`id`),
    KEY `idx_api_id` (`api_id`),
    KEY `idx_created_at` (`created_at`),
    CONSTRAINT `fk_api_call_logs_api_id` FOREIGN KEY (`api_id`) REFERENCES `platform_apis` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='接口调用日志表'; 