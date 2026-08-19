#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
config_path="${SIGNALOPS_PGBACKREST_CONFIG_PATH:-/etc/signalops/marketops-pgbackrest/pgbackrest.conf}"
boundary_env=/etc/signalops/marketops-boundary.env
action="${1:-scheduled}"

case "$action" in
  scheduled) if [[ "$(date -u +%d)" == "01" ]]; then backup_type=full; else backup_type=diff; fi ;;
  full|diff|incr) backup_type="$action" ;;
  check) backup_type="" ;;
  *) echo "Usage: ${0##*/} [scheduled|full|diff|incr|check]" >&2; exit 2 ;;
esac

[[ -r "$config_path" && -r "$boundary_env" ]] || { echo "pgBackRest configuration or MarketOps boundary secret is not readable." >&2; exit 3; }
compose=(docker compose -p signalops --env-file "$boundary_env" -f "$root_dir/compose.yaml" -f "$root_dir/compose.marketops-boundary.yaml" -f "$root_dir/compose.marketops-pgbackrest.yaml")
require_live_pgbackrest_service() {
  local service="$1"
  local container_id running
  container_id="$("${compose[@]}" ps -q "$service")"
  [[ -n "$container_id" ]] || { echo "MarketOps pgBackRest service is missing: $service" >&2; exit 4; }
  running="$(docker inspect --format '{{.State.Running}}' "$container_id")"
  [[ "$running" == "true" ]] || { echo "MarketOps pgBackRest service is not running: $service" >&2; exit 4; }
  docker exec --user postgres "$container_id" pgbackrest version >/dev/null || { echo "MarketOps service is not pgBackRest-capable: $service" >&2; exit 4; }
}
require_live_pgbackrest_service marketops-postgres
require_live_pgbackrest_service marketops-timescaledb
targets=("marketops-postgres marketops-primary" "marketops-timescaledb marketops-temporal")
for target in "${targets[@]}"; do
  read -r service stanza <<<"$target"
  if [[ -n "$backup_type" ]]; then
    SIGNALOPS_PGBACKREST_CONFIG_PATH="$config_path" "${compose[@]}" exec -T --user postgres "$service" pgbackrest --stanza="$stanza" --type="$backup_type" backup
  else
    SIGNALOPS_PGBACKREST_CONFIG_PATH="$config_path" "${compose[@]}" exec -T --user postgres "$service" pgbackrest --stanza="$stanza" check
  fi
done
