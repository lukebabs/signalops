-- S5 repair: perform the private-list audit as a separate scoped statement.
-- This preserves the existing function contract while avoiding the PostgreSQL
-- data-modifying-CTE RLS interaction. No broader table privilege is granted.
CREATE OR REPLACE FUNCTION subscriber_add_private_catalog_membership(p_subject text,p_list_id text,p_global_asset_id text,p_correlation_id text)
RETURNS TABLE (tenant_id text,list_id text,global_asset_id text,added_by_subject text,added_at timestamptz,activation_state text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,public AS $$
DECLARE
  v_tenant_id text;
  v_global_asset_id text;
  v_added_by_subject text;
  v_added_at timestamptz;
  v_coverage_state text;
  v_coverage_mode text;
  v_activation_state text := 'active';
BEGIN
  SELECT watchlist.tenant_id INTO v_tenant_id
  FROM public.subscriber_watchlists AS watchlist
  WHERE watchlist.tenant_id=current_setting('signalops.tenant_id',true)
    AND watchlist.list_id=p_list_id
    AND watchlist.list_kind='private'
    AND watchlist.owner_subject=p_subject;
  IF NOT FOUND THEN RETURN; END IF;

  SELECT asset.global_asset_id,COALESCE(coverage.coverage_state,'not_requested'),COALESCE(coverage.execution_mode,'shadow')
  INTO v_global_asset_id,v_coverage_state,v_coverage_mode
  FROM public.subscriber_global_assets AS asset
  LEFT JOIN public.subscriber_global_asset_coverage AS coverage
    ON coverage.global_asset_id=asset.global_asset_id AND coverage.coverage_product='eod_baseline'
  WHERE asset.global_asset_id=p_global_asset_id AND asset.eligibility_status='eligible';
  IF NOT FOUND THEN RETURN; END IF;

  INSERT INTO public.subscriber_watchlist_memberships AS new_membership (tenant_id,list_id,global_asset_id,added_by_subject,provenance)
  VALUES (v_tenant_id,p_list_id,v_global_asset_id,p_subject,jsonb_build_object('schema_version','subscriber.watchlist.v1','surface','subscriber.catalog'))
  ON CONFLICT ON CONSTRAINT subscriber_watchlist_memberships_pkey DO NOTHING
  RETURNING new_membership.added_by_subject,new_membership.added_at INTO v_added_by_subject,v_added_at;

  IF FOUND THEN
    INSERT INTO public.subscriber_watchlist_audit (audit_id,tenant_id,list_id,actor_subject,mutation,global_asset_id,before_value,after_value,correlation_id,occurred_at)
    VALUES ('sublistaudit-'||md5(p_list_id||v_global_asset_id||clock_timestamp()::text),v_tenant_id,p_list_id,p_subject,'add_asset',v_global_asset_id,'{}'::jsonb,jsonb_build_object('global_asset_id',v_global_asset_id),COALESCE(p_correlation_id,''),now());
  ELSE
    SELECT membership.added_by_subject,membership.added_at INTO v_added_by_subject,v_added_at
    FROM public.subscriber_watchlist_memberships AS membership
    WHERE membership.tenant_id=v_tenant_id AND membership.list_id=p_list_id AND membership.global_asset_id=v_global_asset_id;
    IF NOT FOUND THEN RETURN; END IF;
  END IF;

  IF NOT (v_coverage_state='active' AND v_coverage_mode='enabled') THEN
    INSERT INTO public.subscriber_global_coverage_activation_requests (activation_request_id,global_asset_id,request_key,request_state,request_reason,requester_kind,requester_tenant_id,requester_subject,requester_list_id,policy_version,provenance,requested_at)
    VALUES ('subactivation-'||md5(v_global_asset_id||clock_timestamp()::text),v_global_asset_id,'subscriber-eod-activation-v1:'||v_global_asset_id,'queued','subscriber_private_list_cold_asset','user_private_list',v_tenant_id,p_subject,p_list_id,'subscriber-eod-activation-v1',jsonb_build_object('surface','subscriber.catalog','correlation_id',COALESCE(p_correlation_id,'')),now())
    ON CONFLICT (request_key) DO UPDATE SET updated_at=now()
    RETURNING request_state INTO v_activation_state;
  END IF;

  RETURN QUERY SELECT v_tenant_id,p_list_id,v_global_asset_id,v_added_by_subject,v_added_at,v_activation_state;
END;
$$;
ALTER FUNCTION subscriber_add_private_catalog_membership(text,text,text,text) OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON FUNCTION subscriber_add_private_catalog_membership(text,text,text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION subscriber_add_private_catalog_membership(text,text,text,text) TO signalops_subscriber_gateway;
