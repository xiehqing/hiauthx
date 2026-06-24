-- hiauth API metadata initialization
-- MySQL 5.7+
-- Route paths must use Hertz templates (for example /api/v1/users/:id),
-- because the audit cache is indexed by HTTP method + matched route template.

CREATE TABLE IF NOT EXISTS `api` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `created_at` datetime NOT NULL,
  `created_by` varchar(255) NOT NULL,
  `updated_at` datetime NOT NULL,
  `updated_by` varchar(255) NOT NULL,
  `name` varchar(128) NOT NULL COMMENT 'API name',
  `method` varchar(16) NOT NULL COMMENT 'HTTP method',
  `path` varchar(255) NOT NULL COMMENT 'route template',
  `module` varchar(64) NOT NULL COMMENT 'audit module',
  `action` varchar(64) NOT NULL COMMENT 'audit action',
  `description` varchar(255) DEFAULT NULL COMMENT 'audit description',
  `resource_type` varchar(64) NOT NULL DEFAULT 'request' COMMENT 'resource type',
  `status` int NOT NULL DEFAULT 1 COMMENT '0 disabled, 1 enabled',
  `sort` int NOT NULL DEFAULT 0 COMMENT 'sort order',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_api_method_path` (`method`, `path`),
  KEY `idx_api_module` (`module`),
  KEY `idx_api_action` (`action`),
  KEY `idx_api_resource_type` (`resource_type`),
  KEY `idx_api_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

START TRANSACTION;

INSERT INTO `api`
  (`name`, `method`, `path`, `module`, `action`, `description`, `resource_type`, `status`, `sort`, `created_at`, `created_by`, `updated_at`, `updated_by`)
VALUES
  ('健康检查', 'GET', '/api/v1/health', '系统', 'query', '检查服务健康状态', 'system', 1, 10, NOW(), 'system', NOW(), 'system'),

  ('获取登录加密配置', 'GET', '/api/v1/auth/encrypt-config', '认证管理', 'query', '获取登录密码加密配置', 'auth', 1, 100, NOW(), 'system', NOW(), 'system'),
  ('生成 RSA 密钥对', 'POST', '/api/v1/auth/rsa-key-pair', '认证管理', 'create', '生成登录加密 RSA 公私钥对', 'auth', 1, 105, NOW(), 'system', NOW(), 'system'),
  ('用户登录', 'POST', '/api/v1/auth/login', '认证管理', 'login', '用户登录', 'auth', 1, 110, NOW(), 'system', NOW(), 'system'),
  ('用户退出', 'POST', '/api/v1/auth/logout', '认证管理', 'logout', '用户退出登录', 'auth', 1, 120, NOW(), 'system', NOW(), 'system'),
  ('查询当前用户', 'GET', '/api/v1/auth/me', '认证管理', 'query', '查询当前登录用户', 'auth', 1, 130, NOW(), 'system', NOW(), 'system'),

  ('查询用户列表', 'GET', '/api/v1/users', '用户管理', 'query', '查询用户列表', 'user', 1, 200, NOW(), 'system', NOW(), 'system'),
  ('新增用户', 'POST', '/api/v1/users', '用户管理', 'create', '新增用户', 'user', 1, 210, NOW(), 'system', NOW(), 'system'),
  ('查询用户详情', 'GET', '/api/v1/users/:id', '用户管理', 'query', '查询用户详情', 'user', 1, 220, NOW(), 'system', NOW(), 'system'),
  ('更新用户', 'PUT', '/api/v1/users/:id', '用户管理', 'update', '更新用户信息', 'user', 1, 230, NOW(), 'system', NOW(), 'system'),
  ('删除用户', 'DELETE', '/api/v1/users/:id', '用户管理', 'delete', '删除用户', 'user', 1, 240, NOW(), 'system', NOW(), 'system'),

  ('查询角色列表', 'GET', '/api/v1/roles', '角色管理', 'query', '查询角色列表', 'role', 1, 300, NOW(), 'system', NOW(), 'system'),
  ('查询角色选项', 'GET', '/api/v1/roles/options', '角色管理', 'query', '查询角色选项', 'role', 1, 310, NOW(), 'system', NOW(), 'system'),
  ('新增角色', 'POST', '/api/v1/roles', '角色管理', 'create', '新增角色', 'role', 1, 320, NOW(), 'system', NOW(), 'system'),
  ('查询角色详情', 'GET', '/api/v1/roles/:id', '角色管理', 'query', '查询角色详情', 'role', 1, 330, NOW(), 'system', NOW(), 'system'),
  ('更新角色', 'PUT', '/api/v1/roles/:id', '角色管理', 'update', '更新角色信息', 'role', 1, 340, NOW(), 'system', NOW(), 'system'),
  ('删除角色', 'DELETE', '/api/v1/roles/:id', '角色管理', 'delete', '删除角色', 'role', 1, 350, NOW(), 'system', NOW(), 'system'),
  ('查询角色菜单', 'GET', '/api/v1/roles/:id/menus', '角色管理', 'query', '查询角色关联菜单', 'role', 1, 360, NOW(), 'system', NOW(), 'system'),
  ('更新角色菜单', 'PUT', '/api/v1/roles/:id/menus', '角色管理', 'association', '更新角色关联菜单', 'role', 1, 370, NOW(), 'system', NOW(), 'system'),

  ('查询部门列表', 'GET', '/api/v1/departments', '部门管理', 'query', '查询部门列表', 'department', 1, 400, NOW(), 'system', NOW(), 'system'),
  ('查询部门选项', 'GET', '/api/v1/departments/options', '部门管理', 'query', '查询部门选项', 'department', 1, 410, NOW(), 'system', NOW(), 'system'),
  ('新增部门', 'POST', '/api/v1/departments', '部门管理', 'create', '新增部门', 'department', 1, 420, NOW(), 'system', NOW(), 'system'),
  ('查询部门详情', 'GET', '/api/v1/departments/:id', '部门管理', 'query', '查询部门详情', 'department', 1, 430, NOW(), 'system', NOW(), 'system'),
  ('更新部门', 'PUT', '/api/v1/departments/:id', '部门管理', 'update', '更新部门信息', 'department', 1, 440, NOW(), 'system', NOW(), 'system'),
  ('删除部门', 'DELETE', '/api/v1/departments/:id', '部门管理', 'delete', '删除部门', 'department', 1, 450, NOW(), 'system', NOW(), 'system'),

  ('查询菜单列表', 'GET', '/api/v1/menus', '菜单管理', 'query', '查询菜单列表', 'menu', 1, 500, NOW(), 'system', NOW(), 'system'),
  ('新增菜单', 'POST', '/api/v1/menus', '菜单管理', 'create', '新增菜单', 'menu', 1, 510, NOW(), 'system', NOW(), 'system'),
  ('查询菜单详情', 'GET', '/api/v1/menus/:id', '菜单管理', 'query', '查询菜单详情', 'menu', 1, 520, NOW(), 'system', NOW(), 'system'),
  ('更新菜单', 'PUT', '/api/v1/menus/:id', '菜单管理', 'update', '更新菜单信息', 'menu', 1, 530, NOW(), 'system', NOW(), 'system'),
  ('删除菜单', 'DELETE', '/api/v1/menus/:id', '菜单管理', 'delete', '删除菜单', 'menu', 1, 540, NOW(), 'system', NOW(), 'system'),

  ('查询系统配置列表', 'GET', '/api/v1/system-configs', '系统配置管理', 'query', '查询系统配置列表', 'system_config', 1, 600, NOW(), 'system', NOW(), 'system'),
  ('新增系统配置', 'POST', '/api/v1/system-configs', '系统配置管理', 'create', '新增系统配置', 'system_config', 1, 610, NOW(), 'system', NOW(), 'system'),
  ('查询启用系统配置', 'GET', '/api/v1/system-configs/enabled', '系统配置管理', 'query', '查询启用的系统配置', 'system_config', 1, 620, NOW(), 'system', NOW(), 'system'),
  ('查询启用配置映射', 'GET', '/api/v1/system-configs/enabled-map', '系统配置管理', 'query', '查询启用的系统配置映射', 'system_config', 1, 630, NOW(), 'system', NOW(), 'system'),
  ('查询系统设置', 'GET', '/api/v1/system-configs/system-settings', '系统配置管理', 'query', '查询启用的系统设置键值对', 'system_config', 1, 632, NOW(), 'system', NOW(), 'system'),
  ('查询网站配置', 'GET', '/api/v1/system-configs/site-settings', '系统配置管理', 'query', '查询启用的网站配置键值对', 'system_config', 1, 634, NOW(), 'system', NOW(), 'system'),
  ('批量保存系统配置', 'PUT', '/api/v1/system-configs/batch', '系统配置管理', 'update', '批量保存系统配置', 'system_config', 1, 636, NOW(), 'system', NOW(), 'system'),
  ('按键查询系统配置', 'GET', '/api/v1/system-configs/by-key/:key', '系统配置管理', 'query', '按配置键查询系统配置', 'system_config', 1, 640, NOW(), 'system', NOW(), 'system'),
  ('查询系统配置详情', 'GET', '/api/v1/system-configs/:id', '系统配置管理', 'query', '查询系统配置详情', 'system_config', 1, 650, NOW(), 'system', NOW(), 'system'),
  ('更新系统配置', 'PUT', '/api/v1/system-configs/:id', '系统配置管理', 'update', '更新系统配置', 'system_config', 1, 660, NOW(), 'system', NOW(), 'system'),
  ('删除系统配置', 'DELETE', '/api/v1/system-configs/:id', '系统配置管理', 'delete', '删除系统配置', 'system_config', 1, 670, NOW(), 'system', NOW(), 'system'),

  ('查询 API 列表', 'GET', '/api/v1/apis', 'API 管理', 'query', '查询 API 元数据列表', 'api', 1, 700, NOW(), 'system', NOW(), 'system'),
  ('新增 API', 'POST', '/api/v1/apis', 'API 管理', 'create', '新增 API 元数据', 'api', 1, 710, NOW(), 'system', NOW(), 'system'),
  ('查询 API 详情', 'GET', '/api/v1/apis/:id', 'API 管理', 'query', '查询 API 元数据详情', 'api', 1, 720, NOW(), 'system', NOW(), 'system'),
  ('更新 API', 'PUT', '/api/v1/apis/:id', 'API 管理', 'update', '更新 API 元数据', 'api', 1, 730, NOW(), 'system', NOW(), 'system'),
  ('删除 API', 'DELETE', '/api/v1/apis/:id', 'API 管理', 'delete', '删除 API 元数据', 'api', 1, 740, NOW(), 'system', NOW(), 'system'),

  ('查询操作日志列表', 'GET', '/api/v1/operation-logs', '操作日志', 'query', '查询用户操作行为日志列表', 'operation_log', 1, 790, NOW(), 'system', NOW(), 'system'),
  ('查询审计日志列表', 'GET', '/api/v1/audit-logs', '审计日志', 'query', '查询审计日志列表', 'audit_log', 1, 800, NOW(), 'system', NOW(), 'system'),
  ('查询审计日志详情', 'GET', '/api/v1/audit-logs/:id', '审计日志', 'query', '查询审计日志详情', 'audit_log', 1, 810, NOW(), 'system', NOW(), 'system')
ON DUPLICATE KEY UPDATE
  `name` = VALUES(`name`),
  `module` = VALUES(`module`),
  `action` = VALUES(`action`),
  `description` = VALUES(`description`),
  `resource_type` = VALUES(`resource_type`),
  `status` = VALUES(`status`),
  `sort` = VALUES(`sort`),
  `updated_at` = NOW(),
  `updated_by` = 'system';

COMMIT;

-- If this script is executed while hiauth is already running, restart the
-- service (or invoke Router.RefreshAPICache from an administrative workflow)
-- so the in-memory audit metadata snapshot is rebuilt immediately.
