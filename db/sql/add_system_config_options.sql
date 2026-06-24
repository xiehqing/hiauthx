-- Add enum options metadata for system_config.
-- MySQL 5.7+.

ALTER TABLE `system_config`
  ADD COLUMN `options` text DEFAULT NULL COMMENT '配置选项 JSON' AFTER `value_type`;

UPDATE `system_config`
SET `options` = '[{"label":"内存","value":"memory"},{"label":"Redis","value":"redis"}]',
    `updated_at` = NOW(),
    `updated_by` = 'system'
WHERE `config_key` = 'security.token.storage.type';
