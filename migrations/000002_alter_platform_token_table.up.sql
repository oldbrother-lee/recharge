-- 先删除旧表
DROP TABLE IF EXISTS platform_token;

-- 创建新表
CREATE TABLE platform_token (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    task_config_id BIGINT NOT NULL,
    token VARCHAR(255) NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_used_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_task_config_id (task_config_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci; 