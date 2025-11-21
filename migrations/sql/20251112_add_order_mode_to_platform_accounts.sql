-- 添加平台账号模式字段
ALTER TABLE `platform_accounts`
  ADD COLUMN `order_mode` TINYINT NOT NULL DEFAULT 1 COMMENT '账号模式: 1-推单 2-拉单';