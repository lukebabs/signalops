#!/usr/bin/env bash
set -euo pipefail

# Read-only guard for every scheduled job that operates on the dedicated
# MarketOps boundary. Batch runners publish raw events to the broker; the
# continuous writers below must then route MarketOps envelopes to the same
# primary/temporal pair. Without this guard a pull can succeed while all of its
# usable output is silently persisted in the shared legacy stores.
root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root_dir"
# shellcheck source=marketops_schedule_database.sh
source "$root_dir/scripts/marketops_schedule_database.sh"

if [[ "${SIGNALOPS_MARKETOPS_PRIMARY_DB_SERVICE:-}" != "marketops-postgres" ]]; then
  printf '%s\n' 'marketops_data_plane_preflight_skipped_shared_topology'
  exit 0
fi

primary_expected='@marketops-postgres:5432/marketops?sslmode=disable'
temporal_expected='@marketops-timescaledb:5432/marketops_temporal?sslmode=disable'
services=(
  normalizer
  signal-persister
  marketops-signal-assurance-registrar
  marketops-signal-assurance-outbox
)

for service in "${services[@]}"; do
  # Resolve the already-running Compose service through Docker labels. This
  # keeps the topology check independent of Compose interpolation, so a broken
  # or absent secret injection is reported as a data-plane failure rather than
  # masking the actual writer-route state.
  container_id="$(docker ps -q --filter 'label=com.docker.compose.project=signalops' --filter "label=com.docker.compose.service=$service" | head -n 1)"
  [[ -n "$container_id" ]] || {
    printf 'marketops_data_plane_service_missing=%s\n' "$service" >&2
    exit 4
  }
  running="$(docker inspect --format '{{.State.Running}}' "$container_id")"
  [[ "$running" == "true" ]] || {
    printf 'marketops_data_plane_service_not_running=%s\n' "$service" >&2
    exit 4
  }
  env_lines="$(docker inspect --format '{{range .Config.Env}}{{println .}}{{end}}' "$container_id")"
  boundary_required="$(awk -F= '$1 == "SIGNALOPS_MARKETOPS_DATA_BOUNDARY_REQUIRED" {print substr($0, index($0, "=") + 1); exit}' <<<"$env_lines")"
  primary_url="$(awk -F= '$1 == "SIGNALOPS_MARKETOPS_DATABASE_URL" {print substr($0, index($0, "=") + 1); exit}' <<<"$env_lines")"
  temporal_url="$(awk -F= '$1 == "SIGNALOPS_MARKETOPS_TEMPORAL_DATABASE_URL" {print substr($0, index($0, "=") + 1); exit}' <<<"$env_lines")"
  [[ "$boundary_required" == "true" && "$primary_url" == *"$primary_expected" && "$temporal_url" == *"$temporal_expected" ]] || {
    printf 'marketops_data_plane_route_invalid=%s\n' "$service" >&2
    exit 4
  }
done

primary_database="$(marketops_primary_psql -Atc 'select current_database()' | tr -d '[:space:]')"
temporal_database="$(marketops_temporal_psql -Atc 'select current_database()' | tr -d '[:space:]')"
[[ "$primary_database" == "marketops" && "$temporal_database" == "marketops_temporal" ]] || {
  printf 'marketops_data_plane_database_mismatch=primary:%s temporal:%s\n' "$primary_database" "$temporal_database" >&2
  exit 4
}

printf '%s\n' 'marketops_data_plane_preflight_passed'
