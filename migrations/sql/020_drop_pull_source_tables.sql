-- 删除拉单源相关表，整合到平台管理
-- 开发阶段直接删除，不需要数据迁移

-- 删除商品映射表
DROP TABLE IF EXISTS `pull_source_product_map`;

-- 删除变体配置表
DROP TABLE IF EXISTS `pull_source_variants`;

-- 删除拉单源表
DROP TABLE IF EXISTS `pull_sources`;