#!/usr/bin/env bash
set -Eeuo pipefail

fail() {
  printf '%s\n' "$*" >&2
  exit 2
}

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source_env="${SIGNALOPS_PGBACKREST_SOURCE_ENV:-/etc/signalops/pgbackrest-source.env}"
config_path="${SIGNALOPS_PGBACKREST_CONFIG_PATH:-/etc/signalops/pgbackrest.conf}"
project="${SIGNALOPS_COMPOSE_PROJECT:-signalops}"

[[ -r "$source_env" ]] || fail "shared pgBackRest source env is not readable: $source_env"
set -a
# shellcheck disable=SC1090
. "$source_env"
set +a

[[ -x "$root_dir/scripts/refresh_signalops_pgbackrest_credentials.sh" ]] || fail "pgBackRest credential refresher is not executable"
"$root_dir/scripts/refresh_signalops_pgbackrest_credentials.sh"
[[ -r "$config_path" ]] || fail "shared pgBackRest config is not readable after refresh: $config_path"

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
