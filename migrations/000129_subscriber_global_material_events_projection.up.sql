-- Subscriber global analytical-data-plane: point-in-time-known material
-- events are platform-owned evidence. Watchlists authorize symbols at Gateway.

CREATE VIEW subscriber_gateway_global_material_events WITH (security_barrier = true) AS
SELECT DISTINCT ON (record.global_asset_id, record.source_event_id)
  record.global_asset_id,
  asset.canonical_symbol AS symbol,
  record.source_event_id AS event_id,
  record.algorithm_id,
  record.algorithm_version,
  record.session_date,
  record.observed_at,
  record.quality_state,
  record.payload,
  record.evidence_fingerprint,
  record.validation_contract_ref,
  record.immutable_baseline_ref,
  record.provenance
FROM subscriber_global_marketops_evidence_records record
JOIN subscriber_global_marketops_evidence_runs run ON run.evidence_run_id = record.evidence_run_id
JOIN subscriber_global_assets asset ON asset.global_asset_id = record.global_asset_id
WHERE record.evidence_kind = 'material_event'
  AND record.algorithm_id = 'market_data.fmp.earnings_calendar'
  AND run.source_scope = 'global_provider_capture'
ORDER BY record.global_asset_id, record.source_event_id,
  record.observed_at DESC, record.global_evidence_id DESC;

ALTER VIEW subscriber_gateway_global_material_events OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON subscriber_gateway_global_material_events FROM PUBLIC;
GRANT SELECT ON subscriber_gateway_global_material_events TO signalops_subscriber_gateway;
