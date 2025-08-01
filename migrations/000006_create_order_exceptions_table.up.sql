-- 创建订单异常表
CREATE TABLE IF NOT EXISTS order_exceptions (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    order_id BIGINT NOT NULL COMMENT '订单ID',
    order_number VARCHAR(255) NOT NULL COMMENT '订单号',
    exception_type VARCHAR(50) NOT NULL COMMENT '异常类型：balance_verification_failed',
    exception_reason TEXT COMMENT '异常原因详细描述',
    exception_data JSON COMMENT '异常相关数据（JSON格式）',
    status VARCHAR(20) NOT NULL DEFAULT 'pending' COMMENT '处理状态：pending-待处理, processing-处理中, resolved-已解决, ignored-已忽略',
    resolved_by BIGINT COMMENT '处理人ID',
    resolved_at TIMESTAMP NULL COMMENT '处理时间',
    resolved_note TEXT COMMENT '处理备注',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    INDEX idx_order_id (order_id),
    INDEX idx_order_number (order_number),
    INDEX idx_exception_type (exception_type),
    INDEX idx_status (status),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='订单异常记录表';

-- 为订单表添加异常标记字段
ALTER TABLE orders ADD COLUMN has_exception TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否有异常：0-无异常, 1-有异常';

-- 为has_exception字段添加索引
ALTER TABLE orders ADD INDEX idx_has_exception (has_exception);

-- 添加外键约束
ALTER TABLE order_exceptions ADD CONSTRAINT fk_order_exceptions_order_id FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE;