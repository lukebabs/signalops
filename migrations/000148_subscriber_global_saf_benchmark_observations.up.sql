-- SAF-V2: append-only matched benchmark observations for historical outcome
-- evidence.  This never alters a signal outcome or its immutable legacy
-- materialization; each calculation stores its own source price provenance.

CREATE TABLE subscriber_global_saf_benchmark_observations (
  benchmark_observation_id text PRIMARY KEY,
  source_observation_id text NOT NULL,
  global_asset_id text NOT NULL REFERENCES subscriber_global_assets(global_asset_id) ON DELETE RESTRICT,
  benchmark_kind text NOT NULL CHECK (benchmark_kind IN ('broad_market','sector')),
  benchmark_symbol text NOT NULL,
  benchmark_segment_key text NOT NULL DEFAULT '',
  benchmark_resolution_state text NOT NULL CHECK (benchmark_resolution_state IN ('matched','sector_unmapped','origin_price_unavailable','maturity_price_unavailable')),
  origin_session_date date NOT NULL,
  matured_session_date date NOT NULL,
  origin_price double precision,
  matured_price double precision,
  benchmark_return double precision,
  benchmark_relative_return double precision,
  source_origin_event_id text NOT NULL DEFAULT '',
  source_matured_event_id text NOT NULL DEFAULT '',
  source_origin_fingerprint text NOT NULL DEFAULT '',
  source_matured_fingerprint text NOT NULL DEFAULT '',
  selection_policy_version text NOT NULL,
  calculation_version text NOT NULL,
  calculation_run_id text NOT NULL,
  provenance jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(provenance) = 'object'),
  observed_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (source_observation_id, benchmark_kind, calculation_version),
  CHECK ((benchmark_resolution_state = 'matched' AND origin_price > 0 AND matured_price > 0 AND benchmark_return IS NOT NULL AND benchmark_relative_return IS NOT NULL) OR benchmark_resolution_state <> 'matched')
);

CREATE INDEX idx_subscriber_global_saf_benchmark_source
  ON subscriber_global_saf_benchmark_observations (source_observation_id, benchmark_kind, calculation_version);

CREATE FUNCTION subscriber_global_saf_benchmark_observation_immutable_guard()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  RAISE EXCEPTION 'subscriber global SAF benchmark observation is append-only; record a new calculation version instead';
END;
$$;
CREATE TRIGGER trg_subscriber_global_saf_benchmark_observation_immutable
BEFORE UPDATE OR DELETE ON subscriber_global_saf_benchmark_observations
FOR EACH ROW EXECUTE FUNCTION subscriber_global_saf_benchmark_observation_immutable_guard();

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
  broad.benchmark_symbol AS broad_market_benchmark_symbol,
  broad.benchmark_relative_return AS broad_market_relative_return,
  broad.benchmark_resolution_state AS broad_market_benchmark_state,
  sector.benchmark_symbol AS sector_benchmark_symbol,
  sector.benchmark_segment_key AS sector_benchmark_segment_key,
  sector.benchmark_relative_return AS sector_relative_return,
  sector.benchmark_resolution_state AS sector_benchmark_state
FROM subscriber_global_marketops_evidence_records record
JOIN subscriber_global_marketops_evidence_runs run ON run.evidence_run_id = record.evidence_run_id
JOIN subscriber_global_assets asset ON asset.global_asset_id = record.global_asset_id
LEFT JOIN subscriber_global_saf_benchmark_observations broad
  ON broad.source_observation_id = record.source_event_id
 AND broad.benchmark_kind = 'broad_market' AND broad.calculation_version = 'saf_benchmark.v1'
LEFT JOIN subscriber_global_saf_benchmark_observations sector
  ON sector.source_observation_id = record.source_event_id
 AND sector.benchmark_kind = 'sector' AND sector.calculation_version = 'saf_benchmark.v1'
WHERE record.evidence_kind = 'outcome'
  AND record.algorithm_id = 'opportunity'
  AND run.source_scope = 'legacy_materialization'
  AND COALESCE(record.payload->>'source_type', record.payload->'outcome_payload'->>'source_type', record.algorithm_id) = 'opportunity'
  AND COALESCE(record.payload->>'direction', '') IN ('upside', 'downside')
  AND jsonb_typeof(record.payload->'directional_hit') = 'boolean'
ORDER BY record.source_event_id, record.observed_at DESC, record.global_evidence_id DESC;

ALTER TABLE subscriber_global_saf_benchmark_observations OWNER TO signalops_subscriber_migrator;
ALTER VIEW subscriber_gateway_global_signal_assurance_observations OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON subscriber_global_saf_benchmark_observations FROM PUBLIC;
GRANT SELECT, INSERT ON subscriber_global_saf_benchmark_observations TO signalops_subscriber_global_eod;
GRANT SELECT ON subscriber_global_assets, subscriber_gateway_global_signal_assurance_observations TO signalops_subscriber_global_eod;
GRANT SELECT ON subscriber_gateway_global_signal_assurance_observations TO signalops_subscriber_gateway;
REVOKE ALL ON FUNCTION subscriber_global_saf_benchmark_observation_immutable_guard() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION subscriber_global_saf_benchmark_observation_immutable_guard() TO signalops_subscriber_global_eod;
