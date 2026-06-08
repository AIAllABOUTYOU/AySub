# AySub

<div align="center">

[![Go](https://img.shields.io/badge/Go-1.26.3-00ADD8.svg)](https://golang.org/)
[![Vue](https://img.shields.io/badge/Vue-3.4+-4FC08D.svg)](https://vuejs.org/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15+-336791.svg)](https://www.postgresql.org/)
[![Redis](https://img.shields.io/badge/Redis-7+-DC382D.svg)](https://redis.io/)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED.svg)](https://www.docker.com/)

**All Your AI Sub Hub**

[English](README.md) | 中文 | [日本語](README_JA.md)

</div>

AySub 是一个自部署 AI API 网关，覆盖账号调度、OpenAI 兼容 API、NewAPI 风格运营能力和 Grok/xAI 适配。为保持兼容，Go module path 仍是 `github.com/Wei-Shaw/sub2api`；运行时名称、Docker 镜像、容器、服务、网络和卷使用 AySub 命名。

版权归属：`aiaay.com`。


## 当前能力

当前代码库已实现：

- 多平台账号池调度：Anthropic/Claude、OpenAI、Gemini、xAI/Grok、Antigravity，以及 Anthropic Bedrock / Vertex Service Account；支持账号分组、优先级、负载因子、粘性会话、失败切换、并发控制、临时禁调、过载冷却、限流冷却、额度检查和用量计费。
- API 网关：Claude Messages、OpenAI Chat Completions、Responses、Messages 兼容转发、Embeddings、Images、Videos、Audio、LiveKit；Gemini 原生 `/v1beta`；Antigravity 专用 `/antigravity/v1` 与 `/antigravity/v1beta`。
- API Key 管理：模型白名单、端点权限、分组绑定、状态控制、用户侧创建/更新/删除，管理员侧可调整 Key 分组。
- 渠道与价格体系：渠道 CRUD、模型定价/映射、通配符匹配、分组倍率、RPM override、策略视图、用户可用渠道、模型广场和价格计算器。
- 账号运营：OAuth/API Key/Cookie/Service Account/Bedrock 等账号类型，批量导入导出、CRS 同步、上游模型同步、账号测试、刷新、隐私设置、quota/tier 刷新、错误清理和用量统计。
- 用户系统：邮箱注册登录、验证码、忘记/重置密码、JWT refresh/logout、GitHub/Google/OIDC/钉钉/微信/LinuxDo 登录和绑定、邮箱补全、会话撤销。
- 用户工作台：仪表盘、API Key、用量记录、请求日志、每日签到、用户安全中心、通知邮箱、TOTP 与恢复码、敏感操作二次验证、个人资料、账号绑定、兑换码、订阅、订单、邀请返利和自定义菜单页面。
- 管理后台：仪表盘、用户/分组/账号/代理/公告/设置、主页配置、渠道、渠道监控、定时测试、订阅、用量清理、请求日志、兑换码、优惠码、用户属性、API Key 分组管理。
- 运维监控：实时并发、账号可用性、实时流量、QPS WebSocket、告警规则/事件/静默、错误日志、请求错误、上游错误、请求明细、系统日志、运行时日志配置和指标阈值。
- 安全与风控：安全审计日志、事件、策略、封控对象、哈希链完整性校验、审计导出；内容审核配置、状态、日志、用户解封和 flagged hash 清理。
- 支付与订阅：套餐、订单、余额充值、订阅购买、退款申请/处理、订单重试、支付看板、支付实例、可见支付方式、回调；支持 EasyPay、支付宝、微信支付、Stripe、Airwallex。
- 媒体与文件：生成图片/视频本地缓存，`DATA_DIR/files/images`、`DATA_DIR/files/videos` 可在后台列表、单删、按筛选清理和孤儿文件清理；自定义 Markdown 页面和页面图片从 `DATA_DIR/pages` 提供。
- 数据与系统维护：S3/数据源配置、备份任务、定时备份、备份恢复、系统版本检查、应用更新、回滚和重启接口。
- 部署能力：`/setup` 初始化向导、Docker `AUTO_SETUP`、嵌入式前端、simple/backend mode、可选 Privoxy/FlareSolverr/WARP 代理 compose profile。

README 不把以下内容写成已完成：

- Passkey/WebAuthn。
- 通用自定义 OAuth Provider CRUD；当前已有 GitHub、Google、OIDC、钉钉、微信、LinuxDo 等固定流程。
- 独立于现有模型/渠道/价格数据源的完整模型集合模板和供应商目录。
- Grok2API Masonry 画廊、ChatKit 语音 UI 的完整 parity。
- 真实 xAI/Grok Cookie/Console/媒体/LiveKit 账号和真实商户回调的生产验收。

缺口和验收条件见：

- [NewAPI 功能路线](docs/NEWAPI_FEATURE_ROADMAP_CN.md)
- [安全审计路线](docs/SECURITY_AUDIT_ROADMAP_CN.md)
- [Grok2API parity 路线](docs/GROK2API_PARITY_CN.md)

## 主要页面

| 模块 | 页面 |
| --- | --- |
| 初始化 | `/setup` |
| 首页 | `/home` |
| 登录/注册 | `/login`、`/register` |
| 用户仪表盘 | `/dashboard` |
| API Key | `/keys` |
| 每日签到 | `/checkin` |
| 用户用量 | `/usage` |
| 用户请求日志 | `/request-logs` |
| 用户安全中心 | `/security` |
| 模型广场 | `/models` |
| 可用渠道 | `/available-channels` |
| 渠道状态 | `/monitor` |
| 体验中心 | `/playground`（别名 `/experience`） |
| 订阅与购买 | `/subscriptions`、`/purchase` |
| 订单与支付结果 | `/orders`、`/payment/result`、`/payment/stripe`、`/payment/airwallex` |
| 兑换码 | `/redeem` |
| 邀请返利 | `/affiliate` |
| 用户资料 | `/profile` |
| 自定义页面 | `/custom/:id` |
| 公开状态页 | `/status` |
| 法务文档 | `/legal/:documentId` |
| 管理仪表盘 | `/admin/dashboard` |
| 运维监控 | `/admin/ops` |
| 用户/分组/账号 | `/admin/users`、`/admin/groups`、`/admin/accounts` |
| 渠道与监控 | `/admin/channels/pricing`、`/admin/channels/monitor` |
| 订阅管理 | `/admin/subscriptions` |
| 公告与自定义首页 | `/admin/announcements`、`/admin/home-config` |
| 代理管理 | `/admin/proxies` |
| 风控中心 | `/admin/risk-control` |
| 兑换码/优惠码 | `/admin/redeem`、`/admin/promo-codes` |
| 邀请返利记录 | `/admin/affiliates/invites`、`/admin/affiliates/rebates`、`/admin/affiliates/transfers` |
| 支付后台 | `/admin/orders/dashboard`、`/admin/orders`、`/admin/orders/plans` |
| 管理用量/请求日志 | `/admin/usage`、`/admin/request-logs` |
| 安全审计 | `/admin/security` |
| 媒体缓存 | `/admin/media-cache` |
| 系统设置 | `/admin/settings` |

## 接口

AySub 主要接口按用途分为网关、用户、管理、支付和公开接口。完整契约以 `backend/internal/server/routes/*.go` 为准。

网关接口使用 AySub API Key 鉴权：

- `POST /v1/messages`
- `POST /v1/messages/count_tokens`
- `GET /v1/models`
- `GET /v1/usage`
- `POST /v1/chat/completions`
- `POST /v1/responses`
- `POST /v1/responses/*`
- `GET /v1/responses`
- `POST /v1/embeddings`
- `POST /v1/images/generations`
- `POST /v1/images/edits`
- `POST /v1/videos`
- `GET /v1/videos/{id}`
- `GET /v1/videos/{id}/content`
- `POST /v1/audio/speech`
- `POST /v1/audio/transcriptions`
- `POST /v1/audio/translations`
- `POST /v1/livekit/tokens`
- `GET /v1/livekit/rtc`
- `GET /v1/files/image?id=...`
- `GET /v1/files/video?id=...`
- `GET /v1beta/models`
- `GET /v1beta/models/{model}`
- `POST /v1beta/models/*`
- `POST /responses`、`POST /chat/completions` 等无 `/v1` 兼容别名
- `POST /backend-api/codex/responses`、`GET /backend-api/codex/responses`
- `GET /antigravity/models`
- `POST /antigravity/v1/messages`
- `POST /antigravity/v1/messages/count_tokens`
- `GET /antigravity/v1/models`
- `GET /antigravity/v1/usage`
- `GET /antigravity/v1beta/models`
- `GET /antigravity/v1beta/models/{model}`
- `POST /antigravity/v1beta/models/*`

用户、管理、支付和公开接口示例：

- 认证：`/api/v1/auth/register`、`/api/v1/auth/login`、`/api/v1/auth/login/2fa`、`/api/v1/auth/refresh`、`/api/v1/auth/logout`、`/api/v1/auth/forgot-password`、`/api/v1/auth/reset-password`
- OAuth：`/api/v1/auth/oauth/{github,google,linuxdo,wechat,oidc,dingtalk}/start` 与对应 callback / complete / bind / create-account 流程
- 用户：`/api/v1/user/profile`、`/api/v1/user/platform-quotas`、`/api/v1/user/checkin`、`/api/v1/user/security/events`、`/api/v1/user/totp/*`、`/api/v1/user/notify-email/*`
- API Key 与用量：`/api/v1/keys`、`/api/v1/groups/available`、`/api/v1/usage`、`/api/v1/usage/requests`、`/api/v1/usage/dashboard/*`
- 用户渠道与体验中心：`/api/v1/channels/available`、`/api/v1/channel-monitors`、`/api/v1/playground/sessions`
- 公告、兑换、订阅：`/api/v1/announcements`、`/api/v1/redeem`、`/api/v1/subscriptions/*`
- 管理核心：`/api/v1/admin/dashboard/*`、`/api/v1/admin/users/*`、`/api/v1/admin/groups/*`、`/api/v1/admin/accounts/*`、`/api/v1/admin/proxies/*`、`/api/v1/admin/settings/*`
- 管理渠道：`/api/v1/admin/channels/*`、`/api/v1/admin/channel-monitors/*`、`/api/v1/admin/channel-monitor-templates/*`、`/api/v1/admin/scheduled-test-plans/*`
- 管理运营：`/api/v1/admin/ops/*`、`/api/v1/admin/ops/requests`、`/api/v1/admin/usage/*`、`/api/v1/admin/media-cache/*`
- 管理安全/风控：`/api/v1/admin/security/*`、`/api/v1/admin/risk-control/*`、`/api/v1/admin/error-passthrough-rules/*`、`/api/v1/admin/tls-fingerprint-profiles/*`
- 管理商业化：`/api/v1/admin/subscriptions/*`、`/api/v1/admin/payment/*`、`/api/v1/admin/redeem-codes/*`、`/api/v1/admin/promo-codes/*`、`/api/v1/admin/affiliates/*`
- 数据与系统：`/api/v1/admin/data-management/*`、`/api/v1/admin/backups/*`、`/api/v1/admin/system/*`
- 支付：`/api/v1/payment/config`、`/api/v1/payment/plans`、`/api/v1/payment/orders/*`、`/api/v1/payment/public/orders/*`、`/api/v1/payment/webhook/{easypay,alipay,wxpay,stripe,airwallex}`
- 公开：`GET /health`、`GET /setup/status`、`GET /api/v1/settings/public`、`GET /api/v1/status/public`、`GET /api/v1/pages/:slug`、`GET /api/v1/pages/:slug/images/*`

## Docker

GitHub Actions 发布镜像到：

```bash
ghcr.io/aiallaboutyou/aysub:latest
```

Docker 镜像触发条件：

- 推送到 `main`：发布 `ghcr.io/aiallaboutyou/aysub:latest`
- 推送 `v*` 标签：发布对应的 GHCR 版本标签
- 手动运行 Docker workflow

workflow 使用根目录 [`Dockerfile`](Dockerfile)。`deploy/` 下的 compose 文件也从仓库根目录 Dockerfile 构建。

GitHub Release 触发条件：

- 推送 `v0.1.134` 这类版本标签
- 或手动运行 `Release` workflow，并填写一个已存在的 `v*` 标签

普通提交到 `main` 不会创建 GitHub Release。Release 会上传 `aysub_<version>_<os>_<arch>.tar.gz` 二进制包和 `checksums.txt`，供二进制安装脚本和应用内检查更新使用。

```bash
git tag -a v0.1.134 -m "AySub v0.1.134"
git push origin v0.1.134
```

使用 GHCR 镜像：

```bash
git clone https://github.com/AIAllABOUTYOU/AySub.git
cd AySub/deploy
cp .env.example .env
nano .env
mkdir -p data postgres_data redis_data
docker compose -f docker-compose.image.yml pull
docker compose -f docker-compose.image.yml up -d
```

本地源码构建：

```bash
git clone https://github.com/AIAllABOUTYOU/AySub.git
cd AySub/deploy
cp .env.example .env
nano .env
mkdir -p data postgres_data redis_data
docker compose -f docker-compose.local.yml up -d --build
```

生产环境至少配置：

```bash
POSTGRES_PASSWORD=replace-with-random-password
JWT_SECRET=replace-with-openssl-rand-hex-32
TOTP_ENCRYPTION_KEY=replace-with-openssl-rand-hex-32
ADMIN_EMAIL=admin@example.com
ADMIN_PASSWORD=replace-with-admin-password
SERVER_PORT=8080
```

Docker 自动初始化时如果 `ADMIN_PASSWORD` 留空，AySub 会生成管理员密码并写入容器日志。

### Grok WARP / FlareSolverr 可选防封栈

Grok Cookie 账号遇到 Cloudflare challenge 时，可启用 FlareSolverr 自动刷新 `cf_clearance`；WARP 可作为 Grok Cookie 请求的 SOCKS5 出口代理。两者都定义在 `deploy/docker-compose.proxy-profiles.yml`，详细部署说明见 [`deploy/DOCKER.md`](deploy/DOCKER.md)。

同时启用 FlareSolverr 和 WARP：

```bash
cd deploy
GROK_FLARESOLVERR_URL=http://flaresolverr:8191 \
docker compose -f docker-compose.image.yml -f docker-compose.proxy-profiles.yml --profile flaresolverr --profile warp up -d
```

启用 WARP 后，在后台 IP 管理新增代理：

```text
socks5://warp:1080
```

再把该代理绑定到 Grok Cookie 账号。AySub 调用 FlareSolverr 时会复用账号绑定的代理，确保 clearance cookie 与实际 Grok 请求使用同一出口。这个栈只处理 Cloudflare challenge 和出口 IP 问题；如果是 token 失效、账号风控、地区限制或上游拒绝访问，仍需更换账号、Cookie 或代理。

## 从 Sub2API 迁移

AySub 按 Sub2API 的向前兼容升级来处理。原来的用户、API Key、分组、账号、余额、用量日志、settings、PostgreSQL 数据、Redis 数据和 `DATA_DIR` 文件可以继续使用。

已有 Docker 部署的升级步骤：

1. 备份 PostgreSQL 和运行时数据目录。
2. 停掉旧的 Sub2API 应用容器。
3. 保留原 `.env`、PostgreSQL volume、Redis volume 和 `DATA_DIR` volume。
4. 只把应用镜像或 compose 源码构建目标换成 AySub。
5. 启动 AySub。新增迁移和 settings key 会在启动时补齐。

数据库备份示例：

```bash
pg_dump -h 127.0.0.1 -U sub2api -d sub2api > sub2api_backup.sql
```

如果原部署使用 Docker volume，继续把 `postgres_data`、`redis_data`、`data` 挂载到 AySub compose 中。已有数据库不要重新跑 `/setup` 初始化；原管理员账号继续有效。

兼容边界：

- `home_content` 继续兼容。它非空时，`/home` 仍优先展示这段自定义 HTML 或 iframe URL，不会展示 AySub 新增的结构化 `home_config`。
- AySub 会新增表、字段和 settings。升级后不建议把同一份数据库直接回退到旧 Sub2API 版本。
- Go module path 保持 `github.com/Wei-Shaw/sub2api` 只是为了代码和迁移兼容；运行时镜像、容器、服务、网络和卷使用 AySub 命名。

## 源码构建

```bash
git clone https://github.com/AIAllABOUTYOU/AySub.git
cd AySub/frontend
pnpm install
pnpm run build

cd ../backend
go build -tags embed -o aysub ./cmd/server
./aysub
```

本地开发：

```bash
cd backend
go run ./cmd/server

cd ../frontend
pnpm run dev
```

修改 Ent schema 或 Wire provider 后：

```bash
cd backend
go generate ./ent
go generate ./cmd/server
```

## 配置注意

- `DATA_DIR` 控制本地媒体缓存和运行时数据，默认 `./data`。
- `RUN_MODE=simple` 会隐藏 SaaS 相关页面并跳过计费流程；生产环境还需要 `SIMPLE_MODE_CONFIRM=true`。
- 使用 `X-Forwarded-For` 前应配置 `server.trusted_proxies`。
- Nginx 反代需要设置 `underscores_in_headers on;`，否则 `session_id` 这类请求头会被丢弃。
- `security.url_allowlist` 及对应环境变量控制上游 URL 校验。
- `billing.circuit_breaker` 控制计费写入异常时是否 fail closed。
- `turnstile.required` 可在 release 模式强制启用 Turnstile。

## 验证记录

已在当前工作区通过：

```bash
cd backend && go test ./...
pnpm --dir frontend test:run
pnpm --dir frontend build
```

`docker build -t aysub:local-verify .` 未在本机执行：当前工作区没有安装 `docker` 命令（`zsh: command not found: docker`）。根 Dockerfile 路径和 compose 引用已做静态核对。

## 许可证

许可证：[GNU Lesser General Public License v3.0](LICENSE) 或更高版本。

版权归属：`aiaay.com`。

# 🎉致谢

本项目在 [LINUX DO](https://linux.do/) 社区推广，感谢 LINUX DO 社区对开源项目的支持与认可。 学 AI 上 L 站
