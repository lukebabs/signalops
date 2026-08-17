#!/usr/bin/env bash
set -euo pipefail

# One controlled append-only SAF-V2a materialization. It uses only persisted
# normalized EOD history and never calls a market-data provider.
[[ "${EUID}" -eq 0 ]] || { echo "Run this command as root." >&2; exit 2; }
root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$root_dir/scripts/lib/marketops_boundary_env.sh"
boundary_env=/etc/signalops/marketops-boundary.env
runtime_env="${1:-${SIGNALOPS_PRODUCTION_ENV_FILE:-}}"
[[ -r "$boundary_env" && -n "$runtime_env" && -r "$runtime_env" ]] || { echo "Protected boundary or runtime environment is unavailable." >&2; exit 3; }
load_marketops_boundary_env "$boundary_env"
compose=export SIGNALOPS_MARKETOPS_DATABASE_URL="postgres://signalops:@marketops-postgres:5432/marketops?sslmode=disable"
export SIGNALOPS_MARKETOPS_TEMPORAL_DATABASE_URL="postgres://signalops:@marketops-timescaledb:5432/marketops_temporal?sslmode=disable"
(docker compose --env-file "$runtime_env" -p signalops -f "$root_dir/compose.yaml" -f "$root_dir/compose.marketops-boundary.yaml" -f "$root_dir/compose.marketops-writer-cutover.yaml")
before="$("${compose[@]}" exec -T marketops-postgres psql -U signalops -d marketops -Atc "SELECT count(*) FROM subscriber_global_saf_benchmark_observations")"
[[ "$before" == "0" ]] || { echo "Refusing materialization: expected zero benchmark rows, got ${before}." >&2; exit 4; }
"${compose[@]}" --profile subscriber-global-evidence run --rm --build subscriber-global-saf-benchmark-materializer --execute --max-observations 500 --correlation-id saf-v2a-legacy-default-20260817
"${compose[@]}" exec -T marketops-postgres psql -v ON_ERROR_STOP=1 -U signalops -d marketops -c "SELECT benchmark_kind, benchmark_resolution_state, count(*) FROM subscriber_global_saf_benchmark_observations GROUP BY benchmark_kind, benchmark_resolution_state ORDER BY benchmark_kind, benchmark_resolution_state;"
"${compose[@]}" exec -T marketops-postgres psql -v ON_ERROR_STOP=1 -U signalops -d marketops -c "SELECT count(*) AS legacy_outcomes FROM subscriber_gateway_global_signal_assurance_observations; SELECT count(*) AS benchmark_rows FROM subscriber_global_saf_benchmark_observations;"
echo "subscriber_global_saf_benchmark_materialization_verified"
