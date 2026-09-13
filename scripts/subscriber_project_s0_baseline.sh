#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

usage() {
  printf '%s\n' \
    'Usage: scripts/subscriber_project_s0_baseline.sh [--tenant-id ID]' \
    '' \
    'Emits a read-only Markdown baseline of the current tenant-owned MarketOps plane.'
}

tenant_id="${SUBSCRIBER_BASELINE_TENANT_ID:-tenant-local}"
while (($# > 0)); do
  case "$1" in
    --tenant-id)
      [[ $# -ge 2 ]] || { printf 'missing value for --tenant-id\n' >&2; exit 2; }
      tenant_id="$2"
      shift 2
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      printf 'unknown argument: %s\n' "$1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

[[ "$tenant_id" =~ ^[A-Za-z0-9][A-Za-z0-9_-]{0,127}$ ]] || { printf 'invalid tenant id\n' >&2; exit 2; }

if [[ -f .env ]]; then
  set -a
  # shellcheck disable=SC1091
  . ./.env
  set +a
fi

command -v docker >/dev/null 2>&1 || { printf 'docker is required\n' >&2; exit 3; }

primary_query() {
  docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U signalops -d signalops -At -F ' | ' -c "$1"
}

temporal_query() {
  docker compose exec -T timescaledb psql -v ON_ERROR_STOP=1 -U signalops -d signalops_temporal -At -F ' | ' -c "$1"
}

sql_literal() {
  printf '%s' "$1" | sed "s/'/''/g"
}

baseline_tenant="$(sql_literal "$tenant_id")"
generated_at="$(date -u '+%Y-%m-%dT%H:%M:%SZ')"

printf '# Subscriber Project S0 baseline\n\n'
printf 'Generated: `%s`  \nTenant: `%s`  \nMode: read-only\n\n' "$generated_at" "$tenant_id"
printf 'This report performs SELECT-only database queries and reads local job-status files. It does not run a provider pull, scheduler, migration, API mutation, or ownership change.\n\n'

printf '## Storage contracts\n\n'
printf '| Store | Latest applied migration |\n|---|---|\n'
printf '| Primary | %s |\n' "$(primary_query "SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1")"
printf '| Temporal | %s |\n\n' "$(temporal_query "SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1")"

printf '## Current tenant-owned MarketOps universe\n\n'
printf '| Universe group | Active rows |\n|---|---|\n'
primary_query "SELECT universe_group, count(*) FROM marketops_universal_assets WHERE tenant_id='${baseline_tenant}' AND is_active GROUP BY universe_group ORDER BY universe_group" | awk -F ' [|] ' '{ printf "| %s | %s |\n", $1, $2 }'
printf '\n'
printf '| Metric | Value |\n|---|---|\n'
printf '| Distinct active tickers | %s |\n' "$(primary_query "SELECT count(DISTINCT ticker) FROM marketops_universal_assets WHERE tenant_id='${baseline_tenant}' AND is_active")"
printf '| Analyst-watchlist rows | %s |\n\n' "$(primary_query "SELECT count(*) FROM marketops_asset_universe WHERE tenant_id='${baseline_tenant}' AND universe_group='analyst_watchlist' AND is_active")"

printf '## Current shared evidence coverage\n\n'
printf '| Metric | Value |\n|---|---|\n'
printf '| Latest EOD observation date | %s |\n' "$(temporal_query "SELECT COALESCE(max((normalized_payload->>'observation_date')::date)::text, '') FROM normalized_event_ledger WHERE tenant_id='${baseline_tenant}' AND source_id='src-massive' AND dataset='equity_eod_prices'")"
printf '| Options symbols captured | %s |\n' "$(primary_query "SELECT count(DISTINCT symbol) FROM marketops_options_capture_sessions WHERE tenant_id='${baseline_tenant}'")"
printf '| Earliest options capture date | %s |\n' "$(primary_query "SELECT COALESCE(min(session_date)::text, '') FROM marketops_options_capture_sessions WHERE tenant_id='${baseline_tenant}'")"
printf '| Latest options capture date | %s |\n' "$(primary_query "SELECT COALESCE(max(session_date)::text, '') FROM marketops_options_capture_sessions WHERE tenant_id='${baseline_tenant}'")"
printf '| Options analytics-ready captures | %s |\n\n' "$(primary_query "SELECT count(*) FROM marketops_options_capture_sessions WHERE tenant_id='${baseline_tenant}' AND analytics_ready")"

printf '## Access and rollout controls\n\n'
printf '| Control | Current baseline |\n|---|---|\n'
printf '| Gateway authentication environment | `%s` |\n' "${SIGNALOPS_AUTH_ENABLED:-false}"
printf '| Browser authentication environment | `%s` |\n' "${VITE_SIGNALOPS_AUTH_ENABLED:-false}"
printf '| Tenant MarketOps access grants | %s |\n' "$(primary_query "SELECT count(*) FROM tenant_user_access WHERE tenant_id='${baseline_tenant}' AND app_id='marketops'")"
printf '| Subscriber rollout flags | Reserved only; no current runtime behavior |\n\n'

printf '## Recorded scheduled-job status\n\n'
printf '| Job file | Last recorded status |\n|---|---|\n'
shopt -s nullglob
for status_file in runtime/scheduled-jobs/*.json; do
  job_file="$(basename "$status_file")"
  status_value="$(tr '\n' ' ' < "$status_file" | sed -n 's/.*"status":"\([^"]*\)".*/\1/p')"
  printf '| %s | %s |\n' "$job_file" "${status_value:-not recorded}"
done

printf '\n## Reserved flags for later sprints\n\n'
printf '%s\n' \
  '- `SIGNALOPS_SUBSCRIBER_GLOBAL_CATALOG_ENABLED=false`' \
  '- `SIGNALOPS_SUBSCRIBER_LISTS_ENABLED=false`' \
  '- `SIGNALOPS_SUBSCRIBER_GLOBAL_EOD_SHADOW_ENABLED=false`' \
  '- `SIGNALOPS_SUBSCRIBER_GLOBAL_EOD_CANARY_ENABLED=false`' \
  '- `SIGNALOPS_SUBSCRIBER_OPTIONS_DEMAND_ENABLED=false`' \
  '' \
  'These names are contractual placeholders in S0. No current application component reads them; they must remain false until the owning sprint adds server-side enforcement, tests, observability, and rollback.'
