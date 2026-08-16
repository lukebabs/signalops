#!/usr/bin/env bash
set -euo pipefail

[[ "${EUID}" -eq 0 ]] || {
  printf "%s\n" "Run this command as root." >&2
  exit 2
}

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=lib/marketops_boundary_env.sh
source "$root_dir/scripts/lib/marketops_boundary_env.sh"
boundary_env=/etc/signalops/marketops-boundary.env
runtime_env="${1:-${SIGNALOPS_PRODUCTION_ENV_FILE:-}}"

[[ -r "$boundary_env" ]] || {
  printf "%s\n" "Protected MarketOps boundary secret is not readable: $boundary_env" >&2
  exit 3
}
[[ -n "$runtime_env" && -r "$runtime_env" ]] || {
  printf "%s\n" "Provide a readable production Compose environment file as argument 1." >&2
  exit 2
}

# Load only the two allowlisted dedicated-database credentials as literal data.
# The migration runner receives them through Compose; this launcher never prints them.
load_marketops_boundary_env "$boundary_env"
compose=(docker compose --env-file "$runtime_env" -p signalops -f "$root_dir/compose.yaml" -f "$root_dir/compose.marketops-boundary.yaml")

# The historical 000124 repair is needed only when its migration-ledger row is
# absent. Replaying it after later migrations would restore an older constraint
# definition and regress newer evidence kinds.
ledger_000124="$("${compose[@]}" exec -T marketops-postgres psql -v ON_ERROR_STOP=1 -U signalops -d marketops -Atc "SELECT 1 FROM schema_migrations WHERE version='000124_subscriber_global_risk_reward_projection';")"
if [[ "$ledger_000124" != "1" ]]; then
  repair_file="$(mktemp)"
  trap "rm -f \"$repair_file\"" EXIT
  sed "s/^CREATE VIEW /CREATE OR REPLACE VIEW /" "$root_dir/migrations/000124_subscriber_global_risk_reward_projection.up.sql" > "$repair_file"
  "${compose[@]}" exec -T marketops-postgres psql -v ON_ERROR_STOP=1 -U signalops -d marketops < "$repair_file"
  "${compose[@]}" exec -T marketops-postgres psql -v ON_ERROR_STOP=1 -U signalops -d marketops -c "INSERT INTO schema_migrations(version) VALUES ('000124_subscriber_global_risk_reward_projection') ON CONFLICT DO NOTHING;"
  printf "%s\n" "marketops_migration_000124_ledger_reconciled"
else
  printf "%s\n" "marketops_migration_000124_ledger_already_present"
fi

# The standard migration runner is idempotent and records the exact version in
# schema_migrations. This action is intentionally limited to the primary
# MarketOps store; it does not touch the temporal database, Gateway, or jobs.
"${compose[@]}" --profile marketops-boundary run --rm marketops-postgres-migrate

"${compose[@]}" exec -T marketops-postgres \
  psql -v ON_ERROR_STOP=1 -U signalops -d marketops -c "SELECT version, applied_at FROM schema_migrations ORDER BY applied_at DESC LIMIT 2;"
"${compose[@]}" exec -T marketops-postgres \
  psql -v ON_ERROR_STOP=1 -U signalops -d marketops -c "SELECT count(*) AS global_market_state_records, max(session_date) AS latest_session_date FROM subscriber_gateway_global_market_states;"
"${compose[@]}" exec -T marketops-postgres \
  psql -v ON_ERROR_STOP=1 -U signalops -d marketops -c "\dp subscriber_gateway_global_market_states"
printf "%s\n" "marketops_global_market_state_projection_migration_verified"
