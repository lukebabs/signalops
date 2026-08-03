CREATE TABLE IF NOT EXISTS storage_monitor_snapshots (
  snapshot_id bigserial PRIMARY KEY, store_id text NOT NULL CHECK (store_id IN ('postgres','timescaledb','redpanda')),
  observed_at timestamptz NOT NULL DEFAULT now(), used_bytes bigint NOT NULL CHECK (used_bytes >= 0), capacity_bytes bigint NOT NULL CHECK (capacity_bytes >= 0), free_bytes bigint NOT NULL CHECK (free_bytes >= 0),
  status text NOT NULL CHECK (status IN ('healthy','warning','critical','unavailable')), detail jsonb NOT NULL DEFAULT '{}'::jsonb
);
CREATE INDEX IF NOT EXISTS idx_storage_monitor_snapshots_store_observed ON storage_monitor_snapshots (store_id, observed_at DESC);
