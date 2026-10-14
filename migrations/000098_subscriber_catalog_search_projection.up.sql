-- Subscriber Project S5: bounded entitlement-gated catalog projection.
-- This exposes platform metadata only through a SECURITY DEFINER function;
-- the subscriber gateway receives no SELECT grant on global catalog tables.

CREATE FUNCTION subscriber_search_global_catalog(p_query text, p_limit integer)
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
  SELECT asset.global_asset_id, asset.canonical_symbol, asset.company_name,
    asset.asset_type, asset.exchange, asset.sector, asset.eligibility_status,
    COALESCE(coverage.coverage_state, 'not_requested'),
    COALESCE(coverage.execution_mode, 'shadow')
  FROM public.subscriber_global_assets AS asset
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
