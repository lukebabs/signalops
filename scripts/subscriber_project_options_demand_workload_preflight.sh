#!/usr/bin/env bash
set -euo pipefail

usage() {
  printf '%s\n' \
    'Usage: scripts/subscriber_project_options_demand_workload_preflight.sh' \
    '' \
    'Verifies the dedicated S6 Options-demand shadow-planner database login.' \
    'Required: SIGNALOPS_SUBSCRIBER_OPTIONS_DEMAND_DATABASE_URL' \
    'Required: SIGNALOPS_SUBSCRIBER_WORKLOAD_IDENTITY=subscriber-options-demand-planner' \
    '' \
    'The preflight never calls a provider, runs a plan, or writes a snapshot.'
}

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then usage; exit 0; fi
if [[ -f .env ]]; then set -a; . ./.env; set +a; fi
command -v psql >/dev/null 2>&1 || { printf 'psql is required\n' >&2; exit 3; }
database_url="${SIGNALOPS_SUBSCRIBER_OPTIONS_DEMAND_DATABASE_URL:-}"
identity="${SIGNALOPS_SUBSCRIBER_WORKLOAD_IDENTITY:-}"
[[ -n "$database_url" ]] || { printf 'Missing SIGNALOPS_SUBSCRIBER_OPTIONS_DEMAND_DATABASE_URL.\n' >&2; exit 2; }
[[ "$identity" == 'subscriber-options-demand-planner' ]] || { printf 'Options-demand preflight failed: wrong workload identity.\n' >&2; exit 2; }
query(){ psql "$database_url" -X -v ON_ERROR_STOP=1 -qAt -c "$1"; }
role="$(query "SELECT current_user || '|' || rolsuper || '|' || rolcreaterole || '|' || rolbypassrls || '|' || pg_has_role(current_user, 'signalops_subscriber_options_demand', 'member') FROM pg_roles WHERE rolname=current_user")"
[[ "$role" == *'|f|f|f|t' || "$role" == *'|false|false|false|true' ]] || { printf 'Options-demand preflight failed: login must be non-superuser, non-CREATEROLE, NOBYPASSRLS, and in signalops_subscriber_options_demand (got %s).\n' "$role" >&2; exit 4; }
allowed="$(query "SELECT has_function_privilege(current_user, 'subscriber_options_demand_aggregate()', 'EXECUTE')")"
[[ "$allowed" == t || "$allowed" == true ]] || { printf 'Options-demand preflight failed: missing aggregate function execute privilege.\n' >&2; exit 4; }
for table in subscriber_options_demand_snapshot_runs subscriber_options_demand_snapshot_members; do
  allowed="$(query "SELECT has_table_privilege(current_user, '$table', 'SELECT,INSERT')")"
  [[ "$allowed" == t || "$allowed" == true ]] || { printf 'Options-demand preflight failed: missing SELECT,INSERT on %s.\n' "$table" >&2; exit 4; }
  mutable="$(query "SELECT has_table_privilege(current_user, '$table', 'UPDATE,DELETE')")"
  [[ "$mutable" == f || "$mutable" == false ]] || { printf 'Options-demand preflight failed: append-only table %s must not permit UPDATE or DELETE.\n' "$table" >&2; exit 4; }
done
for table in subscriber_tenant_entitlements subscriber_entitlement_capabilities subscriber_watchlists subscriber_watchlist_memberships subscriber_global_assets marketops_options_chain marketops_options_capture_sessions; do
  forbidden="$(query "SELECT has_table_privilege(current_user, '$table', 'SELECT,INSERT,UPDATE,DELETE')")"
  [[ "$forbidden" == f || "$forbidden" == false ]] || { printf 'Options-demand preflight failed: direct access to %s is forbidden.\n' "$table" >&2; exit 4; }
done
printf 'Options-demand workload preflight passed: aggregate-only input, append-only shadow storage, and no direct subscriber or Options-capture data access verified. Provider execution remains disabled.\n'
