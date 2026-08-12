#!/usr/bin/env bash
set -euo pipefail

usage() {
  printf '%s\n' \
    'Usage: scripts/subscriber_project_rls_preflight.sh' \
    '' \
    'Verifies the required Subscriber Project NOLOGIN database group roles.' \
    'Set SIGNALOPS_SUBSCRIBER_RLS_MIGRATOR_DATABASE_URL for a privileged migration connection,' \
    'or run from this repository with the local postgres Compose service available.'
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

database_url="${SIGNALOPS_SUBSCRIBER_RLS_MIGRATOR_DATABASE_URL:-}"

query_roles() {
  local sql="$1"
  if [[ -n "$database_url" ]]; then
    command -v psql >/dev/null 2>&1 || { printf 'psql is required for SIGNALOPS_SUBSCRIBER_RLS_MIGRATOR_DATABASE_URL.\n' >&2; return 3; }
    psql "$database_url" -v ON_ERROR_STOP=1 -At -F '|' -c "$sql"
    return
  fi
  command -v docker >/dev/null 2>&1 || { printf 'docker is required when no RLS migrator database URL is set.\n' >&2; return 3; }
  docker compose exec -T postgres psql -U signalops -d signalops -v ON_ERROR_STOP=1 -At -F '|' -c "$sql"
}

roles=(
  signalops_subscriber_migrator
  signalops_subscriber_gateway
  signalops_subscriber_catalog_sync
  signalops_subscriber_global_eod
  signalops_subscriber_options_demand
  signalops_subscriber_options_capture
)

for role in "${roles[@]}"; do
  row="$(query_roles "SELECT rolcanlogin, rolsuper, rolcreaterole, rolbypassrls FROM pg_roles WHERE rolname='${role}'")"
  [[ -n "$row" ]] || { printf 'RLS preflight failed: required role %s is missing.\n' "$role" >&2; exit 4; }
  [[ "$row" == "f|f|f|f" ]] || { printf 'RLS preflight failed: role %s must be NOLOGIN, non-superuser, non-CREATEROLE, and NOBYPASSRLS (got %s).\n' "$role" "$row" >&2; exit 4; }
done

printf 'Subscriber RLS role preflight passed for %d least-privilege group roles.\n' "${#roles[@]}"
printf 'No subscriber-private table is enabled by this preflight; validate forced RLS policies and workload login grants with the owning migration.\n'
