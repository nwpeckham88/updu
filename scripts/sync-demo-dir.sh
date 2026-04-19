#!/usr/bin/env bash

set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/.." && pwd)"

target_dir="${1:-$repo_root/demo}"
binary_path="${UPDU_BINARY_PATH:-$repo_root/bin/updu}"
demo_config_source="${UPDU_DEMO_CONFIG_SOURCE:-$repo_root/sample.updu.conf}"

relative_path() {
	if command -v python3 >/dev/null 2>&1; then
		python3 - "$1" "$2" <<'PY'
import os
import sys

print(os.path.relpath(sys.argv[1], sys.argv[2]))
PY
		return
	fi

	if command -v node >/dev/null 2>&1; then
		node -e 'const path = require("node:path"); console.log(path.relative(process.argv[2], process.argv[1]));' "$1" "$2"
		return
	fi

	echo "python3 or node required to calculate relative symlink paths" >&2
	exit 1
}

if [[ ! -e "$binary_path" ]]; then
	echo "expected built binary at $binary_path" >&2
	exit 1
fi

if [[ ! -e "$demo_config_source" ]]; then
	echo "expected demo config source at $demo_config_source" >&2
	exit 1
fi

mkdir -p "$target_dir/data"

binary_relative_path="$(relative_path "$binary_path" "$target_dir")"
config_relative_path="$(relative_path "$demo_config_source" "$target_dir")"

ln -sfn "$binary_relative_path" "$target_dir/updu"
ln -sfn "$config_relative_path" "$target_dir/updu.conf"

echo "synced demo workspace in $target_dir"