-- 创建订单表
CREATE TABLE IF NOT EXISTS `orders` (
    `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '订单ID',
    `order_no` varchar(32) NOT NULL COMMENT '订单编号',
    `user_id` bigint(20) NOT NULL COMMENT '用户ID',
    `product_id` bigint(20) NOT NULL COMMENT '商品ID',
    `product_spec_id` bigint(20) NOT NULL COMMENT '商品规格ID',
    `member_grade_id` bigint(20) NOT NULL COMMENT '会员等级ID',
    `price` decimal(10,2) NOT NULL COMMENT '订单金额',
    `status` tinyint(4) NOT NULL DEFAULT '0' COMMENT '订单状态：0-待支付，1-已支付，2-已取消，3-已退款',
    `pay_time` datetime DEFAULT NULL COMMENT '支付时间',
    `pay_method` varchar(20) DEFAULT NULL COMMENT '支付方式',
    `pay_no` varchar(64) DEFAULT NULL COMMENT '支付流水号',
    `cancel_time` datetime DEFAULT NULL COMMENT '取消时间',
    `cancel_reason` varchar(255) DEFAULT NULL COMMENT '取消原因',
    `refund_time` datetime DEFAULT NULL COMMENT '退款时间',
    `refund_reason` varchar(255) DEFAULT NULL COMMENT '退款原因',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_order_no` (`order_no`),
    KEY `idx_user_id` (`user_id`),
    KEY `idx_product_id` (`product_id`),
    KEY `idx_status` (`status`),
    KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='订单表';

-- 创建订单日志表
CREATE TABLE IF NOT EXISTS `order_logs` (
    `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '日志ID',
    `order_id` bigint(20) NOT NULL COMMENT '订单ID',
    `order_no` varchar(32) NOT NULL COMMENT '订单编号',
    `action` varchar(20) NOT NULL COMMENT '操作类型',
    `operator` varchar(32) NOT NULL COMMENT '操作人',
    `content` varchar(255) NOT NULL COMMENT '操作内容',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    PRIMARY KEY (`id`),
    KEY `idx_order_id` (`order_id`),
    KEY `idx_order_no` (`order_no`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='订单日志表'; 