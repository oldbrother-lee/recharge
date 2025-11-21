-- 将 users 表的 last_login 列修改为允许 NULL，去除非法的零日期默认值
ALTER TABLE `users`
  MODIFY COLUMN `last_login` DATETIME NULL DEFAULT NULL;