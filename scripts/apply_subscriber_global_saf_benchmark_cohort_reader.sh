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
latest="$("${compose[@]}" exec -T marketops-postgres psql -U signalops -d marketops -Atc "SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1")"
[[ "$latest" == "000148_subscriber_global_saf_benchmark_observations" ]] || { echo "Refusing migration: expected 000148 as latest applied migration, got ${latest:-none}." >&2; exit 4; }
"${compose[@]}" --profile marketops-boundary run --rm marketops-postgres-migrate
"${compose[@]}" exec -T marketops-postgres psql -v ON_ERROR_STOP=1 -U signalops -d marketops -c "SELECT version, applied_at FROM schema_migrations WHERE version='000149_subscriber_global_saf_benchmark_cohort_reader'; SELECT has_function_privilege('signalops_subscriber_global_eod','subscriber_global_saf_benchmark_legacy_default_members()','EXECUTE') AS cohort_reader_access;"
echo "subscriber_global_saf_benchmark_cohort_reader_verified"
