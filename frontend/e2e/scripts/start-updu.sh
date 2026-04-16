#!/usr/bin/env bash
set -euo pipefail

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
repo_root=$(cd -- "$script_dir/../../.." && pwd)
runtime_dir="$repo_root/.tmp/playwright/app"
binary_path="$repo_root/bin/updu"

app_host=${UPDU_E2E_HOST:-127.0.0.1}
app_port=${UPDU_E2E_PORT:-4010}
app_base_url=${UPDU_E2E_BASE_URL:-http://$app_host:$app_port}
auth_secret=${UPDU_E2E_AUTH_SECRET:-updu-playwright-auth-secret}
admin_user=${UPDU_E2E_ADMIN_USER:-admin}
admin_password=${UPDU_E2E_ADMIN_PASSWORD:-password123}
log_level=${UPDU_E2E_LOG_LEVEL:-warn}

rm -rf "$runtime_dir"
mkdir -p "$runtime_dir"

if [[ ! -x "$binary_path" ]]; then
    echo "missing built app at $binary_path; run pnpm run test:e2e:prepare first" >&2
    exit 1
fi

cd "$runtime_dir"

exec env \
    UPDU_HOST="$app_host" \
    UPDU_PORT="$app_port" \
    UPDU_BASE_URL="$app_base_url" \
    UPDU_DB_PATH="$runtime_dir/updu.db" \
    UPDU_AUTH_SECRET="$auth_secret" \
    UPDU_ADMIN_USER="$admin_user" \
    UPDU_ADMIN_PASSWORD="$admin_password" \
    UPDU_LOG_LEVEL="$log_level" \
    UPDU_ALLOW_LOCALHOST="true" \
    "$binary_path"