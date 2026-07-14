#!/bin/bash

set -euo pipefail

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEPLOY_DIR="$(cd "${TEST_DIR}/.." && pwd)"
SCRIPT="${DEPLOY_DIR}/apple-container.sh"
TEST_ROOT="$(mktemp -d "${TMPDIR:-/tmp}/aysub-apple-test.XXXXXX")"
STATE_DIR="${TEST_ROOT}/state"
ENV_FILE="${TEST_ROOT}/aysub.env"

cleanup() {
    rm -rf "${TEST_ROOT}"
}
trap cleanup EXIT

fail() {
    printf 'FAIL: %s\n' "$*" >&2
    exit 1
}

assert_exists() {
    [[ -e "$1" ]] || fail "Expected path to exist: $1"
}

assert_missing() {
    [[ ! -e "$1" ]] || fail "Expected path to be absent: $1"
}

assert_line() {
    grep -Fqx -- "$2" "$1" || fail "Expected '$2' in $1"
}

export FAKE_CONTAINER_STATE="${STATE_DIR}"
export PATH="${TEST_DIR}/fixtures/bin:${PATH}"
export AYSUB_ENV_FILE="${ENV_FILE}"

mkdir -p "${STATE_DIR}"

"${SCRIPT}" init
[[ "$(stat -f '%Lp' "${ENV_FILE}")" == "600" ]] || fail "init did not create a mode-600 env file"
grep -q '^POSTGRES_PASSWORD=change_this_secure_password$' "${ENV_FILE}" && fail "init retained the placeholder password"

chmod 644 "${ENV_FILE}"
if "${SCRIPT}" up >/dev/null 2>&1; then
    fail "up accepted an insecure env file"
fi
chmod 600 "${ENV_FILE}"

"${SCRIPT}" up
assert_exists "${STATE_DIR}/containers/aysub-apple"
assert_exists "${STATE_DIR}/containers/aysub-apple-postgres"
assert_exists "${STATE_DIR}/containers/aysub-apple-redis"
assert_exists "${STATE_DIR}/running/aysub-apple"
assert_line "${STATE_DIR}/create/aysub-apple.args" "ghcr.io/aiallaboutyou/aysub:latest"
assert_line "${STATE_DIR}/create/aysub-apple.args" "aysub-apple-data:/app/storage"
grep -Fq 'exec su-exec aysub /app/aysub' "${STATE_DIR}/create/aysub-apple.args" || \
    fail "AySub container did not use the expected runtime user and binary"
assert_line "${STATE_DIR}/create/aysub-apple.env" "DATA_DIR=/app/storage/data"
assert_line "${STATE_DIR}/create/aysub-apple.env" "DATABASE_USER=aysub"
assert_line "${STATE_DIR}/create/aysub-apple.env" "DATABASE_DBNAME=aysub"
"${SCRIPT}" status >/dev/null

"${SCRIPT}" up --recreate
assert_exists "${STATE_DIR}/running/aysub-apple"
"${SCRIPT}" down
assert_missing "${STATE_DIR}/running/aysub-apple"
assert_missing "${STATE_DIR}/running/aysub-apple-postgres"
assert_missing "${STATE_DIR}/running/aysub-apple-redis"

"${SCRIPT}" destroy --yes
assert_missing "${STATE_DIR}/containers/aysub-apple"
assert_missing "${STATE_DIR}/networks/aysub-apple"
assert_exists "${STATE_DIR}/volumes/aysub-apple-data"

"${SCRIPT}" up
"${SCRIPT}" destroy --volumes --yes
assert_missing "${STATE_DIR}/volumes/aysub-apple-data"
assert_missing "${STATE_DIR}/volumes/aysub-apple-postgres-data"
assert_missing "${STATE_DIR}/volumes/aysub-apple-redis-data"

touch "${STATE_DIR}/system-running"
touch "${STATE_DIR}/containers/aysub-apple"
touch "${STATE_DIR}/unowned/container/aysub-apple"
if "${SCRIPT}" status >/dev/null 2>&1; then
    fail "status accepted an unowned same-name container"
fi

printf 'Apple container lifecycle tests passed.\n'
