# PaaS 应用侧认证基础能力

此版本实现 PaaS Access Token 校验、退出和应用侧浏览器票据登录。浏览器接入、共享事务存储及平台接口契约见 [浏览器登录说明](paas-browser-login.md)。PaaS 服务端由平台项目实现。

## 接入

原有 `routes.New(db)` 保持本地认证。外部认证使用：

```go
client, err := paas.New(paas.Config{
    BaseURL: "https://paas.example.com",
    IssuerID: "company-paas",
    AppCode: "agent-console",
    AppSecretEnv: "PAAS_APP_SECRET",
})
// 检查 err；应用自行实现 authn.IdentityResolver。
router, err := routes.NewWithExternalAuth(db, routes.ExternalAuth{
    Authenticator: client,
    Resolver: resolver,
    Revoker: client,
})
```

不能忽略初始化错误或在失败后回退到本地登录。PaaS 客户端默认要求 HTTPS；内网 HTTP 部署可显式设置 allow-insecure-http: true，默认超时 3 秒，拒绝所有 HTTP 重定向，不缓存有效性结果，不自动重试。

## 接口约定

默认 `POST /open/auth/check` 和 `POST /open/auth/logout`。应用凭证用 HTTP Basic，正文为 `{"accessToken":"..."}`。端点可配置为同源绝对路径，不能指定其他主机。

check 成功：

```json
{"code":0,"data":{"active":true,"allowed":true,"appCode":"agent-console","issuer":"company-paas","subject":"user_123","expiresAt":null,"user":{"username":"zhangsan","displayName":"张三"}}}
```

active=false 表示失效；active=true 时必须提供 allowed、匹配的 appCode、issuer 及非空 subject。到期时间可为 null，否则必须是 RFC3339 时间。allowed=false 表示准入拒绝。

命名错误码 TOKEN_INVALID/TOKEN_EXPIRED/TOKEN_REVOKED/USER_DISABLED → 401，APP_ACCESS_DENIED → 403；应用凭证错误、未知响应、超时、错误 JSON、重定向和上游故障 → 503。不把上游响应原文输出给浏览器。

logout 的 code=0 或已经过期/撤销/无效的 Token 视为幂等完成。仅撤销当前共享 Token，不承诺退出其他设备。响应和日志应禁止缓存及记录主 Token。

## 上下文和本地用户

IdentityResolver 只处理已认证的 Identity，返回本地用户 ID。Router 读取本地用户确认可用状态，并依据本地 role_admin 角色计算应用管理员；外部用户名 admin 不授予管理权限。

authn.FromContext 可获取 Principal，原 Hertz userId/username/isSystemManager 字段继续兼容。业务审计和请求审计使用同一份可信操作人，不再通过外部 Token 查询本地 hitoken。

`/auth/me` 使用 Principal 组装本地菜单，`/auth/logout` 使用远程 Revoker。PaaS 模式不注册密码登录、登录加密配置、RSA 生成，以及用户/部门身份写入接口；管理接口要求显式本地管理员角色。此阶段没有新增前端角色分配接口，角色关联通过部署初始化管理。

user 表新增 identity_source（默认空）。非空表示外部身份投影，这类用户不能走本地密码登录；IdentityResolver 应正确设置来源且不给默认密码。库不自动按用户名关联用户，也不自动授予角色。

## 验证范围

`go test ./...` 覆盖现有测试，以及 TLS 模拟 PaaS 的撤销可见性、无缓存、错误分类、凭证传递、超时、重定向拒绝、退出幂等、路由关闭、审计上下文与外部 admin 名称隔离。真实 PaaS 和业务数据库联调仍由应用负责。
