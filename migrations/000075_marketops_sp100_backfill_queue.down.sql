ALTER TABLE marketops_asset_backfill_jobs
  DROP CONSTRAINT IF EXISTS marketops_asset_backfill_jobs_universe_group_check;

ALTER TABLE marketops_asset_backfill_jobs
  ADD CONSTRAINT marketops_asset_backfill_jobs_universe_group_check
  CHECK (universe_group = 'analyst_watchlist');
