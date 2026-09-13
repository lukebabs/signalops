#!/usr/bin/env bash
set -Eeuo pipefail

fail() {
  printf '%s
' "$*" >&2
  exit 2
}

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
config_path="${SIGNALOPS_PGBACKREST_CONFIG_PATH:-/etc/signalops/pgbackrest.conf}"
project="${SIGNALOPS_COMPOSE_PROJECT:-signalops}"

[[ -r "$config_path" ]] || fail "shared pgBackRest config is not readable: $config_path"

if [[ -x "$root_dir/scripts/refresh_signalops_pgbackrest_credentials.sh" && -r /etc/signalops/pgbackrest-source.env ]]; then
  "$root_dir/scripts/refresh_signalops_pgbackrest_credentials.sh"
fi

compose=(
  docker compose
  -p "$project"
  -f "$root_dir/compose.yaml"
  -f "$root_dir/compose.pgbackrest.yaml"
)

SIGNALOPS_PGBACKREST_CONFIG_PATH="$config_path" "${compose[@]}" config --quiet
SIGNALOPS_PGBACKREST_CONFIG_PATH="$config_path" "${compose[@]}" up -d --build postgres
SIGNALOPS_PGBACKREST_CONFIG_PATH="$config_path" "${compose[@]}" exec -T --user postgres postgres pgbackrest --stanza=signalops check
"$root_dir/scripts/verify_signalops_shared_postgres_archive_health.sh"
