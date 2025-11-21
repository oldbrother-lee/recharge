-- 回滚：将 users 表的 last_login 列改为非空，并设置一个安全的默认值
ALTER TABLE `users`
  MODIFY COLUMN `last_login` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP;