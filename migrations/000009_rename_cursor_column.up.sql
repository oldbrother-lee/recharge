-- Rename column `cursor` to `cursor_token` in pull_source_variants
ALTER TABLE `pull_source_variants`
  CHANGE COLUMN `cursor` `cursor_token` VARCHAR(255) DEFAULT NULL COMMENT '拉取游标令牌';