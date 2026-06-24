-- Initialize audit log switches.
-- MySQL 5.7+.

INSERT INTO `system_config`
  (`config_key`, `config_value`, `name`, `value_type`, `config_group`, `category`, `description`, `enabled`, `sort`, `created_at`, `created_by`, `updated_at`, `updated_by`)
VALUES
  ('audit.log.enabled', 'true', '审计日志功能开关', 'bool', 'audit_log', 'system', '控制操作日志和审计日志是否记录', 1, 900, NOW(), 'system', NOW(), 'system'),
  ('audit.log.include_query', 'true', '审计日志是否包含查询', 'bool', 'audit_log', 'system', '控制是否记录 GET/query 查询类操作日志', 1, 910, NOW(), 'system', NOW(), 'system')
ON DUPLICATE KEY UPDATE
  `name` = VALUES(`name`),
  `value_type` = VALUES(`value_type`),
  `config_group` = VALUES(`config_group`),
  `category` = VALUES(`category`),
  `description` = VALUES(`description`),
  `enabled` = VALUES(`enabled`),
  `sort` = VALUES(`sort`),
  `updated_at` = NOW(),
  `updated_by` = 'system';
