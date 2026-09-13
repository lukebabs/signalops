#!/usr/bin/env bash
set -euo pipefail
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=marketops_schedule_database.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/marketops_schedule_database.sh"
# shellcheck source=lib/marketops_locks.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib/marketops_locks.sh"
LOCK_FILE="${MARKETOPS_SRI_HOLDINGS_LOCK_FILE:-$(marketops_runtime_lock_file sri-holdings)}"
exec 9>"$LOCK_FILE"
if ! flock -n 9; then
  printf '%s\n' "SRI issuer holdings refresh already running; skipping." >&2
  exit 0
fi
cd "$ROOT_DIR"
marketops_compose --profile marketops-daily run --rm --no-deps marketops-sri-holdings-runner --tenant-id "${SIGNALOPS_SRI_OUTPUT_TENANT_ID:-platform-global}"
