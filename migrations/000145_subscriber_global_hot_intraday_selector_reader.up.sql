-- Constrained cross-tenant aggregate selector boundary.
-- The no-login selector owns only this SECURITY DEFINER aggregation function.
-- It returns aggregate global demand, never tenant, subject, list, or raw market rows.
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = $role$signalops_subscriber_global_selector$role$) THEN
    CREATE ROLE signalops_subscriber_global_selector NOLOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE BYPASSRLS;
  END IF;
END
$$;

GRANT SELECT ON subscriber_watchlist_context_preferences, subscriber_watchlists,
  subscriber_watchlist_memberships, subscriber_global_assets
  TO signalops_subscriber_global_selector;

CREATE OR REPLACE FUNCTION subscriber_global_hot_intraday_selector_rows()
RETURNS TABLE (
  global_asset_id text,
  canonical_symbol text,
  watcher_count bigint,
  first_selected_at timestamptz,
  last_selected_at timestamptz
)
LANGUAGE sql
STABLE
SECURITY DEFINER
SET search_path = pg_catalog, public
AS $selector$
  WITH selected_lists AS (
    SELECT preference.tenant_id, preference.subject, list.list_id, preference.updated_at
    FROM public.subscriber_watchlist_context_preferences AS preference
    JOIN public.subscriber_watchlists AS list
      ON list.tenant_id = preference.tenant_id
     AND ((preference.selection_mode = $$list$$ AND preference.list_id = list.list_id)
       OR preference.selection_mode = $$all$$)
     AND (list.list_kind = $$tenant_default$$
       OR (list.list_kind = $$private$$ AND list.owner_subject = preference.subject))
  )
  SELECT asset.global_asset_id, asset.canonical_symbol,
    count(DISTINCT selected.tenant_id || chr(31) || selected.subject)::bigint AS watcher_count,
    min(membership.added_at) AS first_selected_at,
    max(selected.updated_at) AS last_selected_at
  FROM selected_lists AS selected
  JOIN public.subscriber_watchlist_memberships AS membership
    ON membership.tenant_id = selected.tenant_id AND membership.list_id = selected.list_id
  JOIN public.subscriber_global_assets AS asset ON asset.global_asset_id = membership.global_asset_id
  WHERE asset.eligibility_status = $$eligible$$
  GROUP BY asset.global_asset_id, asset.canonical_symbol
$selector$;

ALTER FUNCTION subscriber_global_hot_intraday_selector_rows()
  OWNER TO signalops_subscriber_global_selector;
REVOKE ALL ON FUNCTION subscriber_global_hot_intraday_selector_rows() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION subscriber_global_hot_intraday_selector_rows()
  TO signalops_subscriber_global_eod;

CREATE OR REPLACE VIEW subscriber_global_hot_intraday_assets WITH (security_barrier = true) AS
SELECT global_asset_id, canonical_symbol, watcher_count, first_selected_at, last_selected_at
FROM subscriber_global_hot_intraday_selector_rows();

ALTER VIEW subscriber_global_hot_intraday_assets OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON subscriber_global_hot_intraday_assets FROM PUBLIC;
GRANT SELECT ON subscriber_global_hot_intraday_assets TO signalops_subscriber_global_eod;
