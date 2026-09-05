-- Governed FMP profile normalization for the unmapped assets represented in
-- the immutable tenant-local legacy SAF cohort. Source outcomes and existing
-- benchmark observations remain untouched.

CREATE TABLE subscriber_global_asset_sector_classifications (
  classification_id text PRIMARY KEY,
  global_asset_id text NOT NULL REFERENCES subscriber_global_assets(global_asset_id) ON DELETE RESTRICT,
  provider text NOT NULL CHECK (provider = 'fmp'),
  provider_endpoint text NOT NULL,
  provider_symbol text NOT NULL,
  provider_sector text NOT NULL,
  provider_industry text NOT NULL,
  provider_exchange text NOT NULL,
  canonical_sector text NOT NULL CHECK (canonical_sector IN ('Materials','Communication Services','Industrials','Technology','Consumer Staples','Consumer Discretionary','Utilities','Healthcare')),
  source_fingerprint text NOT NULL UNIQUE,
  captured_at timestamptz NOT NULL,
  correlation_id text NOT NULL,
  provenance jsonb NOT NULL CHECK (jsonb_typeof(provenance) = 'object'),
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_subscriber_global_asset_sector_classifications_asset_time
  ON subscriber_global_asset_sector_classifications (global_asset_id, captured_at DESC);

CREATE FUNCTION subscriber_global_asset_sector_classification_immutable_guard()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  RAISE EXCEPTION 'subscriber global asset sector classifications are append-only; create a new provider-backed classification instead';
END;
$$;
CREATE TRIGGER trg_subscriber_global_asset_sector_classification_immutable
BEFORE UPDATE OR DELETE ON subscriber_global_asset_sector_classifications
FOR EACH ROW EXECUTE FUNCTION subscriber_global_asset_sector_classification_immutable_guard();

WITH seed(canonical_symbol, provider_sector, provider_industry, provider_exchange, canonical_sector) AS (
  VALUES
    ('FDX',  'Industrials',             'Integrated Freight & Logistics',       'NYSE',   'Industrials'),
    ('GILD', 'Healthcare',              'Drug Manufacturers - General',         'NASDAQ', 'Healthcare'),
    ('LIN',  'Basic Materials',         'Chemicals - Specialty',                 'NASDAQ', 'Materials'),
    ('MCD',  'Consumer Cyclical',       'Restaurants',                           'NYSE',   'Consumer Discretionary'),
    ('META', 'Communication Services',  'Internet Content & Information',        'NASDAQ', 'Communication Services'),
    ('MO',   'Consumer Defensive',      'Tobacco',                               'NYSE',   'Consumer Staples'),
    ('NEE',  'Utilities',               'Regulated Electric',                    'NYSE',   'Utilities'),
    ('NKE',  'Consumer Cyclical',       'Apparel - Footwear & Accessories',      'NYSE',   'Consumer Discretionary'),
    ('PEP',  'Consumer Defensive',      'Beverages - Non-Alcoholic',             'NASDAQ', 'Consumer Staples'),
    ('QCOM', 'Technology',              'Semiconductors',                        'NASDAQ', 'Technology'),
    ('TMUS', 'Communication Services',  'Telecommunications Services',            'NASDAQ', 'Communication Services')
), inserted AS (
  INSERT INTO subscriber_global_asset_sector_classifications
    (classification_id,global_asset_id,provider,provider_endpoint,provider_symbol,provider_sector,provider_industry,provider_exchange,canonical_sector,source_fingerprint,captured_at,correlation_id,provenance)
  SELECT
    'subsector_' || substr(md5(asset.global_asset_id || '|fmp_profile.v1|' || seed.provider_sector || '|' || seed.provider_industry), 1, 24),
    asset.global_asset_id,'fmp','/stable/profile',seed.canonical_symbol,seed.provider_sector,seed.provider_industry,seed.provider_exchange,seed.canonical_sector,
    md5('fmp|/stable/profile|' || seed.canonical_symbol || '|' || seed.provider_sector || '|' || seed.provider_industry || '|' || seed.provider_exchange),
    now(),'saf-v2b-legacy-sector-normalization-20260817',
    jsonb_build_object('schema_version','subscriber_global_sector_normalization.v1','provider','fmp','endpoint','/stable/profile','requested_symbol',seed.canonical_symbol,'response_symbol',seed.canonical_symbol,'profile_sector',seed.provider_sector,'profile_industry',seed.provider_industry,'profile_exchange',seed.provider_exchange,'canonical_sector',seed.canonical_sector,'source_scope','legacy SAF benchmark sector-unmapped remediation')
  FROM seed
  JOIN subscriber_global_assets asset ON asset.canonical_symbol = seed.canonical_symbol
  ON CONFLICT (source_fingerprint) DO NOTHING
  RETURNING global_asset_id,classification_id,provider_symbol,provider_sector,provider_industry,provider_exchange,canonical_sector,source_fingerprint
), latest AS (
  SELECT DISTINCT ON (global_asset_id) global_asset_id,classification_id,provider_symbol,provider_sector,provider_industry,provider_exchange,canonical_sector,source_fingerprint
  FROM inserted
  ORDER BY global_asset_id, classification_id DESC
)
UPDATE subscriber_global_assets asset
SET canonical_symbol = latest.provider_symbol,
    asset_type = CASE WHEN asset.asset_type = '' THEN 'equity' ELSE asset.asset_type END,
    exchange = latest.provider_exchange,
    sector = latest.canonical_sector,
    industry = latest.provider_industry,
    reference_effective_at = now(),
    reference_provenance = jsonb_build_object('authority','fmp_profile','classification_id',latest.classification_id,'source_fingerprint',latest.source_fingerprint,'provider','fmp','endpoint','/stable/profile','provider_sector',latest.provider_sector,'canonical_sector',latest.canonical_sector,'correlation_id','saf-v2b-legacy-sector-normalization-20260817'),
    updated_at = now()
FROM latest
WHERE asset.global_asset_id = latest.global_asset_id;

-- Prefer complete v2 benchmark evidence as it is appended, but retain v1 on a
-- per-observation basis until every legacy observation has been recalculated.
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
  COALESCE(broad_v2.benchmark_symbol, broad_v1.benchmark_symbol) AS broad_market_benchmark_symbol,
  COALESCE(broad_v2.benchmark_relative_return, broad_v1.benchmark_relative_return) AS broad_market_relative_return,
  COALESCE(broad_v2.benchmark_resolution_state, broad_v1.benchmark_resolution_state) AS broad_market_benchmark_state,
  COALESCE(sector_v2.benchmark_symbol, sector_v1.benchmark_symbol) AS sector_benchmark_symbol,
  COALESCE(sector_v2.benchmark_segment_key, sector_v1.benchmark_segment_key) AS sector_benchmark_segment_key,
  COALESCE(sector_v2.benchmark_relative_return, sector_v1.benchmark_relative_return) AS sector_relative_return,
  COALESCE(sector_v2.benchmark_resolution_state, sector_v1.benchmark_resolution_state) AS sector_benchmark_state
FROM subscriber_global_marketops_evidence_records record
JOIN subscriber_global_marketops_evidence_runs run ON run.evidence_run_id = record.evidence_run_id
JOIN subscriber_global_assets asset ON asset.global_asset_id = record.global_asset_id
LEFT JOIN subscriber_global_saf_benchmark_observations broad_v2 ON broad_v2.source_observation_id = record.source_event_id AND broad_v2.benchmark_kind = 'broad_market' AND broad_v2.calculation_version = 'saf_benchmark.v2'
LEFT JOIN subscriber_global_saf_benchmark_observations broad_v1 ON broad_v1.source_observation_id = record.source_event_id AND broad_v1.benchmark_kind = 'broad_market' AND broad_v1.calculation_version = 'saf_benchmark.v1'
LEFT JOIN subscriber_global_saf_benchmark_observations sector_v2 ON sector_v2.source_observation_id = record.source_event_id AND sector_v2.benchmark_kind = 'sector' AND sector_v2.calculation_version = 'saf_benchmark.v2'
LEFT JOIN subscriber_global_saf_benchmark_observations sector_v1 ON sector_v1.source_observation_id = record.source_event_id AND sector_v1.benchmark_kind = 'sector' AND sector_v1.calculation_version = 'saf_benchmark.v1'
WHERE record.evidence_kind = 'outcome'
  AND record.algorithm_id = 'opportunity'
  AND run.source_scope = 'legacy_materialization'
  AND COALESCE(record.payload->>'source_type', record.payload->'outcome_payload'->>'source_type', record.algorithm_id) = 'opportunity'
  AND COALESCE(record.payload->>'direction', '') IN ('upside', 'downside')
  AND jsonb_typeof(record.payload->'directional_hit') = 'boolean'
ORDER BY record.source_event_id, record.observed_at DESC, record.global_evidence_id DESC;

ALTER TABLE subscriber_global_asset_sector_classifications OWNER TO signalops_subscriber_migrator;
ALTER VIEW subscriber_gateway_global_signal_assurance_observations OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON subscriber_global_asset_sector_classifications FROM PUBLIC;
GRANT SELECT, INSERT ON subscriber_global_asset_sector_classifications TO signalops_subscriber_catalog_sync;
GRANT SELECT ON subscriber_global_asset_sector_classifications TO signalops_subscriber_global_eod;
REVOKE ALL ON FUNCTION subscriber_global_asset_sector_classification_immutable_guard() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION subscriber_global_asset_sector_classification_immutable_guard() TO signalops_subscriber_catalog_sync;
GRANT SELECT ON subscriber_gateway_global_signal_assurance_observations TO signalops_subscriber_gateway;
