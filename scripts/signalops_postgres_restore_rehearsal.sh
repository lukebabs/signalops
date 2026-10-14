#!/usr/bin/env bash
set -euo pipefail

[[ "${EUID}" -eq 0 ]] || {
  printf 'Run this restore rehearsal as root because it creates an isolated Docker volume.\n' >&2
  exit 2
}

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
config_path="${SIGNALOPS_PGBACKREST_CONFIG_PATH:-/etc/signalops/pgbackrest.conf}"
restore_volume="signalops-pgbackrest-restore-rehearsal-$(date -u +%Y%m%d%H%M%S)"
restore_container="signalops-pgbackrest-restore-rehearsal"
image="signalops-postgres-pgbackrest:16"

[[ -r "$config_path" ]] || { printf 'missing rendered pgBackRest configuration: %s\n' "$config_path" >&2; exit 3; }

compose=(docker compose -f "$root_dir/compose.yaml" -f "$root_dir/compose.pgbackrest.yaml")
SIGNALOPS_PGBACKREST_CONFIG_PATH="$config_path" "${compose[@]}" exec -T --user postgres postgres \
  pgbackrest --stanza=signalops check

docker volume create "$restore_volume" >/dev/null
cleanup() {
  docker rm -f "$restore_container" >/dev/null 2>&1 || true
  docker volume rm "$restore_volume" >/dev/null 2>&1 || true
}
trap cleanup EXIT

docker run --rm --user root -v "$restore_volume:/var/lib/postgresql/data" "$image" \
  sh -ceu 'chown -R 70:70 /var/lib/postgresql/data'
docker run --rm --user 70:70 \
  -v "$config_path:/etc/pgbackrest/pgbackrest.conf:ro" \
  -v "$restore_volume:/var/lib/postgresql/data" \
  "$image" pgbackrest --stanza=signalops restore
docker run -d --name "$restore_container" \
  -e POSTGRES_PASSWORD=signalops-restore-rehearsal \
  -v "$restore_volume:/var/lib/postgresql/data" \
  "$image" postgres >/dev/null

for attempt in $(seq 1 30); do
  if docker exec "$restore_container" pg_isready -U signalops -d signalops >/dev/null; then
    break
  fi
  sleep 2
done
docker exec "$restore_container" pg_isready -U signalops -d signalops >/dev/null
docker exec "$restore_container" psql -U signalops -d signalops -tAc 'SELECT 1' | rg -qx '1'
printf '%s\n' 'Restore rehearsal passed: isolated database started and accepted a validation query. The temporary container and volume will now be removed.'
