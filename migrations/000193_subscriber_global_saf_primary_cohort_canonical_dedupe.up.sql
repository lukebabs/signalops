-- The global catalog can retain multiple identities for a canonical symbol.
-- SAF must evaluate one governed global identity per primary ticker.
CREATE OR REPLACE FUNCTION subscriber_global_saf_benchmark_primary_members()
RETURNS TABLE(global_asset_id text)
LANGUAGE sql
STABLE
SECURITY DEFINER
SET search_path = public
AS $$
  SELECT asset.global_asset_id
  FROM (
    SELECT DISTINCT ON (upper(primary_asset.ticker))
      upper(primary_asset.ticker) AS ticker,
      primary_asset.universe_priority,
      primary_asset.rank
    FROM marketops_primary_assets primary_asset
    WHERE primary_asset.tenant_id = 'tenant-local'
      AND primary_asset.is_active = true
    ORDER BY upper(primary_asset.ticker), primary_asset.universe_priority, primary_asset.rank NULLS LAST
  ) primary_asset
  JOIN subscriber_global_assets asset
    ON upper(asset.canonical_symbol) = primary_asset.ticker
  ORDER BY primary_asset.universe_priority, primary_asset.rank NULLS LAST, primary_asset.ticker
$$;

INSERT INTO schema_migrations(version, applied_at)
VALUES ('000193_subscriber_global_saf_primary_cohort_canonical_dedupe', now())
ON CONFLICT (version) DO NOTHING;
