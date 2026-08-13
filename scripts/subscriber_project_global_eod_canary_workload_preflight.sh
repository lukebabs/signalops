#!/usr/bin/env bash
set -euo pipefail

usage() {
  printf '%s\n' \
    'Usage: scripts/subscriber_project_global_eod_canary_workload_preflight.sh' \
    '' \
    'Verifies the dedicated global-EOD workload login for the disabled S4 canary gate.' \
    'Required: SIGNALOPS_SUBSCRIBER_GLOBAL_EOD_DATABASE_URL' \
    'Required: SIGNALOPS_SUBSCRIBER_WORKLOAD_IDENTITY=subscriber-global-eod-reconciler' \
    '' \
    'This preflight never makes a market-data/provider request and does not arm collection.'
}

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
  usage
  exit 0
fi

if [[ -f .env ]]; then
  set -a
  # shellcheck disable=SC1091
  . ./.env
  set +a
fi

command -v psql >/dev/null 2>&1 || { printf 'psql is required\n' >&2; exit 3; }
database_url="${SIGNALOPS_SUBSCRIBER_GLOBAL_EOD_DATABASE_URL:-}"
workload_identity="${SIGNALOPS_SUBSCRIBER_WORKLOAD_IDENTITY:-}"
[[ -n "$database_url" ]] || { printf 'Missing SIGNALOPS_SUBSCRIBER_GLOBAL_EOD_DATABASE_URL.\n' >&2; exit 2; }
[[ "$workload_identity" == "subscriber-global-eod-reconciler" ]] || { printf 'Global EOD canary preflight failed: SIGNALOPS_SUBSCRIBER_WORKLOAD_IDENTITY must be subscriber-global-eod-reconciler.\n' >&2; exit 2; }

query() {
  psql "$database_url" -X -v ON_ERROR_STOP=1 -qAt -c "$1"
}

identity="$(query "SELECT current_user || '|' || rolsuper || '|' || rolcreaterole || '|' || rolbypassrls || '|' || pg_has_role(current_user, 'signalops_subscriber_global_eod', 'member') FROM pg_roles WHERE rolname=current_user")"
[[ "$identity" == *"|f|f|f|t" || "$identity" == *"|false|false|false|true" ]] || {
  printf 'Global EOD canary preflight failed: login must be non-superuser, non-CREATEROLE, NOBYPASSRLS, and a member of signalops_subscriber_global_eod (got %s).\n' "$identity" >&2
  exit 4
}

for table in subscriber_global_assets subscriber_global_asset_source_links subscriber_global_asset_reference_observations subscriber_global_catalog_seed_runs; do
  allowed="$(query "SELECT has_table_privilege(current_user, '${table}', 'SELECT')")"
  [[ "$allowed" == "t" || "$allowed" == "true" ]] || { printf 'Global EOD canary preflight failed: missing SELECT on %s.\n' "$table" >&2; exit 4; }
done

for table in subscriber_global_eod_canary_runs subscriber_global_eod_canary_members subscriber_global_eod_canary_execution_plans subscriber_global_eod_canary_execution_members subscriber_global_eod_canary_evidence_events; do
  allowed="$(query "SELECT has_table_privilege(current_user, '${table}', 'SELECT,INSERT')")"
  [[ "$allowed" == "t" || "$allowed" == "true" ]] || { printf 'Global EOD canary preflight failed: missing SELECT,INSERT on %s.\n' "$table" >&2; exit 4; }
done

coverage_allowed="$(query "SELECT has_table_privilege(current_user, 'subscriber_global_asset_coverage', 'SELECT,UPDATE')")"
[[ "$coverage_allowed" == "t" || "$coverage_allowed" == "true" ]] || { printf 'Global EOD canary preflight failed: missing SELECT,UPDATE on subscriber_global_asset_coverage.\n' >&2; exit 4; }

for table in subscriber_global_eod_canary_execution_plans subscriber_global_eod_canary_execution_members subscriber_global_eod_canary_evidence_events; do
  mutable="$(query "SELECT has_table_privilege(current_user, '${table}', 'UPDATE,DELETE')")"
  [[ "$mutable" == "f" || "$mutable" == "false" ]] || { printf 'Global EOD canary preflight failed: append-only table %s must not permit UPDATE or DELETE.\n' "$table" >&2; exit 4; }
done

for table in subscriber_watchlists subscriber_watchlist_memberships subscriber_tenant_entitlements marketops_universal_assets; do
  forbidden="$(query "SELECT has_table_privilege(current_user, '${table}', 'SELECT,INSERT,UPDATE,DELETE')")"
  [[ "$forbidden" == "f" || "$forbidden" == "false" ]] || { printf 'Global EOD canary preflight failed: worker must not access %s.\n' "$table" >&2; exit 4; }
done

printf 'Global EOD canary workload preflight passed: dedicated identity, least-privilege database access, and append-only evidence controls verified. Provider execution remains disabled.\n'
