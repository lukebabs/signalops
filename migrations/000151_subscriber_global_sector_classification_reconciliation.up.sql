-- Reconcile historical evidence asset identities with the FMP-authoritative
-- canonical-symbol classification. The prior migration classified the one
-- catalog identity selected by the unique profile fingerprint; legacy evidence
-- still references older identities for the same canonical symbols. Preserve
-- those immutable evidence rows and normalize the reference metadata on every
-- matching catalog identity instead.

WITH latest AS (
  SELECT DISTINCT ON (classification.provider_symbol)
    classification.classification_id,
    classification.provider_symbol,
    classification.provider_sector,
    classification.provider_industry,
    classification.provider_exchange,
    classification.canonical_sector,
    classification.source_fingerprint,
    classification.correlation_id
  FROM subscriber_global_asset_sector_classifications classification
  ORDER BY classification.provider_symbol, classification.captured_at DESC, classification.classification_id DESC
)
UPDATE subscriber_global_assets asset
SET asset_type = CASE WHEN asset.asset_type = '' THEN 'equity' ELSE asset.asset_type END,
    exchange = latest.provider_exchange,
    sector = latest.canonical_sector,
    industry = latest.provider_industry,
    reference_effective_at = now(),
    reference_provenance = jsonb_build_object(
      'authority', 'fmp_profile',
      'classification_id', latest.classification_id,
      'source_fingerprint', latest.source_fingerprint,
      'provider', 'fmp',
      'endpoint', '/stable/profile',
      'provider_sector', latest.provider_sector,
      'canonical_sector', latest.canonical_sector,
      'correlation_id', latest.correlation_id,
      'classification_scope', 'canonical_symbol_reconciliation.v1'
    ),
    updated_at = now()
FROM latest
WHERE asset.canonical_symbol = latest.provider_symbol
  AND (
    asset.sector IS DISTINCT FROM latest.canonical_sector
    OR asset.industry IS DISTINCT FROM latest.provider_industry
    OR asset.exchange IS DISTINCT FROM latest.provider_exchange
    OR COALESCE(asset.reference_provenance->>'classification_id', '') IS DISTINCT FROM latest.classification_id
  );

-- Version three supersedes the incomplete v2 calculation while keeping v1 and
-- v2 append-only evidence available for audit and comparison.
CREATE OR REPLACE VIEW subscriber_gateway_global_signal_assurance_observations WITH (security_barrier = true) AS
SELECT DISTINCT ON (record.source_event_id)
  record.source_event_id AS observation_id,
  record.global_asset_id,
  asset.canonical_symbol AS symbol,
  COALESCE(record.payload->>'source_id', '') AS source_id,
  COALESCE(record.payload->>'source_type', record.payload->'outcome_payload'->>'source_type', record.algorithm_id) AS source_type,
  COALESCE(record.payload->>'direction', '') AS direction,
  COALESCE((record.payload->>'horizon_sessions')::integer, 0) AS horizon_sessions,
  record.session_date AS origin_session_date,
  NULLIF(record.payload->>'matured_session_date', '')::date AS matured_session_date,
  NULLIF(record.payload->>'directional_hit', '')::boolean AS directional_hit,
  NULLIF(record.payload->>'forward_return', '')::double precision AS forward_return,
  NULLIF(record.payload->>'max_favorable_excursion', '')::double precision AS mfe,
  NULLIF(record.payload->>'max_adverse_excursion', '')::double precision AS mae,
  record.algorithm_version AS calculation_version,
  COALESCE(record.payload->>'calculation_run_id', '') AS calculation_run_id,
  record.quality_state,
  record.validation_contract_ref,
  record.immutable_baseline_ref,
  record.provenance,
  record.observed_at,
  COALESCE(broad_v3.benchmark_symbol, broad_v2.benchmark_symbol, broad_v1.benchmark_symbol) AS broad_market_benchmark_symbol,
  COALESCE(broad_v3.benchmark_relative_return, broad_v2.benchmark_relative_return, broad_v1.benchmark_relative_return) AS broad_market_relative_return,
  COALESCE(broad_v3.benchmark_resolution_state, broad_v2.benchmark_resolution_state, broad_v1.benchmark_resolution_state) AS broad_market_benchmark_state,
  COALESCE(sector_v3.benchmark_symbol, sector_v2.benchmark_symbol, sector_v1.benchmark_symbol) AS sector_benchmark_symbol,
  COALESCE(sector_v3.benchmark_segment_key, sector_v2.benchmark_segment_key, sector_v1.benchmark_segment_key) AS sector_benchmark_segment_key,
  COALESCE(sector_v3.benchmark_relative_return, sector_v2.benchmark_relative_return, sector_v1.benchmark_relative_return) AS sector_relative_return,
  COALESCE(sector_v3.benchmark_resolution_state, sector_v2.benchmark_resolution_state, sector_v1.benchmark_resolution_state) AS sector_benchmark_state
FROM subscriber_global_marketops_evidence_records record
JOIN subscriber_global_marketops_evidence_runs run ON run.evidence_run_id = record.evidence_run_id
JOIN subscriber_global_assets asset ON asset.global_asset_id = record.global_asset_id
LEFT JOIN subscriber_global_saf_benchmark_observations broad_v3 ON broad_v3.source_observation_id = record.source_event_id AND broad_v3.benchmark_kind = 'broad_market' AND broad_v3.calculation_version = 'saf_benchmark.v3'
LEFT JOIN subscriber_global_saf_benchmark_observations broad_v2 ON broad_v2.source_observation_id = record.source_event_id AND broad_v2.benchmark_kind = 'broad_market' AND broad_v2.calculation_version = 'saf_benchmark.v2'
LEFT JOIN subscriber_global_saf_benchmark_observations broad_v1 ON broad_v1.source_observation_id = record.source_event_id AND broad_v1.benchmark_kind = 'broad_market' AND broad_v1.calculation_version = 'saf_benchmark.v1'
LEFT JOIN subscriber_global_saf_benchmark_observations sector_v3 ON sector_v3.source_observation_id = record.source_event_id AND sector_v3.benchmark_kind = 'sector' AND sector_v3.calculation_version = 'saf_benchmark.v3'
LEFT JOIN subscriber_global_saf_benchmark_observations sector_v2 ON sector_v2.source_observation_id = record.source_event_id AND sector_v2.benchmark_kind = 'sector' AND sector_v2.calculation_version = 'saf_benchmark.v2'
LEFT JOIN subscriber_global_saf_benchmark_observations sector_v1 ON sector_v1.source_observation_id = record.source_event_id AND sector_v1.benchmark_kind = 'sector' AND sector_v1.calculation_version = 'saf_benchmark.v1'
WHERE record.evidence_kind = 'outcome'
  AND record.algorithm_id = 'opportunity'
  AND run.source_scope = 'legacy_materialization'
  AND COALESCE(record.payload->>'source_type', record.payload->'outcome_payload'->>'source_type', record.algorithm_id) = 'opportunity'
  AND COALESCE(record.payload->>'direction', '') IN ('upside', 'downside')
  AND jsonb_typeof(record.payload->'directional_hit') = 'boolean'
ORDER BY record.source_event_id, record.observed_at DESC, record.global_evidence_id DESC;

ALTER VIEW subscriber_gateway_global_signal_assurance_observations OWNER TO signalops_subscriber_migrator;
GRANT SELECT ON subscriber_gateway_global_signal_assurance_observations TO signalops_subscriber_gateway;
