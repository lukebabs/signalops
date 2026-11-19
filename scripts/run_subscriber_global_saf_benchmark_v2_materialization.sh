#!/usr/bin/env bash
set -euo pipefail

# One controlled append-only SAF-V2b benchmark materialization after the
# provider-backed sector classifications have been applied. No provider calls.
[[ "${EUID}" -eq 0 ]] || { echo "Run this command as root." >&2; exit 2; }
root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$root_dir/scripts/lib/marketops_boundary_env.sh"
boundary_env=/etc/signalops/marketops-boundary.env
runtime_env="${1:-${SIGNALOPS_PRODUCTION_ENV_FILE:-}}"
[[ -r "$boundary_env" && -n "$runtime_env" && -r "$runtime_env" ]] || { echo "Protected boundary or runtime environment is unavailable." >&2; exit 3; }
load_marketops_boundary_env "$boundary_env"
export SIGNALOPS_MARKETOPS_DATABASE_URL="postgres://signalops:${SIGNALOPS_MARKETOPS_POSTGRES_PASSWORD}@marketops-postgres:5432/marketops?sslmode=disable"
export SIGNALOPS_MARKETOPS_TEMPORAL_DATABASE_URL="postgres://signalops:${SIGNALOPS_MARKETOPS_TEMPORAL_PASSWORD}@marketops-timescaledb:5432/marketops_temporal?sslmode=disable"
compose=(docker compose --env-file "$runtime_env" -p signalops -f "$root_dir/compose.yaml" -f "$root_dir/compose.marketops-boundary.yaml" -f "$root_dir/compose.marketops-writer-cutover.yaml")
classified="$("${compose[@]}" exec -T marketops-postgres psql -U signalops -d marketops -Atc "SELECT count(*) FROM subscriber_global_asset_sector_classifications WHERE correlation_id='saf-v2b-legacy-sector-normalization-20260817'")"
[[ "$classified" == "11" ]] || { echo "Refusing v2 materialization: expected eleven governed classifications, got ${classified:-0}." >&2; exit 4; }
before="$("${compose[@]}" exec -T marketops-postgres psql -U signalops -d marketops -Atc "SELECT count(*) FROM subscriber_global_saf_benchmark_observations WHERE calculation_version='saf_benchmark.v2'")"
[[ "$before" == "0" ]] || { echo "Refusing v2 materialization: expected zero saf_benchmark.v2 rows, got $before." >&2; exit 5; }
"${compose[@]}" --profile subscriber-global-evidence run --rm --build subscriber-global-saf-benchmark-materializer --execute --calculation-version saf_benchmark.v2 --max-observations 500 --correlation-id saf-v2b-legacy-default-20260817
"${compose[@]}" exec -T marketops-postgres psql -v ON_ERROR_STOP=1 -U signalops -d marketops -c "SELECT benchmark_kind, benchmark_resolution_state, count(*) FROM subscriber_global_saf_benchmark_observations WHERE calculation_version='saf_benchmark.v2' GROUP BY benchmark_kind, benchmark_resolution_state ORDER BY benchmark_kind, benchmark_resolution_state; SELECT sector_benchmark_state,count(*) FROM subscriber_gateway_global_signal_assurance_observations GROUP BY sector_benchmark_state ORDER BY sector_benchmark_state;"
echo "subscriber_global_saf_benchmark_v2_materialization_verified"
