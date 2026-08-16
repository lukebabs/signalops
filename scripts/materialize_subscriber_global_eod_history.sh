#!/usr/bin/env bash
set -euo pipefail

[[ "${EUID}" -eq 0 ]] || {
  printf '%s\n' 'Run this command as root.' >&2
  exit 2
}

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=lib/marketops_boundary_env.sh
source "$root_dir/scripts/lib/marketops_boundary_env.sh"
boundary_env=/etc/signalops/marketops-boundary.env
runtime_env="${1:-${SIGNALOPS_PRODUCTION_ENV_FILE:-}}"

[[ -r "$boundary_env" ]] || {
  printf '%s\n' "Protected MarketOps boundary secret is not readable: $boundary_env" >&2
  exit 3
}
[[ -n "$runtime_env" && -r "$runtime_env" ]] || {
  printf '%s\n' 'Provide a readable production Compose environment file as argument 1.' >&2
  exit 2
}

# Parse only the dedicated primary/temporal credentials as literal dotenv data.
# The values are supplied directly to Compose and are never written or printed.
load_marketops_boundary_env "$boundary_env"
compose=(docker compose --env-file "$runtime_env" -p signalops -f "$root_dir/compose.yaml" -f "$root_dir/compose.marketops-boundary.yaml" -f "$root_dir/compose.marketops-writer-cutover.yaml")
run=("${compose[@]}" --profile subscriber-global-evidence run --rm subscriber-global-eod-history-materializer)

# A bounded dry-run proves the exact retained-source coverage before the
# append-only execution. The worker does not call an external provider.
"${run[@]}" --dry-run --limit 50000
correlation_id="subglobaleodhist-$(date -u +%Y%m%dT%H%M%SZ)"
"${run[@]}" --execute --limit 50000 --correlation-id "$correlation_id"

# Verify the globally owned output by its fixed algorithm identity. This does
# not read raw tenant evidence and does not activate an HTTP reader.
"${compose[@]}" exec -T marketops-postgres psql -v ON_ERROR_STOP=1 -U signalops -d marketops -c "
  SELECT count(*) AS eod_bar_records,
         count(DISTINCT global_asset_id) AS covered_assets,
         min(session_date) AS first_session,
         max(session_date) AS last_session
  FROM subscriber_global_marketops_evidence_records
  WHERE evidence_kind = 'eod_bar'
    AND algorithm_id = 'marketops.equity_eod.initial_capture'
    AND algorithm_version = 'v1';"
printf '%s\n' "subscriber_global_eod_history_materialization_verified correlation_id=$correlation_id"
