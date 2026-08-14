#!/usr/bin/env bash
# Shell helpers for MarketOps scheduled jobs. Source from a job script after
# changing to the repository root.
marketops_compose() {
  local root_dir
  root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

  # Scheduled jobs run on the dedicated boundary after cutover. Make that
  # compose topology explicit instead of relying on inherited COMPOSE_FILE,
  # which can silently route a job back to the shared database.
  if [[ "${SIGNALOPS_MARKETOPS_PRIMARY_DB_SERVICE:-}" == "marketops-postgres" ]]; then
    docker compose \
      -f "$root_dir/compose.yaml" \
      -f "$root_dir/compose.marketops-boundary.yaml" \
      -f "$root_dir/compose.marketops-scheduled-cutover.yaml" \
      "$@"
    return
  fi

  docker compose "$@"
}

marketops_primary_psql() {
  marketops_compose exec -T "${SIGNALOPS_MARKETOPS_PRIMARY_DB_SERVICE:-postgres}" \
    psql -U signalops -d "${SIGNALOPS_MARKETOPS_PRIMARY_DATABASE:-signalops}" "$@"
}
marketops_temporal_psql() {
  marketops_compose exec -T "${SIGNALOPS_MARKETOPS_TEMPORAL_DB_SERVICE:-timescaledb}" \
    psql -U signalops -d "${SIGNALOPS_MARKETOPS_TEMPORAL_DATABASE:-signalops_temporal}" "$@"
}
