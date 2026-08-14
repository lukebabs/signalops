#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
config_path="${SIGNALOPS_PGBACKREST_CONFIG_PATH:-/etc/signalops/pgbackrest.conf}"
action="${1:-scheduled}"

case "$action" in
  scheduled) if [[ "$(date -u +%d)" == "01" ]]; then backup_type=full; else backup_type=diff; fi ;;
  full|diff|incr) backup_type="$action" ;;
  check) backup_type="" ;;
  *) echo "Usage: ${0##*/} [scheduled|full|diff|incr|check]" >&2; exit 2 ;;
esac

[[ -r "$config_path" ]] || { echo "pgBackRest configuration is not readable: $config_path" >&2; exit 3; }
compose=(docker compose -f "$root_dir/compose.yaml" -f "$root_dir/compose.marketops-boundary.yaml" -f "$root_dir/compose.marketops-pgbackrest.yaml")
targets=("marketops-postgres marketops-primary" "marketops-timescaledb marketops-temporal")
for target in "${targets[@]}"; do
  read -r service stanza <<<"$target"
  if [[ -n "$backup_type" ]]; then
    SIGNALOPS_PGBACKREST_CONFIG_PATH="$config_path" "${compose[@]}" exec -T --user postgres "$service" pgbackrest --stanza="$stanza" --type="$backup_type" backup
  else
    SIGNALOPS_PGBACKREST_CONFIG_PATH="$config_path" "${compose[@]}" exec -T --user postgres "$service" pgbackrest --stanza="$stanza" check
  fi
done
