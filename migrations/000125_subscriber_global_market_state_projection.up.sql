-- Subscriber global analytical-data-plane: Market State is the first
-- parity-approved analytical reader. Tenant watchlist membership authorizes
-- symbols in the Gateway; this view never consults a tenant-local state row.

CREATE VIEW subscriber_gateway_global_market_states WITH (security_barrier = true) AS
SELECT DISTINCT ON (record.global_asset_id, record.session_date)
  record.source_event_id AS market_state_id,
  record.global_asset_id,
  asset.canonical_symbol AS symbol,
  record.session_date,
  record.observed_at AS as_of_time,
  record.algorithm_version AS state_schema_version,
  record.quality_state,
  record.payload,
  record.evidence_fingerprint,
  record.validation_contract_ref,
  record.immutable_baseline_ref,
  record.provenance
FROM subscriber_global_marketops_evidence_records record
JOIN subscriber_global_marketops_evidence_runs run ON run.evidence_run_id = record.evidence_run_id
JOIN subscriber_global_assets asset ON asset.global_asset_id = record.global_asset_id
WHERE record.evidence_kind = 'market_state'
  AND run.source_scope IN ('global_provider_capture', 'legacy_materialization')
ORDER BY record.global_asset_id, record.session_date, record.observed_at DESC, record.global_evidence_id DESC;

ALTER VIEW subscriber_gateway_global_market_states OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON subscriber_gateway_global_market_states FROM PUBLIC;
GRANT SELECT ON subscriber_gateway_global_market_states TO signalops_subscriber_gateway;
