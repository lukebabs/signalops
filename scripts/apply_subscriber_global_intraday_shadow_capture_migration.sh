#!/usr/bin/env bash
set -euo pipefail

[[ "${EUID}" -eq 0 ]] || { echo "Run this command as root." >&2; exit 2; }
root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$root_dir/scripts/lib/marketops_boundary_env.sh"
boundary_env=/etc/signalops/marketops-boundary.env
runtime_env="${1:-${SIGNALOPS_PRODUCTION_ENV_FILE:-}}"
[[ -r "$boundary_env" && -n "$runtime_env" && -r "$runtime_env" ]] || { echo "Protected boundary or runtime environment is unavailable." >&2; exit 3; }
load_marketops_boundary_env "$boundary_env"
compose=(docker compose --env-file "$runtime_env" -p signalops -f "$root_dir/compose.yaml" -f "$root_dir/compose.marketops-boundary.yaml")
"${compose[@]}" --profile marketops-boundary run --rm marketops-postgres-migrate
"${compose[@]}" exec -T marketops-postgres psql -v ON_ERROR_STOP=1 -U signalops -d marketops -c "SELECT version, applied_at FROM schema_migrations WHERE version='000146_subscriber_global_intraday_shadow_capture';"
"${compose[@]}" exec -T marketops-postgres psql -v ON_ERROR_STOP=1 -U signalops -d marketops -c "SELECT has_table_privilege('signalops_subscriber_global_eod','subscriber_global_intraday_shadow_capture_runs','SELECT,INSERT') AS worker_ledger_grant, has_table_privilege('signalops_subscriber_global_eod','subscriber_watchlist_memberships','SELECT') AS worker_raw_watchlist_read;"
echo "subscriber_global_intraday_shadow_capture_migration_verified"
