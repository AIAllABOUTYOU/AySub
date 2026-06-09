# AySub

<div align="center">

[![Go](https://img.shields.io/badge/Go-1.26.3-00ADD8.svg)](https://golang.org/)
[![Vue](https://img.shields.io/badge/Vue-3.4+-4FC08D.svg)](https://vuejs.org/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15+-336791.svg)](https://www.postgresql.org/)
[![Redis](https://img.shields.io/badge/Redis-7+-DC382D.svg)](https://redis.io/)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED.svg)](https://www.docker.com/)

**All Your AI Sub Hub**

[English](README.md) | [中文](README_CN.md) | 日本語

</div>

AySub は、セルフホスト向けの AI API ゲートウェイです。アカウントスケジューリング、OpenAI 互換 API、NewAPI 風の運用機能、Grok/xAI アダプターを扱います。互換性維持のため Go module path は `github.com/Wei-Shaw/sub2api` のままです。実行時名、Docker image、container、service、network、volume は AySub 名を使います。

著作権表記: `aiaay.com`。

## 現在の機能

現在のコードベースで実装済み:

- Anthropic/Claude、OpenAI、Gemini、xAI/Grok、Antigravity、Anthropic Bedrock / Vertex Service Account のマルチプラットフォームアカウントプール。group、priority、load factor、sticky session、failover、concurrency、temporary unscheduling、overload/rate-limit cooldown、quota check、usage billing に対応。
- API gateway: Claude Messages、OpenAI Chat Completions、Responses、Messages compatible forwarding、Embeddings、Images、Videos、Audio、LiveKit、Gemini native `/v1beta`、Antigravity 専用 `/antigravity/v1` と `/antigravity/v1beta`。
- API Key 管理: model allowlist、endpoint permission、group binding、status control、user CRUD、admin-side group reassignment。
- Channel / pricing: channel CRUD、model pricing/mapping、wildcard matching、group multiplier、RPM override、strategy view、user available channels、model marketplace、price calculator。
- Account operations: OAuth/API Key/Cookie/Service Account/Bedrock account、batch import/export、CRS sync、upstream model sync、account test、account inspection、refresh、privacy setup、quota/tier refresh、error cleanup、usage stats。Account inspection は Codex/OpenAI/all targets、concurrency、timeout、sampling、filter、keep/enable/disable/delete/reauth suggestions に対応。
- User system: email registration/login、verification code、forgot/reset password、JWT refresh/logout、GitHub/Google/OIDC/DingTalk/WeChat/LinuxDo login and binding、email completion、session revocation。
- User workspace: dashboard、API Keys、usage records、request logs、daily check-in、security center、notification email、TOTP and recovery codes、sensitive-operation verification、profile、account bindings、redeem codes、subscriptions、orders、affiliate rebates、custom menu pages。
- Admin console: dashboard、users、groups、accounts、account inspection、proxies、announcements、settings、home configuration、channels、channel monitors、scheduled tests、subscriptions、usage cleanup、request logs、redeem codes、promo codes、user attributes、API Key group management。
- Ops monitoring: realtime concurrency、account availability、realtime traffic、QPS WebSocket、alert rules/events/silences、error logs、request errors、upstream errors、request drilldown、system logs、runtime log config、metric thresholds。
- Security / risk control: audit logs、incidents、policies、subject locks、hash-chain integrity checks、audit exports、content moderation config/status/logs、user unban、flagged-hash cleanup。
- Payment / subscription: plans、balance recharge、subscription purchase、orders、refund request/processing、order retry、payment dashboard、provider instances、visible payment methods、callbacks、EasyPay、Alipay、WeChat Pay、Stripe、Airwallex。
- Media / files: generated image/video local cache under `DATA_DIR/files/images` and `DATA_DIR/files/videos`、admin listing/deletion/filtered cleanup/orphan cleanup、custom Markdown pages and images served from `DATA_DIR/pages`。
- Data / system maintenance: S3/source profiles、backup jobs、scheduled backups、backup restore、system version check、app update、rollback、restart endpoints。
- Grok/xAI compatibility: Grok Cookie request 用の optional dynamic `x-statsig-id` compatibility header を global または account 単位で有効化できます。
- Deployment features: `/setup` wizard、Docker `AUTO_SETUP`、embedded frontend、simple/backend modes、optional Privoxy/FlareSolverr/WARP proxy compose profiles。

README で完了済みとは扱わない項目:

- Passkey/WebAuthn。
- 汎用 custom OAuth Provider CRUD。現在は GitHub、Google、OIDC、DingTalk、WeChat、LinuxDo の固定フロー。
- 既存のモデル/チャネル/価格データとは別の完全なモデル集合テンプレートと provider catalog。
- Grok2API の Masonry gallery と ChatKit voice UI の完全 parity。
- 実 xAI/Grok Cookie/Console/media/LiveKit アカウント、実決済事業者 callback の本番検証。

詳細は以下に記録しています:

- [NewAPI feature roadmap](docs/NEWAPI_FEATURE_ROADMAP_CN.md)
- [Security audit roadmap](docs/SECURITY_AUDIT_ROADMAP_CN.md)
- [Grok2API parity roadmap](docs/GROK2API_PARITY_CN.md)

## 主なページ

| 領域 | ページ |
| --- | --- |
| セットアップ | `/setup` |
| Home | `/home` |
| Login/Register | `/login`, `/register` |
| User dashboard | `/dashboard` |
| API Key | `/keys` |
| API Key usage lookup | `/key-usage` |
| デイリーチェックイン | `/checkin` |
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
| Account inspection | `/admin/accounts/inspection` |
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

ゲートウェイルートは AySub API Key で認証します:

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

User/admin/payment/public API examples:

- Auth: `/api/v1/auth/register`, `/api/v1/auth/login`, `/api/v1/auth/login/2fa`, `/api/v1/auth/refresh`, `/api/v1/auth/logout`, `/api/v1/auth/forgot-password`, `/api/v1/auth/reset-password`
- OAuth: `/api/v1/auth/oauth/{github,google,linuxdo,wechat,oidc,dingtalk}/start` plus callback / complete / bind / create-account flows
- User: `/api/v1/user/profile`, `/api/v1/user/platform-quotas`, `/api/v1/user/checkin`, `/api/v1/user/security/events`, `/api/v1/user/totp/*`, `/api/v1/user/notify-email/*`
- API Keys and usage: `/api/v1/keys`, `/api/v1/groups/available`, `/api/v1/usage`, `/api/v1/usage/requests`, `/api/v1/usage/dashboard/*`
- User channels and playground: `/api/v1/channels/available`, `/api/v1/channel-monitors`, `/api/v1/playground/sessions`
- Announcements, redeem, subscriptions: `/api/v1/announcements`, `/api/v1/redeem`, `/api/v1/subscriptions/*`
- Admin core: `/api/v1/admin/dashboard/*`, `/api/v1/admin/users/*`, `/api/v1/admin/groups/*`, `/api/v1/admin/accounts/*`, `/api/v1/admin/accounts/inspection/run`, `/api/v1/admin/proxies/*`, `/api/v1/admin/settings/*`
- Admin channels: `/api/v1/admin/channels/*`, `/api/v1/admin/channel-monitors/*`, `/api/v1/admin/channel-monitor-templates/*`, `/api/v1/admin/scheduled-test-plans/*`
- Admin operations: `/api/v1/admin/ops/*`, `/api/v1/admin/ops/requests`, `/api/v1/admin/usage/*`, `/api/v1/admin/media-cache/*`
- Admin security/risk: `/api/v1/admin/security/*`, `/api/v1/admin/risk-control/*`, `/api/v1/admin/error-passthrough-rules/*`, `/api/v1/admin/tls-fingerprint-profiles/*`
- Admin monetization: `/api/v1/admin/subscriptions/*`, `/api/v1/admin/payment/*`, `/api/v1/admin/redeem-codes/*`, `/api/v1/admin/promo-codes/*`, `/api/v1/admin/affiliates/*`
- Data and system: `/api/v1/admin/data-management/*`, `/api/v1/admin/backups/*`, `/api/v1/admin/system/*`
- Payment: `/api/v1/payment/config`, `/api/v1/payment/plans`, `/api/v1/payment/orders/*`, `/api/v1/payment/public/orders/*`, `/api/v1/payment/webhook/{easypay,alipay,wxpay,stripe,airwallex}`
- Public: `GET /health`, `GET /setup/status`, `GET /api/v1/settings/public`, `GET /api/v1/status/public`, `GET /api/v1/pages/:slug`, `GET /api/v1/pages/:slug/images/*`

## Docker

GitHub Actions は以下の image を公開します:

```bash
ghcr.io/aiallaboutyou/aysub:latest
```

Docker image のトリガー:

- `main` への push: `ghcr.io/aiallaboutyou/aysub:latest` を公開
- `v*` tag: 対応する GHCR version tag を公開
- Docker workflow の手動実行

workflow はルートの [`Dockerfile`](Dockerfile) を使用します。`deploy/` の compose ファイルもリポジトリルートの Dockerfile からビルドします。

GitHub Release のトリガー:

- `v0.1.134` のような version tag を push
- または既存の `v*` tag を指定して `Release` workflow を手動実行

通常の `main` への commit では GitHub Release は作成されません。Release には `aysub_<version>_<os>_<arch>.tar.gz` binary と `checksums.txt` がアップロードされ、binary installer とアプリ内 update check が使用します。

```bash
git tag -a v0.1.134 -m "AySub v0.1.134"
git push origin v0.1.134
```

GHCR image を使う場合:

```bash
git clone https://github.com/AIAllABOUTYOU/AySub.git
cd AySub/deploy
cp .env.example .env
nano .env
mkdir -p data postgres_data redis_data
docker compose -f docker-compose.image.yml pull
docker compose -f docker-compose.image.yml up -d
```

ローカルソースからビルドする場合:

```bash
git clone https://github.com/AIAllABOUTYOU/AySub.git
cd AySub/deploy
cp .env.example .env
nano .env
mkdir -p data postgres_data redis_data
docker compose -f docker-compose.local.yml up -d --build
```

本番では最低限以下を設定してください:

```bash
POSTGRES_PASSWORD=replace-with-random-password
JWT_SECRET=replace-with-openssl-rand-hex-32
TOTP_ENCRYPTION_KEY=replace-with-openssl-rand-hex-32
ADMIN_EMAIL=admin@example.com
ADMIN_PASSWORD=replace-with-admin-password
SERVER_PORT=8080
```

Docker auto setup で `ADMIN_PASSWORD` が空の場合、AySub は管理者パスワードを生成して container log に出力します。

### 任意の Grok WARP / FlareSolverr スタック

Grok Cookie account で Cloudflare challenge が検出された場合、FlareSolverr で `cf_clearance` を自動更新できます。WARP は Grok request 用の SOCKS5 egress proxy として利用できます。どちらも `deploy/docker-compose.proxy-profiles.yml` に定義されています。詳細は [`deploy/DOCKER.md`](deploy/DOCKER.md) を参照してください。

FlareSolverr と WARP を一緒に有効化する例:

```bash
cd deploy
GROK_FLARESOLVERR_URL=http://flaresolverr:8191 \
docker compose -f docker-compose.image.yml -f docker-compose.proxy-profiles.yml --profile flaresolverr --profile warp up -d
```

WARP 起動後、admin proxy page で active proxy を追加します:

```text
socks5://warp:1080
```

その proxy を Grok Cookie account に紐付けてください。AySub は account proxy を FlareSolverr に渡すため、clearance cookie は実際の Grok request と同じ egress path で解決されます。この stack が扱うのは Cloudflare challenge と egress IP の問題だけです。token 失効、account risk control、region restriction、upstream access denial の場合は、有効な account、Cookie、proxy が別途必要です。

## Sub2API からの移行

AySub は既存 Sub2API デプロイからの forward-compatible upgrade として扱えます。既存の user、API Key、group、account、balance、usage log、settings、PostgreSQL data、Redis data、`DATA_DIR` files は再利用できます。

既存 Docker デプロイの移行手順:

1. PostgreSQL と runtime data directory をバックアップします。
2. 旧 Sub2API application container を停止します。
3. 既存の `.env`、PostgreSQL volume、Redis volume、`DATA_DIR` volume を保持します。
4. application image または compose の source build target だけを AySub に差し替えます。
5. AySub を起動します。新しい migration と settings key は起動時に追加されます。

database backup の例:

```bash
pg_dump -h 127.0.0.1 -U sub2api -d sub2api > sub2api_backup.sql
```

Docker volume を使っている場合は、`postgres_data`、`redis_data`、`data` を AySub compose にそのまま mount してください。既存 database に対して `/setup` を再実行しないでください。既存の admin user はそのまま有効です。

互換性メモ:

- `home_content` は引き続き互換です。値が空でない場合、`/home` は AySub の構造化 `home_config` より先に custom HTML または iframe URL を表示します。
- AySub は table、column、settings を追加します。upgrade 後に同じ database を古い Sub2API build へ直接戻すことは推奨しません。
- Go module path は code と migration compatibility のため `github.com/Wei-Shaw/sub2api` のままです。runtime image、container、service、network、volume は AySub 名を使います。

## ソースビルド

```bash
git clone https://github.com/AIAllABOUTYOU/AySub.git
cd AySub/frontend
pnpm install
pnpm run build

cd ../backend
go build -tags embed -o aysub ./cmd/server
./aysub
```

ローカル開発:

```bash
cd backend
go run ./cmd/server

cd ../frontend
pnpm run dev
```

Ent schema または Wire provider を変更した場合:

```bash
cd backend
go generate ./ent
go generate ./cmd/server
```

## 設定メモ

- `DATA_DIR` はローカルメディアキャッシュとランタイムデータを制御します。デフォルトは `./data`。
- `RUN_MODE=simple` は SaaS 系画面を隠し、課金フローをスキップします。本番では `SIMPLE_MODE_CONFIRM=true` も必要です。
- `X-Forwarded-For` を信頼する前に `server.trusted_proxies` を設定してください。
- Nginx reverse proxy では `underscores_in_headers on;` が必要です。`session_id` などのヘッダーが落ちます。
- `security.url_allowlist` と関連環境変数は上流 URL 検証を制御します。
- `billing.circuit_breaker` は課金書き込み失敗時の fail closed を制御します。
- `turnstile.required` は release mode で Turnstile を強制できます。
- `grok.dynamic_statsig_enabled` は Grok Cookie account の dynamic `x-statsig-id` compatibility header を有効化します。account credentials または extra の `dynamic_statsig_enabled` / `grok_dynamic_statsig_enabled` で global 設定を上書きできます。

## 検証

この workspace で通過済み:

```bash
cd backend && go test ./...
pnpm --dir frontend test:run
pnpm --dir frontend build
```

`docker build -t aysub:local-verify .` はこのローカル環境では未実行です。`docker` CLI がインストールされていません（`zsh: command not found: docker`）。Dockerfile path と compose references は静的に確認済みです。

## ライセンス

License: [GNU Lesser General Public License v3.0](LICENSE) or later.

Copyright: `aiaay.com`.

# 🎉致谢

本项目在 [LINUX DO](https://linux.do/) 社区推广，感谢 LINUX DO 社区对开源项目的支持与认可。 学 AI 上 L 站
