-- 回滚：删除notification_records表的订单快照和目标状态字段
ALTER TABLE `notification_records` 
DROP COLUMN `order_snapshot`,
DROP COLUMN `target_status`;