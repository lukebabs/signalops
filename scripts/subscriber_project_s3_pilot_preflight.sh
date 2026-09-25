#!/usr/bin/env bash
set -euo pipefail

usage() {
  printf '%s
'     'Usage: scripts/subscriber_project_s3_pilot_preflight.sh --tenant-id <tenant-id>'     ''     'Validates that a named Subscriber S3 pilot can be enabled safely.'     'It must run while SIGNALOPS_SUBSCRIBER_LISTS_ENABLED is false.'     'Required local/deployment configuration:'     '  SIGNALOPS_SUBSCRIBER_GATEWAY_DATABASE_URL'     '  SIGNALOPS_SUBSCRIBER_LISTS_PILOT_TENANTS (must include --tenant-id)'
}

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
  usage
  exit 0
fi

tenant_id=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    --tenant-id)
      tenant_id="${2:-}"
      shift 2
      ;;
    *)
      usage >&2
      exit 2
      ;;
  esac
done

[[ "$tenant_id" =~ ^[A-Za-z0-9][A-Za-z0-9_-]{0,127}$ ]] || {
  printf 'A valid --tenant-id is required.
' >&2
  exit 2
}

if [[ -f .env ]]; then
  set -a
  # shellcheck disable=SC1091
  . ./.env
  set +a
fi

case "${SIGNALOPS_SUBSCRIBER_LISTS_ENABLED:-false}" in
  0|false|FALSE|no|NO|off|OFF|"") ;;
  *)
    printf 'Pilot preflight must run before enablement; SIGNALOPS_SUBSCRIBER_LISTS_ENABLED is already true.
' >&2
    exit 4
    ;;
esac

command -v psql >/dev/null 2>&1 || { printf 'psql is required
' >&2; exit 3; }
database_url="${SIGNALOPS_SUBSCRIBER_GATEWAY_DATABASE_URL:-}"
[[ -n "$database_url" ]] || { printf 'Missing SIGNALOPS_SUBSCRIBER_GATEWAY_DATABASE_URL.
' >&2; exit 2; }

pilot_tenants=",${SIGNALOPS_SUBSCRIBER_LISTS_PILOT_TENANTS:-},"
[[ "$pilot_tenants" == *",$tenant_id,"* ]] || {
  printf 'SIGNALOPS_SUBSCRIBER_LISTS_PILOT_TENANTS must include %s before pilot validation.
' "$tenant_id" >&2
  exit 4
}

bash ./scripts/subscriber_project_gateway_workload_preflight.sh

query() {
  psql "$database_url" -X -v ON_ERROR_STOP=1 -qAt -c "$1"
}

schema_count="$(query "SELECT count(*) FROM pg_class WHERE relkind='r' AND relname IN ('subscriber_watchlists','subscriber_watchlist_memberships','subscriber_watchlist_audit')")"
[[ "$schema_count" == "3" ]] || {
  printf 'S3 pilot preflight failed: all three S3 list tables must exist.
' >&2
  exit 4
}

state="$(query "BEGIN;
SELECT set_config('signalops.tenant_id', '$tenant_id', true);
SELECT COALESCE((SELECT status FROM subscriber_tenant_entitlements WHERE tenant_id='$tenant_id'), 'missing');
SELECT count(*) FROM subscriber_watchlists WHERE tenant_id='$tenant_id' AND list_kind='tenant_default';
ROLLBACK;")"
entitlement_status="$(printf '%s
' "$state" | sed -n '2p')"
default_list_count="$(printf '%s
' "$state" | sed -n '3p')"

[[ "$entitlement_status" == "active" ]] || {
  printf 'S3 pilot preflight failed: tenant %s needs an active Subscriber entitlement (got %s).
' "$tenant_id" "$entitlement_status" >&2
  exit 4
}
[[ "$default_list_count" == "1" ]] || {
  printf 'S3 pilot preflight failed: tenant %s needs exactly one provisioned tenant-default list (got %s).
' "$tenant_id" "$default_list_count" >&2
  exit 4
}

printf 'S3 pilot preflight passed for %s: disabled flag posture, dedicated gateway login, forced RLS, active entitlement, and tenant-default list are ready.
' "$tenant_id"
