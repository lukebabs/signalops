-- Live governed primary MarketOps universe for production EOD work.
-- The projection is evaluated at query time so list changes are picked up on
-- the next run without changing worker code.
CREATE OR REPLACE VIEW marketops_primary_assets
WITH (security_barrier = true) AS
SELECT DISTINCT ON (tenant_id, ticker)
  tenant_id, app_id, domain, use_case, source_id, universe_group, rank,
  ticker, ticker_key, company, company_key, asset_type, exchange, sector,
  sector_key, industry, industry_key, is_active, metadata, universe_priority
FROM marketops_universal_assets
WHERE is_active = true
ORDER BY tenant_id, ticker, universe_priority, rank NULLS LAST, universe_group;

ALTER VIEW marketops_primary_assets OWNER TO signalops;
REVOKE ALL ON marketops_primary_assets FROM PUBLIC;
GRANT SELECT ON marketops_primary_assets TO signalops_subscriber_migrator;
GRANT SELECT ON marketops_primary_assets TO signalops_subscriber_global_eod;
GRANT SELECT ON marketops_primary_assets TO signalops_subscriber_gateway_runtime;
COMMENT ON VIEW marketops_primary_assets IS
  'Live governed primary MarketOps universe; scheduled EOD and downstream analysis resolve symbols here.';
