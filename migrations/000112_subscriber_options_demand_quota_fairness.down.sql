-- Restore the original aggregate projection without numeric quota selection or
-- deferred-age carry-forward. This is a pre-capture shadow-only rollback.
CREATE OR REPLACE FUNCTION subscriber_options_demand_aggregate()
RETURNS TABLE (global_asset_id text, highest_tier_rank integer, eligible_tenant_count integer, eligible_watcher_count integer, deferred_sessions integer)
LANGUAGE sql STABLE SECURITY DEFINER
SET search_path = pg_catalog, public
AS $$
  WITH eligible_demands AS (
    SELECT membership.global_asset_id, entitlement.tenant_id,
      CASE WHEN watchlist.list_kind='tenant_default' THEN entitlement.tenant_id || ':tenant-default'
           ELSE entitlement.tenant_id || ':private:' || watchlist.owner_subject END AS watcher_key,
      CASE entitlement.product_tier WHEN 'enterprise' THEN 30 WHEN 'professional' THEN 20 WHEN 'standard' THEN 10 ELSE 1 END AS tier_rank
    FROM public.subscriber_tenant_entitlements entitlement
    JOIN public.subscriber_entitlement_capabilities capability ON capability.tenant_id=entitlement.tenant_id
    JOIN public.subscriber_watchlists watchlist ON watchlist.tenant_id=entitlement.tenant_id
    JOIN public.subscriber_watchlist_memberships membership ON membership.tenant_id=watchlist.tenant_id AND membership.list_id=watchlist.list_id
    JOIN public.subscriber_global_assets asset ON asset.global_asset_id=membership.global_asset_id
    WHERE entitlement.status='active' AND capability.capability='options_demand'
      AND capability.enabled AND capability.quota_limit>0 AND asset.eligibility_status='eligible'
  )
  SELECT global_asset_id, max(tier_rank)::integer, count(DISTINCT tenant_id)::integer,
    count(DISTINCT watcher_key)::integer, 0::integer
  FROM eligible_demands GROUP BY global_asset_id ORDER BY global_asset_id;
$$;
ALTER FUNCTION subscriber_options_demand_aggregate() OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON FUNCTION subscriber_options_demand_aggregate() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION subscriber_options_demand_aggregate() TO signalops_subscriber_options_demand;
