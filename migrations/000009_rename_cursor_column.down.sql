-- Rollback: rename column `cursor_token` back to `cursor` in pull_source_variants
ALTER TABLE `pull_source_variants`
  CHANGE COLUMN `cursor_token` `cursor` VARCHAR(255) DEFAULT NULL COMMENT '拉取游标';