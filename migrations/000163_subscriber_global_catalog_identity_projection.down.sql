CREATE OR REPLACE VIEW subscriber_gateway_global_valuation_results WITH (security_barrier = true) AS
WITH evidence_results AS (
SELECT DISTINCT ON (record.global_asset_id, record.session_date, record.algorithm_id)
  record.source_event_id AS result_id,
  COALESCE(record.payload->>'snapshot_id', '') AS snapshot_id,
  record.global_asset_id,
  asset.canonical_symbol AS symbol,
  record.session_date,
  record.algorithm_id,
  record.algorithm_version AS model_version,
  COALESCE((record.payload->>'score')::double precision, 0) AS score,
  COALESCE((record.payload->>'fair_value')::double precision, 0) AS fair_value,
  COALESCE(record.payload->>'classification', '') AS classification,
  COALESCE((record.payload->>'confidence')::integer, 0) AS confidence,
  COALESCE(record.payload->>'confidence_label', '') AS confidence_label,
  COALESCE(record.payload->>'evaluation_status', '') AS evaluation_status,
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
WHERE record.evidence_kind = 'valuation'
  AND record.algorithm_id IN (
    'signalops.algorithms.valuation_composite_v3',
    'signalops.algorithms.distressed_opportunity_scoring_v3',
    'signalops.algorithms.valuation_composite_v4_annual',
    'signalops.algorithms.distressed_opportunity_scoring_v4_annual'
  )
  AND run.source_scope IN ('global_provider_capture', 'legacy_materialization')
ORDER BY record.global_asset_id, record.session_date, record.algorithm_id,
  record.observed_at DESC, record.global_evidence_id DESC
), tactical_results AS (
SELECT DISTINCT ON (asset.global_asset_id, result.session_date, result.algorithm_id)
  result.result_id,
  result.snapshot_id,
  asset.global_asset_id,
  asset.canonical_symbol AS symbol,
  result.session_date,
  result.algorithm_id,
  result.model_version,
  result.score,
  result.fair_value,
  result.classification,
  result.confidence,
  result.confidence_label,
  result.evaluation_status,
  result.eligible,
  result.result_json,
  result.created_at,
  'legacy-tactical:' || result.result_id AS evidence_fingerprint,
  'marketops-tactical-posture-v1/legacy-projection' AS validation_contract_ref,
  'marketops-dedicated-primary' AS immutable_baseline_ref,
  jsonb_build_object('source_table', 'marketops_valuation_results', 'tenant_id', result.tenant_id, 'projection', 'subscriber_gateway_global_valuation_results') AS provenance
FROM marketops_valuation_results result
JOIN (
  SELECT DISTINCT ON (upper(canonical_symbol)) global_asset_id, canonical_symbol
  FROM subscriber_global_assets
  WHERE canonical_symbol <> ''
  ORDER BY upper(canonical_symbol), global_asset_id
) asset ON upper(asset.canonical_symbol) = upper(result.symbol)
WHERE result.tenant_id = 'tenant-local'
  AND result.algorithm_id = 'signalops.algorithms.tactical_market_posture_v1'
ORDER BY asset.global_asset_id, result.session_date, result.algorithm_id, result.created_at DESC, result.result_id DESC
)
SELECT * FROM evidence_results
UNION ALL
SELECT * FROM tactical_results;

ALTER VIEW subscriber_gateway_global_valuation_results OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON subscriber_gateway_global_valuation_results FROM PUBLIC;
GRANT SELECT ON subscriber_gateway_global_valuation_results TO signalops_subscriber_gateway;

DROP VIEW IF EXISTS subscriber_gateway_global_canonical_assets;
