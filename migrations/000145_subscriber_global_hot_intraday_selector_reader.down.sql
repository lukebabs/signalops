DROP VIEW IF EXISTS subscriber_global_hot_intraday_assets;
CREATE VIEW subscriber_global_hot_intraday_assets WITH (security_barrier = true) AS
WITH selected_lists AS (
  SELECT preference.tenant_id, preference.subject, list.list_id, preference.updated_at
  FROM subscriber_watchlist_context_preferences AS preference
  JOIN subscriber_watchlists AS list ON list.tenant_id = preference.tenant_id
    AND ((preference.selection_mode = $$list$$ AND preference.list_id = list.list_id) OR preference.selection_mode = $$all$$)
    AND (list.list_kind = $$tenant_default$$ OR (list.list_kind = $$private$$ AND list.owner_subject = preference.subject))
)
SELECT asset.global_asset_id, asset.canonical_symbol,
  count(DISTINCT selected.tenant_id || chr(31) || selected.subject)::bigint AS watcher_count,
  min(membership.added_at) AS first_selected_at, max(selected.updated_at) AS last_selected_at
FROM selected_lists AS selected
JOIN subscriber_watchlist_memberships AS membership ON membership.tenant_id = selected.tenant_id AND membership.list_id = selected.list_id
JOIN subscriber_global_assets AS asset ON asset.global_asset_id = membership.global_asset_id
WHERE asset.eligibility_status = $$eligible$$
GROUP BY asset.global_asset_id, asset.canonical_symbol;
ALTER VIEW subscriber_global_hot_intraday_assets OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON subscriber_global_hot_intraday_assets FROM PUBLIC;
GRANT SELECT ON subscriber_global_hot_intraday_assets TO signalops_subscriber_global_eod;
REVOKE EXECUTE ON FUNCTION subscriber_global_hot_intraday_selector_rows() FROM signalops_subscriber_global_eod;
DROP FUNCTION IF EXISTS subscriber_global_hot_intraday_selector_rows();
REVOKE SELECT ON subscriber_watchlist_context_preferences, subscriber_watchlists, subscriber_watchlist_memberships, subscriber_global_assets FROM signalops_subscriber_global_selector;
DROP ROLE IF EXISTS signalops_subscriber_global_selector;
