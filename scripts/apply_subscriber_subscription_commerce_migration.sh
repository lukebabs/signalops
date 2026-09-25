#!/usr/bin/env bash
set -euo pipefail

# Apply only the reviewed Subscriber Subscription Commerce foundation to the
# dedicated MarketOps primary. The runner refuses to proceed unless 000147 is
# exactly the next migration, preventing an accidental bulk apply of future DDL.
[[ "${EUID}" -eq 0 ]] || { echo "Run this command as root." >&2; exit 2; }
root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$root_dir/scripts/lib/marketops_boundary_env.sh"
boundary_env=/etc/signalops/marketops-boundary.env
runtime_env="${1:-${SIGNALOPS_PRODUCTION_ENV_FILE:-}}"
[[ -r "$boundary_env" && -n "$runtime_env" && -r "$runtime_env" ]] || {
  echo "Protected boundary or runtime environment is unavailable." >&2
  exit 3
}
load_marketops_boundary_env "$boundary_env"
compose=(docker compose --env-file "$runtime_env" -p signalops -f "$root_dir/compose.yaml" -f "$root_dir/compose.marketops-boundary.yaml")

latest="$("${compose[@]}" exec -T marketops-postgres psql -U signalops -d marketops -Atc "SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1")"
[[ "$latest" == "000146_subscriber_global_intraday_shadow_capture" ]] || {
  echo "Refusing migration: expected 000146 as latest applied migration, got ${latest:-none}." >&2
  exit 4
}
pending="$("${compose[@]}" exec -T marketops-postgres psql -U signalops -d marketops -Atc "SELECT count(*) FROM schema_migrations WHERE version='000147_subscriber_subscription_commerce_foundation'")"
[[ "$pending" == "0" ]] || {
  echo "Migration 000147 is already recorded; refusing a duplicate run." >&2
  exit 5
}

"${compose[@]}" --profile marketops-boundary run --rm marketops-postgres-migrate
"${compose[@]}" exec -T marketops-postgres psql -v ON_ERROR_STOP=1 -U signalops -d marketops -c "SELECT version, applied_at FROM schema_migrations WHERE version='000147_subscriber_subscription_commerce_foundation';"
"${compose[@]}" exec -T marketops-postgres psql -v ON_ERROR_STOP=1 -U signalops -d marketops -c "
SELECT c.relname AS table_name, c.relrowsecurity AS rls_enabled, c.relforcerowsecurity AS rls_forced, owner.rolname AS owner
FROM pg_class c JOIN pg_roles owner ON owner.oid=c.relowner
WHERE c.relname IN ('subscriber_subject_subscriptions','subscriber_tenant_subscriptions','subscriber_subscription_seats','subscriber_subscription_feature_decisions','subscriber_subscription_audit_events')
ORDER BY c.relname;"
"${compose[@]}" exec -T marketops-postgres psql -v ON_ERROR_STOP=1 -U signalops -d marketops -c "
SELECT has_table_privilege('signalops_subscriber_gateway','subscriber_subscription_products','SELECT') AS products_read,
       has_table_privilege('signalops_subscriber_gateway','subscriber_subject_subscriptions','SELECT,INSERT,UPDATE,DELETE') AS subject_access,
       has_table_privilege('public','subscriber_subscription_products','SELECT') AS public_products_read,
       has_table_privilege('public','subscriber_billing_webhook_events','SELECT') AS public_webhooks_read;"
echo "subscriber_subscription_commerce_migration_verified"
