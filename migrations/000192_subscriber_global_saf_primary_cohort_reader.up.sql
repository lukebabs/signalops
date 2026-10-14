-- Expose only the governed live MarketOps primary cohort to the SAF benchmark
-- worker. Historical legacy-default observations remain immutable and readable
-- through the existing legacy function.
CREATE FUNCTION subscriber_global_saf_benchmark_primary_members()
RETURNS TABLE(global_asset_id text)
LANGUAGE sql
STABLE
SECURITY DEFINER
SET search_path = public
AS $$
  SELECT DISTINCT asset.global_asset_id
  FROM marketops_primary_assets primary_asset
  JOIN subscriber_global_assets asset
    ON upper(asset.canonical_symbol) = upper(primary_asset.ticker)
  WHERE primary_asset.tenant_id = 'tenant-local'
    AND primary_asset.is_active = true
$$;

ALTER FUNCTION subscriber_global_saf_benchmark_primary_members() OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON FUNCTION subscriber_global_saf_benchmark_primary_members() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION subscriber_global_saf_benchmark_primary_members() TO signalops_subscriber_global_eod;

INSERT INTO schema_migrations(version, applied_at)
VALUES ('000192_subscriber_global_saf_primary_cohort_reader', now())
ON CONFLICT (version) DO NOTHING;
