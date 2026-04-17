#!/usr/bin/env bash
set -euo pipefail

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
repo_root=$(cd -- "$script_dir/../../.." && pwd)
auth_mode=${UPDU_E2E_AUTH_MODE:-password}
build_target=build

if [[ "$auth_mode" == "oidc" ]]; then
	build_target=build-oidc
fi

make -C "$repo_root" "$build_target"