-- S5 correction: return one canonical governed security per catalog result.
-- Source rows and canonical-resolution provenance remain immutable and retained.
CREATE OR REPLACE FUNCTION subscriber_search_global_catalog(p_query text, p_limit integer)
RETURNS TABLE (
  global_asset_id text,
  ticker text,
  company_name text,
  asset_type text,
  exchange text,
  sector text,
  eligibility_status text,
  coverage_state text,
  coverage_mode text
)
LANGUAGE sql
STABLE
SECURITY DEFINER
SET search_path = pg_catalog, public
AS $$
  WITH canonical_assets AS (
    SELECT DISTINCT COALESCE(identity_resolution.canonical_global_asset_id, source_asset.global_asset_id) AS global_asset_id
    FROM public.subscriber_global_assets AS source_asset
    LEFT JOIN public.subscriber_global_asset_identity_resolutions AS identity_resolution
      ON identity_resolution.source_global_asset_id = source_asset.global_asset_id
    WHERE source_asset.eligibility_status = 'eligible'
  )
  SELECT asset.global_asset_id, asset.canonical_symbol, asset.company_name,
    asset.asset_type, asset.exchange, asset.sector, asset.eligibility_status,
    COALESCE(coverage.coverage_state, 'not_requested'),
    COALESCE(coverage.execution_mode, 'shadow')
  FROM canonical_assets
  JOIN public.subscriber_global_assets AS asset
    ON asset.global_asset_id = canonical_assets.global_asset_id
  LEFT JOIN public.subscriber_global_asset_coverage AS coverage
    ON coverage.global_asset_id = asset.global_asset_id
   AND coverage.coverage_product = 'eod_baseline'
  WHERE asset.eligibility_status = 'eligible'
    AND (NULLIF(btrim(p_query), '') IS NULL
      OR asset.canonical_symbol ILIKE '%' || btrim(p_query) || '%'
      OR asset.company_name ILIKE '%' || btrim(p_query) || '%')
  ORDER BY
    CASE WHEN upper(asset.canonical_symbol) = upper(btrim(p_query)) THEN 0 ELSE 1 END,
    asset.canonical_symbol, asset.global_asset_id
  LIMIT LEAST(GREATEST(COALESCE(p_limit, 20), 1), 50);
$$;

ALTER FUNCTION subscriber_search_global_catalog(text, integer) OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON FUNCTION subscriber_search_global_catalog(text, integer) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION subscriber_search_global_catalog(text, integer) TO signalops_subscriber_gateway;
