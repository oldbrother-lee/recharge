-- 创建平台账号变体表（替代原来的 pull_source_variants）
CREATE TABLE IF NOT EXISTS `platform_account_variants` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `platform_account_id` BIGINT NOT NULL COMMENT '平台账号ID',
  `isp` INT NOT NULL DEFAULT 0 COMMENT '运营商编码：1移动 2电信 3联通 0未知',
  `face_value` DECIMAL(10,2) NOT NULL DEFAULT 0 COMMENT '面值',
  `product_id` BIGINT DEFAULT NULL COMMENT '关联的商品ID',
  `enabled` TINYINT(1) NOT NULL DEFAULT 1 COMMENT '是否启用',
  `poll_interval_sec` INT NOT NULL DEFAULT 10 COMMENT '轮询间隔秒',
  `concurrency` INT NOT NULL DEFAULT 1 COMMENT '并发度',
  `cursor_token` VARCHAR(255) DEFAULT NULL COMMENT '拉取游标',
  `last_pull_at` DATETIME DEFAULT NULL COMMENT '上次拉取时间',
  `fail_count` INT NOT NULL DEFAULT 0 COMMENT '连续失败计数',
  `notify_url` VARCHAR(255) DEFAULT NULL COMMENT '变体级回调地址（可选）',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_account_isp_value` (`platform_account_id`, `isp`, `face_value`),
  KEY `idx_platform_account_id` (`platform_account_id`),
  KEY `idx_product_id` (`product_id`),
  CONSTRAINT `fk_variant_platform_account` FOREIGN KEY (`platform_account_id`) REFERENCES `platform_accounts`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='平台账号变体配置';

-- 创建平台账号商品映射表（替代原来的 pull_source_product_map）
CREATE TABLE IF NOT EXISTS `platform_account_product_map` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `platform_account_id` BIGINT NOT NULL COMMENT '平台账号ID',
  `key_type` VARCHAR(32) NOT NULL COMMENT '映射键类型：by_external_code | by_isp_face_value',
  `external_code` VARCHAR(64) DEFAULT NULL COMMENT '外部商品代码（可选）',
  `isp` INT DEFAULT NULL COMMENT '运营商编码（可选）',
  `face_value` DECIMAL(10,2) DEFAULT NULL COMMENT '面值（可选）',
  `product_id` BIGINT NOT NULL COMMENT '内部商品ID',
  `amount_override` DECIMAL(10,2) DEFAULT NULL COMMENT '覆盖面值（可选）',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_platform_account_id` (`platform_account_id`),
  UNIQUE KEY `uniq_account_external_code` (`platform_account_id`, `external_code`),
  UNIQUE KEY `uniq_account_isp_face` (`platform_account_id`, `isp`, `face_value`),
  CONSTRAINT `fk_map_platform_account` FOREIGN KEY (`platform_account_id`) REFERENCES `platform_accounts`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='平台账号商品映射';