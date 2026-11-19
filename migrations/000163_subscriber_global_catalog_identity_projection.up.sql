-- Subscriber catalog-governance projection.
-- This does not delete catalog rows and does not rewrite immutable evidence.
-- Source global_asset_id values are resolved through the existing identity
-- resolution table so user-facing projections present one canonical symbol.

CREATE OR REPLACE VIEW subscriber_gateway_global_canonical_assets WITH (security_barrier = true) AS
WITH resolved AS (
  SELECT
    source.global_asset_id AS source_global_asset_id,
    COALESCE(resolution.canonical_global_asset_id, source.global_asset_id) AS canonical_global_asset_id,
    upper(source.canonical_symbol) AS canonical_symbol_key
  FROM subscriber_global_assets source
  LEFT JOIN subscriber_global_asset_identity_resolutions resolution
    ON resolution.source_global_asset_id = source.global_asset_id
  WHERE source.canonical_symbol <> ''
), grouped AS (
  SELECT
    canonical_global_asset_id,
    canonical_symbol_key,
    count(*)::integer AS source_asset_count,
    array_agg(source_global_asset_id ORDER BY source_global_asset_id) AS source_global_asset_ids
  FROM resolved
  GROUP BY canonical_global_asset_id, canonical_symbol_key
)
SELECT
  canonical.global_asset_id,
  canonical.canonical_symbol,
  canonical.company_name,
  canonical.asset_type,
  canonical.exchange,
  canonical.sector,
  canonical.industry,
  canonical.eligibility_status,
  grouped.source_asset_count,
  grouped.source_global_asset_ids,
  CASE WHEN grouped.source_asset_count > 1 THEN 'deduplicated' ELSE 'canonical' END AS canonical_resolution_state
FROM grouped
JOIN subscriber_global_assets canonical
  ON canonical.global_asset_id = grouped.canonical_global_asset_id;

ALTER VIEW subscriber_gateway_global_canonical_assets OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON subscriber_gateway_global_canonical_assets FROM PUBLIC;
GRANT SELECT ON subscriber_gateway_global_canonical_assets TO signalops_subscriber_gateway;

CREATE OR REPLACE VIEW subscriber_gateway_global_valuation_results WITH (security_barrier = true) AS
WITH evidence_results AS (
SELECT DISTINCT ON (canonical.global_asset_id, record.session_date, record.algorithm_id)
  record.source_event_id AS result_id,
  COALESCE(record.payload->>'snapshot_id', '') AS snapshot_id,
  canonical.global_asset_id,
  canonical.canonical_symbol AS symbol,
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
  record.provenance || jsonb_build_object(
    'source_global_asset_id', record.global_asset_id,
    'canonical_global_asset_id', canonical.global_asset_id,
    'canonical_resolution_state', canonical.canonical_resolution_state
  ) AS provenance
FROM subscriber_global_marketops_evidence_records record
JOIN subscriber_global_marketops_evidence_runs run ON run.evidence_run_id = record.evidence_run_id
JOIN subscriber_global_asset_identity_resolutions resolution
  ON resolution.source_global_asset_id = record.global_asset_id
JOIN subscriber_gateway_global_canonical_assets canonical
  ON canonical.global_asset_id = resolution.canonical_global_asset_id
WHERE record.evidence_kind = 'valuation'
  AND record.algorithm_id IN (
    'signalops.algorithms.valuation_composite_v3',
    'signalops.algorithms.distressed_opportunity_scoring_v3',
    'signalops.algorithms.valuation_composite_v4_annual',
    'signalops.algorithms.distressed_opportunity_scoring_v4_annual'
  )
  AND run.source_scope IN ('global_provider_capture', 'legacy_materialization')
  AND (
    record.algorithm_id NOT IN (
      'signalops.algorithms.valuation_composite_v4_annual',
      'signalops.algorithms.distressed_opportunity_scoring_v4_annual'
    )
    OR (record.quality_state = 'usable' AND COALESCE((record.payload->>'eligible')::boolean, false))
  )
ORDER BY canonical.global_asset_id, record.session_date, record.algorithm_id,
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
  jsonb_build_object('source_table', 'marketops_valuation_results', 'tenant_id', result.tenant_id, 'projection', 'subscriber_gateway_global_valuation_results', 'canonical_resolution_state', asset.canonical_resolution_state) AS provenance
FROM marketops_valuation_results result
JOIN subscriber_gateway_global_canonical_assets asset
  ON upper(asset.canonical_symbol) = upper(result.symbol)
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
