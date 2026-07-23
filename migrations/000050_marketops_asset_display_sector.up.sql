ALTER TABLE marketops_asset_universe
  ADD COLUMN IF NOT EXISTS display_sector text NOT NULL DEFAULT '';

ALTER TABLE marketops_asset_universe
  ADD CONSTRAINT marketops_asset_universe_display_sector_length
  CHECK (char_length(display_sector) <= 48);
