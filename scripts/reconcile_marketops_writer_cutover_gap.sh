#!/usr/bin/env bash
# Append only the MarketOps records that appeared in the shared store between
# the initial boundary copy and continuous-writer cutover. This never truncates
# or updates the dedicated boundary.
set -euo pipefail

src_primary=signalops-postgres-1
src_primary_db=signalops
dst_primary=signalops-marketops-postgres-1
dst_primary_db=marketops
src_temporal=signalops-timescaledb-1
src_temporal_db=signalops_temporal
dst_temporal=signalops-marketops-timescaledb-1
dst_temporal_db=marketops_temporal
stage_prefix=marketops_cutover_sync

copy_missing() {
  local source_container="$1" source_db="$2" destination_container="$3" destination_db="$4" table="$5" predicate="$6"
  local stage="${stage_prefix}_${table}_$$" inserted
  docker exec "$destination_container" psql -X -v ON_ERROR_STOP=1 -U signalops -d "$destination_db" -c "CREATE UNLOGGED TABLE public.${stage} (LIKE public.${table} INCLUDING DEFAULTS);"
  cleanup_stage() { docker exec "$destination_container" psql -X -q -U signalops -d "$destination_db" -c "DROP TABLE IF EXISTS public.${stage};" >/dev/null 2>&1 || true; }
  trap cleanup_stage RETURN
  docker exec "$source_container" psql -X -v ON_ERROR_STOP=1 -U signalops -d "$source_db" -c "COPY (SELECT * FROM public.${table} WHERE ${predicate}) TO STDOUT WITH (FORMAT binary)" | docker exec -i "$destination_container" psql -X -v ON_ERROR_STOP=1 -U signalops -d "$destination_db" -c "COPY public.${stage} FROM STDIN WITH (FORMAT binary)"
  inserted="$(docker exec "$destination_container" psql -X -qAt -v ON_ERROR_STOP=1 -U signalops -d "$destination_db" -c "WITH inserted AS (INSERT INTO public.${table} SELECT * FROM public.${stage} ON CONFLICT DO NOTHING RETURNING 1) SELECT count(*) FROM inserted;")"
  printf "Reconciled %s.%s: inserted %s missing rows.\n" "$destination_db" "$table" "$inserted"
}

copy_missing "$src_primary" "$src_primary_db" "$dst_primary" "$dst_primary_db" marketops_dsm_artifacts "true"
copy_missing "$src_primary" "$src_primary_db" "$dst_primary" "$dst_primary_db" marketops_dsm_graph_proposals "true"
copy_missing "$src_primary" "$src_primary_db" "$dst_primary" "$dst_primary_db" signal_ledger "app_id = 'marketops'"
copy_missing "$src_temporal" "$src_temporal_db" "$dst_temporal" "$dst_temporal_db" normalized_event_ledger "app_id = 'marketops'"
copy_missing "$src_temporal" "$src_temporal_db" "$dst_temporal" "$dst_temporal_db" signal_ledger "app_id = 'marketops'"
