#!/usr/bin/env bash
set -euo pipefail

[[ "${EUID}" -eq 0 ]] || { echo "Run this restore rehearsal as root because it creates isolated Docker volumes." >&2; exit 2; }
root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
config_path="${SIGNALOPS_PGBACKREST_CONFIG_PATH:-/etc/signalops/pgbackrest.conf}"
boundary_env=/etc/signalops/marketops-boundary.env
[[ -r "$config_path" && -r "$boundary_env" ]] || { echo "pgBackRest configuration or MarketOps boundary secret is not readable." >&2; exit 3; }
compose=(docker compose -p signalops --env-file "$boundary_env" -f "$root_dir/compose.yaml" -f "$root_dir/compose.marketops-boundary.yaml" -f "$root_dir/compose.marketops-pgbackrest.yaml")
targets=("marketops-postgres marketops-primary marketops signalops-marketops-postgres-pgbackrest:16" "marketops-timescaledb marketops-temporal marketops_temporal signalops-marketops-timescaledb-pgbackrest:2.17.2-pg16")
cleanup_targets=()
cleanup() {
  for target in "${cleanup_targets[@]}"; do
    read -r container volume <<<"$target"
    docker rm -f "$container" >/dev/null 2>&1 || true
    docker volume rm "$volume" >/dev/null 2>&1 || true
  done
}
trap cleanup EXIT
for target in "${targets[@]}"; do
  read -r service stanza database image <<<"$target"
  SIGNALOPS_PGBACKREST_CONFIG_PATH="$config_path" "${compose[@]}" exec -T --user postgres "$service" pgbackrest --stanza="$stanza" check
  suffix="$(date -u +%Y%m%d%H%M%S)-$stanza"
  volume="signalops-$stanza-restore-rehearsal-$suffix"
  container="signalops-$stanza-restore-rehearsal"
  cleanup_targets+=("$container $volume")
  docker volume create "$volume" >/dev/null
  docker run --rm --user root -v "$volume:/var/lib/postgresql/data" "$image" sh -ceu "chown -R postgres:postgres /var/lib/postgresql/data"
  docker run --rm --user postgres -v "$config_path:/etc/pgbackrest/pgbackrest.conf:ro" -v "$volume:/var/lib/postgresql/data" "$image" pgbackrest --stanza="$stanza" restore
  docker run -d --network none --name "$container" -e POSTGRES_PASSWORD=signalops-restore-rehearsal -v "$volume:/var/lib/postgresql/data" "$image" postgres >/dev/null
  for attempt in $(seq 1 30); do
    if docker exec "$container" pg_isready -U signalops -d "$database" >/dev/null; then break; fi
    sleep 2
  done
  docker exec "$container" pg_isready -U signalops -d "$database" >/dev/null
  docker exec "$container" psql -U signalops -d "$database" -tAc "SELECT 1" | rg -qx "1"
  echo "Restore rehearsal passed for $stanza: isolated database started and accepted a validation query."
done
echo "Dedicated MarketOps restore rehearsal passed. Temporary containers and volumes will now be removed."
