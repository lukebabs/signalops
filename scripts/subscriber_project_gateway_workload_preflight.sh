#!/usr/bin/env bash
set -euo pipefail

usage() {
  printf '%s\n' \
    'Usage: scripts/subscriber_project_gateway_workload_preflight.sh' \
    '' \
    'Verifies a dedicated subscriber gateway login can use only the forced-RLS' \
    'entitlement tables and cannot read them without a tenant context.' \
    'Required: SIGNALOPS_SUBSCRIBER_GATEWAY_DATABASE_URL'
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
database_url="${SIGNALOPS_SUBSCRIBER_GATEWAY_DATABASE_URL:-}"
[[ -n "$database_url" ]] || { printf 'Missing SIGNALOPS_SUBSCRIBER_GATEWAY_DATABASE_URL.\n' >&2; exit 2; }

query() {
  psql "$database_url" -X -v ON_ERROR_STOP=1 -qAt -c "$1"
}

identity="$(query "SELECT current_user || '|' || rolsuper || '|' || rolcreaterole || '|' || rolbypassrls || '|' || pg_has_role(current_user, 'signalops_subscriber_gateway', 'member') FROM pg_roles WHERE rolname=current_user")"
[[ "$identity" == *"|f|f|f|t" || "$identity" == *"|false|false|false|true" ]] || {
  printf 'Gateway workload preflight failed: login must be non-superuser, non-CREATEROLE, NOBYPASSRLS, and a member of signalops_subscriber_gateway (got %s).\n' "$identity" >&2
  exit 4
}

for table in subscriber_tenant_entitlements subscriber_entitlement_capabilities subscriber_quota_reservations subscriber_entitlement_decision_audit subscriber_entitlement_provisioning_audit subscriber_quota_reservation_audit subscriber_watchlists subscriber_watchlist_memberships subscriber_watchlist_audit; do
  allowed="$(query "SELECT has_table_privilege(current_user, '${table}', 'SELECT,INSERT,UPDATE,DELETE')")"
  [[ "$allowed" == "t" || "$allowed" == "true" ]] || { printf 'Gateway workload preflight failed: missing entitlement-table privilege on %s.\n' "$table" >&2; exit 4; }
done

legacy_access="$(query "SELECT has_table_privilege(current_user, 'marketops_universal_assets', 'SELECT')")"
[[ "$legacy_access" == "f" || "$legacy_access" == "false" ]] || {
  printf 'Gateway workload preflight failed: subscriber gateway login must not read legacy MarketOps asset ownership tables.\n' >&2
  exit 4
}

no_context="$(query "SELECT (SELECT count(*) FROM subscriber_tenant_entitlements) + (SELECT count(*) FROM subscriber_watchlists) + (SELECT count(*) FROM subscriber_watchlist_memberships) + (SELECT count(*) FROM subscriber_watchlist_audit)")"
[[ "$no_context" == "0" ]] || {
  printf 'Gateway workload preflight failed: tenant-private rows were visible without tenant context.\n' >&2
  exit 4
}

probe_result="$(query "BEGIN;
SET LOCAL signalops.tenant_id = 'subscriber_rls_probe';
INSERT INTO subscriber_tenant_entitlements (tenant_id, provisioning_version, status, provisioned_by)
VALUES ('subscriber_rls_probe', 'preflight-v1', 'active', 'subscriber-workload-preflight');
SELECT count(*) FROM subscriber_tenant_entitlements WHERE tenant_id='subscriber_rls_probe';
SET LOCAL signalops.tenant_id = 'subscriber_rls_probe_other';
SELECT count(*) FROM subscriber_tenant_entitlements WHERE tenant_id='subscriber_rls_probe';
ROLLBACK;")"
[[ "$probe_result" == $'1\n0' ]] || {
  printf 'Gateway workload preflight failed: forced RLS did not isolate the transaction-local tenant probe (got %q).\n' "$probe_result" >&2
  exit 4
}

printf 'Subscriber gateway workload preflight passed: dedicated login is least-privilege and forced-RLS tenant isolation passed.\n'
