DROP FUNCTION IF EXISTS subscriber_visible_watchlist_items(text, text);

DROP POLICY IF EXISTS subscriber_watchlist_memberships_projection_owner_read
  ON subscriber_watchlist_memberships;
DROP POLICY IF EXISTS subscriber_watchlists_projection_owner_read
  ON subscriber_watchlists;
