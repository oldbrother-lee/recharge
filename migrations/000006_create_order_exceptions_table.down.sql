-- 删除订单异常表
DROP TABLE IF EXISTS `order_exceptions`;

-- 删除orders表的has_exception字段
ALTER TABLE `orders` DROP COLUMN IF EXISTS `has_exception`;