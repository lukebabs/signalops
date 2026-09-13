-- Subscriber global analytical-data-plane: EEOM results are platform-owned
-- evidence. Watchlists authorize symbols at the Gateway.

CREATE VIEW subscriber_gateway_global_eeom_results WITH (security_barrier = true) AS
SELECT DISTINCT ON (record.global_asset_id, COALESCE(record.payload->>'earnings_event_id', ''))
  record.source_event_id AS result_id,
  record.global_asset_id,
  asset.canonical_symbol AS symbol,
  COALESCE(record.payload->>'earnings_event_id', '') AS earnings_event_id,
  COALESCE((record.payload->>'earnings_date')::date, record.session_date) AS earnings_date,
  record.session_date,
  record.algorithm_version AS model_version,
  COALESCE((record.payload->>'score')::double precision, 0) AS score,
  COALESCE(record.payload->>'posture', '') AS posture,
  COALESCE(record.payload->>'classification', '') AS classification,
  record.quality_state AS evidence_quality,
  COALESCE((record.payload->>'eligible')::boolean, false) AS eligible,
  COALESCE(record.payload->'result_json', '{}'::jsonb) AS result_json,
  record.observed_at AS created_at,
  record.evidence_fingerprint,
  record.validation_contract_ref,
  record.immutable_baseline_ref,
  record.provenance
FROM subscriber_global_marketops_evidence_records record
JOIN subscriber_global_marketops_evidence_runs run ON run.evidence_run_id = record.evidence_run_id
JOIN subscriber_global_assets asset ON asset.global_asset_id = record.global_asset_id
WHERE record.evidence_kind = 'eeom'
  AND record.algorithm_id = 'earnings_event_opportunity'
  AND run.source_scope IN ('global_provider_capture', 'legacy_materialization')
ORDER BY record.global_asset_id, COALESCE(record.payload->>'earnings_event_id', ''),
  record.session_date DESC, record.observed_at DESC, record.global_evidence_id DESC;

ALTER VIEW subscriber_gateway_global_eeom_results OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON subscriber_gateway_global_eeom_results FROM PUBLIC;
GRANT SELECT ON subscriber_gateway_global_eeom_results TO signalops_subscriber_gateway;
