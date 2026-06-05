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

## Current Scope

Implemented in the current codebase:

- Account pool scheduling, sticky sessions, failover, concurrency controls, rate limits, quota checks, and usage billing.
- OpenAI-compatible `/v1/chat/completions`, `/v1/responses`, `/v1/messages`, `/v1/embeddings`, `/v1/images/*`, `/v1/videos`, `/v1/audio/*`, LiveKit, and Gemini native route handling.
- API Key model and endpoint permissions enforced in backend gateway middleware.
- Request log center and usage reports for users and admins.
- Model marketplace at `/models`, price calculator, available channel view, and admin channel strategy view.
- Experience center at `/playground` for chat, image generation, video tasks, and audio speech/transcription/translation using the user's own API Key.
- Public status page at `/status` backed by anonymous `/api/v1/status/public`.
- Setup wizard at `/setup`, plus Docker `AUTO_SETUP` with `ADMIN_EMAIL` / `ADMIN_PASSWORD`.
- Daily check-in with `checkin_enabled` and `checkin_reward_amount` settings.
- TOTP with recovery codes, sensitive-operation verification, and API Key revocation from the user security page.
- Registration controls: registration switch, email suffix whitelist, email domain blacklist, and email alias restriction.
- Security audit center with audit logs, policy rules, subject locks, and hash-chain integrity check.
- Content moderation/risk control and moderation events connected to audit/logging paths.
- Built-in payment system with plans, orders, provider instances, callbacks, refunds, and payment operation metrics.
- xAI official API Key support and Grok Cookie/Console adapters for chat, responses, messages, images, videos, LiveKit token/RTC, quota, and model listing.
- Local generated media cache under `DATA_DIR/files/images` and `DATA_DIR/files/videos`, with admin cache listing, deletion, filtered cleanup, and orphan cleanup.
- Optional proxy compose profile for Privoxy, FlareSolverr, and WARP in `deploy/docker-compose.proxy-profiles.yml`.

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
| API Keys | `/keys` |
| Daily check-in | `/checkin` |
| User request logs | `/request-logs` |
| User security | `/user/security` |
| Model marketplace | `/models` |
| Experience center | `/playground` |
| Public status | `/status` |
| Admin request logs | `/admin/request-logs` |
| Admin security center | `/admin/security` |
| Admin media cache | `/admin/media-cache` |
| Admin risk control | `/admin/risk-control` |
| Admin payment dashboard | `/admin/orders/dashboard` |

## API Surface

Gateway routes are authenticated with AySub API Keys:

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

User/admin APIs added by the current AySub work:

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
