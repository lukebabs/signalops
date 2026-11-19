-- Subscriber Project S3: bounded watchlist item projection.
-- The gateway may resolve display metadata only for assets already reachable
-- through the caller's tenant-default or subject-owned private list. It does
-- not receive SELECT on the global catalog tables.

CREATE POLICY subscriber_watchlists_projection_owner_read
  ON subscriber_watchlists
  FOR SELECT TO signalops_subscriber_migrator
  USING (true);

CREATE POLICY subscriber_watchlist_memberships_projection_owner_read
  ON subscriber_watchlist_memberships
  FOR SELECT TO signalops_subscriber_migrator
  USING (true);

CREATE FUNCTION subscriber_visible_watchlist_items(p_subject text, p_list_id text)
RETURNS TABLE (
  tenant_id text,
  list_id text,
  list_kind text,
  list_name text,
  global_asset_id text,
  ticker text,
  company_name text,
  asset_type text,
  exchange text,
  sector text,
  eligibility_status text,
  coverage_state text,
  coverage_mode text,
  added_at timestamptz
)
LANGUAGE sql
STABLE
SECURITY DEFINER
SET search_path = pg_catalog, public
AS $$
  SELECT
    list.tenant_id,
    list.list_id,
    list.list_kind,
    list.list_name,
    membership.global_asset_id,
    asset.canonical_symbol,
    asset.company_name,
    asset.asset_type,
    asset.exchange,
    asset.sector,
    asset.eligibility_status,
    COALESCE(coverage.coverage_state, 'not_requested'),
    COALESCE(coverage.execution_mode, 'shadow'),
    membership.added_at
  FROM public.subscriber_watchlists AS list
  JOIN public.subscriber_watchlist_memberships AS membership
    ON membership.tenant_id = list.tenant_id
   AND membership.list_id = list.list_id
  JOIN public.subscriber_global_assets AS asset
    ON asset.global_asset_id = membership.global_asset_id
  LEFT JOIN public.subscriber_global_asset_coverage AS coverage
    ON coverage.global_asset_id = asset.global_asset_id
   AND coverage.coverage_product = 'eod_baseline'
  WHERE list.tenant_id = current_setting('signalops.tenant_id', true)
    AND list.list_id = p_list_id
    AND (
      list.list_kind = 'tenant_default'
      OR (list.list_kind = 'private' AND list.owner_subject = p_subject)
    )
  ORDER BY membership.added_at, membership.global_asset_id;
$$;

ALTER FUNCTION subscriber_visible_watchlist_items(text, text) OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON FUNCTION subscriber_visible_watchlist_items(text, text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION subscriber_visible_watchlist_items(text, text) TO signalops_subscriber_gateway;
