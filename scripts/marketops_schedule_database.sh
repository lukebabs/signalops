#!/usr/bin/env bash
# Shell helpers for MarketOps scheduled jobs. Source from a job script after
# changing to the repository root.
marketops_primary_psql() {
  docker compose exec -T "${SIGNALOPS_MARKETOPS_PRIMARY_DB_SERVICE:-postgres}" \
    psql -U signalops -d "${SIGNALOPS_MARKETOPS_PRIMARY_DATABASE:-signalops}" "$@"
}
marketops_temporal_psql() {
  docker compose exec -T "${SIGNALOPS_MARKETOPS_TEMPORAL_DB_SERVICE:-timescaledb}" \
    psql -U signalops -d "${SIGNALOPS_MARKETOPS_TEMPORAL_DATABASE:-signalops_temporal}" "$@"
}
