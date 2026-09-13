-- Expose annual VC/DOSM v4 as the preferred platform-global valuation evidence.
CREATE OR REPLACE VIEW subscriber_gateway_global_valuation_results WITH (security_barrier = true) AS
SELECT DISTINCT ON (record.global_asset_id, record.session_date, record.algorithm_id)
  record.source_event_id AS result_id,
  COALESCE(record.payload->>'snapshot_id', '') AS snapshot_id,
  record.global_asset_id, asset.canonical_symbol AS symbol, record.session_date,
  record.algorithm_id, record.algorithm_version AS model_version,
  COALESCE((record.payload->>'score')::double precision, 0) AS score,
  COALESCE((record.payload->>'fair_value')::double precision, 0) AS fair_value,
  COALESCE(record.payload->>'classification', '') AS classification,
  COALESCE((record.payload->>'confidence')::integer, 0) AS confidence,
  COALESCE(record.payload->>'confidence_label', '') AS confidence_label,
  COALESCE(record.payload->>'evaluation_status', '') AS evaluation_status,
  COALESCE((record.payload->>'eligible')::boolean, false) AS eligible,
  COALESCE(record.payload->'result_json', '{}'::jsonb) AS result_json,
  record.observed_at AS created_at, record.evidence_fingerprint,
  record.validation_contract_ref, record.immutable_baseline_ref, record.provenance
FROM subscriber_global_marketops_evidence_records record
JOIN subscriber_global_marketops_evidence_runs run ON run.evidence_run_id = record.evidence_run_id
JOIN subscriber_global_assets asset ON asset.global_asset_id = record.global_asset_id
WHERE record.evidence_kind = 'valuation'
  AND record.algorithm_id IN (
    'signalops.algorithms.valuation_composite_v3',
    'signalops.algorithms.distressed_opportunity_scoring_v3',
    'signalops.algorithms.valuation_composite_v4_annual',
    'signalops.algorithms.distressed_opportunity_scoring_v4_annual'
  )
  AND run.source_scope IN ('global_provider_capture', 'legacy_materialization')
ORDER BY record.global_asset_id, record.session_date, record.algorithm_id,
  record.observed_at DESC, record.global_evidence_id DESC;
ALTER VIEW subscriber_gateway_global_valuation_results OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON subscriber_gateway_global_valuation_results FROM PUBLIC;
GRANT SELECT ON subscriber_gateway_global_valuation_results TO signalops_subscriber_gateway;
