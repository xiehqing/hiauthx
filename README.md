# hiauthx

hiauthx 是一个基于 Go、CloudWeGo Hertz 和 GORM 的认证与权限管理后端，提供登录认证、用户与组织管理、角色菜单授权、系统配置以及操作审计能力。可以作为独立 HTTP 服务运行，也可以将路由和服务层集成到现有 Go 工程中。

本仓库包含后端源码与 OpenAPI 文档，不包含前端管理界面。

## 功能概览

| 模块 | 已实现功能 |
| --- | --- |
| 登录认证 | 用户名密码登录、退出登录、查询当前用户、JWT Token、自动续期、并发登录配置 |
| 登录保护 | 可选 RSA-OAEP-SHA256 密码传输加密、RSA 密钥对生成、登录失败计数与临时锁定 |
| 用户管理 | 用户增删改查、启用/禁用、绑定多个角色、绑定部门、分页与条件筛选 |
| 角色管理 | 角色增删改查、角色选项、分配菜单、内置角色删除保护 |
| 部门管理 | 部门树、上下级关系、部门选项；用户可按部门及其下级部门筛选 |
| 菜单管理 | 分组、菜单和按钮管理、树形结构、路由、图标、排序与显示状态 |
| 系统配置 | 配置增删改查、批量保存、分组与分类、启用状态、枚举选项、条件展示元数据 |
| API 元数据 | 维护请求方法、路由模板、业务模块及审计动作等信息，缓存用于审计匹配 |
| 操作审计 | 操作日志、数据变更前后快照、字段差异、关联关系变更、敏感信息脱敏 |
| 会话存储 | 内存或 Redis，可通过数据库中的系统配置切换 |

权限数据采用 `用户 → 角色 → 菜单` 关联模型。登录返回角色标识、菜单树、部门信息和权限列表；权限列表取菜单的非空 `route` 字段并去重。普通用户的菜单会补齐上级节点，用户名为 `admin` 或具有 `role_admin` 角色的用户可获得全部菜单。

**当前 HTTP 管理接口统一检查登录状态，尚未逐接口校验角色或菜单权限。** API 元数据表用于接口信息和审计描述，不会自动执行访问授权。集成时需按业务需求补充接口级权限检查。

## 技术栈

| 组件 | 版本 / 用途 |
| --- | --- |
| Go | `go.mod` 声明 `1.26.2` |
| CloudWeGo Hertz | `v0.10.4`，HTTP 服务与路由 |
| GORM | `v1.31.1`，数据访问及表结构迁移 |
| MySQL | 当前配置使用的业务数据库 |
| hitoken | `v1.0.2`，Token、会话、角色与权限数据存储 |
| infra | `v1.0.3`，配置、日志、HTTP 响应与数据库基础封装 |
| Redis | 可选，会话共享存储 |

## 工程结构

```text
hiauthx/
├── cmd/                 # 独立服务入口与启动配置
├── routes/              # HTTP 路由、登录中间件、审计上下文及响应格式
├── authentication/      # 登录、当前用户、RSA 加密配置、失败锁定
├── authorization/       # 用户、角色、部门、菜单、配置等业务服务
├── db/
│   ├── entity/          # GORM 数据模型
│   ├── queries/         # 数据访问、事务及系统配置缓存
│   └── sql/             # API 初始化与历史配置迁移脚本
├── audit/               # GORM 审计回调、快照与字段差异
├── hitokenx/            # Token 管理器与内存 / Redis 存储适配
├── configx/             # 数据库系统配置读取与类型转换
├── rsax/                # RSA 密钥、PEM 处理与加解密
├── etc/
│   ├── config.yml       # 服务、日志和数据库配置
│   └── rsa/             # 仓库中的 RSA 文件；登录流程不自动加载这些文件
└── docs/swagger.yaml    # OpenAPI 3.0.3 接口说明
```

请求主要经过 `routes → authentication / authorization → db/queries → MySQL`。审计中间件补充请求和操作者信息，GORM 回调记录数据变更。

## 快速开始

### 1. 准备环境

安装满足 `go.mod` 要求的 Go 工具链，准备可连接的 MySQL 实例。默认使用内存会话存储，首次运行不需要 Redis。

在工程根目录下载依赖：

```sh
go mod download
```

### 2. 创建数据库并配置连接

在 MySQL 中创建数据库：

```sql
CREATE DATABASE IF NOT EXISTS hi_auth CHARACTER SET utf8mb4;
```

编辑 `etc/config.yml`，将连接信息替换为自己的环境。下面为本地配置示例，密码需自行替换：

```yaml
server:
  host: '0.0.0.0'
  port: 7798
  write-timeout: 10000
  idle-timeout: 12000
  shutdown-timeout: 10000
log:
  level: 'info'
  output: 'stdout'
db:
  db-type: 'mysql'
  debug: false
  host: '127.0.0.1'
  port: 3306
  username: 'hiauth'
  password: '<your-database-password>'
  database: 'hi_auth'
  charset: utf8mb4
  append-params: 'parseTime=True&loc=Local'
  max-lifetime: 100
```

HTTP 超时配置单位为毫秒，例如 `idle-timeout: 12000` 为 12 秒。数据库账号需具备启动时建表和更新表结构所需权限。

### 3. 启动服务

在工程根目录执行：

```sh
go run ./cmd --config-dir=./etc --config-file=config --config-type=yml
```

当前配置加载器使用 Viper `SetConfigName`，应传入不带扩展名的 `config`。入口声明的默认值为 `config.yml`，因此建议显式覆盖，避免查找配置文件失败。

| 参数 | 环境变量 | 入口默认值 |
| --- | --- | --- |
| `--config-dir` | `CONFIG_DIR` | `./etc` |
| `--config-file` | `CONFIG_FILE` | `config.yml`；运行时建议设为 `config` |
| `--config-type` | `CONFIG_TYPE` | `yml` |

入口还声明了 `--time-zone` / `TZ`，但目前未使用该参数设置程序时区。

启动时通过 GORM `AutoMigrate` 创建或更新 `user`、`role`、`department`、`menu`、`system_config`、`api`、`audit_log`、`audit_change` 及关联表。数据库本身需要预先创建。当前调用方未处理 `AutoMigrate` 返回的错误，首次启动后应确认表结构已成功生成。

健康检查：

```sh
curl http://localhost:7798/api/v1/health
```

正常返回的数据中包含 `data.status = "ok"`。该接口仅检查 HTTP 服务响应，不执行数据库连通性检测。

### 4. 初始化首个管理员

**工程没有自动创建管理员，也没有预设的登录账号密码。** 用户创建接口需要登录，因此空库首次运行需要通过数据库初始化首个用户。

确认表结构已生成后，在目标数据库执行以下示例。先将密码占位符替换为自己的密码；此语句仅用于首次初始化，不覆盖已有 `admin`：

```sql
USE hi_auth;

INSERT INTO `user`
  (`username`, `password`, `nickname`, `uid`, `status`, `department_id`,
   `created_at`, `created_by`, `updated_at`, `updated_by`)
SELECT
  'admin', MD5('<replace-with-your-own-password>'), '系统管理员', '', 1, 0,
  NOW(), 'bootstrap', NOW(), 'bootstrap'
WHERE NOT EXISTS (
  SELECT 1 FROM `user` WHERE `username` = 'admin'
);
```

此处 MD5 与当前代码的密码存储方式保持一致。`admin` 不依赖角色绑定即可获得全部已有菜单；空库还没有菜单，需要登录后再创建。该账号不出现在用户分页列表中，不允许删除，且不参与登录失败自动锁定。

### 5. 登录并调用接口

先请求 `GET /api/v1/auth/encrypt-config` 获取登录加密配置。默认关闭加密，此时登录请求体如下：

```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "username": "admin",
  "password": "<your-admin-password>",
  "device": "web"
}
```

从响应的 `data.accessToken` 读取 Token，后续请求放入请求头：

```sh
curl -H "Authorization: Bearer <accessToken>" http://localhost:7798/api/v1/auth/me
```

启用登录加密后，使用返回的公钥对原始密码进行 **RSA-OAEP-SHA256** 加密，再将 Base64 编码后的密文作为 `password`。`POST /api/v1/auth/rsa-key-pair` 只生成并返回密钥，不会自动保存到系统配置。

## 接口概览

默认接口前缀为 `/api/v1`。以下路径均省略此前缀。

| 模块 | 接口 |
| --- | --- |
| 健康检查 | `GET /health` |
| 认证 | `GET /auth/encrypt-config`、`POST /auth/login`、`POST /auth/logout`、`GET /auth/me`、`POST /auth/rsa-key-pair` |
| 用户 | `GET/POST /users`、`GET/PUT/DELETE /users/:id` |
| 角色 | `GET/POST /roles`、`GET /roles/options`、`GET/PUT/DELETE /roles/:id`、`GET/PUT /roles/:id/menus` |
| 部门 | `GET/POST /departments`、`GET /departments/options`、`GET/PUT/DELETE /departments/:id` |
| 菜单 | `GET/POST /menus`、`GET/PUT/DELETE /menus/:id` |
| 系统配置 | `GET/POST /system-configs`、`GET/PUT/DELETE /system-configs/:id`、`PUT /system-configs/batch` |
| 配置读取 | `GET /system-configs/by-key/:key`、`GET /system-configs/enabled`、`GET /system-configs/enabled-map`、`GET /system-configs/system-settings`、`GET /system-configs/site-settings` |
| API 元数据 | `GET/POST /apis`、`GET/PUT/DELETE /apis/:id` |
| 操作日志 | `GET /operation-logs` |
| 审计日志 | `GET /audit-logs`、`GET /audit-logs/:id` |

健康检查、登录、登录加密配置和网站配置 `/system-configs/site-settings` 允许匿名访问，其余上述接口要求登录。`site` 分类用于公开网站设置，应只存放可公开展示的数据。

分页接口使用 `pageNo`、`pageSize`、`keyword`、`sortField`、`sortOrder`；各模块还支持自己的筛选字段。通过通用数据响应处理器输出的时间格式为 `YYYY-MM-DD HH:mm:ss`。

完整请求、响应及筛选字段见 `docs/swagger.yaml`，可导入支持 OpenAPI 的接口工具。文档中的示例服务地址目前为 `http://localhost:8080`，调试时需改为实际地址，仓库配置默认为 `http://localhost:7798`。工程未注册 Swagger UI 路由。

## 系统配置与会话

服务端口、日志和数据库连接来自 YAML；登录策略、会话参数及网站信息来自数据库 `system_config` 表。配置支持 `string`、`number`、`bool`、`json`、`enum` 类型，类别为 `system` 或 `site`。

| 配置键 | 用途 / 未配置时的行为 |
| --- | --- |
| `security.login.encrypt.enabled` | 登录密码 RSA 加密，默认 `false` |
| `security.login.rsa.public_key` | 登录加密公钥 |
| `security.login.rsa.private_key` | 登录解密私钥 |
| `security.login.max_attempts` | 非 admin 用户失败次数阈值，默认 `5` |
| `security.login.locked_minutes` | 临时锁定时间，默认 `30` 分钟 |
| `security.login.concurrent.enabled` | 并发登录，默认 `false` |
| `security.token.expire_minutes` | Token 超时，默认 `1440` 分钟 |
| `security.token.jwt_secret_key` | JWT 签名密钥，部署时需设置自己的值 |
| `security.token.storage.type` | 会话存储类型，默认 `memory`，可设为 `redis` |
| `security.token.storage.redis` | Redis 连接参数，JSON 字符串 |
| `audit.log.enabled` | 是否记录审计，默认 `true` |
| `audit.log.include_query` | 是否记录查询类操作，默认 `true` |
| `site.title` / `site.logo` / `site.favicon` / `site.copyright` | 网站展示信息 |

Redis 配置值示例：

```json
{
  "host": "127.0.0.1",
  "port": 6379,
  "password": "",
  "database": 0,
  "poolSize": 10
}
```

也支持通过 `url` 字段提供 Redis URL，以及连接、读写、连接池和操作超时参数，详见 `hitokenx/storage.go`。旧配置键 `security.token.storage` 仍有兼容逻辑。

通过配置接口修改 Token 存储、有效期、签名密钥或并发登录策略会重建 Token 管理器，可能导致已有会话失效。内存模式下，重启或重建存储会丢失会话；多实例共享会话可配置 Redis。存储初始化失败时当前实现会记录错误并回退到内存。

系统配置和 API 元数据均有进程内缓存。通过服务接口修改会更新相关缓存；直接修改数据库后应重启服务以重新加载，多实例之间没有自动缓存同步机制。

## 数据库脚本与审计

`db/sql/init_api.sql` 用于初始化 API 元数据，可在目标数据库内通过 MySQL 客户端执行：

```sql
USE hi_auth;
SOURCE db/sql/init_api.sql;
```

执行客户端的工作目录需为工程根目录，或将 `SOURCE` 后的路径改为实际文件路径。手动维护的路由应使用 Hertz 模板，例如 `/api/v1/users/:id`，审计缓存按 HTTP 方法与匹配到的路由模板检索元数据。导入后重启服务以刷新缓存。

其他 `add_*.sql` 文件用于历史数据库升级或配置项初始化，**不应作为新库的全量脚本依次执行**。部分脚本重复添加 `options` 字段；当前自动迁移也可能已创建这些列，应根据现有表结构选择所需语句。

审计由两部分组成：

- 请求上下文记录请求 ID、请求方法、路径、IP、User-Agent 和操作者；请求 ID 可由 `X-Request-Id` 传入。
- GORM 回调记录业务数据新增、修改、删除的前后快照及字段差异，用户角色和角色菜单关联变更另行记录。

`audit_log` 保存日志主体，`audit_change` 保存数据变更明细。健康检查及操作日志、审计日志查询接口跳过通用请求日志记录。

## 集成与开发

除独立运行外，可通过 `routes.New(db)` 创建路由对象，调用 `RefreshTokenManager(ctx)` 初始化会话管理，再通过 `Init(server)` 注册默认前缀，或通过 `RegisterRoutes(group)` 注册到自己的 Hertz 路由组。`Authentication()` 和 `Authorization()` 可取得业务服务。

`routes.New(db)` 会注册审计回调并触发表结构迁移。部分审计路径判断写死了 `/api/v1`，自定义路由前缀时需要同步检查这些逻辑。

构建和检查命令：

```sh
go build -o hiauthx ./cmd
go test ./...
go vet ./...
```

Windows 下可使用 `go build -o hiauthx.exe ./cmd`。构建产物不会内嵌 YAML 配置，运行时仍需指定可访问的配置目录。

现有测试覆盖 RSA 处理、登录锁定辅助逻辑、存储参数、系统配置校验和 API 缓存等；完整登录与数据管理流程仍需要数据库环境进行联调。

## 当前实现注意事项

- 密码当前使用未加盐 MD5 存储；RSA 只处理登录请求的密码传输。正式部署前应评估密码存储改造，并配置 HTTPS。
- 仓库配置包含环境相关的数据库连接信息和 RSA 文件，部署时应替换数据库凭据、JWT 签名密钥及密钥材料。
- `security.password.min_length` 已定义，但当前 `checkPwdLength` 对非空配置值会回退到 8，更新用户密码也未复用该长度校验，不能视为完整的可配置密码策略。
- 登录与当前用户查询会检查用户启用状态，通用 `CheckLogin` 中间件尚未检查该状态；禁用用户后的既有 Token 处理需要进一步完善。
