#!/usr/bin/env bash
set -euo pipefail
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
LOCK_FILE="${MARKETOPS_SRI_HOLDINGS_LOCK_FILE:-/tmp/signalops-marketops-sri-holdings.lock}"
exec 9>"$LOCK_FILE"
if ! flock -n 9; then
  printf '%s\n' "SRI issuer holdings refresh already running; skipping." >&2
  exit 0
fi
cd "$ROOT_DIR"
docker compose --profile marketops-daily run --rm marketops-sri-holdings-runner --tenant-id "${SIGNALOPS_TENANT_ID:-tenant-local}"
