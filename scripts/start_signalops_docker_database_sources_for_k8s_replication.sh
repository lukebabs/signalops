#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cutover_env="${SIGNALOPS_MARKETOPS_CUTOVER_ENV:-/etc/signalops/marketops-cutover.env}"

fail() {
  printf 'signalops_k8s_production_database_source_start_failed: %s\n' "$*" >&2
  exit 1
}

[[ -r "$cutover_env" ]] || fail "cutover env is missing or unreadable: $cutover_env"
command -v docker >/dev/null 2>&1 || fail 'docker is required'

"$root_dir/scripts/render_marketops_cutover_env.sh" >/dev/null
compose=(docker compose --env-file "$cutover_env" -p signalops -f "$root_dir/compose.yaml" -f "$root_dir/compose.pgbackrest.yaml" -f "$root_dir/compose.marketops-boundary.yaml" -f "$root_dir/compose.marketops-pgbackrest.yaml")
"${compose[@]}" up -d --no-deps postgres timescaledb marketops-postgres marketops-timescaledb

for container in signalops-postgres-1 signalops-timescaledb-1 signalops-marketops-postgres-1 signalops-marketops-timescaledb-1; do
  running="$(docker inspect --format '{{.State.Running}}' "$container" 2>/dev/null || true)"
  [[ "$running" == "true" ]] || fail "database container did not start: $container"
done

printf 'signalops_k8s_production_database_sources_started\n'
printf 'containers=signalops-postgres-1,signalops-timescaledb-1,signalops-marketops-postgres-1,signalops-marketops-timescaledb-1\n'
printf 'app_traffic_restarted=false\nschedulers_started=false\nprovider_polling=false\n'
