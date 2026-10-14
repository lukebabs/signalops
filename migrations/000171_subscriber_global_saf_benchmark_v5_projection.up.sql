-- Prefer SAF benchmark v5 for operational post-cutoff usefulness while
-- preserving v1-v4 as immutable audit history. The projection uses one
-- deterministic benchmark row per observation/kind.

CREATE OR REPLACE VIEW subscriber_gateway_global_signal_assurance_observations WITH (security_barrier = true) AS
SELECT DISTINCT ON (record.source_event_id)
  record.source_event_id AS observation_id,
  canonical.global_asset_id,
  canonical.canonical_symbol AS symbol,
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
  record.provenance || jsonb_build_object(
    'source_global_asset_id', record.global_asset_id,
    'canonical_global_asset_id', canonical.global_asset_id,
    'canonical_resolution_state', canonical.canonical_resolution_state
  ) AS provenance,
  record.observed_at,
  broad.benchmark_symbol AS broad_market_benchmark_symbol,
  broad.benchmark_relative_return AS broad_market_relative_return,
  broad.benchmark_resolution_state AS broad_market_benchmark_state,
  sector.benchmark_symbol AS sector_benchmark_symbol,
  sector.benchmark_segment_key AS sector_benchmark_segment_key,
  sector.benchmark_relative_return AS sector_relative_return,
  sector.benchmark_resolution_state AS sector_benchmark_state
FROM subscriber_global_marketops_evidence_records record
JOIN subscriber_global_marketops_evidence_runs run
  ON run.evidence_run_id = record.evidence_run_id
JOIN subscriber_global_asset_identity_resolutions resolution
  ON resolution.source_global_asset_id = record.global_asset_id
JOIN subscriber_gateway_global_canonical_assets canonical
  ON canonical.global_asset_id = resolution.canonical_global_asset_id
LEFT JOIN LATERAL (
  SELECT benchmark_symbol, benchmark_relative_return, benchmark_resolution_state
  FROM subscriber_global_saf_benchmark_observations benchmark
  WHERE benchmark.source_observation_id = record.source_event_id
    AND benchmark.benchmark_kind = 'broad_market'
    AND benchmark.calculation_version IN ('saf_benchmark.v5','saf_benchmark.v4','saf_benchmark.v3','saf_benchmark.v2','saf_benchmark.v1')
  ORDER BY CASE benchmark.calculation_version
      WHEN 'saf_benchmark.v5' THEN 0
      WHEN 'saf_benchmark.v4' THEN 1
      WHEN 'saf_benchmark.v3' THEN 2
      WHEN 'saf_benchmark.v2' THEN 3
      ELSE 4
    END,
    benchmark.created_at DESC,
    benchmark.benchmark_observation_id DESC
  LIMIT 1
) broad ON true
LEFT JOIN LATERAL (
  SELECT benchmark_symbol, benchmark_segment_key, benchmark_relative_return, benchmark_resolution_state
  FROM subscriber_global_saf_benchmark_observations benchmark
  WHERE benchmark.source_observation_id = record.source_event_id
    AND benchmark.benchmark_kind = 'sector'
    AND benchmark.calculation_version IN ('saf_benchmark.v5','saf_benchmark.v4','saf_benchmark.v3','saf_benchmark.v2','saf_benchmark.v1')
  ORDER BY CASE benchmark.calculation_version
      WHEN 'saf_benchmark.v5' THEN 0
      WHEN 'saf_benchmark.v4' THEN 1
      WHEN 'saf_benchmark.v3' THEN 2
      WHEN 'saf_benchmark.v2' THEN 3
      ELSE 4
    END,
    CASE benchmark.benchmark_resolution_state WHEN 'matched' THEN 0 ELSE 1 END,
    benchmark.created_at DESC,
    benchmark.benchmark_observation_id DESC
  LIMIT 1
) sector ON true
WHERE record.evidence_kind = 'outcome'
  AND record.algorithm_id = 'opportunity'
  AND run.source_scope = 'legacy_materialization'
  AND COALESCE(record.payload->>'source_type', record.payload->'outcome_payload'->>'source_type', record.algorithm_id) = 'opportunity'
  AND COALESCE(record.payload->>'direction', '') IN ('upside', 'downside')
  AND jsonb_typeof(record.payload->'directional_hit') = 'boolean'
ORDER BY record.source_event_id, record.observed_at DESC, record.global_evidence_id DESC;

ALTER VIEW subscriber_gateway_global_signal_assurance_observations OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON subscriber_gateway_global_signal_assurance_observations FROM PUBLIC;
GRANT SELECT ON subscriber_gateway_global_signal_assurance_observations TO signalops_subscriber_gateway;
GRANT SELECT ON subscriber_gateway_global_signal_assurance_observations TO signalops_subscriber_global_eod;

INSERT INTO schema_migrations (version, applied_at)
VALUES ('000171_subscriber_global_saf_benchmark_v5_projection', now())
ON CONFLICT (version) DO NOTHING;
