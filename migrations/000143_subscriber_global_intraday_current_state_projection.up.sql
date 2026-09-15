-- Restricted, platform-owned intraday current-state projection.
--
-- This reader is intentionally limited to immutable global evidence imported
-- from the tenant-local all_active current-state source. It does not expose a
-- historical intraday series, invoke a provider, enable a scheduler, or alter
-- a MarketOps API route. Freshness is the captured payload as_of_time, never
-- the legacy row created_at retained in evidence_records.observed_at.

CREATE VIEW subscriber_gateway_global_intraday_current_states
  WITH (security_barrier = true) AS
SELECT DISTINCT ON (record.global_asset_id)
  record.global_asset_id,
  asset.canonical_symbol AS symbol,
  record.source_event_id AS snapshot_id,
  (record.payload ->> $$as_of_time$$)::timestamptz AS as_of_time,
  record.session_date,
  COALESCE(record.payload ->> $$universe_group$$, $$all_active$$) AS universe_group,
  COALESCE(record.payload ->> $$market_status$$, $$$$) AS market_status,
  COALESCE((record.payload ->> $$stale$$)::boolean, true) AS stale,
  COALESCE(record.payload -> $$conditions$$, $$[]$$::jsonb) AS conditions,
  COALESCE(record.payload -> $$source_payload$$, $$ {} $$::jsonb) AS source_payload,
  true AS current_only_source,
  record.algorithm_id,
  record.algorithm_version,
  record.quality_state,
  record.evidence_fingerprint,
  record.validation_contract_ref,
  record.immutable_baseline_ref,
  record.provenance
FROM subscriber_global_marketops_evidence_records record
JOIN subscriber_global_marketops_evidence_runs run
  ON run.evidence_run_id = record.evidence_run_id
JOIN subscriber_global_assets asset
  ON asset.global_asset_id = record.global_asset_id
WHERE record.evidence_kind = $$intraday_snapshot$$
  AND run.source_scope = $$legacy_materialization$$
  AND COALESCE((record.payload ->> $$current_only_source$$)::boolean, false)
  AND NULLIF(record.payload ->> $$as_of_time$$, $$$$) IS NOT NULL
ORDER BY record.global_asset_id,
  (record.payload ->> $$as_of_time$$)::timestamptz DESC,
  record.evidence_fingerprint DESC;

ALTER VIEW subscriber_gateway_global_intraday_current_states OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON subscriber_gateway_global_intraday_current_states FROM PUBLIC;
GRANT SELECT ON subscriber_gateway_global_intraday_current_states TO signalops_subscriber_gateway;
