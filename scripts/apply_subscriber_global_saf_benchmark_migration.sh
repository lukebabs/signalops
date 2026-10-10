#!/usr/bin/env bash
set -euo pipefail

# Applies only the reviewed additive SAF benchmark-evidence migration. It
# refuses to run against an unexpected ledger position or more than once.
[[ "${EUID}" -eq 0 ]] || { echo "Run this command as root." >&2; exit 2; }
root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$root_dir/scripts/lib/marketops_boundary_env.sh"
boundary_env=/etc/signalops/marketops-boundary.env
runtime_env="${1:-${SIGNALOPS_PRODUCTION_ENV_FILE:-}}"
[[ -r "$boundary_env" && -n "$runtime_env" && -r "$runtime_env" ]] || {
  echo "Protected boundary or runtime environment is unavailable." >&2
  exit 3
}
load_marketops_boundary_env "$boundary_env"
compose=(docker compose --env-file "$runtime_env" -p signalops -f "$root_dir/compose.yaml" -f "$root_dir/compose.marketops-boundary.yaml")

latest="$("${compose[@]}" exec -T marketops-postgres psql -U signalops -d marketops -Atc "SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1")"
[[ "$latest" == "000147_subscriber_subscription_commerce_foundation" ]] || {
  echo "Refusing migration: expected 000147 as latest applied migration, got ${latest:-none}." >&2
  exit 4
}
pending="$("${compose[@]}" exec -T marketops-postgres psql -U signalops -d marketops -Atc "SELECT count(*) FROM schema_migrations WHERE version='000148_subscriber_global_saf_benchmark_observations'")"
[[ "$pending" == "0" ]] || { echo "Migration 000148 is already recorded; refusing a duplicate run." >&2; exit 5; }

"${compose[@]}" --profile marketops-boundary run --rm marketops-postgres-migrate
"${compose[@]}" exec -T marketops-postgres psql -v ON_ERROR_STOP=1 -U signalops -d marketops -c "SELECT version, applied_at FROM schema_migrations WHERE version='000148_subscriber_global_saf_benchmark_observations';"
"${compose[@]}" exec -T marketops-postgres psql -v ON_ERROR_STOP=1 -U signalops -d marketops -c "SELECT has_table_privilege('signalops_subscriber_global_eod','subscriber_global_saf_benchmark_observations','SELECT,INSERT') AS writer_access, has_table_privilege('signalops_subscriber_gateway','subscriber_gateway_global_signal_assurance_observations','SELECT') AS gateway_access;"
echo "subscriber_global_saf_benchmark_migration_verified"
