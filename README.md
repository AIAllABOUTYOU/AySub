# AySub

<div align="center">

[![Go](https://img.shields.io/badge/Go-1.26.3-00ADD8.svg)](https://golang.org/)
[![Vue](https://img.shields.io/badge/Vue-3.4+-4FC08D.svg)](https://vuejs.org/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15+-336791.svg)](https://www.postgresql.org/)
[![Redis](https://img.shields.io/badge/Redis-7+-DC382D.svg)](https://redis.io/)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED.svg)](https://www.docker.com/)

**All Your AI Sub Hub**

English | [中文](README_CN.md) | [日本語](README_JA.md)

</div>

AySub is an AI API gateway for self-hosted account scheduling, OpenAI-compatible API access, NewAPI-style operations, and Grok/xAI adapters. The Go module path remains `github.com/Wei-Shaw/sub2api` for compatibility; runtime naming, Docker images, containers, services, networks, and volumes use AySub names.

Copyright: `aiaay.com`.


## Current Capabilities

Implemented in the current codebase:

- Multi-platform account pool scheduling for Anthropic/Claude, OpenAI, Gemini, xAI/Grok, Antigravity, and Anthropic Bedrock / Vertex Service Account accounts, with groups, priority, load factor, sticky sessions, failover, concurrency controls, temporary unscheduling, overload/rate-limit cooldowns, quota checks, and usage billing.
- API gateway support for Claude Messages, OpenAI Chat Completions, Responses, Messages-compatible forwarding, Embeddings, Images, Videos, Audio, LiveKit, Gemini native `/v1beta`, and dedicated Antigravity `/antigravity/v1` plus `/antigravity/v1beta` routes.
- API Key management with model allowlists, endpoint permissions, group binding, status control, user CRUD, and admin-side group reassignment.
- Channel and pricing system with channel CRUD, model pricing/mapping, wildcard matching, group multipliers, RPM overrides, strategy views, user available channels, model marketplace, and price calculator.
- Account operations for OAuth/API Key/Cookie/Service Account/Bedrock account types, batch import/export, CRS sync, upstream model sync, account tests, refresh, privacy setup, quota/tier refresh, error cleanup, and usage stats.
- User system with email registration/login, verification codes, forgot/reset password, JWT refresh/logout, GitHub/Google/OIDC/DingTalk/WeChat/LinuxDo login and binding, email completion, and session revocation.
- User workspace with dashboard, API Keys, usage records, request logs, daily check-in, security center, notification email, TOTP and recovery codes, sensitive-operation verification, profile, account bindings, redeem codes, subscriptions, orders, affiliate rebates, and custom menu pages.
- Admin console for dashboard, users, groups, accounts, proxies, announcements, settings, home configuration, channels, channel monitors, scheduled tests, subscriptions, usage cleanup, request logs, redeem codes, promo codes, user attributes, and API Key group management.
- Ops monitoring with realtime concurrency, account availability, realtime traffic, QPS WebSocket, alert rules/events/silences, error logs, request errors, upstream errors, request drilldown, system logs, runtime log config, and metric thresholds.
- Security and risk control with audit logs, incidents, policies, subject locks, hash-chain integrity checks, audit exports, content moderation config/status/logs, user unban, and flagged-hash cleanup.
- Payment and subscription system with plans, balance recharge, subscription purchase, orders, refund request/processing, order retry, payment dashboard, provider instances, visible payment methods, callbacks, and EasyPay, Alipay, WeChat Pay, Stripe, and Airwallex providers.
- Media and files: generated image/video local cache under `DATA_DIR/files/images` and `DATA_DIR/files/videos`, admin listing/deletion/filtered cleanup/orphan cleanup, and custom Markdown pages plus images served from `DATA_DIR/pages`.
- Data and system maintenance with S3/source profiles, backup jobs, scheduled backups, backup restore, system version check, app update, rollback, and restart endpoints.
- Deployment features: `/setup` wizard, Docker `AUTO_SETUP`, embedded frontend, simple/backend modes, and optional Privoxy/FlareSolverr/WARP proxy compose profiles.

Not claimed as complete in README:

- Passkey/WebAuthn.
- Generic custom OAuth Provider CRUD beyond existing GitHub, Google, OIDC, DingTalk, WeChat, and LinuxDo flows.
- Model collection templates and a full provider catalog separate from the existing model/channel/pricing data.
- Full Masonry gallery and ChatKit voice UI parity from Grok2API.
- Production validation with real xAI/Grok Cookie/Console/media/LiveKit accounts and real merchant callbacks.

Those items are tracked in:

- [NewAPI feature roadmap](docs/NEWAPI_FEATURE_ROADMAP_CN.md)
- [Security audit roadmap](docs/SECURITY_AUDIT_ROADMAP_CN.md)
- [Grok2API parity roadmap](docs/GROK2API_PARITY_CN.md)

## Key Pages

| Area | Page |
| --- | --- |
| Setup | `/setup` |
| Home | `/home` |
| Login/Register | `/login`, `/register` |
| User dashboard | `/dashboard` |
| API Keys | `/keys` |
| Daily check-in | `/checkin` |
| User usage | `/usage` |
| User request logs | `/request-logs` |
| User security center | `/security` |
| Model marketplace | `/models` |
| Available channels | `/available-channels` |
| Channel status | `/monitor` |
| Experience center | `/playground` (alias `/experience`) |
| Subscriptions and purchase | `/subscriptions`, `/purchase` |
| Orders and payment result | `/orders`, `/payment/result`, `/payment/stripe`, `/payment/airwallex` |
| Redeem codes | `/redeem` |
| Affiliate | `/affiliate` |
| Profile | `/profile` |
| Custom pages | `/custom/:id` |
| Public status | `/status` |
| Legal documents | `/legal/:documentId` |
| Admin dashboard | `/admin/dashboard` |
| Ops monitoring | `/admin/ops` |
| Users/groups/accounts | `/admin/users`, `/admin/groups`, `/admin/accounts` |
| Channels and monitors | `/admin/channels/pricing`, `/admin/channels/monitor` |
| Subscription admin | `/admin/subscriptions` |
| Announcements and home config | `/admin/announcements`, `/admin/home-config` |
| Proxy admin | `/admin/proxies` |
| Risk control | `/admin/risk-control` |
| Redeem/promo codes | `/admin/redeem`, `/admin/promo-codes` |
| Affiliate records | `/admin/affiliates/invites`, `/admin/affiliates/rebates`, `/admin/affiliates/transfers` |
| Payment admin | `/admin/orders/dashboard`, `/admin/orders`, `/admin/orders/plans` |
| Admin usage/request logs | `/admin/usage`, `/admin/request-logs` |
| Security audit | `/admin/security` |
| Media cache | `/admin/media-cache` |
| System settings | `/admin/settings` |

## API

AySub APIs are grouped into gateway, user, admin, payment, and public surfaces. The source of truth is `backend/internal/server/routes/*.go`.

Gateway routes are authenticated with AySub API Keys:

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
- `/responses`, `/chat/completions`, and other no-`/v1` compatibility aliases
- `POST /backend-api/codex/responses`, `GET /backend-api/codex/responses`
- `GET /antigravity/models`
- `POST /antigravity/v1/messages`
- `POST /antigravity/v1/messages/count_tokens`
- `GET /antigravity/v1/models`
- `GET /antigravity/v1/usage`
- `GET /antigravity/v1beta/models`
- `GET /antigravity/v1beta/models/{model}`
- `POST /antigravity/v1beta/models/*`

Examples from the user, admin, payment, and public APIs:

- Auth: `/api/v1/auth/register`, `/api/v1/auth/login`, `/api/v1/auth/login/2fa`, `/api/v1/auth/refresh`, `/api/v1/auth/logout`, `/api/v1/auth/forgot-password`, `/api/v1/auth/reset-password`
- OAuth: `/api/v1/auth/oauth/{github,google,linuxdo,wechat,oidc,dingtalk}/start` plus callback / complete / bind / create-account flows
- User: `/api/v1/user/profile`, `/api/v1/user/platform-quotas`, `/api/v1/user/checkin`, `/api/v1/user/security/events`, `/api/v1/user/totp/*`, `/api/v1/user/notify-email/*`
- API Keys and usage: `/api/v1/keys`, `/api/v1/groups/available`, `/api/v1/usage`, `/api/v1/usage/requests`, `/api/v1/usage/dashboard/*`
- User channels and playground: `/api/v1/channels/available`, `/api/v1/channel-monitors`, `/api/v1/playground/sessions`
- Announcements, redeem, subscriptions: `/api/v1/announcements`, `/api/v1/redeem`, `/api/v1/subscriptions/*`
- Admin core: `/api/v1/admin/dashboard/*`, `/api/v1/admin/users/*`, `/api/v1/admin/groups/*`, `/api/v1/admin/accounts/*`, `/api/v1/admin/proxies/*`, `/api/v1/admin/settings/*`
- Admin channels: `/api/v1/admin/channels/*`, `/api/v1/admin/channel-monitors/*`, `/api/v1/admin/channel-monitor-templates/*`, `/api/v1/admin/scheduled-test-plans/*`
- Admin operations: `/api/v1/admin/ops/*`, `/api/v1/admin/ops/requests`, `/api/v1/admin/usage/*`, `/api/v1/admin/media-cache/*`
- Admin security/risk: `/api/v1/admin/security/*`, `/api/v1/admin/risk-control/*`, `/api/v1/admin/error-passthrough-rules/*`, `/api/v1/admin/tls-fingerprint-profiles/*`
- Admin monetization: `/api/v1/admin/subscriptions/*`, `/api/v1/admin/payment/*`, `/api/v1/admin/redeem-codes/*`, `/api/v1/admin/promo-codes/*`, `/api/v1/admin/affiliates/*`
- Data and system: `/api/v1/admin/data-management/*`, `/api/v1/admin/backups/*`, `/api/v1/admin/system/*`
- Payment: `/api/v1/payment/config`, `/api/v1/payment/plans`, `/api/v1/payment/orders/*`, `/api/v1/payment/public/orders/*`, `/api/v1/payment/webhook/{easypay,alipay,wxpay,stripe,airwallex}`
- Public: `GET /health`, `GET /setup/status`, `GET /api/v1/settings/public`, `GET /api/v1/status/public`, `GET /api/v1/pages/:slug`, `GET /api/v1/pages/:slug/images/*`

## Docker

GitHub Actions publishes the image to:

```bash
ghcr.io/aiallaboutyou/aysub:latest
```

Docker image trigger:

- push to `main`: publishes `ghcr.io/aiallaboutyou/aysub:latest`
- `v*` tags: publishes the matching GHCR version tag
- manual Docker workflow dispatch

The workflow uses the root [`Dockerfile`](Dockerfile). Compose files in `deploy/` also build from the repository root Dockerfile.

GitHub Release trigger:

- push a version tag such as `v0.1.134`
- or run the `Release` workflow manually with an existing `v*` tag

Normal commits to `main` do not create GitHub Releases. A Release uploads `aysub_<version>_<os>_<arch>.tar.gz` binaries and `checksums.txt`, which are used by the binary installer and the in-app update check.

```bash
git tag -a v0.1.134 -m "AySub v0.1.134"
git push origin v0.1.134
```

Use the prebuilt image:

```bash
git clone https://github.com/AIAllABOUTYOU/AySub.git
cd AySub/deploy
cp .env.example .env
nano .env
mkdir -p data postgres_data redis_data
docker compose -f docker-compose.image.yml pull
docker compose -f docker-compose.image.yml up -d
```

Build locally:

```bash
git clone https://github.com/AIAllABOUTYOU/AySub.git
cd AySub/deploy
cp .env.example .env
nano .env
mkdir -p data postgres_data redis_data
docker compose -f docker-compose.local.yml up -d --build
```

Minimum production values in `.env`:

```bash
POSTGRES_PASSWORD=replace-with-random-password
JWT_SECRET=replace-with-openssl-rand-hex-32
TOTP_ENCRYPTION_KEY=replace-with-openssl-rand-hex-32
ADMIN_EMAIL=admin@example.com
ADMIN_PASSWORD=replace-with-admin-password
SERVER_PORT=8080
```

If `ADMIN_PASSWORD` is empty during Docker auto setup, AySub generates one and writes it to container logs.

### Optional Grok WARP / FlareSolverr Stack

For Grok Cookie accounts, FlareSolverr can refresh `cf_clearance` when a Cloudflare challenge is detected, and WARP can provide a SOCKS5 egress proxy for Grok requests. Both services are defined in `deploy/docker-compose.proxy-profiles.yml`; see [`deploy/DOCKER.md`](deploy/DOCKER.md) for the detailed deployment notes.

Enable FlareSolverr and WARP together:

```bash
cd deploy
GROK_FLARESOLVERR_URL=http://flaresolverr:8191 \
docker compose -f docker-compose.image.yml -f docker-compose.proxy-profiles.yml --profile flaresolverr --profile warp up -d
```

After WARP is running, create an active proxy in the admin proxy page:

```text
socks5://warp:1080
```

Bind that proxy to the Grok Cookie account. AySub passes the account proxy to FlareSolverr so the clearance cookie is solved from the same egress path used by the Grok request. This stack only helps with Cloudflare challenges and egress IP issues; expired tokens, account risk controls, regional restrictions, or upstream access denial still require a valid account, Cookie, or proxy.

## Migrating From Sub2API

AySub is designed as a forward-compatible upgrade from existing Sub2API deployments. Users, API Keys, groups, accounts, balances, usage logs, settings, PostgreSQL data, Redis data, and `DATA_DIR` files can be reused.

Upgrade path for an existing Docker deployment:

1. Back up PostgreSQL and the runtime data directory.
2. Stop the old Sub2API application container.
3. Keep the existing `.env`, PostgreSQL volume, Redis volume, and `DATA_DIR` volume.
4. Replace only the application image or compose source with AySub.
5. Start AySub. New migrations and settings keys are added on startup.

Example database backup:

```bash
pg_dump -h 127.0.0.1 -U sub2api -d sub2api > sub2api_backup.sql
```

If the original deployment uses Docker volumes, keep `postgres_data`, `redis_data`, and `data` mounted into the AySub compose file. Do not run `/setup` again against an existing database; the original admin user remains valid.

Compatibility notes:

- `home_content` remains compatible. When it is non-empty, `/home` still renders that custom HTML or iframe URL before AySub's structured `home_config`.
- AySub adds new tables, columns, and settings. After upgrading, rolling the same database back to an older Sub2API build is not recommended.
- The Go module path remains `github.com/Wei-Shaw/sub2api` only to keep code and migration compatibility; runtime assets use AySub names.

## Build From Source

```bash
git clone https://github.com/AIAllABOUTYOU/AySub.git
cd AySub/frontend
pnpm install
pnpm run build

cd ../backend
go build -tags embed -o aysub ./cmd/server
./aysub
```

For local development:

```bash
cd backend
go run ./cmd/server

cd ../frontend
pnpm run dev
```

When editing Ent schema or Wire providers:

```bash
cd backend
go generate ./ent
go generate ./cmd/server
```

## Configuration Notes

- `DATA_DIR` controls local media cache and runtime data. Default: `./data`.
- `RUN_MODE=simple` hides SaaS-oriented surfaces and skips billing flow; production simple mode also requires `SIMPLE_MODE_CONFIRM=true`.
- `server.trusted_proxies` should be configured before relying on `X-Forwarded-For`.
- Nginx reverse proxies must set `underscores_in_headers on;` if clients rely on headers such as `session_id`.
- `security.url_allowlist` and related environment variables control upstream URL validation.
- `billing.circuit_breaker` controls fail-closed behavior when billing writes fail.
- `turnstile.required` can force Turnstile in release mode.

## Verification

Validated in this workspace:

```bash
cd backend && go test ./...
pnpm --dir frontend test:run
pnpm --dir frontend build
```

`docker build -t aysub:local-verify .` was not executed in this local workspace because the `docker` CLI is not installed (`zsh: command not found: docker`). The Dockerfile path and compose references were checked statically.

## License

License: [GNU Lesser General Public License v3.0](LICENSE) or later.

Copyright: `aiaay.com`.

# 🎉致谢

本项目在 [LINUX DO](https://linux.do/) 社区推广，感谢 LINUX DO 社区对开源项目的支持与认可。 学 AI 上 L 站
