-- Subscriber Project S1: platform-owned global catalog shadow.
-- This migration is additive and has no browser/API grant. Existing tenant-owned
-- MarketOps reads remain the production path.

CREATE TABLE subscriber_global_assets (
  global_asset_id text PRIMARY KEY,
  source_id text NOT NULL,
  provider_symbol text NOT NULL,
  canonical_symbol text NOT NULL,
  company_name text NOT NULL DEFAULT '',
  asset_type text NOT NULL DEFAULT '',
  exchange text NOT NULL DEFAULT '',
  sector text NOT NULL DEFAULT '',
  industry text NOT NULL DEFAULT '',
  eligibility_status text NOT NULL CHECK (eligibility_status IN ('discovered', 'eligible', 'ineligible', 'suspended')),
  reference_effective_at timestamptz,
  reference_provenance jsonb NOT NULL DEFAULT '{}'::jsonb,
  first_seen_at timestamptz NOT NULL DEFAULT now(),
  last_seen_at timestamptz NOT NULL DEFAULT now(),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (source_id, provider_symbol)
);
CREATE INDEX idx_subscriber_global_assets_catalog_lookup ON subscriber_global_assets (eligibility_status, canonical_symbol, global_asset_id);

CREATE TABLE subscriber_global_asset_source_links (
  source_tenant_id text NOT NULL,
  source_universe_group text NOT NULL,
  source_ticker text NOT NULL,
  global_asset_id text NOT NULL REFERENCES subscriber_global_assets(global_asset_id) ON DELETE RESTRICT,
  source_rank integer NOT NULL CHECK (source_rank > 0),
  source_is_active boolean NOT NULL,
  source_metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  source_created_at timestamptz NOT NULL,
  source_updated_at timestamptz NOT NULL,
  first_observed_at timestamptz NOT NULL DEFAULT now(),
  last_observed_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (source_tenant_id, source_universe_group, source_ticker)
);
CREATE INDEX idx_subscriber_global_asset_source_links_global ON subscriber_global_asset_source_links (global_asset_id, source_is_active);

CREATE TABLE subscriber_global_asset_reference_observations (
  observation_id text PRIMARY KEY,
  global_asset_id text NOT NULL REFERENCES subscriber_global_assets(global_asset_id) ON DELETE RESTRICT,
  seed_run_id text NOT NULL,
  source_fingerprint text NOT NULL UNIQUE,
  observed_at timestamptz NOT NULL,
  source_tenant_id text NOT NULL,
  source_universe_group text NOT NULL,
  source_ticker text NOT NULL,
  reference_payload jsonb NOT NULL,
  provenance jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_subscriber_global_asset_reference_observations_asset_time ON subscriber_global_asset_reference_observations (global_asset_id, observed_at DESC);

CREATE TABLE subscriber_global_asset_coverage (
  global_asset_id text NOT NULL REFERENCES subscriber_global_assets(global_asset_id) ON DELETE RESTRICT,
  coverage_product text NOT NULL CHECK (coverage_product IN ('eod_baseline')),
  coverage_state text NOT NULL CHECK (coverage_state IN ('not_requested', 'queued', 'warming_up', 'active', 'degraded', 'suspended')),
  execution_mode text NOT NULL CHECK (execution_mode IN ('shadow', 'enabled')),
  active_source_rows integer NOT NULL DEFAULT 0 CHECK (active_source_rows >= 0),
  coverage_version text NOT NULL,
  observed_at timestamptz NOT NULL,
  reason_code text NOT NULL DEFAULT '',
  provenance jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (global_asset_id, coverage_product)
);
CREATE INDEX idx_subscriber_global_asset_coverage_shadow ON subscriber_global_asset_coverage (execution_mode, coverage_state, observed_at DESC);

CREATE TABLE subscriber_global_catalog_seed_runs (
  seed_run_id text PRIMARY KEY,
  source_tenant_id text NOT NULL,
  actor_identity text NOT NULL,
  correlation_id text NOT NULL DEFAULT '',
  source_rows integer NOT NULL DEFAULT 0 CHECK (source_rows >= 0),
  active_source_rows integer NOT NULL DEFAULT 0 CHECK (active_source_rows >= 0),
  distinct_global_assets integer NOT NULL DEFAULT 0 CHECK (distinct_global_assets >= 0),
  inserted_global_assets integer NOT NULL DEFAULT 0 CHECK (inserted_global_assets >= 0),
  observed_references integer NOT NULL DEFAULT 0 CHECK (observed_references >= 0),
  completed_at timestamptz,
  report jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now()
);

ALTER TABLE subscriber_global_assets OWNER TO signalops_subscriber_migrator;
ALTER TABLE subscriber_global_asset_source_links OWNER TO signalops_subscriber_migrator;
ALTER TABLE subscriber_global_asset_reference_observations OWNER TO signalops_subscriber_migrator;
ALTER TABLE subscriber_global_asset_coverage OWNER TO signalops_subscriber_migrator;
ALTER TABLE subscriber_global_catalog_seed_runs OWNER TO signalops_subscriber_migrator;

REVOKE ALL ON subscriber_global_assets, subscriber_global_asset_source_links, subscriber_global_asset_reference_observations, subscriber_global_asset_coverage, subscriber_global_catalog_seed_runs FROM PUBLIC;
-- The catalog-sync worker receives the minimum source read needed for this controlled seed.
GRANT SELECT ON marketops_asset_universe TO signalops_subscriber_catalog_sync;
-- Shared platform data is never exposed directly to browser/API credentials.
GRANT SELECT, INSERT, UPDATE ON subscriber_global_assets, subscriber_global_asset_source_links, subscriber_global_asset_reference_observations, subscriber_global_asset_coverage, subscriber_global_catalog_seed_runs TO signalops_subscriber_catalog_sync;
GRANT SELECT ON subscriber_global_assets, subscriber_global_asset_source_links, subscriber_global_asset_reference_observations, subscriber_global_catalog_seed_runs TO signalops_subscriber_global_eod;
GRANT SELECT, UPDATE ON subscriber_global_asset_coverage TO signalops_subscriber_global_eod;
