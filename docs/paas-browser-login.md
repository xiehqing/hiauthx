# 应用侧浏览器票据登录

hiauthx/paas 提供 LoginFlow；应用注入共享 LoginStore，并注册 Start、Callback、Bootstrap 三个 Hertz 处理器。用户仍使用 PaaS 原始 Access Token，不签发应用 Token。

## 配置和部署

在 Config 中设置 callback-url，并可覆盖 endpoints.authorize、endpoints.exchange。PaaS 服务和应用浏览器入口默认要求 HTTPS；配置 allow-insecure-http: true 后也支持内网 HTTP。应用前端与 API 仍需同源。示例回调为 `https://agent.example.com/api/v1/auth/paas/callback`。

Client 复用现有 app-code/app-secret-env。LoginFlow 使用应用密钥通过 HMAC-SHA256 派生 AES-GCM 密钥，加密临时存储内容；不同副本需使用一致的应用配置与密钥。密钥轮换会使尚未完成的交接失效，用户重新发起即可，不影响已交接 Token 的 PaaS 有效性判定。

HTTPS 回调使用 __Host- 前缀和 Secure Cookie；显式允许 HTTP 且回调使用 HTTP 时，使用普通 hiauthx- 前缀并关闭 Secure。两者都保留 HttpOnly、SameSite=Lax、Path=/，不设置 Domain。Cookie 属性取决于配置的外部回调地址，不取决于反向代理到后端的协议。HTTP 会明文传输 Token、票据及应用凭证；开启此配置不关闭 HTTPS 证书验证。

## PaaS 服务端契约

### 1. 授权入口

浏览器 GET 默认 `/open/apps/authorize`，查询参数为：

| 参数 | 说明 |
| --- | --- |
| app_code | 当前应用 |
| redirect_uri | 精确登记的回调地址 |
| state | 应用生成的随机登录事务标识 |
| code_challenge | SHA-256(code_verifier) 的无填充 base64url 值 |
| code_challenge_method | 固定 S256 |

PaaS 按自己的认证方式确认登录及应用准入，生成短期一次性票据，绑定原 Token、应用、回调及挑战值，然后跳转 `redirect_uri?ticket=...&state=...`。state 原样返回；任何主 Token 不得拼进 URL。

### 2. 票据兑换

应用后端 POST 默认 `/open/apps/tickets/exchange`，HTTP Basic 为应用标识和密钥，JSON 正文：

```json
{"ticket":"一次性票据","callbackUrl":"https://agent.example.com/api/v1/auth/paas/callback","codeVerifier":"登录事务中的随机证明"}
```

PaaS 验证应用身份、回调、挑战、票据时效及单次消费，重新检查原 Token 和准入，成功返回：

```json
{"code":0,"data":{"accessToken":"原 PaaS Access Token"}}
```

错误码 TICKET_INVALID/TICKET_EXPIRED/TICKET_USED 表示需重新登录；APP_ACCESS_DENIED 表示无准入；其他 Token 错误遵循客户端校验契约。未识别响应和网络错误作为服务故障处理。应用拿到 Token 后还会调用 check 校验，不仅依赖 exchange 响应中的用户信息。

### 3. Token 校验和退出

继续使用 [客户端契约](paas-client.md)。每次业务请求都校验，不缓存有效状态。

## 事务与交接

- Start：创建浏览器绑定的登录事务，保存 state 关联的随机 verifier、站内返回路径，TTL 为 5 分钟，允许用户完成 PaaS 交互登录。票据自己的 TTL 由 PaaS 管理，建议 60 秒。
- Callback：验证 state 与浏览器 Cookie 对应的事务，原子消费后调用 exchange。失败返回固定错误码页面，不回显票据或上游正文。
- 兑换成功：创建新的浏览器绑定交接记录，TTL 为 30 秒，跳转到同源 `/?paas_login=1`。
- Bootstrap：前端同源 POST，携带 `X-PaaS-Bootstrap: 1`，浏览器自然附带 Origin 和 HttpOnly Cookie。服务端严格比对 Origin，原子消费记录，并再次校验 Token，然后返回 accessToken 和 returnTo。
- 前端调用 me 取得用户和菜单，通过后再进入业务页。Token 只在页面内存中保存，刷新后需要重新发起 PaaS 接入。

LoginStore 的 Take 必须跨副本原子删除并返回未过期记录，错误使用 ErrLoginRecordMissing。Put/Take 保存的是加密字节；数据库不能明文保存 Token。应用提供的 MySQL 实现使用行锁和事务消费；所有节点共享数据库。

同一应用每个浏览器支持一个正在进行的登录事务；重新发起将替换 Cookie，旧事务仅保留到 TTL 清理。原子消费后的上游或网络失败不可重试旧票据/旧交接，用户需重新发起登录。

## 日志和缓存

登录、回调、bootstrap 以及 me 响应均禁止缓存。回调设置 no-referrer；网关和 APM 仍需对 callback 查询、exchange/check/logout 正文及 Cookie/Authorization 脱敏，不能记录完整票据和 Token。

测试通过 TLS 模拟 PaaS 覆盖挑战绑定、正确回调、跨浏览器拒绝、跨 Origin 拒绝、重复及并发消费、过期和篡改、交接前 Token 撤销、原 Token 一致性和存储密文。真实 MySQL 多副本及平台对接需部署环境联调。
