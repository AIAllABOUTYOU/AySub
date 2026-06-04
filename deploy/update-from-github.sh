#!/usr/bin/env bash
set -Eeuo pipefail

REPO_URL="${AYSUB_REPO_URL:-https://github.com/AIAllABOUTYOU/AySub.git}"
BRANCH="${AYSUB_BRANCH:-main}"
DEPLOY_ROOT="${AYSUB_DEPLOY_ROOT:-/opt/AySub-current}"
DATA_SOURCE="${AYSUB_DATA_SOURCE:-}"
COMPOSE_FILE="${AYSUB_COMPOSE_FILE:-docker-compose.local.yml}"
IMAGE_NAME="${AYSUB_IMAGE:-aysub:latest}"
HEALTH_TIMEOUT="${AYSUB_HEALTH_TIMEOUT:-180}"
FORCE=false

usage() {
  cat <<'USAGE'
Usage:
  update-from-github.sh [options]

Options:
  --root PATH          Fixed deployment root. Default: /opt/AySub-current
  --branch NAME        Git branch to deploy. Default: main
  --repo URL           Git repository URL. Default: https://github.com/AIAllABOUTYOU/AySub.git
  --data-source PATH   Existing deploy directory to reuse .env and data directories from.
                       Example: /opt/AySub/deploy
  --force              Reset tracked code changes in the fixed deployment root before updating.
  -h, --help           Show this help.

Environment variables:
  AYSUB_DEPLOY_ROOT, AYSUB_BRANCH, AYSUB_REPO_URL, AYSUB_DATA_SOURCE,
  AYSUB_COMPOSE_FILE, AYSUB_IMAGE, AYSUB_HEALTH_TIMEOUT
USAGE
}

log() {
  printf '[AySub update] %s\n' "$*"
}

fail() {
  printf '[AySub update] ERROR: %s\n' "$*" >&2
  exit 1
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --root)
      DEPLOY_ROOT="${2:-}"
      shift 2
      ;;
    --branch)
      BRANCH="${2:-}"
      shift 2
      ;;
    --repo)
      REPO_URL="${2:-}"
      shift 2
      ;;
    --data-source)
      DATA_SOURCE="${2:-}"
      shift 2
      ;;
    --force)
      FORCE=true
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      fail "unknown option: $1"
      ;;
  esac
done

[[ -n "$DEPLOY_ROOT" ]] || fail "--root cannot be empty"
[[ -n "$BRANCH" ]] || fail "--branch cannot be empty"
[[ -n "$REPO_URL" ]] || fail "--repo cannot be empty"

if ! command -v git >/dev/null 2>&1; then
  fail "git is not installed"
fi
if ! command -v docker >/dev/null 2>&1; then
  fail "docker is not installed"
fi
if docker compose version >/dev/null 2>&1; then
  COMPOSE=(docker compose)
elif command -v docker-compose >/dev/null 2>&1; then
  COMPOSE=(docker-compose)
else
  fail "Docker Compose is not installed"
fi

prepare_repo() {
  if [[ -d "$DEPLOY_ROOT/.git" ]]; then
    log "using existing repository: $DEPLOY_ROOT"
    git config --global --add safe.directory "$DEPLOY_ROOT" >/dev/null 2>&1 || true
    git -C "$DEPLOY_ROOT" fetch origin "$BRANCH"

    local dirty
    dirty="$(git -C "$DEPLOY_ROOT" status --porcelain --untracked-files=no)"
    if [[ -n "$dirty" ]]; then
      if [[ "$FORCE" != true ]]; then
        printf '%s\n' "$dirty" >&2
        fail "tracked local changes exist in $DEPLOY_ROOT; commit them or rerun with --force"
      fi
      log "resetting tracked local changes because --force was provided"
      git -C "$DEPLOY_ROOT" reset --hard "origin/$BRANCH"
    fi

    git -C "$DEPLOY_ROOT" checkout "$BRANCH"
    git -C "$DEPLOY_ROOT" pull --ff-only origin "$BRANCH"
    return
  fi

  if [[ -e "$DEPLOY_ROOT" ]] && [[ -n "$(find "$DEPLOY_ROOT" -mindepth 1 -maxdepth 1 2>/dev/null)" ]]; then
    fail "$DEPLOY_ROOT exists but is not an empty Git repository"
  fi

  log "cloning $REPO_URL branch $BRANCH into $DEPLOY_ROOT"
  mkdir -p "$(dirname "$DEPLOY_ROOT")"
  git clone --branch "$BRANCH" "$REPO_URL" "$DEPLOY_ROOT"
  git config --global --add safe.directory "$DEPLOY_ROOT" >/dev/null 2>&1 || true
}

link_or_copy_runtime_file() {
  local name="$1"
  local source="$2"
  local target="$3"

  [[ -e "$target" || -L "$target" ]] && return
  [[ -n "$source" && -e "$source/$name" ]] || return

  if [[ -d "$source/$name" ]]; then
    ln -s "$source/$name" "$target"
    log "linked $target -> $source/$name"
  else
    cp -p "$source/$name" "$target"
    log "copied $source/$name -> $target"
  fi
}

prepare_runtime_files() {
  local deploy_dir="$DEPLOY_ROOT/deploy"
  local source=""
  mkdir -p "$deploy_dir"

  if [[ -n "$DATA_SOURCE" ]]; then
    [[ -d "$DATA_SOURCE" ]] || fail "--data-source does not exist: $DATA_SOURCE"
    source="$DATA_SOURCE"
  fi

  link_or_copy_runtime_file ".env" "$source" "$deploy_dir/.env"
  link_or_copy_runtime_file "data" "$source" "$deploy_dir/data"
  link_or_copy_runtime_file "postgres_data" "$source" "$deploy_dir/postgres_data"
  link_or_copy_runtime_file "redis_data" "$source" "$deploy_dir/redis_data"

  if [[ ! -f "$deploy_dir/.env" ]]; then
    cp "$deploy_dir/.env.example" "$deploy_dir/.env"
    log "created $deploy_dir/.env from .env.example; edit production secrets before first public use"
  fi

  mkdir -p "$deploy_dir/data" "$deploy_dir/postgres_data" "$deploy_dir/redis_data"
}

wait_for_health() {
  local deadline=$((SECONDS + HEALTH_TIMEOUT))
  local status=""

  while (( SECONDS < deadline )); do
    status="$(docker inspect -f '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' aysub 2>/dev/null || true)"
    log "container health: ${status:-unknown}"
    [[ "$status" == "healthy" || "$status" == "running" ]] && return 0
    sleep 3
  done

  return 1
}

smoke_test() {
  local url
  for url in /health /status /models /playground; do
    local code
    code="$(curl -fsS -o /dev/null -w '%{http_code}' "http://127.0.0.1:8080${url}" || true)"
    [[ "$code" == "200" ]] || fail "smoke test failed for ${url}: HTTP ${code:-000}"
    log "smoke ${url}: HTTP $code"
  done
}

main() {
  prepare_repo
  prepare_runtime_files

  cd "$DEPLOY_ROOT/deploy"
  [[ -f "$COMPOSE_FILE" ]] || fail "compose file not found: $COMPOSE_FILE"

  local old_commit new_commit backup_tag
  old_commit="$(git -C "$DEPLOY_ROOT" rev-parse --short HEAD 2>/dev/null || true)"
  new_commit="$(git -C "$DEPLOY_ROOT" rev-parse --short HEAD)"
  backup_tag="aysub:backup-$(date +%Y%m%d%H%M%S)"

  if docker image inspect "$IMAGE_NAME" >/dev/null 2>&1; then
    docker tag "$IMAGE_NAME" "$backup_tag"
    log "tagged previous image as $backup_tag"
  fi

  log "validating compose config"
  "${COMPOSE[@]}" -f "$COMPOSE_FILE" config >/dev/null

  log "building $IMAGE_NAME from $DEPLOY_ROOT"
  "${COMPOSE[@]}" -f "$COMPOSE_FILE" build aysub

  log "starting services"
  "${COMPOSE[@]}" -f "$COMPOSE_FILE" up -d

  wait_for_health || {
    "${COMPOSE[@]}" -f "$COMPOSE_FILE" ps
    "${COMPOSE[@]}" -f "$COMPOSE_FILE" logs --tail=160 aysub || true
    fail "aysub did not become healthy within ${HEALTH_TIMEOUT}s"
  }

  smoke_test
  "${COMPOSE[@]}" -f "$COMPOSE_FILE" ps
  log "deployed commit: $new_commit (previous: ${old_commit:-unknown})"
}

main "$@"
