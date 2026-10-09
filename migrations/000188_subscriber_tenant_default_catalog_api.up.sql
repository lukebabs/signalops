-- Governed tenant-default catalog selection and read-only external manifest.
-- The warm view is the source of eligible 1,000-member candidates; the
-- current ranking snapshot supplies the market-cap ordering metadata.
CREATE OR REPLACE FUNCTION subscriber_search_global_warm_catalog(
  p_tenant_id text, p_list_id text, p_query text, p_preset text, p_limit integer, p_offset integer
)
RETURNS TABLE (
  global_asset_id text, ticker text, company_name text, asset_type text,
  exchange text, sector text, eligibility_status text, coverage_state text,
  coverage_mode text, warm_rank integer, market_cap_rank integer,
  tenant_default_member boolean, legacy_protected boolean
)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,public AS $$
  SELECT asset.global_asset_id, asset.canonical_symbol, asset.company_name,
    asset.asset_type, asset.exchange, asset.sector, asset.eligibility_status,
    COALESCE(coverage.coverage_state, 'not_requested'),
    COALESCE(coverage.execution_mode, 'shadow'),
    candidates.warm_rank,
    ranking.source_rank,
    EXISTS (
      SELECT 1 FROM public.subscriber_watchlist_memberships membership
      WHERE membership.tenant_id = p_tenant_id
        AND membership.list_id = p_list_id
        AND membership.global_asset_id = asset.global_asset_id
    ),
    EXISTS (
      SELECT 1 FROM public.subscriber_watchlist_memberships legacy
      WHERE legacy.tenant_id = p_tenant_id
        AND legacy.list_id = 'sublist-tenant-local-legacy-default'
        AND legacy.global_asset_id = asset.global_asset_id
    )
  FROM (
    SELECT warm.global_asset_id, warm.priority AS warm_rank
    FROM public.subscriber_global_warm_eod_assets warm
    UNION
    SELECT legacy.global_asset_id, NULL::integer AS warm_rank
    FROM public.subscriber_watchlist_memberships legacy
    WHERE legacy.tenant_id = p_tenant_id
      AND legacy.list_id = 'sublist-tenant-local-legacy-default'
  ) candidates
  JOIN public.subscriber_global_assets asset ON asset.global_asset_id = candidates.global_asset_id
  LEFT JOIN public.subscriber_global_asset_coverage coverage
    ON coverage.global_asset_id = asset.global_asset_id
   AND coverage.coverage_product = 'eod_baseline'
  LEFT JOIN public.subscriber_global_ranking_snapshots snapshot ON snapshot.is_current
  LEFT JOIN public.subscriber_global_ranking_snapshot_entries ranking
    ON ranking.ranking_snapshot_id = snapshot.ranking_snapshot_id
   AND ranking.global_asset_id = asset.global_asset_id
  WHERE asset.eligibility_status = 'eligible'
    AND (NULLIF(btrim(p_query), '') IS NULL
      OR asset.canonical_symbol ILIKE '%' || btrim(p_query) || '%'
      OR asset.company_name ILIKE '%' || btrim(p_query) || '%')
    AND (COALESCE(p_preset, '') <> 'snp500_top200'
      OR COALESCE(ranking.source_rank, 2147483647) <= 200
      OR EXISTS (
        SELECT 1 FROM public.subscriber_watchlist_memberships legacy_filter
        WHERE legacy_filter.tenant_id = p_tenant_id
          AND legacy_filter.list_id = 'sublist-tenant-local-legacy-default'
          AND legacy_filter.global_asset_id = asset.global_asset_id
      ))
  ORDER BY COALESCE(ranking.source_rank, 2147483647), warm.priority, asset.canonical_symbol, asset.global_asset_id
  LIMIT LEAST(GREATEST(COALESCE(p_limit, 100), 1), 1000)
  OFFSET GREATEST(COALESCE(p_offset, 0), 0)
$$;

ALTER FUNCTION subscriber_search_global_warm_catalog(text,text,text,text,integer,integer) OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON FUNCTION subscriber_search_global_warm_catalog(text,text,text,text,integer,integer) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION subscriber_search_global_warm_catalog(text,text,text,text,integer,integer) TO signalops_subscriber_gateway;

CREATE OR REPLACE FUNCTION subscriber_add_tenant_default_catalog_membership(
  p_subject text, p_list_id text, p_global_asset_id text, p_correlation_id text
)
RETURNS TABLE (tenant_id text, list_id text, global_asset_id text, added_by_subject text, added_at timestamptz, activation_state text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,public AS $$
DECLARE
  v_tenant text;
  v_added_by text;
  v_added_at timestamptz;
  v_state text := 'active';
BEGIN
  SELECT list.tenant_id INTO v_tenant
  FROM public.subscriber_watchlists list
  WHERE list.tenant_id = current_setting('signalops.tenant_id', true)
    AND list.list_id = p_list_id AND list.list_kind = 'tenant_default';
  IF NOT FOUND THEN RETURN; END IF;

  IF NOT EXISTS (
    SELECT 1 FROM public.subscriber_global_assets asset
    WHERE asset.global_asset_id = p_global_asset_id AND asset.eligibility_status = 'eligible'
  ) THEN RETURN; END IF;

  INSERT INTO public.subscriber_watchlist_memberships
    (tenant_id, list_id, global_asset_id, added_by_subject, provenance)
  VALUES (v_tenant, p_list_id, p_global_asset_id, p_subject,
    jsonb_build_object('schema_version','subscriber.watchlist.v1','surface','subscriber.admin.coverage','correlation_id',coalesce(p_correlation_id,'')))
  ON CONFLICT (list_id, global_asset_id) DO NOTHING
  RETURNING added_by_subject, added_at INTO v_added_by, v_added_at;

  IF NOT FOUND THEN
    SELECT membership.added_by_subject, membership.added_at
      INTO v_added_by, v_added_at
    FROM public.subscriber_watchlist_memberships membership
    WHERE membership.tenant_id = v_tenant AND membership.list_id = p_list_id AND membership.global_asset_id = p_global_asset_id;
  END IF;

  INSERT INTO public.subscriber_watchlist_audit
    (audit_id, tenant_id, list_id, actor_subject, mutation, global_asset_id, after_value, correlation_id)
  VALUES ('sublistaudit-' || md5(p_list_id || p_global_asset_id || clock_timestamp()::text), v_tenant, p_list_id,
    p_subject, 'add_asset', p_global_asset_id,
    jsonb_build_object('global_asset_id', p_global_asset_id, 'promotion', 'hot'), coalesce(p_correlation_id,''));

  INSERT INTO public.subscriber_global_coverage_activation_requests
    (activation_request_id, global_asset_id, request_key, request_state, request_reason, requester_kind,
     requester_tenant_id, requester_subject, requester_list_id, policy_version, provenance, requested_at)
  VALUES ('subactivation-' || md5(v_tenant || ':' || p_global_asset_id), p_global_asset_id,
    'subscriber-tenant-default-hot-v1:' || v_tenant || ':' || p_global_asset_id, 'queued',
    'subscriber_tenant_default_hot_asset', 'tenant_default_list', v_tenant, p_subject, p_list_id,
    'subscriber-tenant-default-hot-v1', jsonb_build_object('surface','subscriber.admin.coverage','correlation_id',coalesce(p_correlation_id,'')), now())
  ON CONFLICT (request_key) DO UPDATE SET updated_at = now();

  RETURN QUERY SELECT v_tenant, p_list_id, p_global_asset_id, v_added_by, v_added_at, v_state;
END;
$$;

ALTER FUNCTION subscriber_add_tenant_default_catalog_membership(text,text,text,text) OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON FUNCTION subscriber_add_tenant_default_catalog_membership(text,text,text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION subscriber_add_tenant_default_catalog_membership(text,text,text,text) TO signalops_subscriber_gateway;
