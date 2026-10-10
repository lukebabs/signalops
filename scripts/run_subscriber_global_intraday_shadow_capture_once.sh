#!/usr/bin/env bash
set -euo pipefail

# This entrypoint is invoked only by the approved one-shot unit. It has no
# retry loop and never enables a recurring central shadow-capture schedule.
[[ "${EUID}" -eq 0 ]] || { echo "Run this command as root." >&2; exit 2; }
root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$root_dir/scripts/lib/marketops_boundary_env.sh"
boundary_env=/etc/signalops/marketops-boundary.env
runtime_env="$root_dir/.env"
[[ -r "$boundary_env" && -r "$runtime_env" ]] || { echo "Protected boundary or runtime environment is unavailable." >&2; exit 3; }
systemctl is-active --quiet signalops-marketops-boundary-intraday.timer || { echo "legacy_intraday_timer_inactive" >&2; exit 4; }
load_marketops_boundary_env "$boundary_env"
export SIGNALOPS_MARKETOPS_DATABASE_URL="postgres://signalops:${SIGNALOPS_MARKETOPS_POSTGRES_PASSWORD}@marketops-postgres:5432/marketops?sslmode=disable"
export SIGNALOPS_MARKETOPS_TEMPORAL_DATABASE_URL="postgres://signalops:${SIGNALOPS_MARKETOPS_TEMPORAL_PASSWORD}@marketops-timescaledb:5432/marketops_temporal?sslmode=disable"
docker compose --env-file "$runtime_env" -p signalops \
  -f "$root_dir/compose.yaml" \
  -f "$root_dir/compose.marketops-boundary.yaml" \
  -f "$root_dir/compose.marketops-scheduled-cutover.yaml" \
  --profile subscriber-global-evidence \
  run --rm --no-build subscriber-global-intraday-shadow-capture \
    --execute --correlation-id "approved-luke-20260816-one-shot"
