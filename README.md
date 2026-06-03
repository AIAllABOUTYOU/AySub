# AySub

<div align="center">

[![Go](https://img.shields.io/badge/Go-1.25.7-00ADD8.svg)](https://golang.org/)
[![Vue](https://img.shields.io/badge/Vue-3.4+-4FC08D.svg)](https://vuejs.org/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15+-336791.svg)](https://www.postgresql.org/)
[![Redis](https://img.shields.io/badge/Redis-7+-DC382D.svg)](https://redis.io/)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED.svg)](https://www.docker.com/)

**Three-in-one AI API gateway based on Sub2API, NewAPI, and Grok2API**

English | [中文](README_CN.md) | [日本語](README_JA.md)

</div>

> **AySub is a modified integration fork based on Sub2API.** The Go module and some legacy systemd/helper paths currently still use the upstream `sub2api` identifier for compatibility. Docker images, services, containers, networks, and volumes use the AySub naming by default.

---

## Project Status

AySub is currently implemented in `/Volumes/llovky/AitchTey/code/AySub`. The main gateway, NewAPI-style business operations, xAI/Grok official API Key support, Grok Cookie reverse adapter, console model adapter, and local generated-media cache have been merged into the current codebase.

There is no dedicated public AySub demo in this repository yet. Self-hosted deployments should create their own admin account during setup or through the configured `ADMIN_EMAIL` / `ADMIN_PASSWORD`.

## Overview

AySub is an AI API gateway platform that keeps the Sub2API scheduling core and adds NewAPI-style commercial operations plus a Go implementation of Grok2API reverse access. Users call upstream AI services through platform-generated API Keys, while AySub handles authentication, account scheduling, model pricing, billing, quota control, media generation routing, and request forwarding.

## Features

- **Sub2API Scheduling Core** - Account pool scheduling, sticky sessions, circuit breaking, concurrency limits, rate limits, and token-level usage records.
- **NewAPI-style Operations** - User balance, platform quota, groups, API key permissions, orders, recharge plans, payment instances, payment callbacks, and payment dashboard.
- **Model and Pricing Management** - Default model pricing, channel model allowlists, model sync candidates, per-channel pricing UI, token/image/request billing modes, and user-facing available-channel pricing display.
- **Multi-Provider Gateway** - OpenAI-compatible, Claude/Anthropic-compatible, Gemini-compatible, Antigravity, OpenAI OAuth/API Key, Claude OAuth/API Key, Gemini OAuth/API Key, and custom upstream channels.
- **xAI Official API Key** - xAI platform support with Chat Completions, Responses fallback, Anthropic Messages fallback, model listing, quota attribution, and channel monitoring.
- **Grok Cookie Reverse Adapter** - Grok Web Cookie accounts support Chat Completions, Responses, Anthropic Messages, search references, thinking streams, multimodal image upload, image generation/edit, video generation, and quota lookup.
- **Grok Console Models** - Supports `console.x.ai/v1/responses` routing for `grok-4.3-console`, `grok-4.3-low/medium/high`, `grok-4.20-*`, `grok-4.20-multi-agent-*`, and `grok-build-console` models through Chat, Responses, and Messages entries.
- **Generated Media Cache** - Generated Grok images/videos can be saved under `DATA_DIR/files/images` and `DATA_DIR/files/videos`, then served through `/v1/files/image` and `/v1/files/video`.
- **Built-in Payment System** - Supports EasyPay, Alipay, WeChat Pay, Stripe, and related self-service recharge flows without a separate payment deployment ([Configuration Guide](docs/PAYMENT.md)).
- **Admin Dashboard** - Web interface for users, accounts, channels, pricing, groups, payment, usage, monitoring, data management, and system settings.
- **External System Integration** - Embed external systems such as ticketing through iframe entries in the admin dashboard.

## Integration Notes

| Area | Current state |
|------|---------------|
| Sub2API core | Main scheduling, account pool, sticky session, billing records, rate limit, and gateway routes are retained. |
| NewAPI model/pricing | Main path is available: model pricing, channel allowlists, pricing sync, available channels, and admin channel pricing. A full standalone NewAPI-style “model marketplace” product page is not yet a separate finished module. |
| NewAPI business operations | Users, balance, quotas, groups, orders, recharge plans, payment instances, callbacks, and payment dashboard are present. Real merchant configuration still needs production validation. |
| Grok2API parity | Chat, Responses, Messages, Images, Videos, quota, model list, LiveKit token, LiveKit RTC proxy, console models, and local media cache are implemented in Go. Media link/upscale, Masonry/ChatKit/Admin WebUI, WARP, and FlareSolverr stacks are not directly imported. |
| End-to-end validation | Unit tests and frontend type checks have passed for the implemented paths. Real xAI/Grok Cookie/Console/media/LiveKit account validation is still required before production claims. |

## Tech Stack

| Component | Technology |
|-----------|------------|
| Backend | Go 1.25.7, Gin, Ent |
| Frontend | Vue 3.4+, Vite 5+, TailwindCSS |
| Database | PostgreSQL 15+ |
| Cache/Queue | Redis 7+ |

---

## Nginx Reverse Proxy Note

When using Nginx as a reverse proxy for AySub (or CRS) with Codex CLI, add the following to the `http` block in your Nginx configuration:

```nginx
underscores_in_headers on;
```

Nginx drops headers containing underscores by default (e.g. `session_id`), which breaks sticky session routing in multi-account setups.

---

## Deployment

### Method 1: Script Installation (Upstream Compatibility)

The one-click installer currently follows upstream Sub2API release artifacts. Use it only when you intentionally want the upstream-compatible binary layout. For the current AySub modified code, use Docker Compose or source build from this repository.

#### Prerequisites

- Linux server (amd64 or arm64)
- PostgreSQL 15+ (installed and running)
- Redis 7+ (installed and running)
- Root privileges

#### Installation Steps

```bash
curl -sSL https://raw.githubusercontent.com/AIAllABOUTYOU/AySub/main/deploy/install.sh | sudo bash
```

The script will:
1. Detect your system architecture
2. Download the latest release
3. Install binary to `/opt/sub2api`
4. Create systemd service
5. Configure system user and permissions

#### Post-Installation

```bash
# 1. Start the service
sudo systemctl start sub2api

# 2. Enable auto-start on boot
sudo systemctl enable sub2api

# 3. Open Setup Wizard in browser
# http://YOUR_SERVER_IP:8080
```

The Setup Wizard will guide you through:
- Database configuration
- Redis configuration
- Admin account creation

#### Upgrade

You can upgrade directly from the **Admin Dashboard** by clicking the **Check for Updates** button in the top-left corner.

The web interface will:
- Check for new versions automatically
- Download and apply updates with one click
- Support rollback if needed

#### Useful Commands

```bash
# Check status
sudo systemctl status sub2api

# View logs
sudo journalctl -u sub2api -f

# Restart service
sudo systemctl restart sub2api

# Uninstall
curl -sSL https://raw.githubusercontent.com/AIAllABOUTYOU/AySub/main/deploy/install.sh | sudo bash -s -- uninstall -y
```

---

### Method 2: Docker Compose (Recommended for AySub)

Deploy AySub with Docker Compose, including PostgreSQL and Redis containers. You can either use the GitHub-built GHCR image or build `aysub:latest` from this repository.

#### Prerequisites

- Docker 20.10+
- Docker Compose v2+

#### Quick Start (One-Click Deployment)

Use the GitHub-built image:

```bash
# Clone AySub
git clone https://github.com/AIAllABOUTYOU/AySub.git

# Prepare deployment config
cd AySub/deploy
cp .env.example .env
nano .env
mkdir -p data postgres_data redis_data

# Pull the GHCR image and start services
docker compose -f docker-compose.image.yml pull
docker compose -f docker-compose.image.yml up -d

# View logs
docker compose -f docker-compose.image.yml logs -f aysub
```

GitHub Actions builds `ghcr.io/aiallaboutyou/aysub:latest` on pushes to `main`, version tags matching `v*`, and manual workflow runs. If the package is not publicly pullable after the first publish, set the GitHub Packages visibility to public.

Build the current repository image locally:

```bash
# Clone AySub
git clone https://github.com/AIAllABOUTYOU/AySub.git

# Prepare deployment config
cd AySub/deploy
cp .env.example .env
nano .env
mkdir -p data postgres_data redis_data

# Build and start services with local PostgreSQL and Redis
docker compose -f docker-compose.local.yml up -d --build

# View logs
docker compose -f docker-compose.local.yml logs -f aysub
```

Generate secure values for `JWT_SECRET`, `TOTP_ENCRYPTION_KEY`, and `POSTGRES_PASSWORD` before production use.

#### Manual Deployment

If you prefer manual setup:

```bash
# 1. Clone the repository
git clone https://github.com/AIAllABOUTYOU/AySub.git
cd AySub/deploy

# 2. Copy environment configuration
cp .env.example .env

# 3. Edit configuration (generate secure passwords)
nano .env
```

**Required configuration in `.env`:**

```bash
# PostgreSQL password (REQUIRED)
POSTGRES_PASSWORD=your_secure_password_here

# JWT Secret (RECOMMENDED - keeps users logged in after restart)
JWT_SECRET=your_jwt_secret_here

# TOTP Encryption Key (RECOMMENDED - preserves 2FA after restart)
TOTP_ENCRYPTION_KEY=your_totp_key_here

# Optional: Admin account
ADMIN_EMAIL=admin@example.com
ADMIN_PASSWORD=your_admin_password

# Optional: Custom port
SERVER_PORT=8080
```

**Generate secure secrets:**
```bash
# Generate JWT_SECRET
openssl rand -hex 32

# Generate TOTP_ENCRYPTION_KEY
openssl rand -hex 32

# Generate POSTGRES_PASSWORD
openssl rand -hex 32
```

```bash
# 4. Create data directories (for local version)
mkdir -p data postgres_data redis_data

# 5. Build and start all services
# Option A: Local directory version (recommended - easy migration)
docker compose -f docker-compose.local.yml up -d --build

# Option B: Named volumes version (simple setup)
docker compose up -d --build

# 6. Check status
docker compose -f docker-compose.local.yml ps

# 7. View logs
docker compose -f docker-compose.local.yml logs -f aysub
```

#### Deployment Versions

| Version | Data Storage | Migration | Best For |
|---------|-------------|-----------|----------|
| **docker-compose.image.yml** | Local directories | ✅ Easy (tar entire directory) | Production from GitHub-built images |
| **docker-compose.local.yml** | Local directories | ✅ Easy (tar entire directory) | Production, frequent backups |
| **docker-compose.yml** | Named volumes | ⚠️ Requires docker commands | Simple setup |

**Recommendation:** Use `docker-compose.image.yml` for GitHub-built images, or `docker-compose.local.yml` when building from local source.

#### Access

Open `http://YOUR_SERVER_IP:8080` in your browser.

If admin password was auto-generated, find it in logs:
```bash
docker compose -f docker-compose.local.yml logs aysub | grep "admin password"
```

#### Upgrade

```bash
# Pull latest AySub source and rebuild/recreate container
git pull
docker compose -f docker-compose.local.yml up -d --build

# Or update from the GitHub-built image
docker compose -f docker-compose.image.yml pull aysub
docker compose -f docker-compose.image.yml up -d
```

#### Easy Migration (Local Directory Version)

When using `docker-compose.local.yml`, migrate to a new server easily:

```bash
# On source server
docker compose -f docker-compose.local.yml down
cd ..
tar czf aysub-complete.tar.gz AySub/

# Transfer to new server
scp aysub-complete.tar.gz user@new-server:/path/

# On new server
tar xzf aysub-complete.tar.gz
cd AySub/deploy/
docker compose -f docker-compose.local.yml up -d --build
```

#### Useful Commands

```bash
# Stop all services
docker compose -f docker-compose.local.yml down

# Restart
docker compose -f docker-compose.local.yml restart

# View all logs
docker compose -f docker-compose.local.yml logs -f

# Remove all data (caution!)
docker compose -f docker-compose.local.yml down
rm -rf data/ postgres_data/ redis_data/
```

---

### Method 3: Build from Source

Build and run from source code for development or customization.

#### Prerequisites

- Go 1.21+
- Node.js 18+
- PostgreSQL 15+
- Redis 7+

#### Build Steps

```bash
# 1. Clone the repository
git clone https://github.com/AIAllABOUTYOU/AySub.git
cd AySub

# 2. Install pnpm (if not already installed)
npm install -g pnpm

# 3. Build frontend
cd frontend
pnpm install
pnpm run build
# Output will be in ../backend/internal/web/dist/

# 4. Build backend with embedded frontend
cd ../backend
go build -tags embed -o aysub ./cmd/server

# 5. Create configuration file
cp ../deploy/config.example.yaml ./config.yaml

# 6. Edit configuration
nano config.yaml
```

> **Note:** The `-tags embed` flag embeds the frontend into the binary. Without this flag, the binary will not serve the frontend UI.

**Key configuration in `config.yaml`:**

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

### Sora Status (Temporarily Unavailable)

> ⚠️ Sora-related features are temporarily unavailable due to technical issues in upstream integration and media delivery.
> Please do not rely on Sora in production at this time.
> Existing `gateway.sora_*` configuration keys are reserved and may not take effect until these issues are resolved.

Additional security-related options are available in `config.yaml`:

- `cors.allowed_origins` for CORS allowlist
- `security.url_allowlist` for upstream/pricing/CRS host allowlists
- `security.url_allowlist.enabled` to disable URL validation (use with caution)
- `security.url_allowlist.allow_insecure_http` to allow HTTP URLs when validation is disabled
- `security.url_allowlist.allow_private_hosts` to allow private/local IP addresses
- `security.response_headers.enabled` to enable configurable response header filtering (disabled uses default allowlist)
- `security.csp` to control Content-Security-Policy headers
- `billing.circuit_breaker` to fail closed on billing errors
- `server.trusted_proxies` to enable X-Forwarded-For parsing
- `turnstile.required` to require Turnstile in release mode

**⚠️ Security Warning: HTTP URL Configuration**

When `security.url_allowlist.enabled=false`, the system performs minimal URL validation by default, **rejecting HTTP URLs** and only allowing HTTPS. To allow HTTP URLs (e.g., for development or internal testing), you must explicitly set:

```yaml
security:
  url_allowlist:
    enabled: false                # Disable allowlist checks
    allow_insecure_http: true     # Allow HTTP URLs (⚠️ INSECURE)
```

**Or via environment variable:**

```bash
SECURITY_URL_ALLOWLIST_ENABLED=false
SECURITY_URL_ALLOWLIST_ALLOW_INSECURE_HTTP=true
```

**Risks of allowing HTTP:**
- API keys and data transmitted in **plaintext** (vulnerable to interception)
- Susceptible to **man-in-the-middle (MITM) attacks**
- **NOT suitable for production** environments

**When to use HTTP:**
- ✅ Development/testing with local servers (http://localhost)
- ✅ Internal networks with trusted endpoints
- ✅ Testing account connectivity before obtaining HTTPS
- ❌ Production environments (use HTTPS only)

**Example error without this setting:**
```
Invalid base URL: invalid url scheme: http
```

If you disable URL validation or response header filtering, harden your network layer:
- Enforce an egress allowlist for upstream domains/IPs
- Block private/loopback/link-local ranges
- Enforce TLS-only outbound traffic
- Strip sensitive upstream response headers at the proxy

```bash
# 6. Run the application
./aysub
```

#### Development Mode

```bash
# Backend (with hot reload)
cd backend
go run ./cmd/server

# Frontend (with hot reload)
cd frontend
pnpm run dev
```

#### Code Generation

When editing `backend/ent/schema`, regenerate Ent + Wire:

```bash
cd backend
go generate ./ent
go generate ./cmd/server
```

---

## Simple Mode

Simple Mode is designed for individual developers or internal teams who want quick access without full SaaS features.

- Enable: Set environment variable `RUN_MODE=simple`
- Difference: Hides SaaS-related features and skips billing process
- Security note: In production, you must also set `SIMPLE_MODE_CONFIRM=true` to allow startup

---

## Antigravity Support

AySub supports [Antigravity](https://antigravity.so/) accounts. After authorization, dedicated endpoints are available for Claude and Gemini models.

### Dedicated Endpoints

| Endpoint | Model |
|----------|-------|
| `/antigravity/v1/messages` | Claude models |
| `/antigravity/v1beta/` | Gemini models |

### Claude Code Configuration

```bash
export ANTHROPIC_BASE_URL="http://localhost:8080/antigravity"
export ANTHROPIC_AUTH_TOKEN="sk-xxx"
```

### Hybrid Scheduling Mode

Antigravity accounts support optional **hybrid scheduling**. When enabled, the general endpoints `/v1/messages` and `/v1beta/` will also route requests to Antigravity accounts.

> **⚠️ Warning**: Anthropic Claude and Antigravity Claude **cannot be mixed within the same conversation context**. Use groups to isolate them properly.

### Known Issues

In Claude Code, Plan Mode cannot exit automatically. (Normally when using the native Claude API, after planning is complete, Claude Code will pop up options for users to approve or reject the plan.)

**Workaround**: Press `Shift + Tab` to manually exit Plan Mode, then type your response to approve or reject the plan.

---

## Project Structure

```
AySub/
├── backend/                  # Go backend service
│   ├── cmd/server/           # Application entry
│   ├── internal/             # Internal modules
│   │   ├── config/           # Configuration
│   │   ├── model/            # Data models
│   │   ├── service/          # Business logic
│   │   ├── handler/          # HTTP handlers
│   │   └── gateway/          # API gateway core
│   └── resources/            # Static resources
│
├── frontend/                 # Vue 3 frontend
│   └── src/
│       ├── api/              # API calls
│       ├── stores/           # State management
│       ├── views/            # Page components
│       └── components/       # Reusable components
│
└── deploy/                   # Deployment files
    ├── docker-compose.yml    # Docker Compose configuration
    ├── .env.example          # Environment variables for Docker Compose
    ├── config.example.yaml   # Full config file for binary deployment
    └── install.sh            # One-click installation script
```

## Disclaimer

> **Please read carefully before using this project:**
>
> :rotating_light: **Terms of Service Risk**: Using this project may violate Anthropic's Terms of Service. Please read Anthropic's user agreement carefully before use. All risks arising from the use of this project are borne solely by the user.
>
> :book: **Disclaimer**: This project is for technical learning and research purposes only. The author assumes no responsibility for account suspension, service interruption, or any other losses caused by the use of this project.

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

## License

This project is licensed under the [GNU Lesser General Public License v3.0](LICENSE) (or later).

---

<div align="center">

**If you find this project useful, please give it a star!**

</div>
