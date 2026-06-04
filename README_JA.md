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

## 現在の範囲

現在のコードベースで実装済み:

- アカウントプール、スティッキーセッション、フェイルオーバー、同時実行制御、レート制限、クォータ確認、使用量課金。
- OpenAI 互換 `/v1/chat/completions`、`/v1/responses`、`/v1/messages`、`/v1/embeddings`、`/v1/images/*`、`/v1/videos`、`/v1/audio/*`、LiveKit、Gemini native routes。
- API Key 単位のモデル/エンドポイント権限。判定はバックエンドのゲートウェイミドルウェアで強制。
- ユーザー/管理者向けのリクエストログと使用量レポート。
- `/models` モデルマーケット、価格計算機、利用可能チャネル、管理側チャネル戦略ビュー。
- `/playground` 体験センター。ユーザー自身の API Key でチャット、画像生成、動画タスク、音声 speech/transcription/translation を実行。
- `/status` 公開ステータスページと匿名 read-only `/api/v1/status/public`。
- `/setup` セットアップウィザード。Docker では `AUTO_SETUP` と `ADMIN_EMAIL` / `ADMIN_PASSWORD` を利用可能。
- デイリーチェックイン。設定キーは `checkin_enabled` と `checkin_reward_amount`。
- TOTP リカバリーコード、センシティブ操作の再認証、ユーザー側 API Key 失効。
- 登録制御: 登録スイッチ、メールドメイン許可リスト、メールドメインブロックリスト、メールエイリアス制限。
- セキュリティ監査センター: 監査ログ、ポリシールール、ロック対象、ハッシュチェーン整合性チェック。
- コンテンツモデレーション/リスク管理。モデレーションイベントは監査/ログ経路へ接続。
- 内蔵決済: プラン、注文、決済インスタンス、callback、返金、決済運用メトリクス。
- xAI 公式 API Key。Grok Cookie/Console アダプターは chat、responses、messages、images、videos、LiveKit token/RTC、quota、model list に対応。
- 生成メディアのローカルキャッシュ: `DATA_DIR/files/images`、`DATA_DIR/files/videos`。管理画面で一覧、削除、条件付きクリーンアップ、孤児ファイル削除が可能。
- 任意の proxy compose profile: `deploy/docker-compose.proxy-profiles.yml` に Privoxy、FlareSolverr、WARP を定義。

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
| API Key | `/keys` |
| デイリーチェックイン | `/checkin` |
| ユーザーリクエストログ | `/request-logs` |
| ユーザーセキュリティ | `/user/security` |
| モデルマーケット | `/models` |
| 体験センター | `/playground` |
| 公開ステータス | `/status` |
| 管理リクエストログ | `/admin/request-logs` |
| 管理セキュリティセンター | `/admin/security` |
| メディアキャッシュ | `/admin/media-cache` |
| リスク管理 | `/admin/risk-control` |
| 決済ダッシュボード | `/admin/orders/dashboard` |

## API

ゲートウェイルートは AySub API Key で認証します:

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

現在の AySub 追加 API:

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
