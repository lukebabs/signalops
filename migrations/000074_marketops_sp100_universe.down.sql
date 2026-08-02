DROP VIEW IF EXISTS marketops_universal_assets;
CREATE VIEW marketops_universal_assets AS
SELECT marketops_asset_universe.*,
  CASE universe_group
    WHEN 'top50_megacap' THEN 1
    WHEN 'analyst_watchlist' THEN 2
    ELSE 99
  END AS universe_priority
FROM marketops_asset_universe
WHERE universe_group IN ('top50_megacap', 'analyst_watchlist');

DELETE FROM marketops_asset_universe
WHERE tenant_id='tenant-local' AND universe_group='sp100';
