#!/usr/bin/env bash
# Read-only real-browser validation of the live SAF benchmark-coverage panel.
set -Eeuo pipefail

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
dotenv_path="${SIGNALOPS_E2E_ENV_FILE:-$repo_dir/.env}"
source "$repo_dir/scripts/lib/dotenv.sh"
load_dotenv "$dotenv_path"

: "${SIGNALOPS_WEB_ADMIN:?SIGNALOPS_WEB_ADMIN must identify the tenant-local SAF QA account}"
: "${SIGNALOPS_WEB_PASS_ADMIN:?SIGNALOPS_WEB_PASS_ADMIN must be set}"
export SIGNALOPS_E2E_ADMIN_USERNAME="$SIGNALOPS_WEB_ADMIN"
export SIGNALOPS_E2E_ADMIN_PASSWORD="$SIGNALOPS_WEB_PASS_ADMIN"
export SIGNALOPS_E2E_ARTIFACT_DIR="${SIGNALOPS_E2E_ARTIFACT_DIR:-/tmp/signalops-e2e-artifacts}"

exec "$repo_dir/.venv/bin/python" -m pytest -q "$repo_dir/python/tests/test_signal_assurance_benchmark_ui_smoke.py"
