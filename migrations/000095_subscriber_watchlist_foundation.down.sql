-- S3 lists are additive. Production rollback must first disable the feature flag
-- and preserve tenant preference/audit evidence. This down migration is only
-- safe before any dependent S3+ tables or list records exist.

DROP TABLE IF EXISTS subscriber_watchlist_audit;
DROP TABLE IF EXISTS subscriber_watchlist_memberships;
DROP TABLE IF EXISTS subscriber_watchlists;

REVOKE REFERENCES (global_asset_id) ON subscriber_global_assets FROM signalops_subscriber_gateway;
