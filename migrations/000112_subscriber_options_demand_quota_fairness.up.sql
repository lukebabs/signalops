-- S6 quota enforcement and deferred-age fairness for the aggregate-only
-- shadow planner. No entitlement, capture, scheduler, or provider path changes.

CREATE OR REPLACE FUNCTION subscriber_options_demand_aggregate()
RETURNS TABLE (global_asset_id text, highest_tier_rank integer, eligible_tenant_count integer, eligible_watcher_count integer, deferred_sessions integer)
LANGUAGE sql STABLE SECURITY DEFINER
SET search_path = pg_catalog, public
AS $$
  WITH tenant_watcher_demand AS (
    SELECT DISTINCT entitlement.tenant_id, entitlement.product_tier, capability.quota_limit,
      membership.global_asset_id,
      CASE WHEN watchlist.list_kind='tenant_default' THEN entitlement.tenant_id || ':tenant-default'
           ELSE entitlement.tenant_id || ':private:' || watchlist.owner_subject END AS watcher_key
    FROM public.subscriber_tenant_entitlements entitlement
    JOIN public.subscriber_entitlement_capabilities capability ON capability.tenant_id=entitlement.tenant_id
    JOIN public.subscriber_watchlists watchlist ON watchlist.tenant_id=entitlement.tenant_id
    JOIN public.subscriber_watchlist_memberships membership ON membership.tenant_id=watchlist.tenant_id AND membership.list_id=watchlist.list_id
    JOIN public.subscriber_global_assets asset ON asset.global_asset_id=membership.global_asset_id
    WHERE entitlement.status='active' AND capability.capability='options_demand'
      AND capability.enabled AND capability.quota_limit>0 AND asset.eligibility_status='eligible'
  ), tenant_asset_demand AS (
    SELECT DISTINCT tenant_id, product_tier, quota_limit, global_asset_id
    FROM tenant_watcher_demand
  ), ranked_tenant_assets AS (
    SELECT *, row_number() OVER (PARTITION BY tenant_id ORDER BY global_asset_id) AS tenant_asset_position
    FROM tenant_asset_demand
  ), quota_eligible AS (
    SELECT watcher.* FROM tenant_watcher_demand watcher
    JOIN ranked_tenant_assets ranked ON ranked.tenant_id=watcher.tenant_id AND ranked.global_asset_id=watcher.global_asset_id
    WHERE ranked.tenant_asset_position <= ranked.quota_limit
  ), latest_member_state AS (
    SELECT DISTINCT ON (member.global_asset_id) member.global_asset_id, member.selection_state, member.deferred_sessions
    FROM public.subscriber_options_demand_snapshot_members member
    JOIN public.subscriber_options_demand_snapshot_runs run ON run.snapshot_run_id=member.snapshot_run_id
    ORDER BY member.global_asset_id, run.planned_at DESC, run.snapshot_run_id DESC
  )
  SELECT demand.global_asset_id,
    max(CASE demand.product_tier WHEN 'enterprise' THEN 30 WHEN 'professional' THEN 20 WHEN 'standard' THEN 10 ELSE 1 END)::integer,
    count(DISTINCT demand.tenant_id)::integer,
    count(DISTINCT demand.watcher_key)::integer,
    max(CASE WHEN previous.selection_state='deferred' THEN previous.deferred_sessions + 1 ELSE 0 END)::integer
  FROM quota_eligible demand
  LEFT JOIN latest_member_state previous ON previous.global_asset_id=demand.global_asset_id
  GROUP BY demand.global_asset_id
  ORDER BY demand.global_asset_id;
$$;
ALTER FUNCTION subscriber_options_demand_aggregate() OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON FUNCTION subscriber_options_demand_aggregate() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION subscriber_options_demand_aggregate() TO signalops_subscriber_options_demand;
