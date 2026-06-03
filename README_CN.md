# AySub

<div align="center">

[![Go](https://img.shields.io/badge/Go-1.25.7-00ADD8.svg)](https://golang.org/)
[![Vue](https://img.shields.io/badge/Vue-3.4+-4FC08D.svg)](https://vuejs.org/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15+-336791.svg)](https://www.postgresql.org/)
[![Redis](https://img.shields.io/badge/Redis-7+-DC382D.svg)](https://redis.io/)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED.svg)](https://www.docker.com/)

**基于 Sub2API + NewAPI + Grok2API 的三合一 AI API 网关**

[English](README.md) | 中文 | [日本語](README_JA.md)

</div>

> **AySub 是基于 Sub2API 的二次魔改整合版本。** 为兼容现有代码，Go module 与部分旧 systemd/helper 路径当前仍保留 `sub2api` 标识。Docker 镜像、服务、容器、网络和卷默认使用 AySub 命名。
---

## 当前状态

当前项目落点：`/Volumes/llovky/AitchTey/code/AySub`。

主网关、NewAPI 风格运营能力、xAI/Grok 官方 API Key、Grok Cookie 逆向适配器、Console 免费模型适配和生成媒体本地缓存已结合进当前代码库。当前仓库暂未提供独立 AySub 公共演示环境；自部署时请通过初始化向导或 `ADMIN_EMAIL` / `ADMIN_PASSWORD` 创建管理员账号。

## 项目概述

AySub 是一个 AI API 聚合网关：保留 Sub2API 的调度内核，叠加 NewAPI 风格的商用运营体系，并把 Grok2API 的 Grok Cookie 逆向能力改写为 Go 内置适配器。用户通过平台生成的 API Key 调用上游 AI 服务，平台负责鉴权、账号调度、模型定价、计费、额度控制、媒体生成路由和请求转发。

## 核心功能

- **Sub2API 调度内核** - 保留账号池调度、粘性会话、熔断、并发控制、速率限制和 Token 级用量记录主路径。
- **NewAPI 风格运营体系** - 支持用户余额、平台额度、分组、API Key 权限、订单、充值套餐、支付实例、支付回调和运营支付看板。
- **模型与定价管理** - 支持默认模型定价、渠道模型白名单、模型同步候选、渠道定价 UI、Token/图片/按次计费模式，以及用户侧可用渠道和价格展示。
- **多厂商网关** - 支持 OpenAI 兼容、Claude/Anthropic 兼容、Gemini 兼容、Antigravity、OpenAI OAuth/API Key、Claude OAuth/API Key、Gemini OAuth/API Key 和自定义上游渠道。
- **xAI 官方 API Key** - 新增 xAI 平台，支持 Chat Completions、Responses fallback、Anthropic Messages fallback、模型列表、端点统计归因和渠道监控。
- **Grok Cookie 逆向适配器** - Grok Web Cookie 账号支持 Chat Completions、Responses、Anthropic Messages、联网搜索引用、thinking 流、多模态图片上传、图片生成/编辑、视频生成和 quota 查询。
- **Grok Console 免费模型** - 支持 Cookie 账号经 `console.x.ai/v1/responses` 调用 `grok-4.3-console`、`grok-4.3-low/medium/high`、`grok-4.20-*`、`grok-4.20-multi-agent-*`、`grok-build-console`，并转换到 Chat、Responses、Messages 三类入口。
- **生成媒体本地缓存** - Grok 生成图片/视频可落盘到 `DATA_DIR/files/images` 与 `DATA_DIR/files/videos`，并通过 `/v1/files/image`、`/v1/files/video` 读取。
- **内置支付系统** - 支持 EasyPay 易支付、支付宝官方、微信官方、Stripe 等自助充值能力，无需独立部署支付服务（[配置指南](docs/PAYMENT_CN.md)）。
- **管理后台** - Web 界面覆盖用户、账号、渠道、定价、分组、支付、用量、监控、数据管理和系统设置。
- **外部系统集成** - 支持通过 iframe 嵌入外部系统（如工单等），扩展管理后台功能。

## 整合说明

| 模块 | 当前状态 |
|------|----------|
| Sub2API 主体 | 调度、账号池、粘性会话、用量计费、限流和网关路由主路径已保留。 |
| NewAPI 模型/定价 | 已具备主路径：模型定价、渠道白名单、定价同步、可用渠道展示、后台渠道定价。完整独立的 NewAPI 风格“模型广场”产品页尚未作为单独模块完成。 |
| NewAPI 商用运营 | 用户、余额、额度、分组、订单、充值套餐、支付实例、回调和支付看板已具备；真实商户配置仍需生产环境验收。 |
| Grok2API parity | Chat、Responses、Messages、Images、Videos、Usage、模型列表、LiveKit token、LiveKit RTC 代理、Console 免费模型和本地媒体缓存已转 Go。media link/upscale、Masonry/ChatKit/Admin WebUI、WARP/FlareSolverr 防封栈未直接整体并入。 |
| 端到端验证 | 已完成单元测试和前端类型检查；真实 xAI/Grok Cookie/Console/媒体/LiveKit 账号链路仍需上线前逐项验收。 |

## 技术栈

| 组件 | 技术 |
|------|------|
| 后端 | Go 1.25.7, Gin, Ent |
| 前端 | Vue 3.4+, Vite 5+, TailwindCSS |
| 数据库 | PostgreSQL 15+ |
| 缓存/队列 | Redis 7+ |

---

## Nginx 反向代理注意事项

通过 Nginx 反向代理 AySub（或 CRS 服务）并搭配 Codex CLI 使用时，需要在 Nginx 配置的 `http` 块中添加：

```nginx
underscores_in_headers on;
```

Nginx 默认会丢弃名称中含下划线的请求头（如 `session_id`），这会导致多账号环境下的粘性会话功能失效。

---

## 部署方式

### 方式一：脚本安装（上游兼容）

一键安装脚本当前仍跟随上游 Sub2API Release 产物。只有在你明确需要上游兼容二进制布局时才使用；部署当前 AySub 魔改代码请优先使用 Docker Compose 或从当前仓库源码编译。

#### 前置条件

- Linux 服务器（amd64 或 arm64）
- PostgreSQL 15+（已安装并运行）
- Redis 7+（已安装并运行）
- Root 权限

#### 安装步骤

```bash
curl -sSL https://raw.githubusercontent.com/AIAllABOUTYOU/AySub/main/deploy/install.sh | sudo bash
```

脚本会自动：
1. 检测系统架构
2. 下载最新版本
3. 安装二进制文件到 `/opt/sub2api`
4. 创建 systemd 服务
5. 配置系统用户和权限

#### 安装后配置

```bash
# 1. 启动服务
sudo systemctl start sub2api

# 2. 设置开机自启
sudo systemctl enable sub2api

# 3. 在浏览器中打开设置向导
# http://你的服务器IP:8080
```

设置向导将引导你完成：
- 数据库配置
- Redis 配置
- 管理员账号创建

#### 升级

可以直接在 **管理后台** 左上角点击 **检测更新** 按钮进行在线升级。

网页升级功能支持：
- 自动检测新版本
- 一键下载并应用更新
- 支持回滚

#### 常用命令

```bash
# 查看状态
sudo systemctl status sub2api

# 查看日志
sudo journalctl -u sub2api -f

# 重启服务
sudo systemctl restart sub2api

# 卸载
curl -sSL https://raw.githubusercontent.com/AIAllABOUTYOU/AySub/main/deploy/install.sh | sudo bash -s -- uninstall -y
```

---

### 方式二：Docker Compose（AySub 推荐）

使用 Docker Compose 部署 AySub，包含 PostgreSQL 和 Redis 容器。可以直接使用 GitHub 打包好的 GHCR 镜像，也可以从当前仓库源码构建 `aysub:latest`。

#### 前置条件

- Docker 20.10+
- Docker Compose v2+

#### 快速开始

直接使用 GitHub 打包好的镜像：

```bash
# 克隆 AySub
git clone https://github.com/AIAllABOUTYOU/AySub.git

# 准备部署配置
cd AySub/deploy
cp .env.example .env
nano .env
mkdir -p data postgres_data redis_data

# 拉取 GHCR 镜像并启动服务
docker compose -f docker-compose.image.yml pull
docker compose -f docker-compose.image.yml up -d

# 查看日志
docker compose -f docker-compose.image.yml logs -f aysub
```

GitHub Actions 会在推送到 `main`、推送 `v*` 版本标签或手动运行 workflow 时构建 `ghcr.io/aiallaboutyou/aysub:latest`。首次发布后如果外部无法拉取，需要在 GitHub Packages 中把包可见性设为 public。

从当前仓库源码本地构建：

```bash
# 克隆 AySub
git clone https://github.com/AIAllABOUTYOU/AySub.git

# 准备部署配置
cd AySub/deploy
cp .env.example .env
nano .env
mkdir -p data postgres_data redis_data

# 构建并启动服务
docker compose -f docker-compose.local.yml up -d --build

# 查看日志
docker compose -f docker-compose.local.yml logs -f aysub
```

生产环境请先为 `JWT_SECRET`、`TOTP_ENCRYPTION_KEY`、`POSTGRES_PASSWORD` 生成安全值。

#### 手动部署

如果你希望手动配置：

```bash
# 1. 克隆仓库
git clone https://github.com/AIAllABOUTYOU/AySub.git
cd AySub/deploy

# 2. 复制环境配置文件
cp .env.example .env

# 3. 编辑配置（生成安全密码）
nano .env
```

**`.env` 必须配置项：**

```bash
# PostgreSQL 密码（必需）
POSTGRES_PASSWORD=your_secure_password_here

# JWT 密钥（推荐 - 重启后保持用户登录状态）
JWT_SECRET=your_jwt_secret_here

# TOTP 加密密钥（推荐 - 重启后保留双因素认证）
TOTP_ENCRYPTION_KEY=your_totp_key_here

# 可选：管理员账号
ADMIN_EMAIL=admin@example.com
ADMIN_PASSWORD=your_admin_password

# 可选：自定义端口
SERVER_PORT=8080
```

**生成安全密钥：**
```bash
# 生成 JWT_SECRET
openssl rand -hex 32

# 生成 TOTP_ENCRYPTION_KEY
openssl rand -hex 32

# 生成 POSTGRES_PASSWORD
openssl rand -hex 32
```

```bash
# 4. 创建数据目录（本地版）
mkdir -p data postgres_data redis_data

# 5. 构建并启动所有服务
# 选项 A：本地目录版（推荐 - 易于迁移）
docker compose -f docker-compose.local.yml up -d --build

# 选项 B：命名卷版（简单设置）
docker compose up -d --build

# 6. 查看状态
docker compose -f docker-compose.local.yml ps

# 7. 查看日志
docker compose -f docker-compose.local.yml logs -f aysub
```

#### 部署版本对比

| 版本 | 数据存储 | 迁移便利性 | 适用场景 |
|------|---------|-----------|---------|
| **docker-compose.image.yml** | 本地目录 | ✅ 简单（打包整个目录） | 使用 GitHub 打包镜像的生产部署 |
| **docker-compose.local.yml** | 本地目录 | ✅ 简单（打包整个目录） | 生产环境、频繁备份 |
| **docker-compose.yml** | 命名卷 | ⚠️ 需要 docker 命令 | 简单设置 |

**推荐：** 使用 GitHub 打包镜像时选 `docker-compose.image.yml`；需要从本地源码构建时选 `docker-compose.local.yml`。

#### 启用“数据管理”功能（datamanagementd）

如需启用管理后台“数据管理”，需要额外部署宿主机数据管理进程 `datamanagementd`。

关键点：

- 主进程固定探测：`/tmp/sub2api-datamanagement.sock`
- 只有该 Socket 可连通时，数据管理功能才会开启
- Docker 场景需将宿主机 Socket 挂载到容器同路径

详细部署步骤见：`deploy/DATAMANAGEMENTD_CN.md`

#### 访问

在浏览器中打开 `http://你的服务器IP:8080`

如果管理员密码是自动生成的，在日志中查找：
```bash
docker compose -f docker-compose.local.yml logs aysub | grep "admin password"
```

#### 升级

```bash
# 拉取最新 AySub 源码并重新构建/重建容器
git pull
docker compose -f docker-compose.local.yml up -d --build

# 或使用 GitHub 打包镜像更新
docker compose -f docker-compose.image.yml pull aysub
docker compose -f docker-compose.image.yml up -d
```

#### 轻松迁移（本地目录版）

使用 `docker-compose.local.yml` 时，可以轻松迁移到新服务器：

```bash
# 源服务器
docker compose -f docker-compose.local.yml down
cd ..
tar czf aysub-complete.tar.gz AySub/

# 传输到新服务器
scp aysub-complete.tar.gz user@new-server:/path/

# 新服务器
tar xzf aysub-complete.tar.gz
cd AySub/deploy/
docker compose -f docker-compose.local.yml up -d --build
```

#### 常用命令

```bash
# 停止所有服务
docker compose -f docker-compose.local.yml down

# 重启
docker compose -f docker-compose.local.yml restart

# 查看所有日志
docker compose -f docker-compose.local.yml logs -f

# 删除所有数据（谨慎！）
docker compose -f docker-compose.local.yml down
rm -rf data/ postgres_data/ redis_data/
```

---

### 方式三：源码编译

从源码编译安装，适合开发或定制需求。

#### 前置条件

- Go 1.21+
- Node.js 18+
- PostgreSQL 15+
- Redis 7+

#### 编译步骤

```bash
# 1. 克隆仓库
git clone https://github.com/AIAllABOUTYOU/AySub.git
cd AySub

# 2. 安装 pnpm（如果还没有安装）
npm install -g pnpm

# 3. 编译前端
cd frontend
pnpm install
pnpm run build
# 构建产物输出到 ../backend/internal/web/dist/

# 4. 编译后端（嵌入前端）
cd ../backend
go build -tags embed -o aysub ./cmd/server

# 5. 创建配置文件
cp ../deploy/config.example.yaml ./config.yaml

# 6. 编辑配置
nano config.yaml
```

> **注意：** `-tags embed` 参数会将前端嵌入到二进制文件中。不使用此参数编译的程序将不包含前端界面。

**`config.yaml` 关键配置：**

```yaml
server:
  host: "0.0.0.0"
  port: 8080
  mode: "release"

database:
  host: "localhost"
  port: 5432
  user: "postgres"
  password: "your_password"
  dbname: "aysub"

redis:
  host: "localhost"
  port: 6379
  password: ""

jwt:
  secret: "change-this-to-a-secure-random-string"
  expire_hour: 24

default:
  user_concurrency: 5
  user_balance: 0
  api_key_prefix: "sk-"
  rate_multiplier: 1.0
```

### Sora 功能状态（暂不可用）

> ⚠️ 当前 Sora 相关功能因上游接入与媒体链路存在技术问题，暂时不可用。
> 现阶段请勿在生产环境依赖 Sora 能力。
> 文档中的 `gateway.sora_*` 配置仅作预留，待技术问题修复后再恢复可用。

### Sora 媒体签名 URL（功能恢复后可选）

当配置 `gateway.sora_media_signing_key` 且 `gateway.sora_media_signed_url_ttl_seconds > 0` 时，网关会将 Sora 输出的媒体地址改写为临时签名 URL（`/sora/media-signed/...`）。这样无需 API Key 即可在浏览器中直接访问，且具备过期控制与防篡改能力（签名包含 path + query）。

```yaml
gateway:
  # /sora/media 是否强制要求 API Key（默认 false）
  sora_media_require_api_key: false
  # 媒体临时签名密钥（为空则禁用签名）
  sora_media_signing_key: "your-signing-key"
  # 临时签名 URL 有效期（秒）
  sora_media_signed_url_ttl_seconds: 900
```

> 若未配置签名密钥，`/sora/media-signed` 将返回 503。  
> 如需更严格的访问控制，可将 `sora_media_require_api_key` 设为 true，仅允许携带 API Key 的 `/sora/media` 访问。

访问策略说明：
- `/sora/media`：内部调用或客户端携带 API Key 才能下载
- `/sora/media-signed`：外部可访问，但有签名 + 过期控制

`config.yaml` 还支持以下安全相关配置：

- `cors.allowed_origins` 配置 CORS 白名单
- `security.url_allowlist` 配置上游/价格数据/CRS 主机白名单
- `security.url_allowlist.enabled` 可关闭 URL 校验（慎用）
- `security.url_allowlist.allow_insecure_http` 关闭校验时允许 HTTP URL
- `security.url_allowlist.allow_private_hosts` 允许私有/本地 IP 地址
- `security.response_headers.enabled` 可启用可配置响应头过滤（关闭时使用默认白名单）
- `security.csp` 配置 Content-Security-Policy
- `billing.circuit_breaker` 计费异常时 fail-closed
- `server.trusted_proxies` 启用可信代理解析 X-Forwarded-For
- `turnstile.required` 在 release 模式强制启用 Turnstile

**网关防御纵深建议（重点）**

- `gateway.upstream_response_read_max_bytes`：限制非流式上游响应读取大小（默认 `8MB`），用于防止异常响应导致内存放大。
- `gateway.proxy_probe_response_read_max_bytes`：限制代理探测响应读取大小（默认 `1MB`）。
- `gateway.gemini_debug_response_headers`：默认 `false`，仅在排障时短时开启，避免高频请求日志开销。
- `/auth/register`、`/auth/login`、`/auth/login/2fa`、`/auth/send-verify-code` 已提供服务端兜底限流（Redis 故障时 fail-close）。
- 推荐将 WAF/CDN 作为第一层防护，服务端限流与响应读取上限作为第二层兜底；两层同时保留，避免旁路流量与误配置风险。

**⚠️ 安全警告：HTTP URL 配置**

当 `security.url_allowlist.enabled=false` 时，系统默认执行最小 URL 校验，**拒绝 HTTP URL**，仅允许 HTTPS。要允许 HTTP URL（例如用于开发或内网测试），必须显式设置：

```yaml
security:
  url_allowlist:
    enabled: false                # 禁用白名单检查
    allow_insecure_http: true     # 允许 HTTP URL（⚠️ 不安全）
```

**或通过环境变量：**

```bash
SECURITY_URL_ALLOWLIST_ENABLED=false
SECURITY_URL_ALLOWLIST_ALLOW_INSECURE_HTTP=true
```

**允许 HTTP 的风险：**
- API 密钥和数据以**明文传输**（可被截获）
- 易受**中间人攻击 (MITM)**
- **不适合生产环境**

**适用场景：**
- ✅ 开发/测试环境的本地服务器（http://localhost）
- ✅ 内网可信端点
- ✅ 获取 HTTPS 前测试账号连通性
- ❌ 生产环境（仅使用 HTTPS）

**未设置此项时的错误示例：**
```
Invalid base URL: invalid url scheme: http
```

如关闭 URL 校验或响应头过滤，请加强网络层防护：
- 出站访问白名单限制上游域名/IP
- 阻断私网/回环/链路本地地址
- 强制仅允许 TLS 出站
- 在反向代理层移除敏感响应头

```bash
# 6. 运行应用
./aysub
```

#### HTTP/2 (h2c) 与 HTTP/1.1 回退

后端明文端口默认支持 h2c，并保留 HTTP/1.1 回退用于 WebSocket 与旧客户端。浏览器通常不支持 h2c，性能收益主要在反向代理或内网链路。

**反向代理示例（Caddy）：**

```caddyfile
transport http {
	versions h2c h1
}
```

**验证：**

```bash
# h2c prior knowledge
curl --http2-prior-knowledge -I http://localhost:8080/health
# HTTP/1.1 回退
curl --http1.1 -I http://localhost:8080/health
# WebSocket 回退验证（需管理员 token）
websocat -H="Sec-WebSocket-Protocol: sub2api-admin, jwt.<ADMIN_TOKEN>" ws://localhost:8080/api/v1/admin/ops/ws/qps
```

#### 开发模式

```bash
# 后端（支持热重载）
cd backend
go run ./cmd/server

# 前端（支持热重载）
cd frontend
pnpm run dev
```

#### 代码生成

修改 `backend/ent/schema` 后，需要重新生成 Ent + Wire：

```bash
cd backend
go generate ./ent
go generate ./cmd/server
```

---

## 简易模式

简易模式适合个人开发者或内部团队快速使用，不依赖完整 SaaS 功能。

- 启用方式：设置环境变量 `RUN_MODE=simple`
- 功能差异：隐藏 SaaS 相关功能，跳过计费流程
- 安全注意事项：生产环境需同时设置 `SIMPLE_MODE_CONFIRM=true` 才允许启动

---

## Antigravity 使用说明

AySub 支持 [Antigravity](https://antigravity.so/) 账户，授权后可通过专用端点访问 Claude 和 Gemini 模型。

### 专用端点

| 端点 | 模型 |
|------|------|
| `/antigravity/v1/messages` | Claude 模型 |
| `/antigravity/v1beta/` | Gemini 模型 |

### Claude Code 配置示例

```bash
export ANTHROPIC_BASE_URL="http://localhost:8080/antigravity"
export ANTHROPIC_AUTH_TOKEN="sk-xxx"
```

### 混合调度模式

Antigravity 账户支持可选的**混合调度**功能。开启后，通用端点 `/v1/messages` 和 `/v1beta/` 也会调度该账户。

> **⚠️ 注意**：Anthropic Claude 和 Antigravity Claude **不能在同一上下文中混合使用**，请通过分组功能做好隔离。


### 已知问题
在 Claude Code 中，无法自动退出Plan Mode。（正常使用原生Claude Api时，Plan 完成后，Claude Code会弹出弹出选项让用户同意或拒绝Plan。） 
解决办法：shift + Tab，手动退出Plan mode，然后输入内容 告诉 Claude Code 同意或拒绝 Plan
---

## 项目结构

```
AySub/
├── backend/                  # Go 后端服务
│   ├── cmd/server/           # 应用入口
│   ├── internal/             # 内部模块
│   │   ├── config/           # 配置管理
│   │   ├── model/            # 数据模型
│   │   ├── service/          # 业务逻辑
│   │   ├── handler/          # HTTP 处理器
│   │   └── gateway/          # API 网关核心
│   └── resources/            # 静态资源
│
├── frontend/                 # Vue 3 前端
│   └── src/
│       ├── api/              # API 调用
│       ├── stores/           # 状态管理
│       ├── views/            # 页面组件
│       └── components/       # 通用组件
│
└── deploy/                   # 部署文件
    ├── docker-compose.yml    # Docker Compose 配置
    ├── .env.example          # Docker Compose 环境变量
    ├── config.example.yaml   # 二进制部署完整配置文件
    └── install.sh            # 一键安装脚本
```

## 免责声明

> **使用本项目前请仔细阅读：**
>
> :rotating_light: **服务条款风险**: 使用本项目可能违反 Anthropic 的服务条款。请在使用前仔细阅读 Anthropic 的用户协议，使用本项目的一切风险由用户自行承担。
>
> :book: **免责声明**: 本项目仅供技术学习和研究使用，作者不对因使用本项目导致的账户封禁、服务中断或其他损失承担任何责任。

---

## Star History

<a href="https://star-history.com/#AIAllABOUTYOU/AySub&Date">
 <picture>
   <source media="(prefers-color-scheme: dark)" srcset="https://api.star-history.com/svg?repos=AIAllABOUTYOU/AySub&type=Date&theme=dark" />
   <source media="(prefers-color-scheme: light)" srcset="https://api.star-history.com/svg?repos=AIAllABOUTYOU/AySub&type=Date" />
   <img alt="Star History Chart" src="https://api.star-history.com/svg?repos=AIAllABOUTYOU/AySub&type=Date" />
 </picture>
</a>

---

## 许可证

本项目基于 [GNU 宽通用公共许可证 v3.0](LICENSE)（或更高版本）授权。

---

<div align="center">

**如果觉得有用，请给个 Star 支持一下！**

</div>
