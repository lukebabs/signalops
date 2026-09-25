#!/usr/bin/env bash
set -euo pipefail

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
  printf '%s\n' 'Usage: scripts/subscriber_project_options_capture_workload_preflight.sh' 'Required: SIGNALOPS_SUBSCRIBER_OPTIONS_CAPTURE_DATABASE_URL' 'Required: SIGNALOPS_SUBSCRIBER_WORKLOAD_IDENTITY=subscriber-options-capture' 'Read-only: no provider, gate, capture, or scheduler operation is performed.'
  exit 0
fi
if [[ -f .env ]]; then set -a; . ./.env; set +a; fi
command -v psql >/dev/null 2>&1 || { printf 'psql is required\n' >&2; exit 3; }
database_url="${SIGNALOPS_SUBSCRIBER_OPTIONS_CAPTURE_DATABASE_URL:-}"
identity="${SIGNALOPS_SUBSCRIBER_WORKLOAD_IDENTITY:-}"
[[ -n "$database_url" && "$identity" == 'subscriber-options-capture' ]] || { printf 'Options-capture preflight failed: dedicated database URL and exact workload identity are required.\n' >&2; exit 2; }
query(){ psql "$database_url" -X -v ON_ERROR_STOP=1 -qAt -c "$1"; }
role="$(query "SELECT current_user || '|' || rolsuper || '|' || rolcreaterole || '|' || rolbypassrls || '|' || pg_has_role(current_user, 'signalops_subscriber_options_capture', 'member') FROM pg_roles WHERE rolname=current_user")"
[[ "$role" == *'|f|f|f|t' || "$role" == *'|false|false|false|true' ]] || { printf 'Options-capture preflight failed: non-admin capture group membership is required (got %s).\n' "$role" >&2; exit 4; }
for table in subscriber_options_demand_snapshot_runs subscriber_options_demand_snapshot_members subscriber_global_assets; do
  allowed="$(query "SELECT has_table_privilege(current_user, '$table', 'SELECT')")"; [[ "$allowed" == t || "$allowed" == true ]] || { printf 'Options-capture preflight failed: missing SELECT on %s.\n' "$table" >&2; exit 4; }
done
for table in subscriber_options_capture_canary_plans subscriber_options_capture_canary_members subscriber_options_capture_canary_evidence_events; do
  allowed="$(query "SELECT has_table_privilege(current_user, '$table', 'SELECT,INSERT')")"; [[ "$allowed" == t || "$allowed" == true ]] || { printf 'Options-capture preflight failed: missing SELECT,INSERT on %s.\n' "$table" >&2; exit 4; }
  mutable="$(query "SELECT has_table_privilege(current_user, '$table', 'UPDATE,DELETE')")"; [[ "$mutable" == f || "$mutable" == false ]] || { printf 'Options-capture preflight failed: append-only table %s must not permit UPDATE or DELETE.\n' "$table" >&2; exit 4; }
done
for table in subscriber_watchlists subscriber_watchlist_memberships subscriber_tenant_entitlements subscriber_entitlement_capabilities marketops_options_chain_daily marketops_options_capture_sessions; do
  forbidden="$(query "SELECT has_table_privilege(current_user, '$table', 'SELECT,INSERT,UPDATE,DELETE')")"; [[ "$forbidden" == f || "$forbidden" == false ]] || { printf 'Options-capture preflight failed: direct access to %s is forbidden.\n' "$table" >&2; exit 4; }
done
printf 'Options-capture workload preflight passed: aggregate snapshot input, append-only disabled gate access, and no direct subscriber or Options evidence access verified. Provider execution remains disabled.\n'
