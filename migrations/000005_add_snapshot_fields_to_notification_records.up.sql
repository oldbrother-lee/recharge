-- 为notification_records表添加订单快照和目标状态字段
ALTER TABLE `notification_records` 
ADD COLUMN `order_snapshot` TEXT COMMENT '序列化的订单快照',
ADD COLUMN `target_status` TINYINT COMMENT '目标状态';