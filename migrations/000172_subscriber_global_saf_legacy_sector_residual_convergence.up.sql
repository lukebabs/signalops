-- Converge any remaining tenant-local legacy SAF observations that still
-- project as sector_unmapped when a same-symbol sector-bearing catalog identity
-- already exists. This is projection metadata only; immutable evidence and
-- benchmark observations are not rewritten.

WITH residual_symbols AS (
  SELECT DISTINCT observation.symbol
  FROM subscriber_gateway_global_signal_assurance_observations observation
  JOIN subscriber_global_saf_benchmark_legacy_default_members() member
    ON member.global_asset_id = observation.global_asset_id
  WHERE observation.matured_session_date >= DATE '2026-08-20'
    AND observation.sector_benchmark_state = 'sector_unmapped'
), preferred AS (
  SELECT DISTINCT ON (asset.canonical_symbol)
    asset.canonical_symbol,
    asset.global_asset_id AS canonical_global_asset_id
  FROM subscriber_global_assets asset
  JOIN residual_symbols residual
    ON residual.symbol = asset.canonical_symbol
  WHERE btrim(asset.sector) <> ''
    AND lower(asset.sector) !~ 'services-|blank check|shell|holding company'
  ORDER BY asset.canonical_symbol,
    CASE
      WHEN asset.reference_provenance->>'source_group' = 'sp100' THEN 0
      WHEN asset.reference_provenance->>'authority' = 'fmp_profile' THEN 1
      WHEN asset.reference_provenance->>'authority' = 'internal_catalog_governance' THEN 2
      ELSE 3
    END,
    asset.reference_effective_at DESC NULLS LAST,
    asset.updated_at DESC,
    asset.global_asset_id
), source_rows AS (
  SELECT source.global_asset_id AS source_global_asset_id,
    preferred.canonical_global_asset_id
  FROM subscriber_global_assets source
  JOIN preferred
    ON preferred.canonical_symbol = source.canonical_symbol
)
INSERT INTO subscriber_global_asset_identity_resolutions
  (source_global_asset_id, canonical_global_asset_id, resolution_version, resolution_reason, resolved_at)
SELECT
  source_global_asset_id,
  canonical_global_asset_id,
  'canonical_sector_identity_convergence.v2',
  'SAF legacy residual sector reconciliation: converge same-symbol identities to sector-bearing canonical asset',
  now()
FROM source_rows
ON CONFLICT (source_global_asset_id) DO UPDATE
SET canonical_global_asset_id = EXCLUDED.canonical_global_asset_id,
    resolution_version = EXCLUDED.resolution_version,
    resolution_reason = EXCLUDED.resolution_reason,
    resolved_at = EXCLUDED.resolved_at
WHERE subscriber_global_asset_identity_resolutions.canonical_global_asset_id IS DISTINCT FROM EXCLUDED.canonical_global_asset_id
   OR subscriber_global_asset_identity_resolutions.resolution_version IS DISTINCT FROM EXCLUDED.resolution_version;

INSERT INTO schema_migrations (version, applied_at)
VALUES ('000172_subscriber_global_saf_legacy_sector_residual_convergence', now())
ON CONFLICT (version) DO NOTHING;
