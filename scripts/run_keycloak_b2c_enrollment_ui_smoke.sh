#!/usr/bin/env bash
# Run the read-only Keycloak B2C enrollment browser smoke.
# The dotenv file is parsed as literal data; it is never executed as shell code.
set -Eeuo pipefail

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
dotenv_path="${SIGNALOPS_E2E_ENV_FILE:-$repo_dir/.env}"

# shellcheck source=./lib/dotenv.sh
source "$repo_dir/scripts/lib/dotenv.sh"
load_dotenv "$dotenv_path"

export SIGNALOPS_E2E_BASE_URL="${SIGNALOPS_E2E_BASE_URL:-https://signalops.syncratic.io}"
export SIGNALOPS_E2E_AUTH_HOST="${SIGNALOPS_E2E_AUTH_HOST:-auth.syncratic.co}"
export SIGNALOPS_E2E_CLIENT_ID="${SIGNALOPS_E2E_CLIENT_ID:-signalops-web}"
export SIGNALOPS_E2E_ARTIFACT_DIR="${SIGNALOPS_E2E_ARTIFACT_DIR:-/tmp/signalops-enrollment-e2e-artifacts}"

exec "$repo_dir/.venv/bin/python" -m pytest -q \
  "$repo_dir/python/tests/test_keycloak_b2c_enrollment_ui.py"
