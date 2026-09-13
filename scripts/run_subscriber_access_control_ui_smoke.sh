#!/usr/bin/env bash
# Run read-only browser/API access-control smoke for tenant isolation. Dotenv is
# parsed as literal data only; the smoke performs no mutations.
set -Eeuo pipefail

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
dotenv_path="${SIGNALOPS_E2E_ENV_FILE:-$repo_dir/.env}"
source "$repo_dir/scripts/lib/dotenv.sh"
load_dotenv "$dotenv_path"

: "${SIGNALOPS_WEB:?SIGNALOPS_WEB must identify the tenant-pilot-b QA account}"
: "${SIGNALOPS_WEB_PASS:?SIGNALOPS_WEB_PASS must be set}"
: "${SIGNALOPS_WEB_ADMIN:?SIGNALOPS_WEB_ADMIN must identify the tenant-local administrator QA account}"
: "${SIGNALOPS_WEB_PASS_ADMIN:?SIGNALOPS_WEB_PASS_ADMIN must be set}"
export SIGNALOPS_E2E_PILOT_USERNAME="$SIGNALOPS_WEB"
export SIGNALOPS_E2E_PILOT_PASSWORD="$SIGNALOPS_WEB_PASS"
export SIGNALOPS_E2E_ADMIN_USERNAME="$SIGNALOPS_WEB_ADMIN"
export SIGNALOPS_E2E_ADMIN_PASSWORD="$SIGNALOPS_WEB_PASS_ADMIN"
export SIGNALOPS_E2E_ARTIFACT_DIR="${SIGNALOPS_E2E_ARTIFACT_DIR:-/tmp/signalops-e2e-artifacts}"

exec "$repo_dir/.venv/bin/python" -m pytest -q "$repo_dir/python/tests/test_subscriber_access_control_ui.py"
