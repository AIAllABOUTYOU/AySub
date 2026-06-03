# AySub

<div align="center">

[![Go](https://img.shields.io/badge/Go-1.25.7-00ADD8.svg)](https://golang.org/)
[![Vue](https://img.shields.io/badge/Vue-3.4+-4FC08D.svg)](https://vuejs.org/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15+-336791.svg)](https://www.postgresql.org/)
[![Redis](https://img.shields.io/badge/Redis-7+-DC382D.svg)](https://redis.io/)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED.svg)](https://www.docker.com/)

**AySub 三位一体 AI API ゲートウェイ**

[English](README.md) | [中文](README_CN.md) | 日本語

</div>

> AySub はアカウントスケジューリング、NewAPI 風の運用機能、Grok リバースアダプターを統合しています。既存コードとの互換性のため、Go module は上流の `github.com/Wei-Shaw/sub2api` import path を維持しますが、実行時名、Docker image、service、container、network、volume はデフォルトで AySub 名を使用します。

---

## 現在の状態

現在の実装場所: `/Volumes/llovky/AitchTey/code/AySub`。

メインゲートウェイ、NewAPI 風の運用機能、xAI/Grok 公式 API Key、Grok Cookie リバースアダプター、Console モデルアダプター、生成メディアのローカルキャッシュは現在のコードベースに統合済みです。このリポジトリにはまだ AySub 専用の公開デモはありません。セルフホスト時はセットアップウィザード、または `ADMIN_EMAIL` / `ADMIN_PASSWORD` で管理者アカウントを作成してください。

## 概要

AySub は、アカウントスケジューリング、NewAPI 風の商用運用機能、Grok Cookie リバースアクセスを Go 実装として統合した AI API ゲートウェイです。ユーザーはプラットフォームが生成した API キーを通じて上流 AI サービスへアクセスし、AySub が認証、アカウントスケジューリング、モデル価格、課金、クォータ制御、メディア生成ルーティング、リクエスト転送を処理します。

## 機能

- **AySub スケジューリングコア** - アカウントプール、スティッキーセッション、サーキットブレーカー、同時実行制御、レート制限、トークン単位の使用量記録をサポート。
- **NewAPI 風の運用機能** - ユーザー残高、プラットフォーム別クォータ、グループ、API Key 権限、注文、チャージプラン、決済インスタンス、決済コールバック、決済ダッシュボードをサポート。
- **モデルと価格管理** - デフォルトモデル価格、チャネル別モデル許可リスト、モデル同期候補、チャネル価格 UI、トークン/画像/リクエスト単位の課金モード、ユーザー向け利用可能チャネルと価格表示に対応。
- **体験センター** - ユーザー向け `/playground` で自分の API Key を使ったチャットと画像生成を試せます。動画と音声タブは予約済みで、現在は coming soon 表示です。
- **マルチプロバイダーゲートウェイ** - OpenAI 互換、Claude/Anthropic 互換、Gemini 互換、Antigravity、OpenAI OAuth/API Key、Claude OAuth/API Key、Gemini OAuth/API Key、カスタム上流チャネルをサポート。
- **xAI 公式 API Key** - xAI プラットフォームを追加し、Chat Completions、Responses fallback、Anthropic Messages fallback、モデル一覧、エンドポイント統計、チャネル監視に対応。
- **Grok Cookie リバースアダプター** - Grok Web Cookie アカウントで Chat Completions、Responses、Anthropic Messages、検索引用、thinking ストリーム、マルチモーダル画像アップロード、画像生成/編集、動画生成、クォータ照会をサポート。
- **Grok Console モデル** - Cookie アカウントから `console.x.ai/v1/responses` 経由で `grok-4.3-console`、`grok-4.3-low/medium/high`、`grok-4.20-*`、`grok-4.20-multi-agent-*`、`grok-build-console` を呼び出し、Chat、Responses、Messages エントリへ変換。
- **生成メディアのローカルキャッシュ** - Grok 生成画像/動画を `DATA_DIR/files/images` と `DATA_DIR/files/videos` に保存し、`/v1/files/image` と `/v1/files/video` から配信可能。
- **内蔵決済システム** - EasyPay、Alipay、WeChat Pay、Stripe などのセルフサービスチャージに対応し、別途決済サービスは不要（[設定ガイド](docs/PAYMENT.md)）。
- **管理ダッシュボード** - ユーザー、アカウント、チャネル、価格、グループ、決済、使用量、監視、データ管理、システム設定を Web UI で管理。
- **外部システム連携** - チケット管理などの外部システムを iframe で管理ダッシュボードに埋め込み可能。

## 統合状況

| 領域 | 現在の状態 |
|------|-------------|
| AySub コア | スケジューリング、アカウントプール、スティッキーセッション、使用量課金、レート制限、ゲートウェイルートは実装済み。 |
| NewAPI モデル/価格 | モデル価格、チャネル許可リスト、価格同期、利用可能チャネル表示、管理側チャネル価格、ユーザー向け `/models` モデルマーケットを実装済み。モデル単位でプラットフォーム、チャネル、グループ、価格を集約表示できます。 |
| NewAPI 体験センター | ユーザー向け `/playground` でチャットと画像生成を試せます。動画と音声の入口は UI に残し、現在は coming soon として表示します。 |
| NewAPI 商用運用 | ユーザー、残高、クォータ、グループ、注文、チャージプラン、決済インスタンス、コールバック、決済ダッシュボードは実装済み。実決済事業者設定は本番環境での検証が必要です。 |
| Grok2API parity | Chat、Responses、Messages、Images、Videos、Usage、モデル一覧、LiveKit token、LiveKit RTC proxy、Console モデル、ローカルメディアキャッシュは Go に移植済み。media link/upscale、Masonry/ChatKit/Admin WebUI、WARP、FlareSolverr は直接統合していません。 |
| E2E 検証 | 単体テストとフロントエンド型チェックは通過済み。実 xAI/Grok Cookie/Console/メディア/LiveKit アカウントでの検証は本番前に必要です。 |

## 技術スタック

| コンポーネント | 技術 |
|-----------|------------|
| バックエンド | Go 1.25.7, Gin, Ent |
| フロントエンド | Vue 3.4+, Vite 5+, TailwindCSS |
| データベース | PostgreSQL 15+ |
| キャッシュ/キュー | Redis 7+ |

---

## Nginx リバースプロキシに関する注意

AySub（または CRS）を Nginx でリバースプロキシし、Codex CLI と組み合わせて使用する場合、Nginx の `http` ブロックに以下の設定を追加してください:

```nginx
underscores_in_headers on;
```

Nginx はデフォルトでアンダースコアを含むヘッダー（例: `session_id`）を破棄するため、マルチアカウント構成でのスティッキーセッションルーティングに支障をきたします。

---

## デプロイ

### 方法1: スクリプトによるインストール

ワンクリックインストーラーは AySub のパスにバイナリ、設定、システムユーザー、systemd サービスを配置します。コンテナデプロイでは Docker Compose を推奨します。

#### 前提条件

- Linux サーバー（amd64 または arm64）
- PostgreSQL 15+（インストール済みかつ稼働中）
- Redis 7+（インストール済みかつ稼働中）
- root 権限

#### インストール手順

```bash
curl -sSL https://raw.githubusercontent.com/AIAllABOUTYOU/AySub/main/deploy/install.sh | sudo bash
```

スクリプトは以下を実行します:
1. システムアーキテクチャの検出
2. 最新リリースのダウンロード
3. バイナリを `/opt/aysub` にインストール
4. systemd サービスの作成
5. システムユーザーと権限の設定

#### インストール後の作業

```bash
# 1. サービスを起動
sudo systemctl start aysub

# 2. 起動時の自動起動を有効化
sudo systemctl enable aysub

# 3. ブラウザでセットアップウィザードを開く
# http://YOUR_SERVER_IP:8080
```

セットアップウィザードでは以下の設定を行います:
- データベース設定
- Redis 設定
- 管理者アカウントの作成

#### アップグレード

**管理ダッシュボード**の左上にある**アップデートを確認**ボタンをクリックすることで、ダッシュボードから直接アップグレードできます。

Web インターフェースでは以下が可能です:
- 新しいバージョンの自動確認
- ワンクリックでのアップデートのダウンロードと適用
- 必要に応じたロールバック

#### よく使うコマンド

```bash
# ステータスを確認
sudo systemctl status aysub

# ログを表示
sudo journalctl -u aysub -f

# サービスを再起動
sudo systemctl restart aysub

# アンインストール
curl -sSL https://raw.githubusercontent.com/AIAllABOUTYOU/AySub/main/deploy/install.sh | sudo bash -s -- uninstall -y
```

---

### 方法2: Docker Compose（AySub 推奨）

AySub を Docker Compose でデプロイします。PostgreSQL と Redis のコンテナも含まれます。GitHub でビルドされた GHCR イメージを使うことも、このリポジトリから `aysub:latest` をローカルビルドすることもできます。

#### 前提条件

- Docker 20.10+
- Docker Compose v2+

#### クイックスタート

GitHub でビルドされたイメージを使う場合:

```bash
# AySub をクローン
git clone https://github.com/AIAllABOUTYOU/AySub.git

# デプロイ設定を準備
cd AySub/deploy
cp .env.example .env
nano .env
mkdir -p data postgres_data redis_data

# GHCR イメージを取得してサービスを起動
docker compose -f docker-compose.image.yml pull
docker compose -f docker-compose.image.yml up -d

# ログを表示
docker compose -f docker-compose.image.yml logs -f aysub
```

GitHub Actions は `main` への push、`v*` タグ、または手動 workflow 実行で `ghcr.io/aiallaboutyou/aysub:latest` をビルドします。初回公開後に外部から pull できない場合は、GitHub Packages で package visibility を public に設定してください。

現在のリポジトリからローカルビルドする場合:

```bash
# AySub をクローン
git clone https://github.com/AIAllABOUTYOU/AySub.git

# デプロイ設定を準備
cd AySub/deploy
cp .env.example .env
nano .env
mkdir -p data postgres_data redis_data

# ビルドしてサービスを起動
docker compose -f docker-compose.local.yml up -d --build

# ログを表示
docker compose -f docker-compose.local.yml logs -f aysub
```

本番環境では、`JWT_SECRET`、`TOTP_ENCRYPTION_KEY`、`POSTGRES_PASSWORD` に安全な値を設定してください。

#### 手動デプロイ

手動でセットアップする場合:

```bash
# 1. リポジトリをクローン
git clone https://github.com/AIAllABOUTYOU/AySub.git
cd AySub/deploy

# 2. 環境設定ファイルをコピー
cp .env.example .env

# 3. 設定を編集（セキュアなパスワードを生成）
nano .env
```

**`.env` の必須設定:**

```bash
# PostgreSQL パスワード（必須）
POSTGRES_PASSWORD=your_secure_password_here

# JWT シークレット（推奨 - 再起動後もユーザーのログイン状態を保持）
JWT_SECRET=your_jwt_secret_here

# TOTP 暗号化キー（推奨 - 再起動後も二要素認証を維持）
TOTP_ENCRYPTION_KEY=your_totp_key_here

# オプション: 管理者アカウント
ADMIN_EMAIL=admin@example.com
ADMIN_PASSWORD=your_admin_password

# オプション: カスタムポート
SERVER_PORT=8080
```

**セキュアなシークレットの生成方法:**
```bash
# JWT_SECRET を生成
openssl rand -hex 32

# TOTP_ENCRYPTION_KEY を生成
openssl rand -hex 32

# POSTGRES_PASSWORD を生成
openssl rand -hex 32
```

```bash
# 4. データディレクトリを作成（ローカルバージョンの場合）
mkdir -p data postgres_data redis_data

# 5. ビルドしてすべてのサービスを起動
# オプション A: ローカルディレクトリバージョン（推奨 - 移行が容易）
docker compose -f docker-compose.local.yml up -d --build

# オプション B: 名前付きボリュームバージョン（シンプルなセットアップ）
docker compose up -d --build

# 6. ステータスを確認
docker compose -f docker-compose.local.yml ps

# 7. ログを表示
docker compose -f docker-compose.local.yml logs -f aysub
```

#### デプロイバージョン

| バージョン | データストレージ | 移行 | 推奨用途 |
|---------|-------------|-----------|----------|
| **docker-compose.image.yml** | ローカルディレクトリ | ✅ 容易（ディレクトリ全体を tar） | GitHub ビルド済みイメージによる本番デプロイ |
| **docker-compose.local.yml** | ローカルディレクトリ | ✅ 容易（ディレクトリ全体を tar） | 本番環境、頻繁なバックアップ |
| **docker-compose.yml** | 名前付きボリューム | ⚠️ docker コマンドが必要 | シンプルなセットアップ |

**推奨:** GitHub ビルド済みイメージを使う場合は `docker-compose.image.yml`、ローカルソースからビルドする場合は `docker-compose.local.yml` を使用してください。

#### アクセス

ブラウザで `http://YOUR_SERVER_IP:8080` を開いてください。

管理者パスワードが自動生成された場合は、ログで確認できます:
```bash
docker compose -f docker-compose.local.yml logs aysub | grep "admin password"
```

#### アップグレード

```bash
# 最新の AySub ソースを取得してコンテナを再ビルド/再作成
git pull
docker compose -f docker-compose.local.yml up -d --build

# または GitHub ビルド済みイメージで更新
docker compose -f docker-compose.image.yml pull aysub
docker compose -f docker-compose.image.yml up -d
```

#### 簡単な移行（ローカルディレクトリバージョン）

`docker-compose.local.yml` を使用している場合、新しいサーバーへの移行が簡単です:

```bash
# 移行元サーバーにて
docker compose -f docker-compose.local.yml down
cd ..
tar czf aysub-complete.tar.gz AySub/

# 新しいサーバーに転送
scp aysub-complete.tar.gz user@new-server:/path/

# 移行先サーバーにて
tar xzf aysub-complete.tar.gz
cd AySub/deploy/
docker compose -f docker-compose.local.yml up -d --build
```

#### よく使うコマンド

```bash
# すべてのサービスを停止
docker compose -f docker-compose.local.yml down

# 再起動
docker compose -f docker-compose.local.yml restart

# すべてのログを表示
docker compose -f docker-compose.local.yml logs -f

# すべてのデータを削除（注意！）
docker compose -f docker-compose.local.yml down
rm -rf data/ postgres_data/ redis_data/
```

---

### 方法3: ソースからビルド

開発やカスタマイズのためにソースコードからビルドして実行します。

#### 前提条件

- Go 1.21+
- Node.js 18+
- PostgreSQL 15+
- Redis 7+

#### ビルド手順

```bash
# 1. リポジトリをクローン
git clone https://github.com/AIAllABOUTYOU/AySub.git
cd AySub

# 2. pnpm をインストール（未インストールの場合）
npm install -g pnpm

# 3. フロントエンドをビルド
cd frontend
pnpm install
pnpm run build
# 出力先: ../backend/internal/web/dist/

# 4. フロントエンドを組み込んだバックエンドをビルド
cd ../backend
go build -tags embed -o aysub ./cmd/server

# 5. 設定ファイルを作成
cp ../deploy/config.example.yaml ./config.yaml

# 6. 設定を編集
nano config.yaml
```

> **注意:** `-tags embed` フラグはフロントエンドをバイナリに組み込みます。このフラグがない場合、バイナリはフロントエンド UI を提供しません。

**`config.yaml` の主要設定:**

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

### Sora ステータス（一時的に利用不可）

> ⚠️ Sora 関連の機能は、上流統合およびメディア配信の技術的問題により一時的に利用できません。
> 現時点では本番環境で Sora に依存しないでください。
> 既存の `gateway.sora_*` 設定キーは予約されていますが、これらの問題が解決されるまで有効にならない場合があります。

`config.yaml` では追加のセキュリティ関連オプションも利用できます:

- `cors.allowed_origins` - CORS 許可リスト
- `security.url_allowlist` - 上流/価格/CRS ホストの許可リスト
- `security.url_allowlist.enabled` - URL バリデーションの無効化（注意して使用）
- `security.url_allowlist.allow_insecure_http` - バリデーション無効時に HTTP URL を許可
- `security.url_allowlist.allow_private_hosts` - プライベート/ローカル IP アドレスを許可
- `security.response_headers.enabled` - 設定可能なレスポンスヘッダーフィルタリングを有効化（無効時はデフォルトの許可リストを使用）
- `security.csp` - Content-Security-Policy ヘッダーの制御
- `billing.circuit_breaker` - 課金エラー時にフェイルクローズ
- `server.trusted_proxies` - X-Forwarded-For パースの有効化
- `turnstile.required` - リリースモードでの Turnstile 必須化

**⚠️ セキュリティ警告: HTTP URL 設定**

`security.url_allowlist.enabled=false` の場合、システムはデフォルトで最小限の URL バリデーションを行い、**HTTP URL を拒否**して HTTPS のみを許可します。HTTP URL を許可するには（開発環境や内部テスト用など）、以下を明示的に設定する必要があります:

```yaml
security:
  url_allowlist:
    enabled: false                # 許可リストチェックを無効化
    allow_insecure_http: true     # HTTP URL を許可（⚠️ セキュリティリスクあり）
```

**または環境変数で設定:**

```bash
SECURITY_URL_ALLOWLIST_ENABLED=false
SECURITY_URL_ALLOWLIST_ALLOW_INSECURE_HTTP=true
```

**HTTP を許可するリスク:**
- API キーとデータが**平文**で送信される（傍受の危険性）
- **中間者攻撃（MITM）**を受けやすい
- **本番環境には不適切**

**HTTP を使用すべき場面:**
- ✅ ローカルサーバーでの開発・テスト（http://localhost）
- ✅ 信頼できるエンドポイントを持つ内部ネットワーク
- ✅ HTTPS 取得前のアカウント接続テスト
- ❌ 本番環境（HTTPS のみを使用）

**この設定なしで表示されるエラー例:**
```
Invalid base URL: invalid url scheme: http
```

URL バリデーションまたはレスポンスヘッダーフィルタリングを無効にする場合は、ネットワーク層を強化してください:
- 上流ドメイン/IP のエグレス許可リストを適用
- プライベート/ループバック/リンクローカル範囲をブロック
- TLS のみのアウトバウンドトラフィックを強制
- プロキシで機密性の高い上流レスポンスヘッダーを除去

```bash
# 6. アプリケーションを実行
./aysub
```

#### 開発モード

```bash
# バックエンド（ホットリロード付き）
cd backend
go run ./cmd/server

# フロントエンド（ホットリロード付き）
cd frontend
pnpm run dev
```

#### コード生成

`backend/ent/schema` を編集した場合、Ent + Wire を再生成してください:

```bash
cd backend
go generate ./ent
go generate ./cmd/server
```

---

## シンプルモード

シンプルモードは、フル SaaS 機能を必要とせず、素早くアクセスしたい個人開発者や社内チーム向けに設計されています。

- 有効化: 環境変数 `RUN_MODE=simple` を設定
- 違い: SaaS 関連機能を非表示にし、課金プロセスをスキップ
- セキュリティに関する注意: 本番環境では `SIMPLE_MODE_CONFIRM=true` も設定する必要があります

---

## Antigravity サポート

AySub は [Antigravity](https://antigravity.so/) アカウントをサポートしています。認証後、Claude および Gemini モデル用の専用エンドポイントが利用可能になります。

### 専用エンドポイント

| エンドポイント | モデル |
|----------|-------|
| `/antigravity/v1/messages` | Claude モデル |
| `/antigravity/v1beta/` | Gemini モデル |

### Claude Code の設定

```bash
export ANTHROPIC_BASE_URL="http://localhost:8080/antigravity"
export ANTHROPIC_AUTH_TOKEN="sk-xxx"
```

### ハイブリッドスケジューリングモード

Antigravity アカウントはオプションの**ハイブリッドスケジューリング**をサポートしています。有効にすると、汎用エンドポイント `/v1/messages` および `/v1beta/` も Antigravity アカウントにリクエストをルーティングします。

> **⚠️ 警告**: Anthropic Claude と Antigravity Claude は**同じ会話コンテキスト内で混在させることはできません**。グループを使用して適切に分離してください。

### 既知の問題

Claude Code では、Plan Mode を自動的に終了できません。（通常、ネイティブの Claude API を使用する場合、計画が完了すると Claude Code はユーザーに計画を承認または拒否するオプションをポップアップ表示します。）

**回避策**: `Shift + Tab` を押して手動で Plan Mode を終了し、計画を承認または拒否するためのレスポンスを入力してください。

---

## プロジェクト構成

```
AySub/
├── backend/                  # Go バックエンドサービス
│   ├── cmd/server/           # アプリケーションエントリ
│   ├── internal/             # 内部モジュール
│   │   ├── config/           # 設定
│   │   ├── model/            # データモデル
│   │   ├── service/          # ビジネスロジック
│   │   ├── handler/          # HTTP ハンドラー
│   │   └── gateway/          # API ゲートウェイコア
│   └── resources/            # 静的リソース
│
├── frontend/                 # Vue 3 フロントエンド
│   └── src/
│       ├── api/              # API 呼び出し
│       ├── stores/           # 状態管理
│       ├── views/            # ページコンポーネント
│       └── components/       # 再利用可能なコンポーネント
│
└── deploy/                   # デプロイファイル
    ├── docker-compose.yml    # Docker Compose 設定
    ├── .env.example          # Docker Compose 用環境変数
    ├── config.example.yaml   # バイナリデプロイ用フル設定ファイル
    └── install.sh            # ワンクリックインストールスクリプト
```

## 免責事項

> **本プロジェクトをご利用の前に、以下をよくお読みください:**
>
> :rotating_light: **利用規約違反のリスク**: 本プロジェクトの使用は Anthropic の利用規約に違反する可能性があります。使用前に Anthropic のユーザー契約をよくお読みください。本プロジェクトの使用に起因するすべてのリスクは、ユーザー自身が負うものとします。
>
> :book: **免責事項**: 本プロジェクトは技術的な学習および研究目的のみで提供されています。作者は、本プロジェクトの使用によるアカウント停止、サービス中断、その他の損失について一切の責任を負いません。

---

## スター履歴

<a href="https://star-history.com/#AIAllABOUTYOU/AySub&Date">
 <picture>
   <source media="(prefers-color-scheme: dark)" srcset="https://api.star-history.com/svg?repos=AIAllABOUTYOU/AySub&type=Date&theme=dark" />
   <source media="(prefers-color-scheme: light)" srcset="https://api.star-history.com/svg?repos=AIAllABOUTYOU/AySub&type=Date" />
   <img alt="Star History Chart" src="https://api.star-history.com/svg?repos=AIAllABOUTYOU/AySub&type=Date" />
 </picture>
</a>

---

## ライセンス

本プロジェクトは [GNU Lesser General Public License v3.0](LICENSE)（またはそれ以降のバージョン）の下でライセンスされています。

---

<div align="center">

**このプロジェクトが役に立ったら、ぜひスターをお願いします！**

</div>
