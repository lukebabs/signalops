#!/usr/bin/env bash
# Run the read-only Subscriber Project browser smoke with the pilot QA identity.
# The dotenv file is parsed as literal data; it is never executed as shell code.
set -Eeuo pipefail

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
dotenv_path="${SIGNALOPS_E2E_ENV_FILE:-$repo_dir/.env}"

# shellcheck source=./lib/dotenv.sh
source "$repo_dir/scripts/lib/dotenv.sh"
load_dotenv "$dotenv_path"

: "${SIGNALOPS_WEB:?SIGNALOPS_WEB must identify the tenant-pilot-b QA account}"
: "${SIGNALOPS_WEB_PASS:?SIGNALOPS_WEB_PASS must be set for the tenant-pilot-b QA account}"

export SIGNALOPS_E2E_USERNAME="$SIGNALOPS_WEB"
export SIGNALOPS_E2E_PASSWORD="$SIGNALOPS_WEB_PASS"
export SIGNALOPS_E2E_WATCHLIST_NAME="${SIGNALOPS_E2E_WATCHLIST_NAME:-First List}"
export SIGNALOPS_E2E_TENANT_ID="${SIGNALOPS_E2E_TENANT_ID:-tenant-pilot-b}"
export SIGNALOPS_E2E_SHARED_TICKERS=""
export SIGNALOPS_E2E_PENDING_TICKERS=""
export SIGNALOPS_E2E_ARTIFACT_DIR="${SIGNALOPS_E2E_ARTIFACT_DIR:-/tmp/signalops-e2e-artifacts}"

# The tenant-local administrator is intentionally not loaded into this test.
# The subscriber smoke must remain a read-only pilot-user journey.
exec "$repo_dir/.venv/bin/python" -m pytest -q \
  "$repo_dir/python/tests/test_subscriber_ui_smoke.py"
