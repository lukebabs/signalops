#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
compose=(docker compose -f "$root_dir/compose.yaml" -f "$root_dir/compose.marketops-boundary.yaml")

for name in SIGNALOPS_MARKETOPS_POSTGRES_PASSWORD SIGNALOPS_MARKETOPS_TEMPORAL_PASSWORD; do
  [[ -n "${!name:-}" ]] || { printf 'missing required secret environment: %s\n' "$name" >&2; exit 2; }
done

assert_zero() {
  local service="$1" database="$2" sql="$3" label="$4" actual
  actual="$("${compose[@]}" exec -T "$service" psql -U signalops -d "$database" -Atc "$sql")"
  [[ "$actual" == "0" ]] || { printf 'Boundary verification failed: %s = %s (expected 0)\n' "$label" "$actual" >&2; exit 4; }
}

assert_zero marketops-postgres marketops "SELECT count(*) FROM cyberops_connect_raw_events;" 'primary CyberOps raw events'
assert_zero marketops-postgres marketops "SELECT count(*) FROM cyberops_connect_outbox;" 'primary CyberOps outbox events'
assert_zero marketops-postgres marketops "SELECT count(*) FROM normalized_event_ledger WHERE app_id <> 'marketops';" 'primary non-MarketOps normalized events'
assert_zero marketops-postgres marketops "SELECT count(*) FROM signal_ledger WHERE app_id <> 'marketops';" 'primary non-MarketOps signals'
assert_zero marketops-timescaledb marketops_temporal "SELECT count(*) FROM normalized_event_ledger WHERE app_id <> 'marketops';" 'temporal non-MarketOps normalized events'
assert_zero marketops-timescaledb marketops_temporal "SELECT count(*) FROM signal_ledger WHERE app_id <> 'marketops';" 'temporal non-MarketOps signals'

"${compose[@]}" exec -T marketops-postgres psql -U signalops -d marketops -Atc "SELECT 'marketops-postgres|' || pg_size_pretty(pg_database_size(current_database()));"
"${compose[@]}" exec -T marketops-timescaledb psql -U signalops -d marketops_temporal -Atc "SELECT 'marketops-timescaledb|' || pg_size_pretty(pg_database_size(current_database()));"
printf '%s\n' 'MarketOps-only data boundary verified: no CyberOps rows and no non-MarketOps shared-ledger rows in either target.'
