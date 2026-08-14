#!/usr/bin/env bash
set -euo pipefail

# Creates a physical MarketOps-only copy. It never deletes, truncates, locks,
# or reconfigures the shared PostgreSQL/TimescaleDB sources. It only writes to
# the two new `marketops-*` services.

[[ "${MARKETOPS_BOUNDARY_ACKNOWLEDGE_WRITES:-}" == "true" ]] || {
  printf '%s\n' 'Refusing to write. Set MARKETOPS_BOUNDARY_ACKNOWLEDGE_WRITES=true after reviewing the boundary runbook.' >&2
  exit 2
}

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
compose=(docker compose -f "$root_dir/compose.yaml" -f "$root_dir/compose.marketops-boundary.yaml")

for command in docker awk sort; do
  command -v "$command" >/dev/null 2>&1 || { printf 'required command not found: %s\n' "$command" >&2; exit 3; }
done
for name in SIGNALOPS_MARKETOPS_POSTGRES_PASSWORD SIGNALOPS_MARKETOPS_TEMPORAL_PASSWORD; do
  [[ -n "${!name:-}" ]] || { printf 'missing required secret environment: %s\n' "$name" >&2; exit 2; }
done
[[ "$SIGNALOPS_MARKETOPS_POSTGRES_PASSWORD" =~ ^[A-Za-z0-9]{32,}$ ]] || {
  printf 'SIGNALOPS_MARKETOPS_POSTGRES_PASSWORD must be a URL-safe, 32-character minimum secret\n' >&2
  exit 2
}
[[ "$SIGNALOPS_MARKETOPS_TEMPORAL_PASSWORD" =~ ^[A-Za-z0-9]{32,}$ ]] || {
  printf 'SIGNALOPS_MARKETOPS_TEMPORAL_PASSWORD must be a URL-safe, 32-character minimum secret\n' >&2
  exit 2
}

"${compose[@]}" up -d --wait marketops-postgres marketops-timescaledb
# Fresh dedicated clusters need the no-login Subscriber migration roles.
"${compose[@]}" cp "$root_dir/deploy/postgres/marketops_boundary_subscriber_roles.sql" marketops-postgres:/tmp/marketops_boundary_subscriber_roles.sql
"${compose[@]}" exec -T marketops-postgres psql -v ON_ERROR_STOP=1 -U signalops -d marketops -f /tmp/marketops_boundary_subscriber_roles.sql
"${compose[@]}" exec -T marketops-postgres rm -f /tmp/marketops_boundary_subscriber_roles.sql

# An interrupted initial 000088 migration can leave only its empty pre-grant tables.
if [[ "$("${compose[@]}" exec -T marketops-postgres psql -U signalops -d marketops -Atc "SELECT count(*) FROM schema_migrations WHERE version = '000088_subscriber_entitlement_foundation';")" == "0" ]]; then
  "${compose[@]}" exec -T marketops-postgres psql -v ON_ERROR_STOP=1 -U signalops -d marketops -c "DROP TABLE IF EXISTS subscriber_entitlement_decision_audit CASCADE; DROP TABLE IF EXISTS subscriber_quota_reservations CASCADE; DROP TABLE IF EXISTS subscriber_entitlement_capabilities CASCADE; DROP TABLE IF EXISTS subscriber_tenant_entitlements CASCADE;"
fi
if [[ -n "${SIGNALOPS_MARKETOPS_PREVIOUS_POSTGRES_PASSWORD:-}" ]]; then
  PGPASSWORD="$SIGNALOPS_MARKETOPS_PREVIOUS_POSTGRES_PASSWORD" "${compose[@]}" exec -T marketops-postgres \
    psql -v ON_ERROR_STOP=1 -U signalops -d postgres -c "ALTER ROLE signalops PASSWORD '$SIGNALOPS_MARKETOPS_POSTGRES_PASSWORD';"
fi
if [[ -n "${SIGNALOPS_MARKETOPS_PREVIOUS_TEMPORAL_PASSWORD:-}" ]]; then
  PGPASSWORD="$SIGNALOPS_MARKETOPS_PREVIOUS_TEMPORAL_PASSWORD" "${compose[@]}" exec -T marketops-timescaledb \
    psql -v ON_ERROR_STOP=1 -U signalops -d postgres -c "ALTER ROLE signalops PASSWORD '$SIGNALOPS_MARKETOPS_TEMPORAL_PASSWORD';"
fi
"${compose[@]}" --profile marketops-boundary run --rm marketops-postgres-migrate
"${compose[@]}" --profile marketops-boundary run --rm marketops-timescaledb-migrate

primary_patterns=(
  '--table=public.marketops_*'
  '--table=public.sri_*'
  '--table=public.signal_assertions'
  '--table=public.signal_assertion_events'
  '--table=public.signal_assertion_evaluations'
  '--table=public.signal_assurance_registration_inbox'
  '--table=public.signal_effectiveness_snapshots'
  '--table=public.signal_validation_contracts'
  '--table=public.algorithm_*'
  '--table=public.platform_primitive_*'
  '--table=public.catalog_*'
  '--table=public.subscriber_*'
  '--table=public.registered_use_case_profiles'
  '--table=public.tenant_user_access'
  '--table=public.administration_notifications'
  '--table=public.administration_notification_*'
)

# Migration seed rows can conflict with the authoritative source data. Only
# target tables that are part of the explicit MarketOps copy are reset; shared
# source databases are never modified.
target_truncate_sql="$("${compose[@]}" exec -T marketops-postgres psql -U signalops -d marketops -Atc "
SELECT 'TRUNCATE TABLE ' || string_agg(format('%I.%I', schemaname, tablename), ', ') || ' RESTART IDENTITY CASCADE;'
FROM pg_tables
WHERE schemaname='public'
  AND (tablename LIKE 'marketops\\_%' ESCAPE '\\'
       OR tablename LIKE 'sri\\_%' ESCAPE '\\'
       OR tablename LIKE 'subscriber\\_%' ESCAPE '\\'
       OR tablename LIKE 'algorithm\\_%' ESCAPE '\\'
       OR tablename LIKE 'platform_primitive\\_%' ESCAPE '\\'
       OR tablename LIKE 'catalog\\_%' ESCAPE '\\'
       OR tablename IN ('signal_assertions','signal_assertion_events','signal_assertion_evaluations','signal_assurance_registration_inbox','signal_effectiveness_snapshots','signal_validation_contracts','registered_use_case_profiles','tenant_user_access','administration_notifications','administration_notification_deliveries','administration_notification_inbox_state'));" )"
[[ -n "$target_truncate_sql" ]] && "${compose[@]}" exec -T marketops-postgres psql -v ON_ERROR_STOP=1 -U signalops -d marketops -c "$target_truncate_sql"
# Reset the two target-only filtered ledgers before a retry. They never contain CyberOps rows.
"${compose[@]}" exec -T marketops-postgres psql -v ON_ERROR_STOP=1 -U signalops -d marketops -c "TRUNCATE TABLE public.normalized_event_ledger, public.signal_ledger RESTART IDENTITY CASCADE;"

audit_trigger_disabled=false
restore_audit_trigger() {
  if [[ "$audit_trigger_disabled" == "true" ]]; then
    "${compose[@]}" exec -T marketops-postgres psql -v ON_ERROR_STOP=1 -U signalops -d marketops -c "ALTER TABLE public.platform_primitive_definitions ENABLE TRIGGER USER;" || true
  fi
}
trap restore_audit_trigger EXIT
"${compose[@]}" exec -T marketops-postgres psql -v ON_ERROR_STOP=1 -U signalops -d marketops -c "ALTER TABLE public.platform_primitive_definitions DISABLE TRIGGER USER;"
audit_trigger_disabled=true
# The large shared ledgers are copied only for MarketOps rows. Binary COPY
# preserves types and avoids an intermediate plaintext file.
for table in normalized_event_ledger signal_ledger; do
  "${compose[@]}" exec -T postgres psql -U signalops -d signalops -c "COPY (SELECT * FROM public.$table WHERE app_id = 'marketops') TO STDOUT WITH (FORMAT binary)" \
    | "${compose[@]}" exec -T marketops-postgres psql -v ON_ERROR_STOP=1 -U signalops -d marketops -c "COPY public.$table FROM STDIN WITH (FORMAT binary)"
done

"${compose[@]}" exec -T postgres pg_dump -U signalops -d signalops --data-only --no-owner --no-privileges "${primary_patterns[@]}" \
  | "${compose[@]}" exec -T marketops-postgres psql -v ON_ERROR_STOP=1 -U signalops -d marketops

# Timescale data is MarketOps data by contract, apart from the two shared
"${compose[@]}" exec -T marketops-postgres psql -v ON_ERROR_STOP=1 -U signalops -d marketops -c "ALTER TABLE public.platform_primitive_definitions ENABLE TRIGGER USER;"
audit_trigger_disabled=false

"${compose[@]}" exec -T marketops-timescaledb psql -v ON_ERROR_STOP=1 -U signalops -d marketops_temporal -c "TRUNCATE TABLE public.marketdata_equity_eod_prices, public.marketdata_option_contracts_daily, public.normalized_event_ledger, public.signal_ledger RESTART IDENTITY;"
# ledgers which receive the same strict app_id filter.
"${compose[@]}" exec -T timescaledb pg_dump -U signalops -d signalops_temporal --data-only --no-owner --no-privileges \
  --table=public.marketdata_equity_eod_prices --table=public.marketdata_option_contracts_daily \
  | "${compose[@]}" exec -T marketops-timescaledb psql -v ON_ERROR_STOP=1 -U signalops -d marketops_temporal
for table in normalized_event_ledger signal_ledger; do
  "${compose[@]}" exec -T timescaledb psql -U signalops -d signalops_temporal -c "COPY (SELECT * FROM public.$table WHERE app_id = 'marketops') TO STDOUT WITH (FORMAT binary)" \
    | "${compose[@]}" exec -T marketops-timescaledb psql -v ON_ERROR_STOP=1 -U signalops -d marketops_temporal -c "COPY public.$table FROM STDIN WITH (FORMAT binary)"
done

"$root_dir/scripts/verify_marketops_database_boundary.sh"
printf '%s\n' 'MarketOps-only database boundary bootstrap completed. No application has been cut over; keep all production reads and writes on the shared services until dual-store validation is approved.'
