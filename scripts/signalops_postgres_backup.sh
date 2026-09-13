#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
config_path="${SIGNALOPS_PGBACKREST_CONFIG_PATH:-/etc/signalops/pgbackrest.conf}"
action="${1:-scheduled}"

case "$action" in
  scheduled)
    if [[ "$(date -u +%d)" == "01" ]]; then
      backup_type=full
    else
      backup_type=diff
    fi
    ;;
  full|diff|incr) backup_type="$action" ;;
  check)
    backup_type=""
    ;;
  *)
    printf 'Usage: %s [scheduled|full|diff|incr|check]\n' "${0##*/}" >&2
    exit 2
    ;;
esac

[[ -f "$config_path" && -r "$config_path" ]] || {
  printf 'pgBackRest configuration is not readable: %s\n' "$config_path" >&2
  exit 3
}

compose=(docker compose -f "$root_dir/compose.yaml" -f "$root_dir/compose.pgbackrest.yaml")
if [[ -n "$backup_type" ]]; then
  SIGNALOPS_PGBACKREST_CONFIG_PATH="$config_path" "${compose[@]}" exec -T --user postgres postgres \
    pgbackrest --stanza=signalops --type="$backup_type" backup
else
  SIGNALOPS_PGBACKREST_CONFIG_PATH="$config_path" "${compose[@]}" exec -T --user postgres postgres \
    pgbackrest --stanza=signalops check
fi
