-- Reconcile SAF benchmark sector coverage for post-cutoff tenant-local legacy
-- observations without rewriting immutable evidence or existing benchmark rows.
--
-- Part 1 resolves duplicate same-symbol global asset identities to a governed
-- canonical asset row that already carries usable sector metadata.
-- Part 2 records explicit internal classifications for seven legacy watchlist
-- rows that only carried broad SIC-style "SERVICES-..." metadata.

ALTER TABLE subscriber_global_asset_sector_classifications
  DROP CONSTRAINT IF EXISTS subscriber_global_asset_sector_classifications_provider_check;
ALTER TABLE subscriber_global_asset_sector_classifications
  ADD CONSTRAINT subscriber_global_asset_sector_classifications_provider_check
  CHECK (provider IN ('fmp', 'internal_catalog_governance'));

ALTER TABLE subscriber_global_asset_sector_classifications
  DROP CONSTRAINT IF EXISTS subscriber_global_asset_sector_classific_canonical_sector_check;
ALTER TABLE subscriber_global_asset_sector_classifications
  ADD CONSTRAINT subscriber_global_asset_sector_classific_canonical_sector_check
  CHECK (canonical_sector IN (
    'Materials',
    'Communication Services',
    'Industrials',
    'Technology',
    'Consumer Staples',
    'Consumer Discretionary',
    'Utilities',
    'Healthcare',
    'Financials',
    'Energy',
    'Real Estate'
  ));

WITH unmapped AS (
  SELECT DISTINCT observation.global_asset_id, observation.symbol
  FROM subscriber_gateway_global_signal_assurance_observations observation
  WHERE observation.matured_session_date >= DATE '2026-08-20'
    AND observation.sector_benchmark_state = 'sector_unmapped'
), ranked_candidates AS (
  SELECT DISTINCT ON (unmapped.global_asset_id)
    unmapped.global_asset_id AS source_global_asset_id,
    candidate.global_asset_id AS canonical_global_asset_id,
    candidate.sector
  FROM unmapped
  JOIN subscriber_global_assets candidate
    ON candidate.canonical_symbol = unmapped.symbol
   AND candidate.global_asset_id <> unmapped.global_asset_id
  WHERE btrim(candidate.sector) <> ''
    AND lower(candidate.sector) !~ 'services-|blank check|shell|holding company'
  ORDER BY unmapped.global_asset_id,
    CASE
      WHEN candidate.reference_provenance->>'source_group' = 'sp100' THEN 0
      WHEN candidate.reference_provenance->>'authority' = 'fmp_profile' THEN 1
      ELSE 2
    END,
    candidate.reference_effective_at DESC NULLS LAST,
    candidate.updated_at DESC,
    candidate.global_asset_id
)
INSERT INTO subscriber_global_asset_identity_resolutions
  (source_global_asset_id, canonical_global_asset_id, resolution_version, resolution_reason, resolved_at)
SELECT
  source_global_asset_id,
  canonical_global_asset_id,
  'canonical_sector_reconciliation.v1',
  'SAF sector benchmark reconciliation: same canonical symbol has governed sector metadata',
  now()
FROM ranked_candidates
ON CONFLICT (source_global_asset_id) DO UPDATE
SET canonical_global_asset_id = EXCLUDED.canonical_global_asset_id,
    resolution_version = EXCLUDED.resolution_version,
    resolution_reason = EXCLUDED.resolution_reason,
    resolved_at = EXCLUDED.resolved_at
WHERE subscriber_global_asset_identity_resolutions.canonical_global_asset_id IS DISTINCT FROM EXCLUDED.canonical_global_asset_id;

WITH seed(canonical_symbol, provider_sector, provider_industry, provider_exchange, canonical_sector) AS (
  VALUES
    ('ABNB', 'Internal Governance', 'Lodging marketplace services', 'XNAS', 'Consumer Discretionary'),
    ('AKAM', 'Internal Governance', 'Internet infrastructure and security services', 'XNAS', 'Technology'),
    ('DASH', 'Internal Governance', 'Local commerce delivery marketplace services', 'XNAS', 'Consumer Discretionary'),
    ('EBAY', 'Internal Governance', 'Online marketplace retail services', 'XNAS', 'Consumer Discretionary'),
    ('ETSY', 'Internal Governance', 'Online marketplace retail services', 'XNYS', 'Consumer Discretionary'),
    ('GPN',  'Internal Governance', 'Payment technology services', 'XNYS', 'Financials'),
    ('PYPL', 'Internal Governance', 'Payment technology services', 'XNAS', 'Financials')
), inserted AS (
  INSERT INTO subscriber_global_asset_sector_classifications
    (classification_id, global_asset_id, provider, provider_endpoint, provider_symbol,
     provider_sector, provider_industry, provider_exchange, canonical_sector,
     source_fingerprint, captured_at, correlation_id, provenance)
  SELECT
    'subsector_' || substr(md5(asset.global_asset_id || '|internal_sector_reconciliation.v1|' || seed.canonical_sector), 1, 24),
    asset.global_asset_id,
    'internal_catalog_governance',
    'internal/catalog-governance',
    seed.canonical_symbol,
    seed.provider_sector,
    seed.provider_industry,
    seed.provider_exchange,
    seed.canonical_sector,
    md5('internal|catalog-governance|' || asset.global_asset_id || '|' || seed.canonical_symbol || '|' || seed.canonical_sector),
    now(),
    'saf-sector-reconciliation-20260905',
    jsonb_build_object(
      'schema_version', 'subscriber_global_sector_reconciliation.v1',
      'authority', 'internal_catalog_governance',
      'source_scope', 'SAF post-cutoff sector_unmapped remediation',
      'source_reference', asset.reference_provenance,
      'input_sector', asset.sector,
      'input_industry', asset.industry,
      'canonical_sector', seed.canonical_sector,
      'no_provider_polling', true
    )
  FROM seed
  JOIN subscriber_global_assets asset
    ON asset.canonical_symbol = seed.canonical_symbol
  WHERE NOT EXISTS (
    SELECT 1
    FROM subscriber_global_assets candidate
    WHERE candidate.canonical_symbol = seed.canonical_symbol
      AND candidate.global_asset_id <> asset.global_asset_id
      AND btrim(candidate.sector) <> ''
      AND lower(candidate.sector) !~ 'services-|blank check|shell|holding company'
  )
  ON CONFLICT (source_fingerprint) DO NOTHING
  RETURNING global_asset_id, classification_id, provider_symbol, provider_industry,
    provider_exchange, canonical_sector, source_fingerprint
), latest AS (
  SELECT DISTINCT ON (global_asset_id)
    global_asset_id, classification_id, provider_symbol, provider_industry,
    provider_exchange, canonical_sector, source_fingerprint
  FROM inserted
  ORDER BY global_asset_id, classification_id DESC
)
UPDATE subscriber_global_assets asset
SET asset_type = CASE WHEN asset.asset_type = '' THEN 'equity' ELSE asset.asset_type END,
    exchange = latest.provider_exchange,
    sector = latest.canonical_sector,
    industry = latest.provider_industry,
    reference_effective_at = now(),
    reference_provenance = jsonb_build_object(
      'authority', 'internal_catalog_governance',
      'classification_id', latest.classification_id,
      'source_fingerprint', latest.source_fingerprint,
      'provider_symbol', latest.provider_symbol,
      'canonical_sector', latest.canonical_sector,
      'correlation_id', 'saf-sector-reconciliation-20260905',
      'classification_scope', 'saf_post_cutoff_sector_reconciliation.v1',
      'no_provider_polling', true
    ),
    updated_at = now()
FROM latest
WHERE asset.global_asset_id = latest.global_asset_id;

INSERT INTO schema_migrations (version, applied_at)
VALUES ('000169_subscriber_global_saf_sector_reconciliation', now())
ON CONFLICT (version) DO NOTHING;
