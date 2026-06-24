-- Add conditional visibility metadata for system_config and split token storage type from redis options.
-- MySQL 5.7+.

ALTER TABLE `system_config`
  ADD COLUMN `options` text DEFAULT NULL COMMENT '配置选项 JSON' AFTER `value_type`,
  ADD COLUMN `visible_when_key` varchar(128) DEFAULT NULL COMMENT '展示条件配置键' AFTER `category`,
  ADD COLUMN `visible_when_value` varchar(64) DEFAULT NULL COMMENT '展示条件配置值' AFTER `visible_when_key`,
  ADD INDEX `idx_system_config_visible_when_key` (`visible_when_key`);

START TRANSACTION;

INSERT INTO `system_config`
  (`config_key`, `config_value`, `name`, `value_type`, `options`, `config_group`, `category`, `visible_when_key`, `visible_when_value`, `description`, `enabled`, `sort`, `created_at`, `created_by`, `updated_at`, `updated_by`)
VALUES
  ('security.token.storage.type', 'memory', 'Token 存储类型', 'enum', '[{"label":"内存","value":"memory"},{"label":"Redis","value":"redis"}]', 'token_storage', 'system', NULL, NULL, '支持 memory、redis', 1, 100, NOW(), 'system', NOW(), 'system'),
  ('security.token.storage.redis', '{}', 'Token Redis 配置', 'json', NULL, 'token_storage', 'system', 'security.token.storage.type', 'redis', '当 Token 存储类型为 redis 时使用的 Redis 连接配置 JSON', 1, 110, NOW(), 'system', NOW(), 'system')
ON DUPLICATE KEY UPDATE
  `name` = VALUES(`name`),
  `value_type` = VALUES(`value_type`),
  `options` = VALUES(`options`),
  `config_group` = VALUES(`config_group`),
  `category` = VALUES(`category`),
  `visible_when_key` = VALUES(`visible_when_key`),
  `visible_when_value` = VALUES(`visible_when_value`),
  `description` = VALUES(`description`),
  `enabled` = VALUES(`enabled`),
  `sort` = VALUES(`sort`),
  `updated_at` = NOW(),
  `updated_by` = 'system';

-- Optional best-effort migration from the legacy JSON key:
-- If security.token.storage exists and stores {"type":"redis", ...}, copy that JSON
-- to the new redis config and switch the type to redis.
UPDATE `system_config` AS target
JOIN `system_config` AS legacy
  ON legacy.`config_key` = 'security.token.storage'
SET target.`config_value` = 'redis',
    target.`updated_at` = NOW(),
    target.`updated_by` = 'system'
WHERE target.`config_key` = 'security.token.storage.type'
  AND JSON_VALID(legacy.`config_value`)
  AND JSON_UNQUOTE(JSON_EXTRACT(legacy.`config_value`, '$.type')) = 'redis';

UPDATE `system_config` AS target
JOIN `system_config` AS legacy
  ON legacy.`config_key` = 'security.token.storage'
SET target.`config_value` = legacy.`config_value`,
    target.`updated_at` = NOW(),
    target.`updated_by` = 'system'
WHERE target.`config_key` = 'security.token.storage.redis'
  AND JSON_VALID(legacy.`config_value`)
  AND JSON_UNQUOTE(JSON_EXTRACT(legacy.`config_value`, '$.type')) = 'redis';

COMMIT;
