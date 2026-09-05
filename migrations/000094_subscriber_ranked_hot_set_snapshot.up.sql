CREATE TABLE subscriber_global_ranking_snapshots (
  ranking_snapshot_id text PRIMARY KEY,
  source_label text NOT NULL,
  source_sha256 text NOT NULL,
  as_of_date date NOT NULL,
  requested_capacity integer NOT NULL CHECK (requested_capacity > 0 AND requested_capacity <= 1000),
  source_rows_examined integer NOT NULL CHECK (source_rows_examined >= 0),
  distinct_symbols_selected integer NOT NULL CHECK (distinct_symbols_selected >= 0 AND distinct_symbols_selected <= requested_capacity),
  duplicate_symbols_skipped integer NOT NULL DEFAULT 0 CHECK (duplicate_symbols_skipped >= 0),
  imported_by text NOT NULL,
  provenance jsonb NOT NULL DEFAULT '{}'::jsonb,
  is_current boolean NOT NULL DEFAULT false,
  imported_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX idx_subscriber_global_ranking_snapshots_current ON subscriber_global_ranking_snapshots (is_current) WHERE is_current;

CREATE TABLE subscriber_global_ranking_snapshot_entries (
  ranking_snapshot_id text NOT NULL REFERENCES subscriber_global_ranking_snapshots(ranking_snapshot_id) ON DELETE RESTRICT,
  selection_rank integer NOT NULL CHECK (selection_rank > 0 AND selection_rank <= 1000),
  source_rank integer NOT NULL CHECK (source_rank > 0),
  global_asset_id text NOT NULL REFERENCES subscriber_global_assets(global_asset_id) ON DELETE RESTRICT,
  provider_symbol text NOT NULL,
  company_name text NOT NULL,
  market_cap_raw text NOT NULL,
  revenue_raw text NOT NULL,
  source_row_sha256 text NOT NULL,
  PRIMARY KEY (ranking_snapshot_id, selection_rank),
  UNIQUE (ranking_snapshot_id, global_asset_id)
);
CREATE INDEX idx_subscriber_global_ranking_snapshot_entries_asset ON subscriber_global_ranking_snapshot_entries (global_asset_id, ranking_snapshot_id DESC);

ALTER TABLE subscriber_global_ranking_snapshots OWNER TO signalops_subscriber_migrator;
ALTER TABLE subscriber_global_ranking_snapshot_entries OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON subscriber_global_ranking_snapshots, subscriber_global_ranking_snapshot_entries FROM PUBLIC;
GRANT SELECT, INSERT, UPDATE ON subscriber_global_ranking_snapshots, subscriber_global_ranking_snapshot_entries TO signalops_subscriber_catalog_sync;
GRANT SELECT ON subscriber_global_ranking_snapshots, subscriber_global_ranking_snapshot_entries TO signalops_subscriber_global_eod;
GRANT INSERT, UPDATE ON subscriber_global_asset_identity_resolutions TO signalops_subscriber_catalog_sync;
