-- Existing rows are classified as system settings by default.
ALTER TABLE `system_config`
  ADD COLUMN `category` varchar(32) NOT NULL DEFAULT 'system'
    COMMENT '配置类别：system系统设置，site网站配置' AFTER `config_group`,
  ADD INDEX `idx_system_config_category` (`category`);
