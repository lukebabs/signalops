#!/usr/bin/env bash
# Execute the temporary Subscription enforcement UI canary. This uses the
# existing controlled pilot identity only; it never creates a user, changes a
# tenant contract, or invokes a market-data provider.
set -Eeuo pipefail

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
dotenv_path="${SIGNALOPS_E2E_ENV_FILE:-$repo_dir/.env}"
source "$repo_dir/scripts/lib/dotenv.sh"
load_dotenv "$dotenv_path"

: "${SIGNALOPS_WEB:?SIGNALOPS_WEB must identify the controlled Explorer pilot account}"
: "${SIGNALOPS_WEB_PASS:?SIGNALOPS_WEB_PASS must be set}"
: "${SIGNALOPS_WEB_ADMIN:?SIGNALOPS_WEB_ADMIN must identify the subscription administrator}"
: "${SIGNALOPS_WEB_PASS_ADMIN:?SIGNALOPS_WEB_PASS_ADMIN must be set}"
export SIGNALOPS_E2E_PILOT_USERNAME="$SIGNALOPS_WEB"
export SIGNALOPS_E2E_PILOT_PASSWORD="$SIGNALOPS_WEB_PASS"
export SIGNALOPS_E2E_ADMIN_USERNAME="$SIGNALOPS_WEB_ADMIN"
export SIGNALOPS_E2E_ADMIN_PASSWORD="$SIGNALOPS_WEB_PASS_ADMIN"
export SIGNALOPS_E2E_ARTIFACT_DIR="${SIGNALOPS_E2E_ARTIFACT_DIR:-/tmp/signalops-e2e-artifacts}"

exec "$repo_dir/.venv/bin/python" -m pytest -q "$repo_dir/python/tests/test_subscription_enforcement_canary_ui.py"
