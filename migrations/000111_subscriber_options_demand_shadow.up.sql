-- S6 aggregate-only Options-demand planning. The worker has execute access
-- only to this non-identifying projection; it cannot read tenant/list rows.

CREATE POLICY subscriber_options_demand_migrator_entitlements ON subscriber_tenant_entitlements FOR SELECT TO signalops_subscriber_migrator USING (true);
CREATE POLICY subscriber_options_demand_migrator_capabilities ON subscriber_entitlement_capabilities FOR SELECT TO signalops_subscriber_migrator USING (true);
CREATE POLICY subscriber_options_demand_migrator_watchlists ON subscriber_watchlists FOR SELECT TO signalops_subscriber_migrator USING (true);
CREATE POLICY subscriber_options_demand_migrator_memberships ON subscriber_watchlist_memberships FOR SELECT TO signalops_subscriber_migrator USING (true);

CREATE FUNCTION subscriber_options_demand_aggregate()
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

CREATE TABLE subscriber_options_demand_snapshot_runs (
  snapshot_run_id text PRIMARY KEY, planner_version text NOT NULL, session_date date NOT NULL,
  execution_mode text NOT NULL CHECK (execution_mode = 'shadow'),
  max_symbols integer NOT NULL CHECK (max_symbols > 0 AND max_symbols <= 1000),
  source_demand_count integer NOT NULL CHECK (source_demand_count >= 0), candidate_count integer NOT NULL CHECK (candidate_count >= 0),
  selected_count integer NOT NULL CHECK (selected_count >= 0 AND selected_count <= max_symbols), deferred_count integer NOT NULL CHECK (deferred_count >= 0),
  report jsonb NOT NULL DEFAULT '{}'::jsonb, planned_by text NOT NULL CHECK (planned_by = 'subscriber-options-demand-planner'),
  correlation_id text NOT NULL DEFAULT '', planned_at timestamptz NOT NULL, created_at timestamptz NOT NULL DEFAULT now(),
  CHECK (candidate_count = selected_count + deferred_count)
);
CREATE TABLE subscriber_options_demand_snapshot_members (
  snapshot_run_id text NOT NULL REFERENCES subscriber_options_demand_snapshot_runs(snapshot_run_id) ON DELETE RESTRICT,
  global_asset_id text NOT NULL REFERENCES subscriber_global_assets(global_asset_id) ON DELETE RESTRICT,
  priority integer NOT NULL CHECK (priority > 0), selection_state text NOT NULL CHECK (selection_state IN ('selected', 'deferred')),
  highest_tier_rank integer NOT NULL CHECK (highest_tier_rank >= 0), eligible_tenant_count integer NOT NULL CHECK (eligible_tenant_count > 0),
  eligible_watcher_count integer NOT NULL CHECK (eligible_watcher_count > 0), deferred_sessions integer NOT NULL CHECK (deferred_sessions >= 0),
  selection_reason text NOT NULL CHECK (selection_reason = 'entitled_watchlist_demand_union'),
  PRIMARY KEY (snapshot_run_id, global_asset_id), UNIQUE (snapshot_run_id, priority)
);
ALTER TABLE subscriber_options_demand_snapshot_runs OWNER TO signalops_subscriber_migrator;
ALTER TABLE subscriber_options_demand_snapshot_members OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON subscriber_options_demand_snapshot_runs, subscriber_options_demand_snapshot_members FROM PUBLIC;
GRANT SELECT, INSERT ON subscriber_options_demand_snapshot_runs, subscriber_options_demand_snapshot_members TO signalops_subscriber_options_demand;
