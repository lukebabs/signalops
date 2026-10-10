#!/usr/bin/env bash
set -euo pipefail

# Applies only the reviewed append-only FMP-backed sector classification slice.
[[ "${EUID}" -eq 0 ]] || { echo "Run this command as root." >&2; exit 2; }
root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$root_dir/scripts/lib/marketops_boundary_env.sh"
boundary_env=/etc/signalops/marketops-boundary.env
runtime_env="${1:-${SIGNALOPS_PRODUCTION_ENV_FILE:-}}"
[[ -r "$boundary_env" && -n "$runtime_env" && -r "$runtime_env" ]] || { echo "Protected boundary or runtime environment is unavailable." >&2; exit 3; }
load_marketops_boundary_env "$boundary_env"
compose=(docker compose --env-file "$runtime_env" -p signalops -f "$root_dir/compose.yaml" -f "$root_dir/compose.marketops-boundary.yaml")
latest="$("${compose[@]}" exec -T marketops-postgres psql -U signalops -d marketops -Atc "SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1")"
[[ "$latest" == "000149_subscriber_global_saf_benchmark_cohort_reader" ]] || { echo "Refusing migration: expected 000149 as latest applied migration, got ${latest:-none}." >&2; exit 4; }
existing="$("${compose[@]}" exec -T marketops-postgres psql -U signalops -d marketops -Atc "SELECT count(*) FROM schema_migrations WHERE version='000150_subscriber_global_sector_normalization'")"
[[ "$existing" == "0" ]] || { echo "Migration 000150 is already recorded; refusing a duplicate run." >&2; exit 5; }
"${compose[@]}" --profile marketops-boundary run --rm marketops-postgres-migrate
"${compose[@]}" exec -T marketops-postgres psql -v ON_ERROR_STOP=1 -U signalops -d marketops -c "SELECT version, applied_at FROM schema_migrations WHERE version='000150_subscriber_global_sector_normalization'; SELECT canonical_sector,count(*) FROM subscriber_global_asset_sector_classifications GROUP BY canonical_sector ORDER BY canonical_sector; SELECT count(*) AS classified_legacy_assets FROM subscriber_global_assets WHERE reference_provenance->>'correlation_id'='saf-v2b-legacy-sector-normalization-20260817';"
echo "subscriber_global_sector_normalization_migration_verified"
