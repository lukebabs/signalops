-- Collapse duplicate historical global identities after the primary ticker
-- mapping as well; SAF must have exactly one row per governed ticker.
CREATE OR REPLACE FUNCTION subscriber_global_saf_benchmark_primary_members()
RETURNS TABLE(global_asset_id text)
LANGUAGE sql
STABLE
SECURITY DEFINER
SET search_path = public
AS $$
  SELECT DISTINCT ON (upper(asset.canonical_symbol)) asset.global_asset_id
  FROM marketops_primary_assets primary_asset
  JOIN subscriber_global_assets asset
    ON upper(asset.canonical_symbol) = upper(primary_asset.ticker)
  WHERE primary_asset.tenant_id = 'tenant-local'
    AND primary_asset.is_active = true
  ORDER BY upper(asset.canonical_symbol), asset.global_asset_id
$$;

INSERT INTO schema_migrations(version, applied_at)
VALUES ('000194_subscriber_global_saf_primary_global_identity_dedupe', now())
ON CONFLICT (version) DO NOTHING;
