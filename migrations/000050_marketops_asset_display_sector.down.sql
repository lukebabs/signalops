ALTER TABLE marketops_asset_universe
  DROP CONSTRAINT IF EXISTS marketops_asset_universe_display_sector_length;

ALTER TABLE marketops_asset_universe
  DROP COLUMN IF EXISTS display_sector;
