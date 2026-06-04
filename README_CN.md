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

## 当前范围

当前代码库已实现：

- 账号池调度、粘性会话、失败切换、并发控制、限流、额度检查和用量计费。
- OpenAI 兼容 `/v1/chat/completions`、`/v1/responses`、`/v1/messages`、`/v1/embeddings`、`/v1/images/*`、`/v1/videos`、`/v1/audio/*`、LiveKit 与 Gemini 原生路由。
- API Key 级模型和端点权限，权限在后端网关中间件强制执行。
- 用户和管理员请求日志中心、用量报表和运营统计。
- `/models` 模型广场、价格计算器、可用渠道视图和后台渠道策略视图。
- `/playground` 体验中心：使用用户自己的 API Key 体验聊天、图片生成、视频任务和音频 speech/transcription/translation。
- `/status` 公开状态页，对应匿名只读 `/api/v1/status/public`。
- `/setup` 初始化向导；Docker 下支持 `AUTO_SETUP` 和 `ADMIN_EMAIL` / `ADMIN_PASSWORD`。
- 每日签到，配置项为 `checkin_enabled` 和 `checkin_reward_amount`。
- TOTP 恢复码、敏感操作二次验证、用户侧 API Key 撤销。
- 注册控制：注册开关、邮箱后缀白名单、邮箱域名黑名单、邮箱别名限制。
- 安全审计中心：审计日志、策略规则、封控对象、哈希链完整性校验。
- 内容审核/风控中心，审核事件接入审计和日志链路。
- 内置支付：套餐、订单、支付实例、回调、退款和支付运营指标。
- xAI 官方 API Key；Grok Cookie/Console 适配聊天、Responses、Messages、图片、视频、LiveKit token/RTC、quota 和模型列表。
- 生成媒体本地缓存：`DATA_DIR/files/images`、`DATA_DIR/files/videos`；后台支持列表、单删、按筛选清理和孤儿文件清理。
- 可选代理 compose profile：`deploy/docker-compose.proxy-profiles.yml` 包含 Privoxy、FlareSolverr、WARP。

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
| API Key | `/keys` |
| 每日签到 | `/checkin` |
| 用户请求日志 | `/request-logs` |
| 用户安全 | `/user/security` |
| 模型广场 | `/models` |
| 体验中心 | `/playground` |
| 公开状态页 | `/status` |
| 管理请求日志 | `/admin/request-logs` |
| 管理安全中心 | `/admin/security` |
| 媒体缓存 | `/admin/media-cache` |
| 风控中心 | `/admin/risk-control` |
| 支付看板 | `/admin/orders/dashboard` |

## 接口

网关接口使用 AySub API Key 鉴权：

- `POST /v1/chat/completions`
- `POST /v1/responses`
- `POST /v1/responses/*`
- `GET /v1/responses`
- `POST /v1/messages`
- `GET /v1/models`
- `GET /v1/usage`
- `POST /v1/embeddings`
- `POST /v1/images/generations`
- `POST /v1/images/edits`
- `POST /v1/videos`
- `GET /v1/videos/{id}`
- `GET /v1/videos/{id}/content`
- `POST /v1/audio/speech`
- `POST /v1/audio/transcriptions`
- `POST /v1/audio/translations`
- `GET /v1/files/image?id=...`
- `GET /v1/files/video?id=...`

本轮已落地的用户/管理接口：

- `GET /api/v1/user/checkin`
- `GET /api/v1/user/checkin/status`
- `POST /api/v1/user/checkin`
- `GET /api/v1/user/security/events`
- `POST /api/v1/user/security/verify-sensitive-operation`
- `POST /api/v1/user/security/api-keys/:id/revoke`
- `GET /api/v1/admin/security/audit-logs`
- `GET /api/v1/admin/security/incidents`
- `GET /api/v1/admin/security/policies`
- `POST /api/v1/admin/security/policies`
- `PUT /api/v1/admin/security/policies/:id`
- `DELETE /api/v1/admin/security/policies/:id`
- `GET /api/v1/admin/security/locks`
- `POST /api/v1/admin/security/locks`
- `POST /api/v1/admin/security/locks/:id/unlock`
- `GET /api/v1/admin/security/integrity/check`
- `GET /api/v1/admin/media-cache`
- `POST /api/v1/admin/media-cache/cleanup`
- `POST /api/v1/admin/media-cache/orphans/cleanup`
- `DELETE /api/v1/admin/media-cache/:type/:id`
- `GET /api/v1/status/public`

## Docker

GitHub Actions 发布镜像到：

```bash
ghcr.io/aiallaboutyou/aysub:latest
```

触发条件：

- 推送到 `main`
- 推送 `v*` 标签
- 手动运行 workflow

workflow 使用根目录 [`Dockerfile`](Dockerfile)。`deploy/` 下的 compose 文件也从仓库根目录 Dockerfile 构建。

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
