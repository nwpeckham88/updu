#!/usr/bin/env bash
set -euo pipefail

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
repo_root=$(cd -- "$script_dir/../../.." && pwd)
runtime_dir="$repo_root/.tmp/playwright/app"

app_host=${UPDU_E2E_HOST:-127.0.0.1}
app_port=${UPDU_E2E_PORT:-4010}
app_base_url=${UPDU_E2E_BASE_URL:-http://$app_host:$app_port}
auth_secret=${UPDU_E2E_AUTH_SECRET:-updu-playwright-auth-secret}
admin_user=${UPDU_E2E_ADMIN_USER:-admin}
admin_password=${UPDU_E2E_ADMIN_PASSWORD:-password123}
log_level=${UPDU_E2E_LOG_LEVEL:-warn}
auth_mode=${UPDU_E2E_AUTH_MODE:-password}
oidc_issuer=${UPDU_E2E_OIDC_ISSUER:-http://127.0.0.1:${UPDU_E2E_OIDC_PORT:-4012}}
oidc_client_id=${UPDU_E2E_OIDC_CLIENT_ID:-updu-playwright-client}
oidc_client_secret=${UPDU_E2E_OIDC_CLIENT_SECRET:-updu-playwright-secret}
oidc_redirect_url=${UPDU_E2E_OIDC_REDIRECT_URL:-$app_base_url/api/v1/auth/oidc/callback}
oidc_auto_register=${UPDU_E2E_OIDC_AUTO_REGISTER:-true}

binary_name=updu
prepare_hint='pnpm run test:e2e:prepare'

if [[ "$auth_mode" == "oidc" ]]; then
    binary_name=updu-oidc
    prepare_hint='pnpm run test:e2e:oidc:prepare'
fi

binary_path="$repo_root/bin/$binary_name"

rm -rf "$runtime_dir"
mkdir -p "$runtime_dir"

if [[ ! -x "$binary_path" ]]; then
    echo "missing built app at $binary_path; run $prepare_hint first" >&2
    exit 1
fi

cd "$runtime_dir"

if [[ "$auth_mode" == "oidc" ]]; then
    exec env \
        UPDU_HOST="$app_host" \
        UPDU_PORT="$app_port" \
        UPDU_BASE_URL="$app_base_url" \
        UPDU_DB_PATH="$runtime_dir/updu.db" \
        UPDU_AUTH_SECRET="$auth_secret" \
        UPDU_LOG_LEVEL="$log_level" \
        UPDU_ALLOW_LOCALHOST="true" \
        UPDU_OIDC_ISSUER="$oidc_issuer" \
        UPDU_OIDC_CLIENT_ID="$oidc_client_id" \
        UPDU_OIDC_CLIENT_SECRET="$oidc_client_secret" \
        UPDU_OIDC_REDIRECT_URL="$oidc_redirect_url" \
        UPDU_OIDC_AUTO_REGISTER="$oidc_auto_register" \
        "$binary_path"
fi

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