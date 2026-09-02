#!/usr/bin/env bash
# Run the authenticated Keycloak B2C enrollment resolver browser smoke.
# This uses an existing B2C QA account and may trigger idempotent enrollment
# provisioning for that subject. It never creates a Keycloak user or Stripe state.
set -Eeuo pipefail

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
dotenv_path="${SIGNALOPS_E2E_ENV_FILE:-$repo_dir/.env}"

# shellcheck source=./lib/dotenv.sh
source "$repo_dir/scripts/lib/dotenv.sh"
load_dotenv "$dotenv_path"

export SIGNALOPS_B2C_WEB="${SIGNALOPS_B2C_WEB:-${SYNCRATIC_QA_CLIENT:-}}"
export SIGNALOPS_B2C_WEB_PASS="${SIGNALOPS_B2C_WEB_PASS:-${SYNCRATIC_QA_PASS:-}}"

: "${SIGNALOPS_B2C_ENROLLMENT_SMOKE_ACK:?set SIGNALOPS_B2C_ENROLLMENT_SMOKE_ACK=approved to run the authenticated B2C enrollment smoke}"
if [[ "$SIGNALOPS_B2C_ENROLLMENT_SMOKE_ACK" != "approved" ]]; then
  printf 'Refusing authenticated B2C enrollment smoke: SIGNALOPS_B2C_ENROLLMENT_SMOKE_ACK must equal approved.\n' >&2
  exit 2
fi

: "${SIGNALOPS_B2C_WEB:?SIGNALOPS_B2C_WEB must identify the existing B2C QA account}"
: "${SIGNALOPS_B2C_WEB_PASS:?SIGNALOPS_B2C_WEB_PASS must be set for the existing B2C QA account}"

export SIGNALOPS_E2E_BASE_URL="${SIGNALOPS_E2E_BASE_URL:-https://signalops.syncratic.io}"
export SIGNALOPS_E2E_B2C_TENANT_ID="${SIGNALOPS_E2E_B2C_TENANT_ID:-tenant-b2c}"
export SIGNALOPS_E2E_ENROLLMENT_EXPECTED_STATE="${SIGNALOPS_E2E_ENROLLMENT_EXPECTED_STATE:-marketops_ready}"
export SIGNALOPS_E2E_ARTIFACT_DIR="${SIGNALOPS_E2E_ARTIFACT_DIR:-/tmp/signalops-b2c-enrollment-e2e-artifacts}"

exec "$repo_dir/.venv/bin/python" -m pytest -q \
  "$repo_dir/python/tests/test_keycloak_b2c_enrollment_authenticated_ui.py"
