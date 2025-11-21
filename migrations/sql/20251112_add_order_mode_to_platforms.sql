-- 为平台表添加平台级订单模式字段
ALTER TABLE `platforms`
  ADD COLUMN `order_mode` TINYINT NOT NULL DEFAULT 1 COMMENT '平台模式：1-推单；2-拉单' AFTER `status`;