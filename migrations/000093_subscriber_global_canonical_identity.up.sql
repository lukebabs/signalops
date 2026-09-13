-- S2 correction: source provenance is not a global security identity.
-- Retain every source record, but resolve all aliases to one deterministic head.
CREATE TABLE subscriber_global_asset_identity_resolutions (
  source_global_asset_id text PRIMARY KEY REFERENCES subscriber_global_assets(global_asset_id) ON DELETE RESTRICT,
  canonical_global_asset_id text NOT NULL REFERENCES subscriber_global_assets(global_asset_id) ON DELETE RESTRICT,
  resolution_version text NOT NULL,
  resolution_reason text NOT NULL,
  resolved_at timestamptz NOT NULL DEFAULT now(),
  CHECK (length(trim(resolution_version)) > 0),
  CHECK (length(trim(resolution_reason)) > 0)
);
CREATE INDEX idx_subscriber_global_asset_identity_resolutions_canonical
  ON subscriber_global_asset_identity_resolutions (canonical_global_asset_id, source_global_asset_id);

-- src-massive is preferred when it is present because Massive is the governed
-- reference provider. Stable lexical ordering is the deterministic fallback.
INSERT INTO subscriber_global_asset_identity_resolutions
  (source_global_asset_id, canonical_global_asset_id, resolution_version, resolution_reason)
SELECT global_asset_id,
  first_value(global_asset_id) OVER (
    PARTITION BY canonical_symbol
    ORDER BY CASE WHEN source_id='src-massive' THEN 0 ELSE 1 END, source_id, global_asset_id
  ),
  's2-canonical-security-v1',
  'canonical_symbol_provider_source_resolution'
FROM subscriber_global_assets;

ALTER TABLE subscriber_global_asset_identity_resolutions OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON subscriber_global_asset_identity_resolutions FROM PUBLIC;
GRANT SELECT ON subscriber_global_asset_identity_resolutions TO signalops_subscriber_catalog_sync;
GRANT SELECT ON subscriber_global_asset_identity_resolutions TO signalops_subscriber_global_eod;
