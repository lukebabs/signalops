#!/usr/bin/env bash
# Run the authenticated tenant-local Market Intelligence timestamp regression.
set -Eeuo pipefail

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
dotenv_path="${SIGNALOPS_E2E_ENV_FILE:-$repo_dir/.env}"

# shellcheck source=./lib/dotenv.sh
source "$repo_dir/scripts/lib/dotenv.sh"
load_dotenv "$dotenv_path"

: "${SIGNALOPS_WEB_ADMIN:?SIGNALOPS_WEB_ADMIN must identify the tenant-local QA account}"
: "${SIGNALOPS_WEB_PASS_ADMIN:?SIGNALOPS_WEB_PASS_ADMIN must be set}"

export SIGNALOPS_E2E_ADMIN_USERNAME="$SIGNALOPS_WEB_ADMIN"
export SIGNALOPS_E2E_ADMIN_PASSWORD="$SIGNALOPS_WEB_PASS_ADMIN"

"$repo_dir/.venv/bin/python" -m pytest -q \
  "$repo_dir/python/tests/test_marketops_intelligence_timestamp_ui.py"
printf '%s\n' 'marketops_intelligence_timestamp_ui_smoke_passed'
