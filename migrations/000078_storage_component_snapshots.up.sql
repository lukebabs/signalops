CREATE TABLE IF NOT EXISTS storage_component_snapshots (
  snapshot_id bigint NOT NULL REFERENCES storage_monitor_snapshots(snapshot_id) ON DELETE CASCADE,
  store_id text NOT NULL CHECK (store_id IN ('postgres','timescaledb','redpanda')),
  component_kind text NOT NULL CHECK (component_kind IN ('table','hypertable','topic','system')),
  component_name text NOT NULL,
  app_id text NOT NULL,
  domain text NOT NULL,
  attribution_method text NOT NULL CHECK (attribution_method IN ('exact','estimated','shared','unattributed')),
  physical_bytes bigint NOT NULL CHECK (physical_bytes >= 0),
  attributed_bytes bigint NOT NULL CHECK (attributed_bytes >= 0),
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  PRIMARY KEY (snapshot_id, component_kind, component_name, app_id, domain)
);
CREATE INDEX IF NOT EXISTS idx_storage_component_snapshots_store_observed ON storage_component_snapshots (store_id, snapshot_id DESC);
CREATE INDEX IF NOT EXISTS idx_storage_component_snapshots_owner ON storage_component_snapshots (app_id, domain, snapshot_id DESC);
